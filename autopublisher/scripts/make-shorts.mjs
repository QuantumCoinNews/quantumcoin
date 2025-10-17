// scripts/make-shorts.mjs
// End-to-end: Topic+script (OpenAI → fallback) → TTS (ElevenLabs → OpenAI → Windows SAPI) → SRT → FFmpeg render → YouTube upload

import fs from "fs";
import path from "path";
import { spawn } from "child_process";
import { fileURLToPath } from "url";
import { google } from "googleapis";
import OpenAI from "openai";
import dotenv from "dotenv";

dotenv.config({ path: path.resolve(process.cwd(), ".env") });

// ---------- Paths & bootstrap ----------
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const TMP  = path.join(ROOT, "tmp");
fs.mkdirSync(TMP, { recursive: true });

function log(...a){ console.log("[make-shorts]", ...a); }
function warn(...a){ console.warn("[make-shorts][WARN]", ...a); }

// ---------- ENV & defaults ----------
const ENV = process.env;
const OPENAI_MODEL = ENV.OPENAI_MODEL || "gpt-4o-mini";
const OPENAI_API_KEY = ENV.OPENAI_API_KEY || "";
const ELEVEN_API_KEY = ENV.ELEVEN_API_KEY || "";
const ELEVEN_VOICE_ID = ENV.ELEVEN_VOICE_ID || "21m00Tcm4TlvDq8ikWAM"; // örnek "Rachel"

const SECRET_PATH = ENV.YOUTUBE_CLIENT_SECRET_PATH || path.join(ROOT, "secrets", "client_secret.json");
const TOKEN_PATH  = ENV.YOUTUBE_TOKEN_PATH || path.join(ROOT, "secrets", "youtube_token.json");
const PRIVACY = (process.argv.includes("--privacy") ? process.argv[process.argv.indexOf("--privacy")+1] : ENV.YOUTUBE_PRIVACY) || "unlisted";

// Render ayarları
const CANVAS_SIZE = ENV.SHORTS_SIZE || "1080x1920";
const BG_COLOR_HEX = ENV.SHORTS_BG || "0x111827"; // koyu gri/mavi
const WAVE_HEIGHT = Number(ENV.SHORTS_WAVE_HEIGHT || 300);
const WAVE_MARGIN_BOTTOM = Number(ENV.SHORTS_WAVE_MARGIN || 120);
const LOGO_PATH = path.join(ROOT, "assets", "logo.png");

// ---------- small utils ----------
function safeForFFmpegSubPath(p){
  // ffmpeg subtitles filtresi için: backslash → slash, tek tırnak kaçış
  return p.replace(/\\/g, "/").replace(/'/g, "\\'");
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
  // cues: [{start,end,text}]
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
  // metni cümle/parçaya böl, süreye eşit yay
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

// ---------- 1) OpenAI: topic + script (with fallback) ----------
async function makeScript(){
  // Offline fallback (OpenAI yoksa/429 olursa)
  const fallback = () => {
    const bank = [
      {
        title: "What drives crypto in the next 30 days?",
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
        bullets: [
          "Position small; scale with confirmation.",
          "Define invalidation before you enter.",
          "Avoid leverage into major events.",
          "Context beats headlines every time.",
          "Follow QuantumCoin for daily insights."
        ]
      }
    ];
    const pick = bank[Math.floor(Math.random()*bank.length)];
    const script = pick.bullets.join(" ");
    return {
      title: pick.title.slice(0,80),
      description: "60s crypto explainer — no hype, just signals.",
      hashtags: ["#shorts","#crypto","#QuantumCoin"],
      script
    };
  };

  // İstersen offline’ı zorla: .env -> OFFLINE_SHORTS=1
  if ((process.env.OFFLINE_SHORTS || "") === "1") return fallback();

  try{
    if(!OPENAI_API_KEY) throw new Error("OPENAI_API_KEY missing");
    const client = new OpenAI({ apiKey: OPENAI_API_KEY });
    const sys = `You are a concise YouTube Shorts writer for a crypto news channel named QuantumCoin.
- Output short, punchy English narration (90–120 words).
- Avoid hype or financial advice. No price targets. Be factual and clear.
- End with a single-sentence CTA like: "Follow QuantumCoin for daily insights."`;
    const user = `Write a new English Shorts voiceover about a timely cryptocurrency topic (choose one concrete theme yourself).
Return **only** this JSON object:

{
  "title": "<max 80 chars>",
  "description": "<max 150 chars>",
  "hashtags": ["#shorts","#crypto","#QuantumCoin"],
  "script": "<90-120 words of voiceover>"
}`;

    const r = await client.chat.completions.create({
      model: OPENAI_MODEL,
      temperature: 0.7,
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
    // 429 / key yok / parse hatası → offline üret
    return fallback();
  }
}

// ---------- 2) TTS with multi-fallback ----------
async function makeTTS(text, outMp3Path){
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
      // devam — OpenAI TTS / SAPI'ye düş
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
      // devam — Windows SAPI'ye düş
    }
  }

  // 2.3 Windows SAPI → WAV üret, sonra MP3'e çevir
  const txtPath = path.join(TMP, "tts.txt");
  const wavPath = path.join(TMP, "tts.wav");
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

// ---------- 3) Render (waveform + optional BG video/image + optional logo + subtitles) ----------
async function renderVideo(audioPath, srtPath, outPath, durationSec){
  const dur = Math.max(10, Math.ceil(durationSec)); // en az 10 sn
  const [W,H] = CANVAS_SIZE.split("x").map(n=>parseInt(n,10));
  const bgMp4 = path.join(ROOT, "assets", "bg.mp4");
  const bgImg = path.join(ROOT, "assets", "bg.jpg");
  const hasLogo = fs.existsSync(LOGO_PATH);

  // Subtitles (ASS) stil dizgesi
  const style = "FontName=Arial,Fontsize=54,PrimaryColour=&H00FFFFFF&,OutlineColour=&H000000&,BorderStyle=1,Outline=2,Shadow=0,Alignment=2,MarginV=80";
  const subFilter = `subtitles='${safeForFFmpegSubPath(srtPath)}':force_style='${style}'`;

  // 0:a → waveform video üret
  const wave = `[0:a]showwaves=s=${W}x${WAVE_HEIGHT}:mode=cline:rate=25:colors=FFFFFF@0.9,format=rgba[wave];`;

  // Logo (varsa) hazırlık
  const logoPrep = hasLogo ? `[2:v]scale=720:-1:force_original_aspect_ratio=decrease,format=rgba[logo];` : "";
  const logoOverlay = hasLogo ? `[v2][logo]overlay=30:30[v3];` : "";

  // Girişler: 0 = audio, 1 = background (varsa), 2 = logo (varsa)
  let args = ["-y", "-i", audioPath];
  let filter = wave;
  let finalMapVideo = "[vout]";

  if (fs.existsSync(bgMp4)) {
    // arka plan video loop
    args.push("-stream_loop","-1","-t", String(dur), "-i", bgMp4);
    if (hasLogo) args.push("-i", LOGO_PATH);

    filter +=
      `[1:v]scale=${W}:${H}:force_original_aspect_ratio=increase,crop=1080:1920,setsar=1[vbg];` +
      `[vbg][wave]overlay=(W-w)/2:${H - WAVE_HEIGHT - WAVE_MARGIN_BOTTOM}[v1];` +
      (hasLogo ? logoPrep + `[v1][logo]overlay=(W-w)/2:(H-h)/2[v2];` : "") +
      `[${hasLogo ? "v2" : "v1"}]${subFilter}[vout]`;

    args.push(
      "-filter_complex", filter,
      "-map", finalMapVideo, "-map", "0:a",
      "-c:v","libx264","-pix_fmt","yuv420p","-r","30",
      "-c:a","aac","-b:a","128k",
      "-shortest",
      outPath
    );

  } else if (fs.existsSync(bgImg)) {
    // arka plan tek görsel
    args.push("-loop","1","-t", String(dur), "-i", bgImg);
    if (hasLogo) args.push("-i", LOGO_PATH);

    filter +=
      `[1:v]scale=${W}:${H}:force_original_aspect_ratio=increase,crop=1080:1920,setsar=1[vbg];` +
      `[vbg][wave]overlay=(W-w)/2:${H - WAVE_HEIGHT - WAVE_MARGIN_BOTTOM}[v1];` +
      (hasLogo ? logoPrep + `[v1][logo]overlay=(W-w)/2:(H-h)/2[v2];` : "") +
      `[${hasLogo ? "v2" : "v1"}]${subFilter}[vout]`;

    args.push(
      "-filter_complex", filter,
      "-map", finalMapVideo, "-map", "0:a",
      "-c:v","libx264","-pix_fmt","yuv420p","-r","30",
      "-c:a","aac","-b:a","128k",
      "-shortest",
      outPath
    );

  } else {
    // renkli arka plan
    args.push("-f","lavfi","-t", String(dur), "-i", `color=c=${BG_COLOR_HEX}:s=${CANVAS_SIZE}`);
    if (hasLogo) args.push("-i", LOGO_PATH);

    filter +=
      `[1:v]format=rgba[vbg];` +
      `[vbg][wave]overlay=(W-w)/2:${H - WAVE_HEIGHT - WAVE_MARGIN_BOTTOM}[v1];` +
      (hasLogo ? logoPrep + `[v1][logo]overlay=(W-w)/2:(H-h)/2[v2];` : "") +
      `[${hasLogo ? "v2" : "v1"}]${subFilter}[vout]`;

    args.push(
      "-filter_complex", filter,
      "-map", finalMapVideo, "-map", "0:a",
      "-c:v","libx264","-pix_fmt","yuv420p","-r","30",
      "-c:a","aac","-b:a","128k",
      "-shortest",
      outPath
    );
  }

  // 1. deneme: altyazılı
  try {
    await runFFmpeg(args);
  } catch(e) {
    warn("Subtitle render failed, retrying without subtitles. Reason:", e?.message || e);

    // 2. deneme: altyazısız ama waveform ve arka plan aynen
    const i = args.indexOf("-filter_complex");
    if (i > -1) {
      let f = args[i+1];
      // 'subtitles' kısmını kaba şekilde budayalım: ...[v1]subtitles... [vout] → ...[vout]
      f = f.replace(/(\[v\d\])subtitles=[^\[]+\[vout\]/, "$1copy[vout]"); // geçici yer tutucu
      f = f.replace("copy", "format=rgba"); // görüntü zinciri bozulmasın
      args[i+1] = f;
    }
    try {
      await runFFmpeg(args);
    } catch(e2) {
      throw e2;
    }
  }
}

// ---------- 4) Upload (googleapis) ----------
async function uploadToYouTube(filePath, title, desc, privacy){
  assertFile(SECRET_PATH, "YOUTUBE_CLIENT_SECRET_PATH");
  assertFile(TOKEN_PATH, "YOUTUBE_TOKEN_PATH");

  const sec = JSON.parse(fs.readFileSync(SECRET_PATH, "utf8")).installed;
  const tok = JSON.parse(fs.readFileSync(TOKEN_PATH, "utf8"));

  const oauth2 = new google.auth.OAuth2(
    sec.client_id,
    sec.client_secret,
    process.env.GOOGLE_REDIRECT_URI || "http://localhost"
  );
  oauth2.setCredentials(tok);
  const youtube = google.youtube({ version:"v3", auth: oauth2 });

  const res = await youtube.videos.insert({
    part: ["snippet","status"],
    requestBody: {
      snippet: { title, description: desc },
      status: { privacyStatus: privacy }
    },
    media: { body: fs.createReadStream(filePath) }
  });
  return `https://www.youtube.com/watch?v=${res.data.id}`;
}

// ---------- MAIN ----------
(async () => {
  try {
    // Önden hızlı kontroller
    if (!await checkCmd("ffmpeg")) warn("ffmpeg bulunamadı; PATH'e ekli olduğundan emin ol.");
    if (!await checkCmd("ffprobe")) warn("ffprobe bulunamadı; PATH'e ekli olduğundan emin ol.");

    log("Generating topic & script...");
    const gen = await makeScript();
    log("Title:", gen.title);

    const voicePath = path.join(TMP, "short-voice.mp3");
    log("TTS...");
    await makeTTS(gen.script, voicePath);

    log("ffprobe duration...");
    const dur = await runFFprobeDuration(voicePath);
    log("Audio duration:", dur.toFixed(2), "s");

    log("Build SRT...");
    const cues = splitCaptions(gen.script, dur);
    const srt = toSrt(cues);
    const srtPath = path.join(TMP, "shorts.srt");
    fs.writeFileSync(srtPath, srt, "utf8");

    const outPath = path.join(TMP, "shorts-out.mp4");
    log("Rendering video...");
    await renderVideo(voicePath, srtPath, outPath, dur);

    const hashtags = Array.isArray(gen.hashtags) ? gen.hashtags.join(" ") : "#shorts #crypto #QuantumCoin";
    const desc = `${gen.description}\n\n${hashtags}`;
    log("Uploading to YouTube...", PRIVACY);
    const url = await uploadToYouTube(outPath, gen.title, desc, PRIVACY);

    console.log("\n=== DONE ===");
    console.log("Video:", outPath);
    console.log("YouTube:", url);

    // Opsiyonel duyuru: .env içine ENABLE_ANNOUNCE=1 koyarsan otomatik gönderir
    if ((process.env.ENABLE_ANNOUNCE || "").toString().trim() === "1") {
      const announce = `New Shorts: ${gen.title} ${url} #shorts #crypto`;
      try {
        await new Promise((resolve, reject) => {
          const ps = spawn(
            "pwsh",
            [
              "-ExecutionPolicy","Bypass",
              "-File", path.join(ROOT, "publish-all.ps1"),
              "-Text", announce,
              "-Platforms", "telegram,x",
            ],
            { stdio: "inherit" }
          );
          ps.on("close", (code) => code === 0 ? resolve() : reject(new Error("publish-all.ps1 exit " + code)));
        });
      } catch (annErr) {
        console.error("Announce failed:", annErr?.message || annErr);
      }
    }
  } catch (e) {
    console.error("FATAL:", e?.response?.data || e?.message || e);
    process.exit(1);
  }
})();





