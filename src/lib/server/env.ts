import { env } from '$env/dynamic/private';

function required(name: string): string {
  const value = env[name]?.trim();
  if (!value) throw new Error(`Missing required environment variable: ${name}`);
  return value;
}

export const config = {
  get databaseUrl() {
    return required('DATABASE_URL');
  },
  get oidcIssuer() {
    return required('OIDC_ISSUER');
  },
  get oidcClientId() {
    return required('OIDC_CLIENT_ID');
  },
  get oidcClientSecret() {
    return required('OIDC_CLIENT_SECRET');
  },
  get oidcRedirectUri() {
    return required('OIDC_REDIRECT_URI');
  },
  get oidcScopes() {
    return env.OIDC_SCOPES?.trim() || 'openid email profile groups';
  },
  get oidcAdminGroup() {
    return env.OIDC_ADMIN_GROUP?.trim() || 'svc:invite-admin:admin';
  },
  get rauthyApiBase() {
    return required('RAUTHY_API_BASE').replace(/\/$/, '');
  },
  get rauthyApiKeyName() {
    return required('RAUTHY_API_KEY_NAME');
  },
  get rauthyApiKeySecret() {
    return required('RAUTHY_API_KEY_SECRET');
  },
  get rauthyLanguage() {
    return env.RAUTHY_DEFAULT_LANGUAGE?.trim() || 'en';
  },
  get rauthyTimezone() {
    return env.RAUTHY_DEFAULT_TIMEZONE?.trim() || 'UTC';
  },
  get inviteHmacKey() {
    const value = required('INVITE_HMAC_KEY');
    if (value.length < 32) throw new Error('INVITE_HMAC_KEY must be at least 32 characters');
    return value;
  },
  get sessionTtlHours() {
    const parsed = Number.parseInt(env.SESSION_TTL_HOURS || '12', 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 12;
  },
  get secureCookies() {
    return env.NODE_ENV === 'production';
  }
};
