import * as oidc from 'openid-client';
import { config } from './env';

let cached: Promise<oidc.Configuration> | undefined;

export function oidcConfiguration(): Promise<oidc.Configuration> {
  cached ??= oidc.discovery(
    new URL(config.oidcIssuer),
    config.oidcClientId,
    config.oidcClientSecret
  );
  return cached;
}

export function normalizeGroups(claim: unknown): string[] {
  if (Array.isArray(claim)) return claim.filter((value): value is string => typeof value === 'string');
  if (typeof claim === 'string') return claim.split(/[ ,]+/).filter(Boolean);
  return [];
}
