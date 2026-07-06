// SPDX-License-Identifier: GPL-3.0-or-later
// Lint + test workflow: frontend check/build, then the Go suite against both
// database engines (SQLite in-process, PostgreSQL via the service container).
local lib = import 'lib.libsonnet';

{
  labels: lib.labels,
  when: [
    { event: ['push', 'pull_request', 'tag', 'manual'] },
  ],

  services: [
    {
      name: 'postgres',
      image: lib.postgresImage,
      environment: {
        POSTGRES_PASSWORD: 'test',
        POSTGRES_DB: 'idp_register_test',
      },
    },
  ],

  steps: [
    {
      name: 'frontend',
      image: lib.nodeImage,
      commands: [
        'corepack enable',
        'pnpm install --filter idp-register-web... --frozen-lockfile',
        'pnpm --filter idp-register-web check',
        // Builds into internal/web/assets/spa so the Go steps compile the
        // same embedded tree a release image would.
        'pnpm --filter idp-register-web build',
      ],
    },
    {
      name: 'go',
      image: lib.goImage,
      environment: {
        // Hostname = service name. The store suite runs every test on SQLite
        // and, because this is set, on PostgreSQL too (see forEachStore).
        TEST_POSTGRES_DSN: 'postgres://postgres:test@postgres:5432/idp_register_test?sslmode=disable',
      },
      commands: [
        'test -z "$(gofmt -l cmd internal)" || { gofmt -l cmd internal; exit 1; }',
        'go vet ./...',
        // The service container starts asynchronously; wait before the store
        // suite dials it (busybox nc/timeout ship with alpine).
        'timeout 60 sh -c \'until nc -z postgres 5432; do sleep 1; done\'',
        'go test ./...',
        'go build -trimpath ./cmd/server',
      ],
    },
  ],
}
