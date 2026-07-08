<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Reverse Proxy

Run `idp-register` on its own origin, such as `https://register.example.com`.
The application currently expects to live at the domain root, not under a path
prefix like `/register`.

Keep these values aligned:

- public URL: `https://register.example.com`
- `ORIGIN=https://register.example.com`
- `OIDC_REDIRECT_URI=https://register.example.com/auth/callback`
- OIDC client redirect URI in the IdP: the same callback URL

## Caddy Example

```caddyfile
register.example.com {
  reverse_proxy 127.0.0.1:8080
}
```

## Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name register.example.com;

    ssl_certificate /etc/letsencrypt/live/register.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/register.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## Cloudflare

When proxied through Cloudflare, keep `TRUST_CF_CONNECTING_IP=true` if you want
logs and audit metadata to use the visitor IP from `CF-Connecting-IP`.

Cloudflare Tunnel is also fine: point the tunnel service at
`http://127.0.0.1:8080` and keep `ORIGIN` set to the public hostname.

## Path Prefixes

Sub-path deployment is not currently supported. Use a dedicated hostname or
subdomain instead of trying to serve the app at `https://example.com/register`.

The frontend, API routes, auth routes, cookies, and OIDC callback are all built
around root-path deployment today.
