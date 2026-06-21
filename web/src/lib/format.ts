// SPDX-License-Identifier: GPL-3.0-or-later
const dt = new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' });

/** Format an epoch-millisecond timestamp; null/undefined render as an em dash. */
export function fmtDate(ms: number | null | undefined): string {
  if (ms === null || ms === undefined) return '—';
  return dt.format(new Date(ms));
}
