#!/usr/bin/env node
// youtube-upload.mjs — Tek seferlik yükleme. .env’deki path’ler kullanılır.
import 'dotenv/config';
import { google } from 'googleapis';
import fs from 'fs';
import path from 'path';
import { spawn } from 'child_process';

const CLIENT_SECRET_PATH = process.env.YOUTUBE_CLIENT_SECRET_PATH;     // secrets/client_secret.json
const TOKEN_PATH = process.env.YOUTUBE_TOKEN_PATH || 'secrets/youtube_token.json';
const PRIVACY = (process.env.YOUTUBE_PRIVACY || 'unlisted').toLowerCase(); // public|private|unlisted

if (!CLIENT_SECRET_PATH || !fs.existsSync(CLIENT_SECRET_PATH)) {
  console.error('ERROR: YOUTUBE_CLIENT_SECRET_PATH yok veya dosya bulunamadı:', CLIENT_SECRET_PATH);
  process.exit(1);
}
if (!fs.existsSync(TOKEN_PATH)) {
  console.error('ERROR: Token yok:', TOKEN_PATH, '\nÖnce "node youtube-auth.mjs" çalıştır.');
  process.exit(1);
}

function loadClientSecrets(p) {
  const raw = JSON.parse(fs.readFileSync(p, 'utf8'));
  const d = raw.installed || raw.web || raw;
  const client_id = d.client_id || d.clientId;
  const client_secret = d.client_secret || d.clientSecret;
  const redirects = d.redirect_uris || d.redirectUris || [];
  const redirect = process.env.GOOGLE_REDIRECT_URI
    || redirects.find(u => u.startsWith('http://127.0.0.1'))
    || 'http://127.0.0.1:53682/oauth2callback';
  if (!client_id || !client_secret) throw new Error('client_id/client_secret yok.');
  return { client_id, client_secret, redirect };
}

const { client_id, client_secret, redirect } = loadClientSecrets(CLIENT_SECRET_PATH);

function arg(name, def = null) {
  const i = process.argv.indexOf(`--${name}`);
  return (i !== -1 && i + 1 < process.argv.length) ? process.argv[i + 1] : def;
}

const makeSample = process.argv.includes('--make-sample');
const fileArg = arg('file', null);
const title = arg('title', `QC Autopublisher Test ${new Date().toISOString().slice(0,19).replace('T',' ')}`);
const desc = arg('description', 'Automated test upload (QuantumCoin Autopublisher).');
const privacy = (arg('privacy', PRIVACY) || PRIVACY);
const tags = (arg('tags', 'QuantumCoin,QC,Blockchain,AI,Mining').split(',').map(t=>t.trim()).filter(Boolean));
const categoryId = arg('category', '28'); // Science & Technology

async function makeSampleMp4(outPath) {
  return new Promise((resolve, reject) => {
    const ff = spawn('ffmpeg', [
      '-y',
      '-f','lavfi','-i','color=c=black:s=1080x1920:d=6',      // 9:16, 6 sn
      '-f','lavfi','-i','anullsrc=channel_layout=stereo:sample_rate=44100',
      '-shortest',
      '-c:v','libx264','-pix_fmt','yuv420p','-preset','veryfast','-crf','23',
      '-c:a','aac','-b:a','128k',
      outPath
    ], { stdio: 'inherit' });
    ff.on('error', reject);
    ff.on('close', c => c === 0 ? resolve() : reject(new Error('ffmpeg failed')));
  });
}

async function main() {
  const tokens = JSON.parse(fs.readFileSync(TOKEN_PATH, 'utf8'));
  const oauth2 = new google.auth.OAuth2(client_id, client_secret, redirect);
  oauth2.setCredentials(tokens);

  let videoPath = fileArg;
  let createdTemp = false;
  if (!videoPath) {
    if (!makeSample) throw new Error('Video dosyası yok. --file <path> veya --make-sample kullan.');
    videoPath = path.join(process.cwd(), 'test_short.mp4');
    console.log('→ Deneme videosu üretiliyor:', videoPath);
    await makeSampleMp4(videoPath);
    createdTemp = true;
  }
  if (!fs.existsSync(videoPath)) throw new Error('Dosya bulunamadı: ' + videoPath);

  const youtube = google.youtube({ version: 'v3', auth: oauth2 });
  console.log('→ Yükleniyor:', path.basename(videoPath));
  const res = await youtube.videos.insert({
    part: 'snippet,status',
    requestBody: {
      snippet: { title, description: desc, tags, categoryId },
      status: { privacyStatus: privacy, selfDeclaredMadeForKids: false }
    },
    media: { body: fs.createReadStream(videoPath) }
  });

  console.log('✓ Yüklendi → https://www.youtube.com/watch?v=' + res.data.id);
  if (createdTemp) { try { fs.unlinkSync(videoPath); console.log('Temizlik: test video silindi.'); } catch {} }
}
main().catch(e => { console.error('Yükleme hatası:', e?.response?.data || e); process.exit(1); });
