// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '$lib/api';
import type { PermissionProfile } from '$lib/types';

export const load: PageLoad = async ({ fetch }) => {
  const profiles = await adminGet<PermissionProfile[]>(fetch, '/api/admin/profiles');
  return { profiles: profiles ?? [] };
};
