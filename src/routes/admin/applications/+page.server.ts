import type { PageServerLoad } from './$types';
import { query } from '$lib/server/db';

const allowedStatuses = new Set(['pending', 'provisioning', 'provisioning_failed', 'approved', 'rejected', 'needs_changes']);

export const load: PageServerLoad = async ({ url }) => {
  const status = url.searchParams.get('status') || 'pending';
  const selected = allowedStatuses.has(status) ? status : 'pending';
  const result = await query<{
    id: string; email: string; username: string; review_text: string; requested_services: string[];
    status: string; created_at: Date; email_verified_at: Date | null; invite_id: string | null;
  }>(`
    SELECT id, email, username, review_text, requested_services, status, created_at,
           email_verified_at, invite_id
    FROM applications
    WHERE status = $1
    ORDER BY created_at ASC
    LIMIT 250
  `, [selected]);
  return { applications: result.rows, status: selected };
};
