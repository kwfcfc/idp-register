import { redirect, type Handle } from '@sveltejs/kit';
import { loadSession, sessionCookieName } from '$lib/server/auth';

export const handle: Handle = async ({ event, resolve }) => {
  event.locals.user = null;
  event.locals.sessionId = null;

  const token = event.cookies.get(sessionCookieName());
  if (token) {
    const session = await loadSession(token);
    if (session) {
      event.locals.user = session.user;
      event.locals.sessionId = session.id;
    } else {
      event.cookies.delete(sessionCookieName(), { path: '/' });
    }
  }

  if (event.url.pathname.startsWith('/admin') && !event.locals.user) {
    const next = `${event.url.pathname}${event.url.search}`;
    throw redirect(303, `/login?next=${encodeURIComponent(next)}`);
  }

  return resolve(event);
};
