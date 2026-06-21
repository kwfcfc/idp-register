// SPDX-License-Identifier: GPL-3.0-or-later
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// In dev, the SvelteKit server runs on :5173 while the Go backend serves the
// API/auth on :8080. Proxy the backend paths so cookies stay same-origin and
// the Origin-based CSRF check in internal/web passes.
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
