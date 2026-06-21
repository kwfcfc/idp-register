// SPDX-License-Identifier: GPL-3.0-or-later
// Pure SPA: everything is client-rendered and the build is a static fallback
// (adapter-static). No prerendering — pages fetch the Go API at runtime.
export const ssr = false;
export const prerender = false;
