import { loadSecrets, saveSecrets } from './secrets.js';
import { PublisherError } from './types.js';

const TOKEN_ENDPOINT = 'https://api.x.com/2/oauth2/token';
const REFRESH_WINDOW_MS = 5 * 60 * 1000;
const MIN_REFRESH_WINDOW_MS = 30 * 1000;
const MAX_TOKEN_LIFETIME_SECONDS = 365 * 24 * 60 * 60;

export interface XOAuth2Tokens {
  access_token: string;
  refresh_token?: string;
  token_type?: string;
  expires_at: number;
  expires_in: number;
  scope?: string;
}

function readTokens(raw: string | undefined): XOAuth2Tokens | undefined {
  if (!raw) return undefined;
  try {
    const value = JSON.parse(raw) as Partial<XOAuth2Tokens>;
    if (typeof value.access_token !== 'string' || !value.access_token ||
        !Number.isSafeInteger(value.expires_at) || (value.expires_at ?? 0) <= 0 ||
        !Number.isSafeInteger(value.expires_in) || (value.expires_in ?? 0) <= 0 || (value.expires_in ?? 0) > MAX_TOKEN_LIFETIME_SECONDS ||
        Math.abs(value.expires_at ?? 0) > 8.64e15 ||
        (value.refresh_token !== undefined && typeof value.refresh_token !== 'string')) return undefined;
    return value as XOAuth2Tokens;
  } catch { return undefined; }
}

export function xOAuth2Configured(env: NodeJS.ProcessEnv): boolean {
  return Boolean(env.X_OAUTH2_CLIENT_ID && readTokens(env.X_OAUTH2_TOKEN_JSON));
}

export function xOAuth2RefreshAt(tokens: XOAuth2Tokens): number {
  const lifetimeMs = tokens.expires_in * 1000;
  const margin = Math.min(REFRESH_WINDOW_MS, Math.max(Math.min(MIN_REFRESH_WINDOW_MS, lifetimeMs / 2), Math.floor(lifetimeMs * 0.1)));
  return tokens.expires_at - margin;
}

export function xOAuth2TokenStatus(env: NodeJS.ProcessEnv, now = Date.now()): 'not-configured' | 'reauthorization-required' | 'refresh-due' | 'valid' {
  if (!env.X_OAUTH2_CLIENT_ID && !env.X_OAUTH2_TOKEN_JSON) return 'not-configured';
  const tokens = readTokens(env.X_OAUTH2_TOKEN_JSON);
  if (!env.X_OAUTH2_CLIENT_ID || !tokens || (!tokens.refresh_token && tokens.expires_at <= now)) return 'reauthorization-required';
  return tokens.expires_at <= now || xOAuth2RefreshAt(tokens) <= now ? 'refresh-due' : 'valid';
}

export function xOAuth2Lifecycle(env: NodeJS.ProcessEnv, now = Date.now()) {
  const tokens = readTokens(env.X_OAUTH2_TOKEN_JSON);
  if (!tokens) return { status: xOAuth2TokenStatus(env, now), expiresAt: undefined, refreshAt: undefined, remainingSeconds: undefined };
  return {
    status: xOAuth2TokenStatus(env, now),
    expiresAt: new Date(tokens.expires_at).toISOString(),
    refreshAt: new Date(xOAuth2RefreshAt(tokens)).toISOString(),
    remainingSeconds: Math.max(0, Math.floor((tokens.expires_at - now) / 1000)),
  };
}

export async function ensureXOAuth2AccessToken(
  env: NodeJS.ProcessEnv,
  options: {
    now?: () => number;
    fetcher?: typeof fetch;
    persist?: (tokens: XOAuth2Tokens) => Promise<void>;
  } = {},
): Promise<string> {
  const tokens = readTokens(env.X_OAUTH2_TOKEN_JSON);
  if (!env.X_OAUTH2_CLIENT_ID || !tokens) throw new PublisherError('PROVIDER_CONFIGURATION_INVALID');
  const now = options.now ?? Date.now;
  if (xOAuth2RefreshAt(tokens) > now()) return tokens.access_token;
  if (!tokens.refresh_token) throw new PublisherError('PROVIDER_CREDENTIAL_REVOKED');

  const body = new URLSearchParams({ grant_type: 'refresh_token', refresh_token: tokens.refresh_token });
  const headers = new Headers({ 'content-type': 'application/x-www-form-urlencoded' });
  if (env.X_OAUTH2_CLIENT_SECRET) {
    headers.set('authorization', `Basic ${Buffer.from(`${env.X_OAUTH2_CLIENT_ID}:${env.X_OAUTH2_CLIENT_SECRET}`).toString('base64')}`);
  } else {
    body.set('client_id', env.X_OAUTH2_CLIENT_ID);
  }

  let response: Response;
  try {
    response = await (options.fetcher ?? fetch)(TOKEN_ENDPOINT, { method: 'POST', headers, body, signal: AbortSignal.timeout(20_000) });
  } catch {
    throw new PublisherError('CREDENTIAL_REFRESH_FAILED');
  }
  if (!response.ok) {
    if (response.status === 400 || response.status === 401) throw new PublisherError('PROVIDER_CREDENTIAL_REVOKED');
    throw new PublisherError(response.status === 429 ? 'CREDENTIAL_REFRESH_RATE_LIMITED' : 'CREDENTIAL_REFRESH_FAILED');
  }

  let fresh: Partial<XOAuth2Tokens>;
  try { fresh = await response.json() as Partial<XOAuth2Tokens>; }
  catch { throw new PublisherError('CREDENTIAL_REFRESH_FAILED'); }
  if (typeof fresh.access_token !== 'string' || !fresh.access_token || !Number.isSafeInteger(fresh.expires_in) ||
      (fresh.expires_in ?? 0) <= 0 || (fresh.expires_in ?? 0) > MAX_TOKEN_LIFETIME_SECONDS ||
      (fresh.refresh_token !== undefined && typeof fresh.refresh_token !== 'string')) {
    throw new PublisherError('CREDENTIAL_REFRESH_FAILED');
  }
  const expiresIn = fresh.expires_in as number;
  const refreshed: XOAuth2Tokens = {
    access_token: fresh.access_token,
    refresh_token: fresh.refresh_token || tokens.refresh_token,
    token_type: fresh.token_type || tokens.token_type || 'bearer',
    expires_in: expiresIn,
    expires_at: now() + expiresIn * 1000,
    ...(typeof fresh.scope === 'string' ? { scope: fresh.scope } : tokens.scope ? { scope: tokens.scope } : {}),
  };

  const persist = options.persist ?? (async value => {
    const file = env.QC_PUBLISHER_SECRET_FILE;
    if (!file) throw new PublisherError('CREDENTIAL_PERSISTENCE_REQUIRED');
    const secrets = await loadSecrets(file);
    await saveSecrets(file, { ...secrets, X_OAUTH2_TOKEN_JSON: JSON.stringify(value) });
  });
  try { await persist(refreshed); }
  catch (error) {
    if (error instanceof PublisherError) throw error;
    throw new PublisherError('CREDENTIAL_PERSISTENCE_FAILED');
  }
  env.X_OAUTH2_TOKEN_JSON = JSON.stringify(refreshed);
  return refreshed.access_token;
}
