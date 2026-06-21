// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '$lib/api';
import type { Application, PermissionProfile } from '$lib/types';

export const load: PageLoad = async ({ fetch, params }) => {
  const [application, profiles] = await Promise.all([
    adminGet<Application>(fetch, `/api/admin/applications/${encodeURIComponent(params.id)}`),
    adminGet<PermissionProfile[]>(fetch, '/api/admin/profiles')
  ]);
  return { application, profiles: profiles ?? [] };
};
