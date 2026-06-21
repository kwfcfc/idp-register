import { redirect, type RequestHandler } from '@sveltejs/kit';
import * as oidc from 'openid-client';
import { oidcConfiguration } from '$lib/server/oidc';
import { config } from '$lib/server/env';

const options = {
  path: '/',
  httpOnly: true,
  secure: config.secureCookies,
  sameSite: 'lax' as const,
  maxAge: 600
};

export const GET: RequestHandler = async ({ cookies, url }) => {
  const configuration = await oidcConfiguration();
  const verifier = oidc.randomPKCECodeVerifier();
  const challenge = await oidc.calculatePKCECodeChallenge(verifier);
  const state = oidc.randomState();
  const nonce = oidc.randomNonce();
  const next = url.searchParams.get('next')?.startsWith('/') ? url.searchParams.get('next')! : '/admin';

  cookies.set('oidc_verifier', verifier, options);
  cookies.set('oidc_state', state, options);
  cookies.set('oidc_nonce', nonce, options);
  cookies.set('oidc_next', next, options);

  const authorizationUrl = oidc.buildAuthorizationUrl(configuration, {
    redirect_uri: config.oidcRedirectUri,
    scope: config.oidcScopes,
    code_challenge: challenge,
    code_challenge_method: 'S256',
    state,
    nonce
  });

  throw redirect(303, authorizationUrl.href);
};
