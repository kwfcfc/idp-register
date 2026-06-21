import { fail, redirect } from '@sveltejs/kit';
import { z } from 'zod';
import type { Actions, PageServerLoad } from './$types';
import { query } from '$lib/server/db';
import { createInvite, revokeInvite } from '$lib/server/invites';

const createSchema = z.object({
  email: z.string().trim().email().or(z.literal('')),
  profileId: z.string().min(1).max(64),
  expiresInDays: z.coerce.number().int().min(1).max(365),
  maxUses: z.coerce.number().int().min(1).max(100)
});

export const load: PageServerLoad = async () => {
  const [invites, profiles] = await Promise.all([
    query<{
      id: string; code_prefix: string; email_constraint: string | null; profile_id: string;
      profile_label: string; expires_at: Date; max_uses: number; used_count: number;
      raw_status: string; computed_status: string; created_at: Date; created_by_email: string;
    }>(`
      SELECT i.*, p.label AS profile_label,
        CASE
          WHEN i.status = 'revoked' THEN 'revoked'
          WHEN i.expires_at <= now() THEN 'expired'
          WHEN i.used_count >= i.max_uses THEN 'exhausted'
          ELSE 'active'
        END AS computed_status,
        i.status AS raw_status
      FROM invites i
      JOIN permission_profiles p ON p.id = i.profile_id
      ORDER BY i.created_at DESC
      LIMIT 200
    `),
    query<{ id: string; label: string; description: string; groups: string[] }>(
      'SELECT id, label, description, groups FROM permission_profiles ORDER BY label'
    )
  ]);
  return { invites: invites.rows, profiles: profiles.rows };
};

export const actions: Actions = {
  create: async ({ request, locals }) => {
    const parsed = createSchema.safeParse(Object.fromEntries(await request.formData()));
    if (!parsed.success) return fail(400, { createError: '请检查邮箱、有效期和使用次数。' });

    const profile = await query('SELECT 1 FROM permission_profiles WHERE id = $1', [parsed.data.profileId]);
    if (!profile.rowCount) return fail(400, { createError: '权限模板不存在。' });

    const invite = await createInvite({
      emailConstraint: parsed.data.email || undefined,
      profileId: parsed.data.profileId,
      expiresAt: new Date(Date.now() + parsed.data.expiresInDays * 86_400_000),
      maxUses: parsed.data.maxUses,
      actor: locals.user!
    });

    return { createdCode: invite.code, createdId: invite.id };
  },
  revoke: async ({ request, locals }) => {
    const id = String((await request.formData()).get('id') || '');
    if (!id) return fail(400, { revokeError: '缺少邀请码 ID。' });
    await revokeInvite(id, locals.user!);
    throw redirect(303, '/admin/invites');
  }
};
