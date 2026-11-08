// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '#lib/api.js';
import type { RegistrationToken, PermissionProfile } from '#lib/types.js';

export const load: PageLoad = async ({ fetch }) => {
  const [tokens, profiles] = await Promise.all([
    adminGet<RegistrationToken[]>(fetch, '/api/admin/tokens'),
    adminGet<PermissionProfile[]>(fetch, '/api/admin/profiles')
  ]);
  return { tokens: tokens ?? [], profiles: profiles ?? [] };
};
