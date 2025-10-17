import 'dotenv/config';
import { TwitterApi } from 'twitter-api-v2';
import { createInterface } from 'node:readline/promises';
import fs from 'node:fs/promises';

const clientId = process.env.X_CLIENT_ID;
const clientSecret = process.env.X_CLIENT_SECRET;
const callback = process.env.X_CALLBACK_URL || 'https://localhost/callback';

if (!clientId || !clientSecret) {
  console.error('X_CLIENT_ID / X_CLIENT_SECRET eksik (.env)');
  process.exit(1);
}

const client = new TwitterApi({ clientId, clientSecret });
const { url, codeVerifier, state } = client.generateOAuth2AuthLink(
  callback,
  { scope: ['tweet.read','tweet.write','users.read','offline.access'] }
);

console.log('\n== Yetkilendirme URL\'si ==\n' + url + '\n');
const rl = createInterface({ input: process.stdin, output: process.stdout });
const code = await rl.question('Tarayıcıda onayla, sonra URL\'deki code=... değerini buraya yapıştır: ');
await rl.close();

const { accessToken, refreshToken, expiresIn } =
  await client.loginWithOAuth2({ code, codeVerifier, redirectUri: callback, state });

await fs.mkdir('secrets', { recursive: true });
await fs.writeFile('secrets/x_oauth2_tokens.json',
  JSON.stringify({ accessToken, refreshToken, expiresAt: Date.now() + expiresIn*1000 }, null, 2)
);

console.log('✅ OK → secrets/x_oauth2_tokens.json kaydedildi');
