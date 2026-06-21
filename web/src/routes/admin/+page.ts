// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '$lib/api';
import type { Application, RegistrationToken } from '$lib/types';

// No server-side aggregate endpoint exists; derive the overview from the list
// endpoints (applications come back newest-first).
export const load: PageLoad = async ({ fetch }) => {
  const [apps, validTokens] = await Promise.all([
    adminGet<Application[]>(fetch, '/api/admin/applications'),
    adminGet<RegistrationToken[]>(fetch, '/api/admin/tokens?valid=true')
  ]);
  const all = apps ?? [];
  const count = (s: string) => all.filter((a) => a.status === s).length;
  return {
    stats: {
      pending: count('pending'),
      approved: count('approved'),
      activeInvites: (validTokens ?? []).length,
      failed: count('provisioning_failed')
    },
    recent: all.slice(0, 8)
  };
};
