// scripts/make-shorts.mjs
// End-to-end: Topic+script (OpenAI → fallback) → TTS (ElevenLabs → OpenAI → Windows SAPI)
// → (optional) SRT → FFmpeg render → YouTube upload
// + NEW: Pool mode (USE_POOL_TTS=1): use tmp/tts.wav AND pool-aligned metadata (title/desc/tags) from pool idx
// + NEW: YouTube schedule (18:00 / 21:00 TR) via QC_SLOT + --schedule / YOUTUBE_SCHEDULE=1

import fs from "fs";
import path from "path";
import { spawn } from "child_process";
import { fileURLToPath } from "url";
import { google } from "googleapis";
import OpenAI from "openai";
import dotenv from "dotenv";
import readline from "readline";
import crypto from "crypto";

dotenv.config({ path: path.resolve(process.cwd(), ".env") });

// ---------- Paths & bootstrap ----------
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const TMP  = path.join(ROOT, "tmp");
fs.mkdirSync(TMP, { recursive: true });

function log(...a){ console.log("[make-shorts]", ...a); }
function warn(...a){ console.warn("[make-shorts][WARN]", ...a); }

// ---------- CLI args ----------
const ARGV = process.argv.slice(2);
const hasFlag = (f) => ARGV.includes(f);
const getArgValue = (key) => {
  const i = ARGV.indexOf(key);
  if (i === -1) return null;
  return ARGV[i+1] ?? null;
};

// ---------- ENV & defaults ----------
const ENV = process.env;

const OPENAI_MODEL = ENV.OPENAI_MODEL || "gpt-4o-mini";
const OPENAI_API_KEY = ENV.OPENAI_API_KEY || "";

const ELEVEN_API_KEY = ENV.ELEVEN_API_KEY || "";
const ELEVEN_VOICE_ID = ENV.ELEVEN_VOICE_ID || "21m00Tcm4TlvDq8ikWAM"; // example voice

const SECRET_PATH = ENV.YOUTUBE_CLIENT_SECRET_PATH || path.join(ROOT, "secrets", "client_secret.json");
const TOKEN_PATH  = ENV.YOUTUBE_TOKEN_PATH || path.join(ROOT, "secrets", "youtube_token.json");

// PRIVACY only used when NOT scheduled
const PRIVACY = (getArgValue("--privacy") || ENV.YOUTUBE_PRIVACY || "unlisted").toLowerCase();

// Render settings
const CANVAS_SIZE = ENV.SHORTS_SIZE || "1080x1920";
const BG_COLOR_HEX = ENV.SHORTS_BG || "0x111827";
const WAVE_HEIGHT = Number(ENV.SHORTS_WAVE_HEIGHT || 300);
const WAVE_MARGIN_BOTTOM = Number(ENV.SHORTS_WAVE_MARGIN || 120);
const LOGO_PATH = path.join(ROOT, "assets", "logo.png");

// Subtitles control
const DISABLE_SUBTITLES = (ENV.DISABLE_SUBTITLES || "").trim() === "1";

// Schedule controls
const SCHEDULE_ENABLED =
  hasFlag("--schedule") ||
  (ENV.YOUTUBE_SCHEDULE || "").toString().trim() === "1";

// NO_UPLOAD_DRY_RUN_V1
// Allows local render tests without uploading to YouTube.
// Usage: --no-upload OR YOUTUBE_UPLOAD=0 OR NO_UPLOAD=1
const NO_UPLOAD =
  hasFlag("--no-upload") ||
  (ENV.YOUTUBE_UPLOAD || "").toString().trim() === "0" ||
  (ENV.NO_UPLOAD || "").toString().trim() === "1";

const QC_TODAY = (ENV.QC_TODAY || "").trim();            // yyyy-MM-dd (optional)
const QC_SLOT  = (ENV.QC_SLOT || "").toString().trim();  // 1 or 2 (optional)
const QC_TZ_OFFSET = (ENV.QC_TZ_OFFSET || "+03:00").trim(); // TR default

// Pool controls
const USE_POOL_TTS = (ENV.USE_POOL_TTS || "").toString().trim() === "1";
const POOL_WAV_PATH = path.join(TMP, "tts.wav"); // run-daily.ps1 already prepares this
const POOL_META_CANDIDATES = [
  path.join(TMP, "pool_meta.json"),
  path.join(TMP, "pool.json"),
  path.join(ROOT, "assets", "pool", "meta.json"),
  path.join(ROOT, "assets", "pool", "items.json"),
];

// ---------- small utils ----------
function safeForFFmpegSubPath(p){
  // ffmpeg subtitles filter on Windows needs:
  // - backslash -> slash
  // - escape single quotes
  // - escape drive letter colon (C:\ -> C\:/) because ":" is option separator in filtergraph
  let s = p.replace(/\\/g, "/").replace(/'/g, "\\'");
  s = s.replace(/^([A-Za-z]):\//, "$1\\:/"); // C:/ -> C\:/ (critical fix)
  return s;
}

async function checkCmd(cmd, args = ["-version"]){
  return new Promise((resolve) => {
    const p = spawn(cmd, args, { stdio: ["ignore", "ignore", "ignore"] });
    p.on("error", () => resolve(false));
    p.on("close", (code) => resolve(code === 0));
  });
}

function assertFile(p, hint){
  if (!fs.existsSync(p)) {
    throw new Error(`${hint} bulunamadı: ${p}`);
  }
}

function ensureDirForFile(filePath){
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
}

function ask(question){
  return new Promise((resolve) => {
    const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
    rl.question(question, (answer) => {
      rl.close();
      resolve((answer || "").trim());
    });
  });
}

function readYouTubeClientSecret(secretPath){
  const raw = JSON.parse(fs.readFileSync(secretPath, "utf8"));
  const sec = raw.installed || raw.web;
  if (!sec?.client_id || !sec?.client_secret) {
    throw new Error("client_secret.json formatı beklenmedik (installed/web yok).");
  }
  return sec;
}

function sha256Hex(s){
  return crypto.createHash("sha256").update(String(s)).digest("hex");
}

function seededPick(arr, seed){
  if (!arr.length) return null;
  const h = sha256Hex(seed);
  const n = parseInt(h.slice(0, 8), 16);
  return arr[n % arr.length];
}

function getTodayYYYYMMDD(){
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth()+1).padStart(2,"0");
  const dd = String(d.getDate()).padStart(2,"0");
  return `${yyyy}-${mm}-${dd}`;
}

function buildPublishAtISO(){
  // Returns ISO 8601 like 2026-02-02T18:00:00+03:00
  const day = QC_TODAY || getTodayYYYYMMDD();

  const slot = (QC_SLOT || "1").trim();
  const hour = slot === "2" ? "21" : "18"; // slot1=18, slot2=21
  const minute = "00";
  const second = "00";

  if (!/^\d{4}-\d{2}-\d{2}$/.test(day)) {
    throw new Error(`QC_TODAY formatı hatalı. Beklenen yyyy-MM-dd. Gelen: ${day}`);
  }
  if (!/^[\+\-]\d{2}:\d{2}$/.test(QC_TZ_OFFSET)) {
    throw new Error(`QC_TZ_OFFSET formatı hatalı. Beklenen +03:00 gibi. Gelen: ${QC_TZ_OFFSET}`);
  }

  return `${day}T${hour}:${minute}:${second}${QC_TZ_OFFSET}`;
}

async function runFFmpeg(args){
  return new Promise((resolve, reject)=>{
    log("ffmpeg", args.join(" "));
    const p = spawn("ffmpeg", args, {stdio:["ignore","inherit","inherit"]});
    p.on("error", (err) => reject(err));
    p.on("close",(code)=> code===0 ? resolve() : reject(new Error("ffmpeg exit "+code)));
  });
}

async function runFFprobeDuration(audioPath){
  return new Promise((resolve, reject)=>{
    const p = spawn("ffprobe", ["-v","error","-show_entries","format=duration","-of","default=noprint_wrappers=1:nokey=1", audioPath], {stdio:["ignore","pipe","pipe"]});
    let out=""; p.stdout.on("data",d=>out+=d.toString());
    let err=""; p.stderr.on("data",d=>err+=d.toString());
    p.on("error", (e) => reject(e));
    p.on("close",(code)=> code===0 ? resolve(Math.max(0, parseFloat(out.trim()||"0"))) : reject(new Error(err||"ffprobe failed")));
  });
}

function toSrt(cues){
  let idx=1, out="";
  const pad=(n)=> String(Math.floor(n)).padStart(2,"0");
  const ms=(t)=>{
    const h = Math.floor(t/3600);
    const m = Math.floor((t%3600)/60);
    const s = Math.floor(t%60);
    const millis = Math.round((t - Math.floor(t))*1000);
    return `${pad(h)}:${pad(m)}:${pad(s)},${String(millis).padStart(3,"0")}`;
  };
  for(const c of cues){
    out += `${idx++}\r\n${ms(c.start)} --> ${ms(c.end)}\r\n${c.text}\r\n\r\n`;
  }
  return out;
}

function splitCaptions(scriptText, totalSec){
  const raw = scriptText
    .replace(/\s+/g," ")
    .trim()
    .split(/(?<=[\.\!\?])\s+|(?<=,)\s+|(?<=;)\s+/g)
    .filter(s=>s && s.length>2);

  const lines = raw.length>0 ? raw : [scriptText];
  const minDur = 1.2, maxDur = 3.0;
  const avg = Math.min(maxDur, Math.max(minDur, totalSec / lines.length));
  const cues=[];
  let t=0;
  for(const l of lines){
    const dur = Math.min(maxDur, Math.max(minDur, avg));
    cues.push({ start:t, end:Math.min(totalSec, t+dur), text:l });
    t += dur;
    if (t >= totalSec) break;
  }
  if(cues.length && cues[cues.length-1].end < totalSec) cues[cues.length-1].end = totalSec;
  return cues;
}

function fileOk(p, minBytes=2048){
  try {
    return fs.existsSync(p) && fs.statSync(p).size >= minBytes;
  } catch { return false; }
}

function readJsonIfExists(p){
  try{
    if(!fs.existsSync(p)) return null;
    return JSON.parse(fs.readFileSync(p, "utf8"));
  } catch {
    return null;
  }
}
function getPoolIdx() {
  // 1) ENV (run-daily’den gelecek)
  const envIdx = parseInt((ENV.POOL_IDX || "").trim(), 10);
  if (Number.isFinite(envIdx) && envIdx > 0) return envIdx;

  // 2) tmp/pool_meta.json (run-daily yazacak)
  const tmpMeta = readJsonIfExists(path.join(TMP, "pool_meta.json"));
  const tmpIdx = Number(tmpMeta?.idx ?? tmpMeta?.current ?? tmpMeta?.index);
  if (Number.isFinite(tmpIdx) && tmpIdx > 0) return tmpIdx;

  // 3) state_slotX.json (SON ÇARE: next-1)
  const slot = (QC_SLOT || "1").trim() || "1";
  const st = readJsonIfExists(path.join(ROOT, "assets", "pool", `state_slot${slot}.json`));
  if (st) {
    const stIdx = Number(st?.idx ?? st?.current ?? st?.index);
    if (Number.isFinite(stIdx) && stIdx > 0) return stIdx;

    const next = Number(st?.next);
    if (Number.isFinite(next) && next > 0) return Math.max(1, next - 1);
  }

  return null;
}

// ---------- Content bank (used for offline + pool metadata alignment) ----------
const BANK = [
  {
    title: "What drives crypto in the next 30 days?",
    description: "60s crypto explainer — no hype, just signals.",
    hashtags: ["#shorts","#crypto","#QuantumCoin"],
    bullets: [
      "Watch macro: rates, liquidity and risk appetite.",
      "Track network activity (users/fees) not just price.",
      "Mind catalysts: upgrades, ETF flows, unlocks.",
      "Stick to risk controls; volatility cuts both ways.",
      "Follow QuantumCoin for daily insights."
    ]
  },
  {
    title: "Understanding Bitcoin funding and liquidity",
    description: "60s crypto explainer — no hype, just signals.",
    hashtags: ["#shorts","#crypto","#QuantumCoin"],
    bullets: [
      "Funding rates hint at long/short bias—extremes fade.",
      "Stablecoin growth can indicate sidelined demand.",
      "On-chain fees show real demand for blockspace.",
      "Zoom out to weekly trends, not 1-min noise.",
      "Follow QuantumCoin for daily insights."
    ]
  },
  {
    title: "3 rules for surviving crypto volatility",
    description: "60s crypto explainer — no hype, just signals.",
    hashtags: ["#shorts","#crypto","#QuantumCoin"],
    bullets: [
      "Position small; scale with confirmation.",
      "Define invalidation before you enter.",
      "Avoid leverage into major events.",
      "Context beats headlines every time.",
      "Follow QuantumCoin for daily insights."
    ]
  },
  {
    title: "Crypto security checklist in 60 seconds",
    description: "60s crypto explainer — no hype, just signals.",
    hashtags: ["#shorts","#crypto","#QuantumCoin"],
    bullets: [
      "Use a hardware wallet for long-term holdings.",
      "Never reuse passwords—use a password manager.",
      "Beware fake airdrops and wallet-drainers.",
      "Verify URLs and contract addresses twice.",
      "Follow QuantumCoin for daily insights."
    ]
  },
  {
    title: "How to read on-chain activity without hype",
    description: "60s crypto explainer — no hype, just signals.",
    hashtags: ["#shorts","#crypto","#QuantumCoin"],
    bullets: [
      "Fees and active addresses show real usage signals.",
      "Watch stablecoin supply for demand shifts.",
      "Track exchange inflows/outflows for pressure.",
      "Compare weekly trends, not single spikes.",
      "Follow QuantumCoin for daily insights."
    ]
  }
];

function getPoolIdxFromState(){
  // State file is managed by your pool system; we only *read* it to align metadata with voice_000X.
  const slot = (QC_SLOT || "1").trim() || "1";
  const p = path.join(ROOT, "assets", "pool", `state_slot${slot}.json`);
  const j = readJsonIfExists(p);
  const idx = Number(j?.idx ?? j?.current ?? j?.index ?? 0);
  return Number.isFinite(idx) && idx > 0 ? idx : null;
}

function resolvePoolMeta(){
  // Priority:
  // 1) ENV (manual override)
  // 2) tmp/pool_meta.json or tmp/pool.json
  // 3) assets/pool/meta.json or assets/pool/items.json
  // 4) fallback: BANK by pool idx (voice_0001 -> BANK[0], voice_0002 -> BANK[1], ...)
  const envTitle = (ENV.POOL_TITLE || "").trim();
  if (envTitle) {
    const desc = (ENV.POOL_DESCRIPTION || "60s crypto explainer — no hype, just signals.").trim().slice(0,150);
    const tagsRaw = (ENV.POOL_HASHTAGS || ENV.POOL_TAGS || "").trim();
    const hashtags = tagsRaw
      ? tagsRaw.split(/[,\s]+/g).filter(Boolean)
      : ["#shorts","#crypto","#QuantumCoin"];
    const script = (ENV.POOL_SCRIPT || "").trim() || "";
    return {
      title: envTitle.slice(0,80),
      description: desc,
      hashtags,
      script
    };
  }

  const poolIdx = getPoolIdx();

  // 2) tmp/meta candidates
  for (const p of POOL_META_CANDIDATES) {
    const j = readJsonIfExists(p);
    if (!j) continue;

    const pickFromItem = (it) => ({
      title: String(it.title || "QuantumCoin Shorts").slice(0,80),
      description: String(it.description || "60s crypto explainer — no hype, just signals.").slice(0,150),
      hashtags: Array.isArray(it.hashtags) ? it.hashtags : ["#shorts","#crypto","#QuantumCoin"],
      script: String(it.script || "")
    });

    // Shape A) { title, description, hashtags, script }
    if (j?.title) return pickFromItem(j);

    // Shape B) { items:[...] }  or  C) [...]
    const arr = Array.isArray(j) ? j : (Array.isArray(j?.items) ? j.items : null);
    if (arr && arr.length) {
      let it = null;
      if (poolIdx) it = arr.find(x => Number(x?.idx) === Number(poolIdx)) || null;
      if (!it) it = arr[0];
      if (it?.title) return pickFromItem(it);
    }
  }

  // 4) fallback: BANK by poolIdx (1-based)
  if (poolIdx && poolIdx >= 1 && poolIdx <= BANK.length) {
    const pick = BANK[poolIdx - 1];
    const script = Array.isArray(pick.bullets) ? pick.bullets.join(" ") : "";
    return {
      title: pick.title.slice(0,80),
      description: pick.description.slice(0,150),
      hashtags: pick.hashtags,
      script
    };
  }

  // 5) safe placeholder (poolIdx var ama BANK dışıysa)
  if (poolIdx) {
    return {
      title: `QuantumCoin Shorts (Pool #${poolIdx})`.slice(0, 80),
      description: `Pool item #${poolIdx} (pre-recorded).`.slice(0, 150),
      hashtags: ["#shorts", "#crypto", "#QuantumCoin"],
      script: ""
    };
  }

  // absolute fallback
  return {
    title: "QuantumCoin Shorts".slice(0, 80),
    description: "60s crypto explainer — no hype, just signals.".slice(0, 150),
    hashtags: ["#shorts", "#crypto", "#QuantumCoin"],
    script: ""
  };
}

// ---------- 1) OpenAI: topic + script (with fallback) ----------
async function makeScriptAIOrFallback(){
  const fallback = () => {
    const seed = `fallback:${QC_TODAY || getTodayYYYYMMDD()}:${QC_SLOT || "1"}`;
    const pick = seededPick(BANK, seed) || BANK[0];
    const script = pick.bullets.join(" ");
    return {
      title: pick.title.slice(0,80),
      description: pick.description.slice(0,150),
      hashtags: pick.hashtags,
      script
    };
  };

  if ((process.env.OFFLINE_SHORTS || "") === "1") return fallback();

  try{
    if(!OPENAI_API_KEY) throw new Error("OPENAI_API_KEY missing");

    const client = new OpenAI({ apiKey: OPENAI_API_KEY });

    const hint = `Today=${QC_TODAY || getTodayYYYYMMDD()} Slot=${QC_SLOT || "1"}`;

    const sys = `You are a concise YouTube Shorts writer for a crypto news channel named QuantumCoin.
- Output short, punchy English narration (90–120 words).
- Avoid hype or financial advice. No price targets. Be factual and clear.
- Use ONE concrete theme and keep it actionable.
- End with a single-sentence CTA: "Follow QuantumCoin for daily insights."`;

    const user = `Write a new English Shorts voiceover about a timely cryptocurrency topic (choose one concrete theme yourself).
Avoid repeating yesterday's idea. Context: ${hint}

Return **only** this JSON object:

{
  "title": "<max 80 chars>",
  "description": "<max 150 chars>",
  "hashtags": ["#shorts","#crypto","#QuantumCoin"],
  "script": "<90-120 words of voiceover>"
}`;

    const r = await client.chat.completions.create({
      model: OPENAI_MODEL,
      temperature: 0.8,
      messages: [{role:"system",content:sys},{role:"user",content:user}]
    });

    const txt = r.choices?.[0]?.message?.content?.trim() || "";
    let json;
    try { json = JSON.parse(txt.replace(/^```json|```$/g,"")); }
    catch { throw new Error("Parse fail"); }

    if(!json?.script) throw new Error("No script");
    json.title = (json.title||"QuantumCoin Shorts").slice(0,80);
    json.description = (json.description||"Crypto insight in 60s").slice(0,150);
    if(!Array.isArray(json.hashtags)) json.hashtags = ["#shorts","#crypto","#QuantumCoin"];
    return json;

  }catch(_){
    return fallback();
  }
}


function resolveContent(){
  // IMPORTANT FIX:
  // If pool WAV is used, also use pool-aligned metadata (title/desc/tags/script),
  // so the YouTube title matches the pre-recorded voice.
  const hasPoolWav = USE_POOL_TTS && fileOk(POOL_WAV_PATH, 2048);
  if (hasPoolWav) {
    const meta = resolvePoolMeta();
    // If script empty and subtitles off, it's ok. If subtitles on, we still try to have script.
    if (!meta.script) meta.script = meta.title; // minimal safe
    return { meta, mode: "pool" };
  }
  return { meta: null, mode: "ai" };
}

// ---------- 2) TTS with multi-fallback ----------
async function makeTTS(text, outMp3Path){
  // 2.0 Pool WAV -> MP3 (skip TTS providers)
  const hasPoolWav = USE_POOL_TTS && fileOk(POOL_WAV_PATH, 2048);
  if (hasPoolWav) {
    log("USE_POOL_TTS=1 -> using existing tmp/tts.wav (skip TTS generation)");
    await runFFmpeg(["-y","-i", POOL_WAV_PATH, "-codec:a","libmp3lame","-b:a","128k", outMp3Path]);
    return outMp3Path;
  }

  // 2.1 ElevenLabs
  if (ELEVEN_API_KEY) {
    try{
      const voice = ELEVEN_VOICE_ID || "21m00Tcm4TlvDq8ikWAM";
      const url = `https://api.elevenlabs.io/v1/text-to-speech/${voice}/stream`;
      const resp = await fetch(url, {
        method:"POST",
        headers:{ "xi-api-key": ELEVEN_API_KEY, "Content-Type":"application/json" },
        body: JSON.stringify({
          text,
          voice_settings:{ stability:0.35, similarity_boost:0.75 },
          model_id:"eleven_multilingual_v2",
          output_format:"mp3_44100_128"
        })
      });
      if(!resp.ok) throw new Error("ElevenLabs TTS failed: " + resp.status);
      const buf = Buffer.from(await resp.arrayBuffer());
      fs.writeFileSync(outMp3Path, buf);
      return outMp3Path;
    }catch(e){
      warn("ElevenLabs TTS error:", e?.message || e);
    }
  }

  // 2.2 OpenAI TTS
  if (OPENAI_API_KEY) {
    try{
      const client = new OpenAI({ apiKey: OPENAI_API_KEY });
      const res = await client.audio.speech.create({
        model: "gpt-4o-mini-tts",
        voice: "alloy",
        input: text,
        format: "mp3"
      });
      const buf = Buffer.from(await res.arrayBuffer());
      fs.writeFileSync(outMp3Path, buf);
      return outMp3Path;
    }catch(e){
      warn("OpenAI TTS error:", e?.message || e);
    }
  }

  // 2.3 Windows SAPI → WAV, then MP3
  const txtPath = path.join(TMP, "tts.txt");
  const wavPath = path.join(TMP, "tts_sapi.wav");
  fs.writeFileSync(txtPath, text, "utf8");

  await new Promise((resolve,reject)=>{
    const ps = spawn("pwsh", ["-NoProfile","-Command", `
      Add-Type -AssemblyName System.Speech;
      $s=New-Object System.Speech.Synthesis.SpeechSynthesizer;
      $s.Rate=0; $s.Volume=100;
      try { $s.SelectVoice('Microsoft David Desktop'); } catch {}
      $s.SetOutputToWaveFile('${wavPath.replace(/\\/g,"/")}');
      $t=[IO.File]::ReadAllText('${txtPath.replace(/\\/g,"/")}');
      $s.Speak($t); $s.Dispose();
    `], {stdio:["ignore","inherit","inherit"]});
    ps.on("close",(c)=> c===0 ? resolve() : reject(new Error("SAPI TTS failed")));
  });

  await runFFmpeg(["-y","-i", wavPath, "-codec:a","libmp3lame","-b:a","128k", outMp3Path]);
  return outMp3Path;
}

// ---------- 3) Render (waveform + optional BG video/image + optional logo + (optional) burned subtitles) ----------
async function renderVideo(audioPath, srtPath, outPath, durationSec){
  const dur = Math.max(10, Math.ceil(durationSec));
  const [W,H] = CANVAS_SIZE.split("x").map(n=>parseInt(n,10));
  const bgMp4 = path.join(ROOT, "assets", "bg.mp4");
  const bgImg = path.join(ROOT, "assets", "bg.jpg");
  const hasLogo = fs.existsSync(LOGO_PATH);
// Logo placement (top-left)
const LOGO_W = Number(ENV.SHORTS_LOGO_W || 220); // width px
const LOGO_X = Number(ENV.SHORTS_LOGO_X || 40);  // left padding
const LOGO_Y = Number(ENV.SHORTS_LOGO_Y || 40);  // top padding
  const wave = `[0:a]showwaves=s=${W}x${WAVE_HEIGHT}:mode=cline:rate=25:colors=FFFFFF@0.9,format=rgba[wave];`;
 const logoPrep = hasLogo ? `[2:v]scale=${LOGO_W}:-1:force_original_aspect_ratio=decrease,format=rgba[logo];` : "";

  let args = ["-y", "-i", audioPath];
  let filter = wave;

  const overlayY = Math.max(0, H - WAVE_HEIGHT - WAVE_MARGIN_BOTTOM);

  const style = "FontName=Arial,Fontsize=54,PrimaryColour=&H00FFFFFF&,OutlineColour=&H000000&,BorderStyle=1,Outline=2,Shadow=0,Alignment=2,MarginV=80";
  const subFilter = `subtitles='${safeForFFmpegSubPath(srtPath)}':force_style='${style}'`;

  const wantSubs = !DISABLE_SUBTITLES;

  if (!wantSubs) {
    log("Subtitles disabled (DISABLE_SUBTITLES=1). Rendering without burned captions.");
  }

  if (fs.existsSync(bgMp4)) {
    args.push("-stream_loop","-1","-t", String(dur), "-i", bgMp4);
    if (hasLogo) args.push("-i", LOGO_PATH);

   filter +=
  `[1:v]scale=${W}:${H}:force_original_aspect_ratio=increase,crop=${W}:${H},setsar=1[vbg];` +
  `[vbg][wave]overlay=(W-w)/2:${overlayY}[v1];` +
  (hasLogo ? (logoPrep + `[v1][logo]overlay=${LOGO_X}:${LOGO_Y}[v2];`) : "");


    if (wantSubs) {
      filter += `[${hasLogo ? "v2" : "v1"}]${subFilter}[vout]`;
    } else {
      filter += `[${hasLogo ? "v2" : "v1"}]format=rgba[vout]`;
    }

    args.push(
      "-filter_complex", filter,
      "-map", "[vout]", "-map", "0:a",
      "-c:v","libx264","-pix_fmt","yuv420p","-r","30",
      "-c:a","aac","-b:a","128k",
      "-shortest",
      outPath
    );

  } else if (fs.existsSync(bgImg)) {
    args.push("-loop","1","-t", String(dur), "-i", bgImg);
    if (hasLogo) args.push("-i", LOGO_PATH);

    filter +=
      `[1:v]scale=${W}:${H}:force_original_aspect_ratio=increase,crop=${W}:${H},setsar=1[vbg];` +
      `[vbg][wave]overlay=(W-w)/2:${overlayY}[v1];` +
      (hasLogo ? (logoPrep + `[v1][logo]overlay=${LOGO_X}:${LOGO_Y}[v2];`) : "")

    if (wantSubs) {
      filter += `[${hasLogo ? "v2" : "v1"}]${subFilter}[vout]`;
    } else {
      filter += `[${hasLogo ? "v2" : "v1"}]format=rgba[vout]`;
    }

    args.push(
      "-filter_complex", filter,
      "-map", "[vout]", "-map", "0:a",
      "-c:v","libx264","-pix_fmt","yuv420p","-r","30",
      "-c:a","aac","-b:a","128k",
      "-shortest",
      outPath
    );

  } else {
    args.push("-f","lavfi","-t", String(dur), "-i", `color=c=${BG_COLOR_HEX}:s=${CANVAS_SIZE}`);
    if (hasLogo) args.push("-i", LOGO_PATH);

`[1:v]format=rgba[vbg];` +
`[vbg][wave]overlay=(W-w)/2:${overlayY}[v1];` +
(hasLogo ? (logoPrep + `[v1][logo]overlay=${LOGO_X}:${LOGO_Y}[v2];`) : "");

    if (wantSubs) {
      filter += `[${hasLogo ? "v2" : "v1"}]${subFilter}[vout]`;
    } else {
      filter += `[${hasLogo ? "v2" : "v1"}]format=rgba[vout]`;
    }

    args.push(
      "-filter_complex", filter,
      "-map", "[vout]", "-map", "0:a",
      "-c:v","libx264","-pix_fmt","yuv420p","-r","30",
      "-c:a","aac","-b:a","128k",
      "-shortest",
      outPath
    );
  }

  try {
    await runFFmpeg(args);
  } catch(e) {
    if (!wantSubs) throw e;

    warn("Subtitle render failed, retrying without subtitles. Reason:", e?.message || e);

    const i = args.indexOf("-filter_complex");
    if (i > -1) {
      let f = args[i+1];
      f = f.replace(/\](subtitles=)'[^']*':force_style='[^']*'\[vout\]/, "]format=rgba[vout]");
      args[i+1] = f;
    }
    await runFFmpeg(args);
  }
}

// ---------- YouTube token bootstrap ----------
async function ensureYouTubeToken(oauth2){
  if (fs.existsSync(TOKEN_PATH)) {
    const tok = JSON.parse(fs.readFileSync(TOKEN_PATH, "utf8"));
    oauth2.setCredentials(tok);
    return;
  }

  log("YouTube OAuth token not found. Starting one-time authorization flow...");
  const scopes = ["https://www.googleapis.com/auth/youtube.upload"];
  const authUrl = oauth2.generateAuthUrl({
    access_type: "offline",
    scope: scopes,
    prompt: "consent",
  });

  console.log("\n=== YOUTUBE AUTH REQUIRED ===");
  console.log("1) Open this URL in your browser:\n");
  console.log(authUrl);
  console.log("\n2) Approve, then copy the 'code' parameter from the redirected URL.\n");

  const code = await ask("Paste code here: ");
  if (!code) throw new Error("OAuth code boş geldi.");

  const { tokens } = await oauth2.getToken(code);
  oauth2.setCredentials(tokens);

  ensureDirForFile(TOKEN_PATH);
  fs.writeFileSync(TOKEN_PATH, JSON.stringify(tokens, null, 2), "utf8");
  log("Saved new token to:", TOKEN_PATH);
}

// ---------- 4) Upload (googleapis) ----------
async function uploadToYouTube(filePath, title, desc, privacy){
  assertFile(SECRET_PATH, "YOUTUBE_CLIENT_SECRET_PATH");

  const sec = readYouTubeClientSecret(SECRET_PATH);
  const redirectUri = (process.env.GOOGLE_REDIRECT_URI || "http://localhost").trim();
  const oauth2 = new google.auth.OAuth2(sec.client_id, sec.client_secret, redirectUri);

  await ensureYouTubeToken(oauth2);

  const youtube = google.youtube({ version:"v3", auth: oauth2 });

  let status = { privacyStatus: privacy };

  if (SCHEDULE_ENABLED) {
    const publishAt = buildPublishAtISO();
    status = { privacyStatus: "private", publishAt };
    log("YouTube schedule ON -> publishAt:", publishAt, "(privacyStatus=private until publish)");
  }

  const res = await youtube.videos.insert({
    part: ["snippet","status"],
    requestBody: {
      snippet: { title, description: desc },
      status
    },
    media: { body: fs.createReadStream(filePath) }
  });

  return `https://www.youtube.com/watch?v=${res.data.id}`;
}

// ---------- MAIN ----------
(async () => {
  try {
    if (!await checkCmd("ffmpeg")) warn("ffmpeg bulunamadı; PATH'e ekli olduğundan emin ol.");
    if (!await checkCmd("ffprobe")) warn("ffprobe bulunamadı; PATH'e ekli olduğundan emin ol.");

    const content = resolveContent();

    let gen;
    if (content.mode === "pool") {
      gen = content.meta;
      log("POOL MODE ON -> using pool-aligned meta (no OpenAI script generation)");
    } else {
      log("Generating topic & script...");
      gen = await makeScriptAIOrFallback();
    }

    log("Title:", gen.title);

    const voicePath = path.join(TMP, "short-voice.mp3");

    log("TTS...");
    await makeTTS(gen.script, voicePath);

    log("ffprobe duration...");
    const dur = await runFFprobeDuration(voicePath);
    log("Audio duration:", dur.toFixed(2), "s");

    const srtPath = path.join(TMP, "shorts.srt");
    if (!DISABLE_SUBTITLES) {
      log("Build SRT...");
      const cues = splitCaptions(gen.script, dur);
      const srt = toSrt(cues);
      fs.writeFileSync(srtPath, srt, "utf8");
    } else {
      fs.writeFileSync(srtPath, "", "utf8");
    }

    const outPath = path.join(TMP, "shorts-out.mp4");
    log("Rendering video...");
    await renderVideo(voicePath, srtPath, outPath, dur);

    const hashtags = Array.isArray(gen.hashtags) ? gen.hashtags.join(" ") : "#shorts #crypto #QuantumCoin";
    const desc = `${gen.description}\n\n${hashtags}`;

    let url = "";

    if (NO_UPLOAD) {
      log("NO_UPLOAD_DRY_RUN_V1 -> render completed; YouTube upload skipped");
      url = "DRY_RUN_NO_UPLOAD";
    } else {
      log("Uploading to YouTube...", SCHEDULE_ENABLED ? "(scheduled)" : PRIVACY);
      url = await uploadToYouTube(outPath, gen.title, desc, PRIVACY);
    }

    console.log("\n=== DONE ===");
    console.log("Video:", outPath);
    console.log("YouTube:", url);

  } catch (e) {
    const msg = e?.response?.data ?? e?.message ?? e;
    console.error("FATAL:", msg);
    process.exit(1);
  }
})();
