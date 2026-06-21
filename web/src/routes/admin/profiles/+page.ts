// SPDX-License-Identifier: GPL-3.0-or-later
import type { PageLoad } from './$types';
import { adminGet } from '$lib/api';
import type { Group, PermissionProfile } from '$lib/types';

export const load: PageLoad = async ({ fetch }) => {
  // Profiles always load. The group catalog is a live call into the target IdP,
  // so a provisioner/API-key failure must not blank the whole page — degrade to
  // an empty catalog and surface the error in the editor instead.
  const profiles = await adminGet<PermissionProfile[]>(fetch, '/api/admin/profiles');
  let groups: Group[] = [];
  let groupsError = '';
  try {
    groups = await adminGet<Group[]>(fetch, '/api/admin/groups');
  } catch (e) {
    groupsError = e instanceof Error ? e.message : '无法从目标 IdP 加载 group 目录。';
  }
  return { profiles: profiles ?? [], groups: groups ?? [], groupsError };
};
