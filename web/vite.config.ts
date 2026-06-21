// SPDX-License-Identifier: GPL-3.0-or-later
import { sveltekit } from '@sveltejs/kit/vite';
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
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/api': { target: backend, changeOrigin: false },
      '/auth': { target: backend, changeOrigin: false },
      '/healthz': { target: backend, changeOrigin: false }
    }
  }
});
