import { createHash, randomBytes } from 'node:crypto';
import { PublisherError } from './types.js';
import type { XOAuth2Tokens } from './xOAuth2.js';

const AUTHORIZE_ENDPOINT = 'https://x.com/i/oauth2/authorize';
const TOKEN_ENDPOINT = 'https://api.x.com/2/oauth2/token';
const REQUIRED_SCOPES = ['tweet.write', 'users.read', 'offline.access'] as const;

export interface XOAuth2AuthorizationRequest {
  authorizationUrl: string;
  state: string;
  codeVerifier: string;
  redirectUri: string;
}

export function createXOAuth2AuthorizationRequest(
  clientId: string,
  redirectUri: string,
  random: (size: number) => Buffer = randomBytes,
): XOAuth2AuthorizationRequest {
  if (!clientId || /\s/.test(clientId)) throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');
  let redirect: URL;
  try { redirect = new URL(redirectUri); } catch { throw new PublisherError('PROVIDER_CONFIGURATION_INVALID'); }
  const loopback = redirect.hostname === '127.0.0.1' || redirect.hostname === 'localhost';
  const port = Number(redirect.port);
  if (redirect.protocol !== 'http:' || !loopback || !Number.isInteger(port) || port < 1024 || port > 65535 ||
      redirect.pathname !== '/callback' || redirect.username || redirect.password || redirect.search || redirect.hash) {
    throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');
  }

  const state = random(32).toString('base64url');
  const codeVerifier = random(32).toString('base64url');
  const codeChallenge = createHash('sha256').update(codeVerifier).digest('base64url');
  const url = new URL(AUTHORIZE_ENDPOINT);
  url.search = new URLSearchParams({
    response_type: 'code',
    client_id: clientId,
    redirect_uri: redirect.toString(),
    scope: REQUIRED_SCOPES.join(' '),
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
  }).toString();
  return { authorizationUrl: url.toString(), state, codeVerifier, redirectUri: redirect.toString() };
}

export function readXOAuth2Callback(callbackUrl: string, request: XOAuth2AuthorizationRequest): string {
  let callback: URL;
  try { callback = new URL(callbackUrl); } catch { throw new PublisherError('PROVIDER_AUTHORIZATION_CALLBACK_INVALID'); }
  if (callback.origin + callback.pathname !== new URL(request.redirectUri).origin + new URL(request.redirectUri).pathname) {
    throw new PublisherError('PROVIDER_AUTHORIZATION_CALLBACK_INVALID');
  }
  if (callback.searchParams.get('state') !== request.state) throw new PublisherError('PROVIDER_AUTHORIZATION_STATE_MISMATCH');
  if (callback.searchParams.has('error') || callback.searchParams.has('error_description')) throw new PublisherError('PROVIDER_AUTHORIZATION_DENIED');
  const code = callback.searchParams.get('code');
  if (!code || code.length > 4096) throw new PublisherError('PROVIDER_AUTHORIZATION_CALLBACK_INVALID');
  return code;
}

export async function exchangeXOAuth2AuthorizationCode(
  clientId: string,
  code: string,
  request: XOAuth2AuthorizationRequest,
  fetcher: typeof fetch = fetch,
): Promise<XOAuth2Tokens> {
  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    code,
    redirect_uri: request.redirectUri,
    code_verifier: request.codeVerifier,
    client_id: clientId,
  });
  let response: Response;
  try {
    response = await fetcher(TOKEN_ENDPOINT, {
      method: 'POST',
      headers: { 'content-type': 'application/x-www-form-urlencoded' },
      body,
      signal: AbortSignal.timeout(20_000),
    });
  } catch { throw new PublisherError('PROVIDER_AUTHORIZATION_EXCHANGE_FAILED'); }
  if (!response.ok) throw new PublisherError('PROVIDER_AUTHORIZATION_EXCHANGE_FAILED');

  let value: Partial<XOAuth2Tokens>;
  try { value = await response.json() as Partial<XOAuth2Tokens>; }
  catch { throw new PublisherError('PROVIDER_AUTHORIZATION_EXCHANGE_FAILED'); }
  const scopes = typeof value.scope === 'string' ? new Set(value.scope.split(/\s+/)) : new Set<string>();
  if (typeof value.access_token !== 'string' || !value.access_token ||
      typeof value.refresh_token !== 'string' || !value.refresh_token ||
      !Number.isSafeInteger(value.expires_in) || (value.expires_in ?? 0) <= 0 ||
      !REQUIRED_SCOPES.every(scope => scopes.has(scope))) {
    throw new PublisherError('PROVIDER_AUTHORIZATION_SCOPE_OR_TOKEN_INVALID');
  }
  return {
    access_token: value.access_token,
    refresh_token: value.refresh_token,
    token_type: value.token_type || 'bearer',
    expires_in: value.expires_in as number,
    expires_at: Date.now() + (value.expires_in as number) * 1000,
    scope: value.scope,
  };
}
