// SPDX-License-Identifier: GPL-3.0-or-later
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

// adapter-static in SPA mode (ADR-0002): the whole app is client-rendered and
// served by the Go backend. The build is written straight into the Go web
// package so `//go:embed all:assets` (internal/web/web.go) picks it up.
/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      // Build into a nested spa/ dir so the committed sentinel at the assets
      // root (internal/web/assets/.gitkeep) survives adapter-static wiping the
      // output dir. Go serves this subtree (see internal/web/web.go).
      pages: '../internal/web/assets/spa',
      assets: '../internal/web/assets/spa',
      fallback: 'index.html',
      precompress: false,
      strict: false
    })
  }
};

export default config;
