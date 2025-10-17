// scripts/post-x.mjs
import 'dotenv/config';
import { TwitterApi } from 'twitter-api-v2';

function parseArg(name){
  const i = process.argv.indexOf(`--${name}`);
  if (i >= 0) return process.argv[i+1];
  return '';
}

const text = parseArg('text') || '';
if (!text) {
  console.error('[X ERROR] missing --text');
  process.exit(1);
}

const ck = process.env.TW_CONSUMER_KEY;
const cs = process.env.TW_CONSUMER_SECRET;
const at = process.env.TW_ACCESS_TOKEN;
const as = process.env.TW_ACCESS_SECRET;

if (!ck || !cs || !at || !as) {
  console.error('[X ERROR] env tokens missing');
  process.exit(1);
}

const client = new TwitterApi({
  appKey: ck,
  appSecret: cs,
  accessToken: at,
  accessSecret: as,
});

function nonce(n=4){
  const chars='0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
  let s=''; for (let i=0;i<n;i++) s+=chars[Math.floor(Math.random()*chars.length)];
  return s;
}

async function post(t){
  const res = await client.v2.tweet(t);
  return res?.data?.id;
}

(async () => {
  try {
    try {
      const id = await post(text);
      console.log(`tweet_id=${id}`);
      process.exit(0);
    } catch (e1) {
      // duplicate guard: retry once with tiny nonce
      const body = e1?.data || e1?.message || '';
      if (String(body).toLowerCase().includes('duplicate')) {
        const t2 = (text.length > 265) ? text.slice(0, 260) + ` [${nonce()}]` : `${text} [${nonce()}]`;
        const id2 = await post(t2);
        console.log(`tweet_id=${id2}`);
        process.exit(0);
      }
      throw e1;
    }
  } catch (e){
    const msg = e?.data?.detail || e?.message || String(e);
    console.error(`[X ERROR] ${msg}`);
    process.exit(2);
  }
})();
