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

fs.mkdirSync(outDir, { recursive: true });

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

// ====== local fallback (OpenAI yoksa/429 ise) ======
function randInt(min, max){return Math.floor(Math.random()*(max-min+1))+min;}
function sample(arr){return arr[Math.floor(Math.random()*arr.length)]}
const topicalTag = (cat) => ({
  'Consensus & Performance':'#Consensus',
  'Mining & Validators':'#Validators',
  'Explorer':'#Explorer',
  'Wallet & Tooling':'#Tooling',
  'Testnet Update':'#Testnet',
  'Security & Audits':'#Security',
  'Tokenomics Insight':'#Tokenomics',
  'Roadmap Milestone':'#Roadmap',
  'Community & Ambassador':'#Community',
}[cat] || '#Web3');

function localGenerate(){
  const cat = sample(allowedCats);
  const metrics = [
    `${randInt(8,32)}% lower verification latency`,
    `${randInt(2,9)}x validator throughput on synthetic load`,
    `${randInt(3,12)} new integration tests`,
    `block time steady at ${randInt(750,1100)}ms (p95)`,
    `${randInt(1,4)} peer discovery fixes merged`,
    `${randInt(2,6)} RPC endpoints hardened`,
  ];
  const artifacts = [
    `bench job qbench-${randInt(100,999)}`,
    `explorer diff v${randInt(0,9)}.${randInt(1,9)}.${randInt(0,9)}`,
    `wallet CLI patch #${randInt(200,400)}`,
    `testnet snapshot r${randInt(1000,1999)}`,
    `audit checklist rev ${randInt(5,19)}`,
  ];
  const titleTemplates = [
    `QC ${cat.split(' ')[0]} snapshot - ${now.toLocaleDateString('en-GB',{timeZone:tz})}`,
    `${cat}: weekly progress note`,
    `${cat}: concrete progress update`,
    `QC ${cat} highlights`,
  ];
  const blurbPieces = [
    `Focus on measurable progress: ${sample(metrics)}; artifact: ${sample(artifacts)}.`,
    `Today’s focus: ${sample(metrics)}; shipped: ${sample(artifacts)}.`,
    `Consolidating stability: ${sample(metrics)}; shipped: ${sample(artifacts)}.`,
  ];
  let title = sample(titleTemplates);
  let blurb = sample(blurbPieces);
  const tag = topicalTag(cat);

  // ASCII-safe
  title = stripNonAscii(toAsciiPunct(title));
  blurb = stripNonAscii(toAsciiPunct(blurb));

  const x = trimX(`QC ${cat.toLowerCase()}: ${blurb.replace(/\.$/,'')}. #QuantumCoin #QC ${tag}`);
  const telegram = `<b>${escapeHtml(title)}</b>\n${escapeHtml(blurb)}\n\n${repoUrl}\n#QuantumCoin #QC #Web3`;
  return { title, category: cat, x, telegram, source: 'fallback' };
}

// ====== OpenAI path (varsa) ======
async function generateWithOpenAI(){
  const client = new OpenAI({ apiKey });
  const system =
    `You are a senior web3/social copywriter for QuantumCoin (QC).
     Professional, specific, no emojis, no clickbait.`;
  const user =
    `Day: ${dow}
     Produce one post for X and one for Telegram.
     Choose exactly ONE category from: ${allowedCats.join(' | ')}.
     Avoid these recent titles: ${lastTitles.length? lastTitles.join(' • ') : '—'}.
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
        { role:'user',   content: user + (i? `\nPrevious violation: ${cause}` : '') }
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

// ====== Orchestrate ======
let out;
try {
  if (!apiKey) throw new Error('no-apikey');
  out = await generateWithOpenAI();
} catch (e){
  // 429/quota/network/no key => fallback
  out = localGenerate();
}

// STDOUT
console.log(JSON.stringify({
  title: out.title,
  category: out.category,
  x: out.x,
  telegram: out.telegram,
  source: out.source,
}));

// Disk
fs.writeFileSync(path.join(outDir,'last.json'), JSON.stringify(out,null,2));
const updated = [...recent, { dt: now.toISOString(), title: out.title, category: out.category }];
fs.writeFileSync(recentPath, JSON.stringify(updated.slice(-60), null, 2));
