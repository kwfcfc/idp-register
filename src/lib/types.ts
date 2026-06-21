export type AdminUser = {
  sub: string;
  email: string;
  displayName: string;
  groups: string[];
};

export type ApplicationStatus =
  | 'pending'
  | 'provisioning'
  | 'provisioning_failed'
  | 'approved'
  | 'rejected'
  | 'needs_changes';

export type InviteStatus = 'active' | 'revoked' | 'exhausted' | 'expired';

export type PermissionProfile = {
  id: string;
  label: string;
  description: string;
  groups: string[];
};
