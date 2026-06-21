import type { AdminUser } from '$lib/types';
import { query } from './db';

export async function audit(
  actor: AdminUser,
  action: string,
  targetType: string,
  targetId: string,
  details: Record<string, unknown> = {}
): Promise<void> {
  await query(
    `INSERT INTO audit_log (actor_sub, actor_email, action, target_type, target_id, details)
     VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
    [actor.sub, actor.email, action, targetType, targetId, JSON.stringify(details)]
  );
}
