import { error, redirect, type RequestHandler } from '@sveltejs/kit';
import * as oidc from 'openid-client';
import { oidcConfiguration, normalizeGroups } from '$lib/server/oidc';
import { config } from '$lib/server/env';
import { createSession } from '$lib/server/auth';

export const GET: RequestHandler = async ({ cookies, url }) => {
  const verifier = cookies.get('oidc_verifier');
  const state = cookies.get('oidc_state');
  const nonce = cookies.get('oidc_nonce');
  const next = cookies.get('oidc_next') || '/admin';

  if (!verifier || !state || !nonce) throw error(400, 'OIDC transaction expired or missing');

  const configuration = await oidcConfiguration();
  const tokens = await oidc.authorizationCodeGrant(configuration, url, {
    pkceCodeVerifier: verifier,
    expectedState: state,
    expectedNonce: nonce
  });
  const claims = tokens.claims();
  if (!claims?.sub || typeof claims.email !== 'string') throw error(403, 'Rauthy did not return sub/email claims');

  const groups = normalizeGroups(claims.groups);
  if (!groups.includes(config.oidcAdminGroup)) throw error(403, 'This account is not in the invite administrator group');

  await createSession(cookies, {
    sub: claims.sub,
    email: claims.email,
    displayName:
      (typeof claims.name === 'string' && claims.name) ||
      (typeof claims.preferred_username === 'string' && claims.preferred_username) ||
      claims.email,
    groups
  });

  for (const name of ['oidc_verifier', 'oidc_state', 'oidc_nonce', 'oidc_next']) {
    cookies.delete(name, { path: '/' });
  }

  throw redirect(303, next.startsWith('/') ? next : '/admin');
};
