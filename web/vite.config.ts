// SPDX-License-Identifier: GPL-3.0-or-later
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

// Two-process dev: `vite dev` (:5173) serves the SPA with HMR while the Go
// backend serves the API/auth on a separate port. Proxy the backend paths
// through Vite so cookies stay same-origin and the Origin-based CSRF check in
// internal/web passes. Point at the backend with BACKEND_ORIGIN
// (e.g. BACKEND_ORIGIN=http://localhost:8081 for the deploy/dev harness).
//
// NOTE: only `vite dev` proxies. `vite preview` does NOT forward /api,/auth —
// SvelteKit's preview middleware intercepts first. To exercise the production
// build, use the embedded single-origin mode (build the SPA, let Go serve it).
const backend = process.env.BACKEND_ORIGIN ?? 'http://localhost:8080';

export default defineConfig({
  plugins: [
    // SvelteKit 3 takes its project config here (svelte.config.js is gone).
    // Non-Kit options such as `preprocess` pass through to vite-plugin-svelte.
    sveltekit({
      preprocess: vitePreprocess(),
      // Kit 3 polls _app/version.json hourly by default; nothing here reads
      // `updated`, so keep Kit 2's no-polling behaviour. The focus-triggered
      // check and the full-reload fallback on a failed navigation still apply.
      version: { pollInterval: 0 },
      // adapter-static in SPA mode (ADR-0002): the whole app is client-rendered
      // and served by the Go backend. The build is written straight into the Go
      // web package so `//go:embed all:assets` (internal/web/web.go) picks it up.
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
    })
  ],
  server: {
    proxy: {
      '/api': { target: backend, changeOrigin: false },
      '/auth': { target: backend, changeOrigin: false },
      '/healthz': { target: backend, changeOrigin: false }
    }
  }
});
