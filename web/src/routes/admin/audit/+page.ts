// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '#lib/api.js';
import type { AuditEntry } from '#lib/types.js';

export const load: PageLoad = async ({ fetch }) => {
  const entries = await adminGet<AuditEntry[]>(fetch, '/api/admin/audit');
  return { entries: entries ?? [] };
};
