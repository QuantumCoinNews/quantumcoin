#!/usr/bin/env node
// publish-youtube.mjs — Generate (OpenAI) → Render (ffmpeg) → Upload (YouTube)
// Çıktılar: ${OUT_DIR}\youtube_{ts}.(json|txt|mp4)
// Mevcut dosya/klasörlere dokunmaz. OUT_DIR varsa kullanır.

import 'dotenv/config';
import fs from 'fs';
import path from 'path';
import { spawn } from 'child_process';
import { google } from 'googleapis';
import OpenAI from 'openai';

const OUT_DIR = process.env.OUT_DIR || 'src/content/out';
const MODEL   = process.env.OPENAI_MODEL || 'gpt-4o-mini';
const LANG    = process.env.CONTENT_LANG || 'en';
const PRIVACY = (process.env.YOUTUBE_PRIVACY || 'unlisted').toLowerCase(); // public|private|unlisted

const CLIENT_SECRET_PATH = process.env.YOUTUBE_CLIENT_SECRET_PATH;     // secrets/client_secret.json
const TOKEN_PATH         = process.env.YOUTUBE_TOKEN_PATH || 'secrets/youtube_token.json';

if (!fs.existsSync(CLIENT_SECRET_PATH)) {
  console.error('ERROR: YOUTUBE_CLIENT_SECRET_PATH bulunamadı:', CLIENT_SECRET_PATH);
  process.exit(1);
}
if (!fs.existsSync(TOKEN_PATH)) {
  console.error('ERROR: Token yok:', TOKEN_PATH, ' — önce youtube-auth.mjs çalıştır.');
  process.exit(1);
}
if (!fs.existsSync(OUT_DIR)) {
  console.error('ERROR: OUT_DIR yok:', OUT_DIR, ' — bu klasörü oluştur (yeni klasör açmak istemediğin için script oluşturmuyor).');
  process.exit(1);
}

function ts() {
  const d = new Date();
  const z = n => String(n).padStart(2,'0');
  return `${d.getFullYear()}${z(d.getMonth()+1)}${z(d.getDate())}-${z(d.getHours())}${z(d.getMinutes())}${z(d.getSeconds())}`;
}

async function generateWithOpenAI() {
  const openai = new OpenAI({ apiKey: process.env.OPENAI_API_KEY });
  const sys = `You are a concise YouTube Shorts writer for the QuantumCoin project. Language: ${LANG}.
Return strict JSON with: title, description, tags (array), script (<= 70 words, 3-6 short lines for on-screen text).`;
  const user = `Create a 12-second educational YouTube Short about QuantumCoin (QC):
- Focus: real-time mining, wallet+miner app, AI-augmented chain, global potential.
- Tone: clear, exciting, non-hype, compliant. No financial advice.
- Add 6-10 comma-separated tags (no #).`;
  const r = await openai.chat.completions.create({
    model: MODEL,
    temperature: 0.7,
    messages: [
      { role: 'system', content: sys },
      { role: 'user', content: user }
    ],
    response_format: { type: "json_object" }
  });
  const data = JSON.parse(r.choices[0].message.content);
  // Basit alan kontrolleri
  data.title ||= 'QuantumCoin — Next-gen blockchain';
  data.description ||= 'QuantumCoin: real-time mining, AI-augmented chain, and a friendly Wallet+Miner.';
  if (!Array.isArray(data.tags) || data.tags.length === 0) data.tags = ['QuantumCoin','QC','Blockchain','Mining','AI','Crypto'];
  data.script = (data.script || 'QuantumCoin\nReal-time mining\nWallet + Miner App\nAI-augmented chain\nJoin the network today!')
    .split(/\r?\n/).map(s => s.trim()).filter(Boolean).join('\n');
  return data;
}

function makeVideo(textLines, outFile) {
  return new Promise((resolve, reject) => {
    const text = textLines.replace(/:/g,'\\:').replace(/'/g,"\\'").replace(/"/g,'\\"');
    const font = process.platform === 'win32'
      ? 'C\\\\:/Windows/Fonts/arial.ttf'
      : '/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf';

    const vf = `drawtext=fontfile='${font}':text='${text}':fontcolor=white:fontsize=58:line_spacing=14:box=1:boxcolor=black@0.45:boxborderw=20:x=(w-text_w)/2:y=(h-text_h)/2`;
    const args = [
      '-y',
      '-f','lavfi','-i','color=c=black:s=1080x1920:d=12',
      '-f','lavfi','-i','anullsrc=channel_layout=stereo:sample_rate=44100',
      '-shortest',
      '-vf', vf,
      '-c:v','libx264','-pix_fmt','yuv420p','-preset','veryfast','-crf','22',
      '-c:a','aac','-b:a','128k',
      outFile
    ];
    const ff = spawn('ffmpeg', args, { stdio: 'inherit' });
    ff.on('error', reject);
    ff.on('close', code => code === 0 ? resolve() : reject(new Error('ffmpeg failed: ' + code)));
  });
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

async function uploadToYouTube(videoPath, meta) {
  const { client_id, client_secret, redirect } = loadClientSecrets(CLIENT_SECRET_PATH);
  const tokens = JSON.parse(fs.readFileSync(TOKEN_PATH, 'utf8'));
  const oauth2 = new google.auth.OAuth2(client_id, client_secret, redirect);
  oauth2.setCredentials(tokens);

  const youtube = google.youtube({ version: 'v3', auth: oauth2 });
  const res = await youtube.videos.insert({
    part: 'snippet,status',
    requestBody: {
      snippet: { title: meta.title, description: meta.description, tags: meta.tags, categoryId: '28' },
      status:  { privacyStatus: PRIVACY, selfDeclaredMadeForKids: false }
    },
    media: { body: fs.createReadStream(videoPath) }
  });
  return res.data.id;
}

(async () => {
  try {
    const t = ts();
    const metaPath   = path.join(OUT_DIR, `youtube_${t}.json`);
    const scriptPath = path.join(OUT_DIR, `youtube_${t}.txt`);
    const videoPath  = path.join(OUT_DIR, `youtube_${t}.mp4`);

    // 1) İçerik üret
    const meta = await generateWithOpenAI();
    fs.writeFileSync(metaPath, JSON.stringify(meta, null, 2));
    fs.writeFileSync(scriptPath, meta.script, 'utf8');

    // 2) Video üret (çok satırlı metni ekrana bas)
    await makeVideo(meta.script, videoPath);

    // 3) Yükle
    const id = await uploadToYouTube(videoPath, meta);
    console.log('✓ Yüklendi → https://www.youtube.com/watch?v=' + id);
    console.log('Files:', { metaPath, scriptPath, videoPath });
  } catch (e) {
    console.error('publish-youtube error:', e?.response?.data || e);
    process.exit(1);
  }
})();
