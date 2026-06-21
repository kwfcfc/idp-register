import type { AdminUser } from '$lib/types';
import { query } from './db';
import { audit } from './audit';

export async function claimForProvisioning(
  applicationId: string,
  profileId: string,
  note: string,
  actor: AdminUser
): Promise<{
  id: string;
  email: string;
  username: string;
  groups: string[];
}> {
  const result = await query<{
    id: string;
    email: string;
    username: string;
    groups: string[];
  }>(
    `WITH selected_profile AS (
       SELECT groups FROM permission_profiles WHERE id = $2
     )
     UPDATE applications a
        SET status = 'provisioning',
            approved_profile_id = $2,
            decision_note = $3,
            reviewed_at = now(),
            reviewed_by_sub = $4,
            reviewed_by_email = $5,
            provisioning_error = NULL,
            updated_at = now()
       FROM selected_profile p
      WHERE a.id = $1
        AND a.status IN ('pending', 'provisioning_failed')
     RETURNING a.id, a.email, a.username, p.groups`,
    [applicationId, profileId, note || null, actor.sub, actor.email]
  );

  const row = result.rows[0];
  if (!row) throw new Error('Application is no longer available for provisioning');
  await audit(actor, 'application.provision.start', 'application', applicationId, { profileId });
  return row;
}

export async function markApproved(
  applicationId: string,
  rauthyUserId: string,
  actor: AdminUser
): Promise<void> {
  await query(
    `UPDATE applications
        SET status = 'approved', rauthy_user_id = $2,
            provisioning_error = NULL, updated_at = now()
      WHERE id = $1`,
    [applicationId, rauthyUserId]
  );
  await audit(actor, 'application.approve', 'application', applicationId, { rauthyUserId });
}

export async function markProvisioningFailed(
  applicationId: string,
  message: string,
  actor: AdminUser,
  rauthyUserId?: string
): Promise<void> {
  await query(
    `UPDATE applications
        SET status = 'provisioning_failed', provisioning_error = $2,
            rauthy_user_id = COALESCE($3, rauthy_user_id), updated_at = now()
      WHERE id = $1`,
    [applicationId, message.slice(0, 4000), rauthyUserId || null]
  );
  await audit(actor, 'application.provision.fail', 'application', applicationId, {
    message: message.slice(0, 1000),
    rauthyUserId: rauthyUserId || null
  });
}

export async function decideApplication(input: {
  id: string;
  status: 'rejected' | 'needs_changes';
  note: string;
  actor: AdminUser;
}): Promise<boolean> {
  const result = await query(
    `UPDATE applications
        SET status = $2, decision_note = $3, reviewed_at = now(),
            reviewed_by_sub = $4, reviewed_by_email = $5, updated_at = now()
      WHERE id = $1 AND status IN ('pending', 'provisioning_failed')
      RETURNING id`,
    [input.id, input.status, input.note, input.actor.sub, input.actor.email]
  );
  if (result.rowCount) {
    await audit(input.actor, `application.${input.status}`, 'application', input.id, {
      note: input.note
    });
  }
  return Boolean(result.rowCount);
}
