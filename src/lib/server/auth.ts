import { createHash, randomBytes } from 'node:crypto';
import type { Cookies } from '@sveltejs/kit';
import type { AdminUser } from '$lib/types';
import { query } from './db';
import { config } from './env';

const SESSION_COOKIE = 'invite_admin_session';

function digest(value: string): string {
  return createHash('sha256').update(value).digest('hex');
}

export async function createSession(cookies: Cookies, user: AdminUser): Promise<void> {
  const token = randomBytes(32).toString('base64url');
  const expiresAt = new Date(Date.now() + config.sessionTtlHours * 60 * 60 * 1000);

  await query(
    `INSERT INTO admin_sessions (token_digest, subject, email, display_name, groups, expires_at)
     VALUES ($1, $2, $3, $4, $5::jsonb, $6)`,
    [digest(token), user.sub, user.email, user.displayName, JSON.stringify(user.groups), expiresAt]
  );

  cookies.set(SESSION_COOKIE, token, {
    path: '/',
    httpOnly: true,
    secure: config.secureCookies,
    sameSite: 'lax',
    expires: expiresAt
  });
}

export async function loadSession(token: string): Promise<{ id: string; user: AdminUser } | null> {
  const result = await query<{
    id: string;
    subject: string;
    email: string;
    display_name: string;
    groups: string[];
  }>(
    `UPDATE admin_sessions
       SET last_seen_at = now()
     WHERE token_digest = $1 AND expires_at > now()
     RETURNING id, subject, email, display_name, groups`,
    [digest(token)]
  );

  const row = result.rows[0];
  if (!row) return null;

  return {
    id: row.id,
    user: {
      sub: row.subject,
      email: row.email,
      displayName: row.display_name,
      groups: Array.isArray(row.groups) ? row.groups : []
    }
  };
}

export async function destroySession(cookies: Cookies, token?: string): Promise<void> {
  const current = token || cookies.get(SESSION_COOKIE);
  if (current) await query('DELETE FROM admin_sessions WHERE token_digest = $1', [digest(current)]);
  cookies.delete(SESSION_COOKIE, { path: '/' });
}

export function sessionCookieName(): string {
  return SESSION_COOKIE;
}
