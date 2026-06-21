// SPDX-License-Identifier: GPL-3.0-or-later
// Thin client for the Go JSON API. The admin surface is session-cookie gated and
// enforces an Origin-based CSRF check on mutations; same-origin fetch satisfies
// both (cookies + Origin are sent automatically).
import { redirect } from '@sveltejs/kit';
import { goto } from '$app/navigation';

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

async function toError(res: Response): Promise<ApiError> {
  let message = `request failed (${res.status})`;
  try {
    const body = (await res.json()) as { error?: string };
    if (body?.error) message = body.error;
  } catch {
    /* non-JSON body; keep the generic message */
  }
  return new ApiError(res.status, message);
}

type Fetch = typeof globalThis.fetch;

/** GET JSON. Throws ApiError on non-2xx. */
export async function apiGet<T>(fetch: Fetch, path: string): Promise<T> {
  const res = await fetch(path, { headers: { accept: 'application/json' } });
  if (!res.ok) throw await toError(res);
  return (await res.json()) as T;
}

/**
 * GET JSON inside an admin loader: an expired/missing session (401) becomes a
 * redirect to the login page instead of an error.
 */
export async function adminGet<T>(fetch: Fetch, path: string): Promise<T> {
  try {
    return await apiGet<T>(fetch, path);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) throw redirect(307, '/login');
    throw e;
  }
}

/** Send a mutating request (POST/PUT/DELETE) with an optional JSON body. */
export async function apiSend<T = unknown>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body !== undefined ? { 'content-type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined
  });
  if (res.status === 401) {
    // Session expired during an action: bounce to login rather than swallow it.
    await goto('/login');
    throw new ApiError(401, 'session expired');
  }
  if (!res.ok) throw await toError(res);
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
