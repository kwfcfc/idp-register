CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS permission_profiles (
  id TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  groups TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS invites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code_digest TEXT NOT NULL UNIQUE,
  code_prefix TEXT NOT NULL,
  email_constraint TEXT,
  profile_id TEXT NOT NULL REFERENCES permission_profiles(id),
  expires_at TIMESTAMPTZ NOT NULL,
  max_uses INTEGER NOT NULL DEFAULT 1 CHECK (max_uses > 0),
  used_count INTEGER NOT NULL DEFAULT 0 CHECK (used_count >= 0),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
  created_by_sub TEXT NOT NULL,
  created_by_email TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ,
  revoked_by_sub TEXT
);

CREATE INDEX IF NOT EXISTS invites_status_expires_idx ON invites(status, expires_at);
CREATE INDEX IF NOT EXISTS invites_email_idx ON invites(lower(email_constraint));

CREATE TABLE IF NOT EXISTS applications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  invite_id UUID REFERENCES invites(id),
  email TEXT NOT NULL,
  email_normalized TEXT NOT NULL,
  username TEXT NOT NULL,
  username_normalized TEXT NOT NULL,
  review_text TEXT NOT NULL,
  requested_services TEXT[] NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'pending' CHECK (
    status IN ('pending', 'provisioning', 'provisioning_failed', 'approved', 'rejected', 'needs_changes')
  ),
  email_verified_at TIMESTAMPTZ,
  captcha_provider TEXT,
  captcha_verified_at TIMESTAMPTZ,
  submitted_ip INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  reviewed_at TIMESTAMPTZ,
  reviewed_by_sub TEXT,
  reviewed_by_email TEXT,
  decision_note TEXT,
  approved_profile_id TEXT REFERENCES permission_profiles(id),
  rauthy_user_id TEXT,
  provisioning_error TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS active_application_email_unique
  ON applications(email_normalized)
  WHERE status IN ('pending', 'provisioning', 'provisioning_failed', 'approved', 'needs_changes');

CREATE UNIQUE INDEX IF NOT EXISTS active_application_username_unique
  ON applications(username_normalized)
  WHERE status IN ('pending', 'provisioning', 'provisioning_failed', 'approved', 'needs_changes');

CREATE INDEX IF NOT EXISTS applications_status_created_idx ON applications(status, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token_digest TEXT NOT NULL UNIQUE,
  subject TEXT NOT NULL,
  email TEXT NOT NULL,
  display_name TEXT NOT NULL,
  groups JSONB NOT NULL DEFAULT '[]'::jsonb,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS admin_sessions_expires_idx ON admin_sessions(expires_at);

CREATE TABLE IF NOT EXISTS audit_log (
  id BIGSERIAL PRIMARY KEY,
  actor_sub TEXT NOT NULL,
  actor_email TEXT NOT NULL,
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO permission_profiles (id, label, description, groups)
VALUES
  ('basic', '基础成员', 'Matrix 与 GoToSocial', ARRAY['svc:matrix:user', 'svc:gotosocial:user']),
  ('developer', '开发者', '基础成员权限，加 Forgejo', ARRAY['svc:matrix:user', 'svc:gotosocial:user', 'svc:forgejo:user']),
  ('community-admin', '社区管理员', '社区服务管理权限，不包含基础设施管理', ARRAY['svc:matrix:user', 'svc:gotosocial:admin', 'svc:forgejo:user'])
ON CONFLICT (id) DO NOTHING;
