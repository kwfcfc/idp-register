import type { PageServerLoad } from './$types';
import { query } from '$lib/server/db';

export const load: PageServerLoad = async () => {
  const [stats, recent] = await Promise.all([
    query<{
      pending: string;
      approved_30d: string;
      active_invites: string;
      failed: string;
    }>(`
      SELECT
        count(*) FILTER (WHERE status = 'pending')::text AS pending,
        count(*) FILTER (WHERE status = 'approved' AND reviewed_at > now() - interval '30 days')::text AS approved_30d,
        (SELECT count(*)::text FROM invites WHERE status = 'active' AND expires_at > now() AND used_count < max_uses) AS active_invites,
        count(*) FILTER (WHERE status = 'provisioning_failed')::text AS failed
      FROM applications
    `),
    query<{
      id: string; email: string; username: string; status: string; created_at: Date; requested_services: string[];
    }>(`
      SELECT id, email, username, status, created_at, requested_services
      FROM applications
      ORDER BY created_at DESC
      LIMIT 8
    `)
  ]);

  const row = stats.rows[0];
  return {
    stats: {
      pending: Number(row.pending),
      approved30d: Number(row.approved_30d),
      activeInvites: Number(row.active_invites),
      failed: Number(row.failed)
    },
    recent: recent.rows
  };
};
