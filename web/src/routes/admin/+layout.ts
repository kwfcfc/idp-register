// SPDX-License-Identifier: GPL-3.0-or-later
import type { LayoutLoad } from './$types';
import { adminGet } from '$lib/api';
import type { AdminUser } from '$lib/types';

// Gate the whole /admin subtree: adminGet redirects to /login on 401.
export const load: LayoutLoad = async ({ fetch }) => {
  const user = await adminGet<AdminUser>(fetch, '/api/admin/me');
  return { user };
};
