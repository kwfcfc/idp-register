// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { apiGet } from '$lib/api';
import type { FormConfig } from '$lib/types';

// Load the anonymous form configuration (the public service options). A failure
// here must not block registration: fall back to an empty option list so the
// form still submits (services are advisory — ADR-0012).
export const load: PageLoad = async ({ fetch }) => {
  try {
    const form = await apiGet<FormConfig>(fetch, '/api/form');
    return { services: form.services ?? [], selectionMode: form.selectionMode };
  } catch {
    return { services: [], selectionMode: 'single' as const };
  }
};
