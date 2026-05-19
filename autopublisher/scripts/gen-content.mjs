// scripts/gen-content.mjs
import 'dotenv/config';
import OpenAI from 'openai';
import fs from 'fs';
import path from 'path';

// ====== ENV & paths ======
const apiKey = process.env.OPENAI_API_KEY;
const model  = process.env.OPENAI_MODEL || 'gpt-4o-mini';
const repoUrl = process.env.REPO_URL || 'https://github.com/QuantumCoinNews/quantumcoin';
const tz      = process.env.TZ || 'Europe/Istanbul';
const X_MAX   = Number(process.env.X_MAX_LEN || 260);

const root    = path.resolve(process.cwd());
const outDir  = path.join(root, 'src', 'content', 'out');
const recentPath = path.join(outDir, 'recent-topics.json');

// AI cache (stabilizasyon)
const today = new Date().toLocaleDateString('en-CA', { timeZone: tz }); // YYYY-MM-DD
const cacheDir = path.join(root, 'ai_cache', today);
const cacheLast = path.join(cacheDir, 'last.json');

fs.mkdirSync(outDir, { recursive: true });
fs.mkdirSync(cacheDir, { recursive: true });

const recent = fs.existsSync(recentPath)
  ? JSON.parse(fs.readFileSync(recentPath, 'utf8') || '[]')
  : [];

// son içeriklerden tekrarı engelle
const lastTitles = recent.slice(-20).map(r => (r.title || '').trim().toLowerCase());
const lastCats   = recent.slice(-7).map(r => r.category).filter(Boolean);
const avoidCats  = Array.from(new Set(lastCats.slice(-5)));

const CATEGORIES = [
  'Consensus & Performance',
  'Mining & Validators',
  'Explorer',
  'Wallet & Tooling',
  'Testnet Update',
  'Security & Audits',
  'Tokenomics Insight',
  'Roadmap Milestone',
  'Community & Ambassador'
];

let allowedCats = CATEGORIES.filter(c => !avoidCats.includes(c));
if (allowedCats.length === 0) allowedCats = [...CATEGORIES];

const now = new Date();
const dow = now.toLocaleDateString('en-US', { weekday: 'long', timeZone: tz });

// ====== helpers (ASCII/HTML sanitize) ======
function toAsciiPunct(s) {
  if (!s) return s;
  return s
    .replace(/[\u2018\u2019\u2032]/g, "'")
    .replace(/[\u201C\u201D\u2033]/g, '"')
    .replace(/[\u2013\u2014\u2212]/g, '-')
    .replace(/\u2026/g, '...')
    .replace(/[\u2190-\u21FF]/g, '->')
    .replace(/\u00A0/g, ' ');
}
function stripNonAscii(s) {
  if (!s) return s;
  return s.normalize('NFKD').replace(/[^\x00-\x7F]/g, '');
}
function escapeHtml(s) {
  if (!s) return s;
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}
const hasLink = (s) => /https?:\/\//i.test(s);
const trimX   = (s) => (s.length > X_MAX ? s.slice(0, X_MAX - 1) + '…' : s);

// doğrulama
const violates = (payload) => {
  if (!payload || typeof payload !== 'object') return 'no-payload';
  const t = (payload.title || '').trim();
  const tl = t.toLowerCase();
  if (!t) return 'empty-title';
  if (lastTitles.includes(tl)) return 'title-duplicate';
  if (!allowedCats.includes(payload.category)) return 'bad-category';
  if (!payload.x || hasLink(payload.x)) return 'x-has-link-or-empty';
  if (payload.x.length > X_MAX + 5) return 'x-too-long';
  if (!payload.telegram || !payload.telegram.includes(repoUrl)) return 'telegram-missing-link';
  return null;
};

// ====== OpenAI path (required) ======
async function generateWithOpenAI(){
  if (!apiKey) throw new Error('no-apikey');

  const client = new OpenAI({ apiKey });
  const system =
    `You are a senior web3/social copywriter for QuantumCoin (QC).
Professional, specific, no emojis, no clickbait.`;

  const user =
    `Day: ${dow}
Produce one post for X and one for Telegram.
Choose exactly ONE category from: ${allowedCats.join(' | ')}.
Avoid these recent titles: ${lastTitles.length ? lastTitles.join(' • ') : '—'}.
X: <=260 chars, 2–3 hashtags (#QuantumCoin #QC #Web3 + a topical tag), no link.
Telegram: HTML (<b>Title</b>\\nBlurb\\n\\n${repoUrl}\\n#QuantumCoin #QC #Web3).
Include one concrete metric/artifact/action. Avoid generic phrasing.`;

  let cause = 'init';
  for (let i=0;i<3;i++){
    const res = await client.chat.completions.create({
      model,
      response_format: { type: 'json_object' },
      temperature: 0.9,
      messages: [
        { role:'system', content: system },
        { role:'user',   content: user + (i ? `\nPrevious violation: ${cause}` : '') }
      ],
    });

    let payload = JSON.parse(res.choices[0].message.content);

    // sanitize + rebuild
    let title = stripNonAscii(toAsciiPunct(payload.title || ''));
    let xcopy = stripNonAscii(toAsciiPunct(payload.x || ''));
    let blurb = payload.telegram?.replace(/<[^>]+>/g,'') || 'Progress update.';
    blurb = stripNonAscii(toAsciiPunct(blurb));

    const telegram = `<b>${escapeHtml(title)}</b>\n${escapeHtml(blurb)}\n\n${repoUrl}\n#QuantumCoin #QC #Web3`;

    const candidate = {
      title,
      category: payload.category || '',
      x: trimX(xcopy),
      telegram,
      source: 'openai'
    };

    const v = violates(candidate);
    if (!v) return candidate;
    cause = v;
  }

  throw new Error('openai-constraints-failed');
}

// ====== cache helpers ======
function readCache(){
  if(!fs.existsSync(cacheLast)) return null;
  try {
    const o = JSON.parse(fs.readFileSync(cacheLast, 'utf8'));
    if(o && o.title && o.x && o.telegram){
      return { ...o, source: 'cache' };
    }
  } catch {}
  return null;
}

function writeAll(out){
  // STDOUT
  console.log(JSON.stringify({
    title: out.title,
    category: out.category,
    x: out.x,
    telegram: out.telegram,
    source: out.source,
  }));

  // Disk (main out)
  fs.writeFileSync(path.join(outDir,'last.json'), JSON.stringify(out, null, 2), 'utf8');

  const updated = [...recent, { dt: now.toISOString(), title: out.title, category: out.category }];
  fs.writeFileSync(recentPath, JSON.stringify(updated.slice(-60), null, 2), 'utf8');

  // Cache (AI success only)
  if(out.source === 'openai'){
    fs.writeFileSync(cacheLast, JSON.stringify(out, null, 2), 'utf8');
  }
}

// ====== Orchestrate (AI required, cache allowed) ======
try {
  const out = await generateWithOpenAI();
  writeAll(out);
} catch (e) {
  // AI fail -> cache fallback (still AI-generated content)
  const cached = readCache();
  if(cached){
    writeAll(cached);
  } else {
    // AI must succeed and no cache => hard fail (scheduler will retry)
   console.error(`[gen-content] AI required. OpenAI failed and no cache available. error=${String(e?.message || e)}`);
process.exit(2);

  }
}
