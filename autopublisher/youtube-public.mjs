#!/usr/bin/env node
// youtube-public.mjs — Public veri çekimi (API key ile). Yeni klasör oluşturmaz.
// Kullanım:
//   node youtube-public.mjs
//   node youtube-public.mjs --max 8
//   node youtube-public.mjs > src/content/out/youtube-latest.json  (klasör zaten varsa)

import 'dotenv/config';
import https from 'https';
import { URL } from 'url';

const API_KEY  = process.env.YOUTUBE_API_KEY;
const HANDLE   = (process.env.YOUTUBE_CHANNEL_HANDLE || '').trim();
const FIXED_ID = (process.env.YOUTUBE_CHANNEL_ID || '').trim();
const MAX_DEF  = Number(process.env.YOUTUBE_PUBLIC_MAX || 5);

function arg(name, def=null) {
  const i = process.argv.indexOf(`--${name}`);
  return (i !== -1 && i + 1 < process.argv.length) ? process.argv[i + 1] : def;
}
const maxResults = Number(arg('max', MAX_DEF)) || 5;

if (!API_KEY) {
  console.error('ERROR: YOUTUBE_API_KEY .env’de yok.');
  process.exit(1);
}
if (!HANDLE && !FIXED_ID) {
  console.error('ERROR: YOUTUBE_CHANNEL_HANDLE veya YOUTUBE_CHANNEL_ID .env’de tanımlı olmalı.');
  process.exit(1);
}

function getJSON(u) {
  return new Promise((resolve, reject) => {
    const req = https.get(u, (res) => {
      let body = '';
      res.setEncoding('utf8');
      res.on('data', (c) => body += c);
      res.on('end', () => {
        try { resolve(JSON.parse(body)); }
        catch { reject(new Error('Invalid JSON from API')); }
      });
    });
    req.on('error', reject);
    req.end();
  });
}

async function resolveByHandle(handle) {
  const clean = handle.startsWith('@') ? handle.slice(1) : handle;
  const url = new URL('https://www.googleapis.com/youtube/v3/channels');
  url.searchParams.set('part', 'snippet,contentDetails');
  url.searchParams.set('forHandle', clean);
  url.searchParams.set('key', API_KEY);
  const data = await getJSON(url);
  const ch = data.items?.[0];
  if (!ch?.id) throw new Error('Handle ile kanal bulunamadı: ' + handle);
  return { channelId: ch.id, uploadsId: ch.contentDetails?.relatedPlaylists?.uploads || null };
}

async function uploadsByChannelId(channelId) {
  const url = new URL('https://www.googleapis.com/youtube/v3/channels');
  url.searchParams.set('part', 'contentDetails');
  url.searchParams.set('id', channelId);
  url.searchParams.set('key', API_KEY);
  const data = await getJSON(url);
  const up = data.items?.[0]?.contentDetails?.relatedPlaylists?.uploads;
  if (!up) throw new Error('uploads playlist bulunamadı (channelId=' + channelId + ')');
  return up;
}

async function listLatest(uploadsId, n) {
  const url = new URL('https://www.googleapis.com/youtube/v3/playlistItems');
  url.searchParams.set('part', 'snippet,contentDetails');
  url.searchParams.set('playlistId', uploadsId);
  url.searchParams.set('maxResults', Math.min(Math.max(n,1), 50).toString());
  url.searchParams.set('key', API_KEY);
  const data = await getJSON(url);
  return (data.items || []).map(it => {
    const vid = it.contentDetails?.videoId || null;
    const sn  = it.snippet || {};
    return {
      videoId: vid,
      title: sn.title || null,
      description: sn.description || null,
      publishedAt: it.contentDetails?.videoPublishedAt || sn.publishedAt || null,
      url: vid ? `https://www.youtube.com/watch?v=${vid}` : null,
      thumbnail: sn.thumbnails?.medium?.url || sn.thumbnails?.default?.url || null
    };
  });
}

(async () => {
  try {
    let channelId = FIXED_ID;
    let uploadsId = null;

    if (channelId) {
      uploadsId = await uploadsByChannelId(channelId);
    } else {
      const r = await resolveByHandle(HANDLE);
      channelId = r.channelId;
      uploadsId = r.uploadsId || await uploadsByChannelId(channelId);
    }

    const items = await listLatest(uploadsId, maxResults);
    const out = { channelId, uploadsId, count: items.length, items };
    process.stdout.write(JSON.stringify(out, null, 2) + '\n');
  } catch (e) {
    console.error('youtube-public.mjs error:', e.message || e);
    process.exit(1);
  }
})();
