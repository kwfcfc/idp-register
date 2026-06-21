import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = ({ locals, url }) => {
  if (locals.user) throw redirect(303, '/admin');
  return { next: url.searchParams.get('next') || '/admin' };
};
