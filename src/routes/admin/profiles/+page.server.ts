import { fail, redirect } from '@sveltejs/kit';
import { z } from 'zod';
import type { Actions, PageServerLoad } from './$types';
import { query } from '$lib/server/db';
import { audit } from '$lib/server/audit';

export const load: PageServerLoad = async () => ({
  profiles: (await query<{ id: string; label: string; description: string; groups: string[] }>(
    'SELECT id, label, description, groups FROM permission_profiles ORDER BY label'
  )).rows
});

const schema = z.object({
  id: z.string().regex(/^[a-z0-9][a-z0-9-]{1,47}$/),
  label: z.string().trim().min(2).max(80),
  description: z.string().trim().max(300),
  groups: z.string().transform((value) => value.split(/\r?\n|,/).map((v) => v.trim()).filter(Boolean))
});

export const actions: Actions = {
  upsert: async ({ request, locals }) => {
    const parsed = schema.safeParse(Object.fromEntries(await request.formData()));
    if (!parsed.success || parsed.data.groups.length === 0) return fail(400, { error: '请填写合法 ID、名称和至少一个 Rauthy group。' });
    await query(`
      INSERT INTO permission_profiles (id, label, description, groups)
      VALUES ($1, $2, $3, $4)
      ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label, description = EXCLUDED.description,
        groups = EXCLUDED.groups, updated_at = now()
    `, [parsed.data.id, parsed.data.label, parsed.data.description, parsed.data.groups]);
    await audit(locals.user!, 'profile.upsert', 'permission_profile', parsed.data.id, { groups: parsed.data.groups });
    throw redirect(303, '/admin/profiles');
  }
};
