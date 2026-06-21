import { createHmac, randomBytes } from 'node:crypto';
import type { AdminUser } from '$lib/types';
import { query } from './db';
import { config } from './env';
import { audit } from './audit';

function digestInvite(code: string): string {
  return createHmac('sha256', config.inviteHmacKey).update(code).digest('hex');
}

export async function createInvite(input: {
  emailConstraint?: string;
  profileId: string;
  expiresAt: Date;
  maxUses: number;
  actor: AdminUser;
}): Promise<{ id: string; code: string }> {
  const code = randomBytes(24).toString('base64url');
  const result = await query<{ id: string }>(
    `INSERT INTO invites (
       code_digest, code_prefix, email_constraint, profile_id, expires_at,
       max_uses, created_by_sub, created_by_email
     ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
     RETURNING id`,
    [
      digestInvite(code),
      code.slice(0, 8),
      input.emailConstraint?.trim().toLowerCase() || null,
      input.profileId,
      input.expiresAt,
      input.maxUses,
      input.actor.sub,
      input.actor.email
    ]
  );
  const id = result.rows[0].id;
  await audit(input.actor, 'invite.create', 'invite', id, {
    profileId: input.profileId,
    emailConstraint: input.emailConstraint || null,
    expiresAt: input.expiresAt.toISOString(),
    maxUses: input.maxUses
  });
  return { id, code };
}

export async function revokeInvite(id: string, actor: AdminUser): Promise<boolean> {
  const result = await query(
    `UPDATE invites
       SET status = 'revoked', revoked_at = now(), revoked_by_sub = $2
     WHERE id = $1 AND status = 'active'
     RETURNING id`,
    [id, actor.sub]
  );
  if (result.rowCount) await audit(actor, 'invite.revoke', 'invite', id);
  return Boolean(result.rowCount);
}
