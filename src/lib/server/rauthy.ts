import { config } from './env';

export type RauthyUser = {
  id: string;
  email: string;
  groups?: string[];
  roles: string[];
};

function authorization(): string {
  return `API-Key ${config.rauthyApiKeyName}$${config.rauthyApiKeySecret}`;
}

async function request<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(`${config.rauthyApiBase}${path}`, {
    ...init,
    headers: {
      accept: 'application/json',
      'content-type': 'application/json',
      authorization: authorization(),
      ...init.headers
    },
    signal: AbortSignal.timeout(12_000)
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`Rauthy ${init.method || 'GET'} ${path} failed (${response.status}): ${body.slice(0, 800)}`);
  }

  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

export async function createRauthyUser(input: {
  email: string;
  groups: string[];
}): Promise<RauthyUser> {
  return request<RauthyUser>('/users', {
    method: 'POST',
    body: JSON.stringify({
      email: input.email,
      family_name: null,
      given_name: null,
      language: config.rauthyLanguage,
      groups: input.groups,
      roles: [],
      user_expires: null,
      tz: config.rauthyTimezone
    })
  });
}

export async function setPreferredUsername(userId: string, username: string): Promise<void> {
  await request<void>(`/users/${encodeURIComponent(userId)}/self/preferred_username`, {
    method: 'PUT',
    body: JSON.stringify({
      preferred_username: username,
      force_overwrite: false
    })
  });
}

export async function findRauthyUserByEmail(email: string): Promise<RauthyUser | null> {
  const response = await fetch(
    `${config.rauthyApiBase}/users/email/${encodeURIComponent(email)}`,
    {
      headers: { accept: 'application/json', authorization: authorization() },
      signal: AbortSignal.timeout(8_000)
    }
  );
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`Rauthy user lookup failed (${response.status})`);
  return (await response.json()) as RauthyUser;
}
