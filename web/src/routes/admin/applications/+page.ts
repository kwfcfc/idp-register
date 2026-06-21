// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '$lib/api';
import type { Application, ApplicationStatus } from '$lib/types';

const allowed = new Set<ApplicationStatus>([
  'pending',
  'provisioning',
  'provisioning_failed',
  'approved',
  'rejected'
]);

export const load: PageLoad = async ({ fetch, url }) => {
  const raw = (url.searchParams.get('status') ?? 'pending') as ApplicationStatus;
  const status = allowed.has(raw) ? raw : 'pending';
  const applications = await adminGet<Application[]>(
    fetch,
    `/api/admin/applications?status=${encodeURIComponent(status)}`
  );
  return { applications: applications ?? [], status };
};
