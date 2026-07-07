// SPDX-License-Identifier: GPL-3.0-or-later
// Manual-only end-to-end smoke: boot a real Rauthy, run the provisioner
// integration test against it, then drive the *published* idp-register image
// through the public register flow. Triggered by hand to validate important
// releases end-to-end.
//
// Crow-native (no docker-in-docker): Rauthy and the app run as Crow-managed
// containers on the agent, reachable by their step/service name as hostname,
// with the cloned repo mounted at $CI_WORKSPACE in every step.
//
// Ordering is a DAG (depends_on), not the default serial run, because two
// startups are fatal-on-missing-dependency and must be gated on readiness:
//   rauthy(detach) -> await-rauthy -> idp-register(detach) -> smoke
// The app aborts at boot if OIDC discovery against Rauthy fails
// (cmd/server: log.Fatal), so it may only start once Rauthy answers.
local lib = import 'lib.libsonnet';

// The image under test: the release already pushed by image.jsonnet. Pin to a
// specific tag (e.g. lib.imageRepo + ':v1.2.3') to validate one release.
local appImage = lib.imageRepo + ':latest';

// Shared test-only credentials (mirrors deploy/e2e): Rauthy bootstraps this API
// key, and both the app's provisioner and the Go integration test authenticate
// with the same name/secret. Kept as locals so the two sides cannot drift.
local rauthyApiBase = 'http://rauthy:8080/auth/v1';
local apiKeyName = 'idp-register';
local apiKeySecret = '75d205eb93bafde5982690d31d35c203d57a7b5c1273d3d9b78879d47b0ddf547da025085dbcd7e4';

// Rauthy is fully env-configured; deploy/e2e/config.toml is only a stub that
// exists because Rauthy 0.35 panics without a config file (see that file).
local rauthyEnv = {
  LISTEN_SCHEME: 'http',
  LISTEN_ADDRESS: '0.0.0.0',
  LISTEN_PORT_HTTP: '8080',
  PUB_URL: 'rauthy:8080',
  INSECURE_COOKIE: 'true',
  RP_ID: 'rauthy',
  RP_ORIGIN: 'http://rauthy:8080',

  HQL_NODE_ID: '1',
  HQL_NODES: '1 localhost:8100 localhost:8200',
  HQL_SECRET_RAFT: '1427209662dbb7ae5a6f1ba1699be92d',
  HQL_SECRET_API: 'bce56762c039d798b5be0269e3c0423f',

  ENC_KEYS: '5fdc10c5/wmGr3cB3Fnm1m12MF7HnJC8eBAsTGRQ0AAMkFgKzQgU=',
  ENC_KEY_ACTIVE: '5fdc10c5',

  BOOTSTRAP_ADMIN_EMAIL: 'admin@localhost',
  BOOTSTRAP_ADMIN_PASSWORD_PLAIN: 'TestAdmin1234!',
  // Provisioning API key ("idp-register"): Users read/create/update, Groups read.
  BOOTSTRAP_API_KEY: 'eyJuYW1lIjoiaWRwLXJlZ2lzdGVyIiwiZXhwIjoyMDAwMDAwMDAwLCJhY2Nlc3MiOlt7Imdyb3VwIjoiVXNlcnMiLCJhY2Nlc3NfcmlnaHRzIjpbInJlYWQiLCJjcmVhdGUiLCJ1cGRhdGUiXX0seyJncm91cCI6Ikdyb3VwcyIsImFjY2Vzc19yaWdodHMiOlsicmVhZCJdfV19',
  BOOTSTRAP_API_KEY_SECRET: apiKeySecret,

  SMTP_URL: 'mailcrab',
  SMTP_PORT: '1025',
  SMTP_DANGER_INSECURE: 'true',
};

local appEnv = {
  ADDR: ':8081',
  ORIGIN: 'http://idp-register:8081',
  TRUST_CF_CONNECTING_IP: 'false',
  SQLITE_PATH: '/tmp/idp-register-e2e.db',

  OIDC_ISSUER: 'http://rauthy:8080/auth/v1/',
  OIDC_CLIENT_ID: 'idp-register',
  OIDC_CLIENT_SECRET: '9c6c109b601b78597d31c91653008e26086e1b0ff89629785cb9226006426e1a',
  OIDC_REDIRECT_URI: 'http://idp-register:8081/auth/callback',
  OIDC_SCOPES: 'openid email profile groups',
  OIDC_GROUPS_CLAIM: 'groups',
  OIDC_ADMIN_EMAILS: 'admin@localhost',

  PROVISIONER: 'rauthy',
  RAUTHY_API_BASE: rauthyApiBase,
  RAUTHY_API_KEY_NAME: apiKeyName,
  RAUTHY_API_KEY_SECRET: apiKeySecret,

  SESSION_TTL_HOURS: '12',
  SECURE_COOKIES: 'false',
  APP_ENV: 'e2e',
};

{
  labels: lib.labels,
  when: [{ event: 'manual' }],

  services: [
    { name: 'mailcrab', image: lib.mailcrabImage },
  ],

  steps: [
    // Rauthy reads ./config.toml from its working directory; point that at the
    // workspace stub via `directory`, overriding the image's default `serve`.
    {
      name: 'rauthy',
      image: lib.rauthyImage,
      detach: true,
      depends_on: [],
      directory: 'deploy/e2e',
      entrypoint: ['/app/rauthy', 'serve', '-c', 'config.toml'],
      environment: rauthyEnv,
    },
    {
      name: 'await-rauthy',
      image: lib.curlImage,
      depends_on: ['rauthy'],
      commands: [
        "timeout 120 sh -c 'until curl -fsS http://rauthy:8080/auth/v1/.well-known/openid-configuration >/dev/null; do sleep 3; done'",
      ],
    },
    // Only now is Rauthy's issuer live, so the app's boot-time OIDC discovery
    // will succeed instead of aborting the process.
    {
      name: 'idp-register',
      image: appImage,
      detach: true,
      depends_on: ['await-rauthy'],
      environment: appEnv,
    },
    {
      name: 'smoke',
      image: lib.goImage,
      depends_on: ['idp-register'],
      environment: {
        // Opts the provisioner suite into hitting the live Rauthy above.
        RAUTHY_INTEGRATION: '1',
        RAUTHY_API_BASE: rauthyApiBase,
        RAUTHY_API_KEY_NAME: apiKeyName,
        RAUTHY_API_KEY_SECRET: apiKeySecret,
      },
      commands: [
        'apk add --no-cache curl >/dev/null',
        "timeout 60 sh -c 'until curl -fsS http://idp-register:8081/healthz >/dev/null; do sleep 2; done'",
        // Provisioner client against the real Rauthy API.
        'go test -count=1 ./internal/provisioner/rauthy',
        // Public form config + a public registration submission end-to-end.
        'curl -fsS http://idp-register:8081/api/form >/dev/null',
        'curl -fsS -H "content-type: application/json"'
        + ' -d \'{"email":"e2e-public@example.test","username":"e2e-public","reviewText":"manual Crow e2e smoke"}\''
        + ' http://idp-register:8081/api/register | grep -q \'"status":"received"\'',
      ],
    },
  ],
}
