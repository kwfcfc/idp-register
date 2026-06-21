// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '$lib/api';
import type { AuditEntry } from '$lib/types';

export const load: PageLoad = async ({ fetch }) => {
  const entries = await adminGet<AuditEntry[]>(fetch, '/api/admin/audit');
  return { entries: entries ?? [] };
};
