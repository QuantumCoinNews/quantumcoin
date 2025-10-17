import 'dotenv/config';
import { TwitterApi } from 'twitter-api-v2';
import fs from 'node:fs/promises';

const clientId = process.env.X_CLIENT_ID;
const clientSecret = process.env.X_CLIENT_SECRET;
const callback = process.env.X_CALLBACK_URL || 'https://localhost/callback';

const textFlag = process.argv.indexOf('--text');
const text = textFlag >= 0 ? process.argv[textFlag + 1] : (process.argv.slice(2).join(' ') || 'Hello from QC');

const tokenPath = 'secrets/x_oauth2_tokens.json';
let tokens = JSON.parse(await fs.readFile(tokenPath, 'utf8'));

const oauth2 = new TwitterApi({ clientId, clientSecret });

// Token süresi bittiyse yenile
if (!tokens.accessToken || (tokens.expiresAt && Date.now() >= tokens.expiresAt - 30000)) {
  const { accessToken, refreshToken, expiresIn } = await oauth2.refreshOAuth2Token(tokens.refreshToken);
  tokens = { accessToken, refreshToken, expiresAt: Date.now() + expiresIn*1000 };
  await fs.writeFile(tokenPath, JSON.stringify(tokens, null, 2));
}

const rw = new TwitterApi(tokens.accessToken).readWrite;
const tweet = await rw.v2.tweet(text);
console.log('tweet_id=' + tweet.data.id);
