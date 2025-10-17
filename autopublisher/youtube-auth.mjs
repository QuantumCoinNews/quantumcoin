#!/usr/bin/env node
// youtube-auth.mjs — YouTube OAuth (Desktop flow), .env’deki path’lerle çalışır.
// Yeni klasör OLUŞTURMAZ; secrets yolu yoksa hata verir.
import 'dotenv/config';
import { google } from 'googleapis';
import http from 'http';
import { URL } from 'url';
import fs from 'fs';
import path from 'path';
import crypto from 'crypto';
import { spawn } from 'child_process';

const CLIENT_SECRET_PATH = process.env.YOUTUBE_CLIENT_SECRET_PATH;  // ör: secrets/client_secret.json
const TOKEN_PATH = process.env.YOUTUBE_TOKEN_PATH || 'secrets/youtube_token.json';
const ENV_REDIRECT = process.env.GOOGLE_REDIRECT_URI;               // opsiyonel

if (!CLIENT_SECRET_PATH || !fs.existsSync(CLIENT_SECRET_PATH)) {
  console.error('ERROR: YOUTUBE_CLIENT_SECRET_PATH yok veya dosya bulunamadı:', CLIENT_SECRET_PATH);
  process.exit(1);
}

// Google client_secret.json içinden kimlik bilgilerini oku
function loadClientSecrets(p) {
  const raw = JSON.parse(fs.readFileSync(p, 'utf8'));
  const d = raw.installed || raw.web || raw;
  const client_id = d.client_id || d.clientId;
  const client_secret = d.client_secret || d.clientSecret;
  const redirects = d.redirect_uris || d.redirectUris || [];
  if (!client_id || !client_secret) throw new Error('client_id/client_secret bulunamadı.');
  // Redirect seçimi: .env öncelikli; yoksa 127.0.0.1 içeren ilk redirect; o da yoksa varsayılan
  let redirect = ENV_REDIRECT
    || redirects.find(u => u.startsWith('http://127.0.0.1'))
    || 'http://127.0.0.1:53682/oauth2callback';
  return { client_id, client_secret, redirect };
}

const { client_id, client_secret, redirect } = loadClientSecrets(CLIENT_SECRET_PATH);

// TOKEN_PATH’in klasörü var mı? Yoksa yeni klasör oluşturmadan dur.
const tokenDir = path.dirname(TOKEN_PATH);
if (!fs.existsSync(tokenDir)) {
  console.error('ERROR: Token için hedef klasör yok:', tokenDir);
  console.error('Lütfen klasörü oluşturun veya .env’de YOUTUBE_TOKEN_PATH’i mevcut bir yola yönlendirin.');
  process.exit(1);
}

const { hostname, port, pathname } = new URL(redirect);
const SCOPES = [
  'https://www.googleapis.com/auth/youtube.upload',
  'https://www.googleapis.com/auth/youtube.readonly',
];

const oauth2 = new google.auth.OAuth2(client_id, client_secret, redirect);
const state = crypto.randomBytes(16).toString('hex');

function openInBrowser(url) {
  try {
    if (process.platform === 'win32') spawn('cmd', ['/c', 'start', '', url], { detached: true, stdio: 'ignore' });
    else if (process.platform === 'darwin') spawn('open', [url], { detached: true, stdio: 'ignore' });
    else spawn('xdg-open', [url], { detached: true, stdio: 'ignore' });
  } catch { console.log('Otomatik açılamazsa URL:\n', url); }
}

async function main() {
  const server = http.createServer(async (req, res) => {
    try {
      const u = new URL(req.url, `http://${req.headers.host}`);
      if (u.pathname !== pathname) { res.writeHead(404); return res.end('Not found'); }

      const code = u.searchParams.get('code');
      const returnedState = u.searchParams.get('state');
      if (!code) { res.writeHead(400); return res.end('Missing code'); }
      if (state !== returnedState) { res.writeHead(400); return res.end('State mismatch'); }

      const { tokens } = await oauth2.getToken(code);
      oauth2.setCredentials(tokens);
      fs.writeFileSync(TOKEN_PATH, JSON.stringify(tokens, null, 2));
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      res.end('<h3>Yetki tamam. Bu pencereyi kapatabilirsiniz.</h3>');

      console.log('✓ Token kaydedildi →', TOKEN_PATH);

      const yt = google.youtube({ version: 'v3', auth: oauth2 });
      const r = await yt.channels.list({ part: 'snippet,statistics', mine: true });
      const ch = r.data.items?.[0];
      if (ch) console.log(`✓ Bağlandı: ${ch.snippet.title} | Videos: ${ch.statistics.videoCount}`);
      else console.log('Uyarı: Kanal bilgisi okunamadı ama yetki alınmış olabilir.');
    } catch (err) {
      console.error('Yetkilendirme hatası:', err?.response?.data || err);
    } finally { server.close(); }
  });

  server.listen(Number(port), hostname, () => {
    const authUrl = oauth2.generateAuthUrl({
      access_type: 'offline',
      prompt: 'consent',
      scope: SCOPES,
      state
    });
    console.log('Tarayıcıda açılıyor. Otomatik açılmazsa bu URL’yi kopyala:\n', authUrl, '\n');
    openInBrowser(authUrl);
    console.log(`Dinleniyor: ${redirect}`);
  });
}

main().catch(e => { console.error(e); process.exit(1); });
