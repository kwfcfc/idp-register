import { error, fail, redirect } from '@sveltejs/kit';
import { z } from 'zod';
import type { Actions, PageServerLoad } from './$types';
import { query } from '$lib/server/db';
import { claimForProvisioning, decideApplication, markApproved, markProvisioningFailed } from '$lib/server/applications';
import { createRauthyUser, setPreferredUsername } from '$lib/server/rauthy';

export const load: PageServerLoad = async ({ params }) => {
  const [application, profiles] = await Promise.all([
    query<{
      id: string; email: string; username: string; review_text: string; requested_services: string[];
      status: string; created_at: Date; updated_at: Date; email_verified_at: Date | null;
      captcha_provider: string | null; captcha_verified_at: Date | null; submitted_ip: string | null;
      decision_note: string | null; provisioning_error: string | null; rauthy_user_id: string | null;
      invite_id: string | null; invite_prefix: string | null; invite_profile_label: string | null;
    }>(`
      SELECT a.*, i.code_prefix AS invite_prefix, p.label AS invite_profile_label
      FROM applications a
      LEFT JOIN invites i ON i.id = a.invite_id
      LEFT JOIN permission_profiles p ON p.id = i.profile_id
      WHERE a.id = $1
    `, [params.id]),
    query<{ id: string; label: string; description: string; groups: string[] }>(
      'SELECT id, label, description, groups FROM permission_profiles ORDER BY label'
    )
  ]);
  if (!application.rows[0]) throw error(404, 'Application not found');
  return { application: application.rows[0], profiles: profiles.rows };
};

const decisionSchema = z.object({ note: z.string().trim().min(3).max(3000) });
const approveSchema = decisionSchema.extend({ profileId: z.string().min(1).max(64) });

export const actions: Actions = {
  approve: async ({ request, params, locals }) => {
    const parsed = approveSchema.safeParse(Object.fromEntries(await request.formData()));
    if (!parsed.success) return fail(400, { error: '请选择权限模板，并填写至少 3 个字符的审核备注。' });

    let claimed: Awaited<ReturnType<typeof claimForProvisioning>>;
    try {
      claimed = await claimForProvisioning(params.id, parsed.data.profileId, parsed.data.note, locals.user!);
    } catch (cause) {
      return fail(409, { error: cause instanceof Error ? cause.message : '申请状态已发生变化。' });
    }

    let userId: string | undefined;
    try {
      const user = await createRauthyUser({ email: claimed.email, groups: claimed.groups });
      userId = user.id;
      await setPreferredUsername(user.id, claimed.username);
      await markApproved(params.id, user.id, locals.user!);
    } catch (cause) {
      const message = cause instanceof Error ? cause.message : 'Unknown Rauthy provisioning error';
      await markProvisioningFailed(params.id, message, locals.user!, userId);
      return fail(502, { error: `Rauthy 创建失败：${message}` });
    }

    throw redirect(303, `/admin/applications/${params.id}?approved=1`);
  },
  reject: async ({ request, params, locals }) => {
    const parsed = decisionSchema.safeParse(Object.fromEntries(await request.formData()));
    if (!parsed.success) return fail(400, { error: '请填写拒绝原因。' });
    if (!await decideApplication({ id: params.id, status: 'rejected', note: parsed.data.note, actor: locals.user! })) {
      return fail(409, { error: '该申请当前无法拒绝。' });
    }
    throw redirect(303, `/admin/applications/${params.id}`);
  },
  changes: async ({ request, params, locals }) => {
    const parsed = decisionSchema.safeParse(Object.fromEntries(await request.formData()));
    if (!parsed.success) return fail(400, { error: '请说明申请人需要补充什么。' });
    if (!await decideApplication({ id: params.id, status: 'needs_changes', note: parsed.data.note, actor: locals.user! })) {
      return fail(409, { error: '该申请当前无法标记为需补充。' });
    }
    throw redirect(303, `/admin/applications/${params.id}`);
  }
};
