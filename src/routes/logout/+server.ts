import { redirect, type RequestHandler } from '@sveltejs/kit';
import { destroySession } from '$lib/server/auth';

export const POST: RequestHandler = async ({ cookies }) => {
  await destroySession(cookies);
  throw redirect(303, '/login');
};
