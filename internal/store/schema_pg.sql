-- SPDX-License-Identifier: GPL-3.0-or-later
-- PostgreSQL schema. Mirrors schema_sqlite.sql exactly in shape and semantics.
-- Differences are types only: epoch-ms timestamps are BIGINT (exceed int32),
-- booleans stay INTEGER 0/1 for cross-engine parity, lists stay JSON-in-TEXT.
-- We deliberately avoid TIMESTAMPTZ / JSONB / TEXT[] / BIGSERIAL / gen_random_uuid()
-- so the same Go code drives both engines.

CREATE TABLE IF NOT EXISTS permission_profiles (
  id                TEXT PRIMARY KEY,
  label             TEXT NOT NULL,
  description       TEXT NOT NULL DEFAULT '',
  groups            TEXT NOT NULL DEFAULT '[]',
  public_selectable INTEGER NOT NULL DEFAULT 0,
  public_label      TEXT NOT NULL DEFAULT '',
  sort_order        INTEGER NOT NULL DEFAULT 0,
  created_at        BIGINT NOT NULL,
  updated_at        BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS registration_tokens (
  id               TEXT PRIMARY KEY,
  token            TEXT NOT NULL UNIQUE,
  uses_allowed     INTEGER,
  pending          INTEGER NOT NULL DEFAULT 0,
  completed        INTEGER NOT NULL DEFAULT 0,
  expiry_time      BIGINT,
  active           INTEGER NOT NULL DEFAULT 1,
  email_constraint TEXT,
  profile_id       TEXT NOT NULL REFERENCES permission_profiles(id),
  note             TEXT NOT NULL DEFAULT '',
  created_by_sub   TEXT NOT NULL,
  created_by_email TEXT NOT NULL,
  created_at       BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS registration_tokens_active_idx
  ON registration_tokens(active, expiry_time);

CREATE TABLE IF NOT EXISTS applications (
  id                  TEXT PRIMARY KEY,
  token_id            TEXT REFERENCES registration_tokens(id),
  email               TEXT NOT NULL,
  email_normalized    TEXT NOT NULL,
  username            TEXT NOT NULL DEFAULT '',
  username_normalized TEXT NOT NULL DEFAULT '',
  review_text         TEXT NOT NULL DEFAULT '',
  requested_services  TEXT NOT NULL DEFAULT '[]',
  status              TEXT NOT NULL DEFAULT 'pending',
  captcha_provider    TEXT,
  captcha_verified_at BIGINT,
  submitted_ip        TEXT,
  approved_profile_id TEXT REFERENCES permission_profiles(id),
  provider_user_id    TEXT,
  provisioning_error  TEXT,
  decision_note       TEXT,
  reviewed_at         BIGINT,
  reviewed_by_sub     TEXT,
  reviewed_by_email   TEXT,
  created_at          BIGINT NOT NULL,
  updated_at          BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS applications_active_email_unique
  ON applications(email_normalized)
  WHERE status IN ('pending', 'provisioning', 'provisioning_failed', 'approved', 'needs_changes');

CREATE UNIQUE INDEX IF NOT EXISTS applications_active_username_unique
  ON applications(username_normalized)
  WHERE username_normalized <> ''
    AND status IN ('pending', 'provisioning', 'provisioning_failed', 'approved', 'needs_changes');

CREATE INDEX IF NOT EXISTS applications_status_created_idx
  ON applications(status, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_sessions (
  id           TEXT PRIMARY KEY,
  token_digest TEXT NOT NULL UNIQUE,
  subject      TEXT NOT NULL,
  email        TEXT NOT NULL,
  display_name TEXT NOT NULL,
  groups       TEXT NOT NULL DEFAULT '[]',
  expires_at   BIGINT NOT NULL,
  created_at   BIGINT NOT NULL,
  last_seen_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS admin_sessions_expires_idx ON admin_sessions(expires_at);

CREATE TABLE IF NOT EXISTS audit_log (
  id          TEXT PRIMARY KEY,
  actor_sub   TEXT NOT NULL,
  actor_email TEXT NOT NULL,
  action      TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id   TEXT NOT NULL,
  details     TEXT NOT NULL DEFAULT '{}',
  created_at  BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS audit_log_created_idx ON audit_log(created_at DESC);
