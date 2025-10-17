import fs from "fs";
import path from "path";
import { google } from "googleapis";

function arg(name, def=""){ const i=process.argv.indexOf("--"+name); return (i!==-1 && process.argv[i+1])? process.argv[i+1] : def; }

const file    = arg("file");
const title   = arg("title", file ? path.basename(file) : "QC Upload");
const desc    = arg("desc", "");
const privacy = arg("privacy", "unlisted");

if (!file || !fs.existsSync(file)) {
  console.error('Usage:\n  node upload-direct.mjs --file <path.mp4> [--title "t"] [--desc "d"] [--privacy public|unlisted|private]');
  process.exit(2);
}

const secretPath = process.env.YOUTUBE_CLIENT_SECRET_PATH || "secrets/client_secret.json";
const tokenPath  = process.env.YOUTUBE_TOKEN_PATH        || "secrets/youtube_token.json";

const sec = JSON.parse(fs.readFileSync(secretPath, "utf8")).installed;
const tok = JSON.parse(fs.readFileSync(tokenPath,  "utf8"));

const oauth2 = new google.auth.OAuth2(sec.client_id, sec.client_secret, process.env.GOOGLE_REDIRECT_URI || "http://localhost");
oauth2.setCredentials(tok);

const youtube = google.youtube({ version: "v3", auth: oauth2 });

const res = await youtube.videos.insert({
  part: ["snippet","status"],
  requestBody: { snippet: { title, description: desc }, status: { privacyStatus: privacy } },
  media: { body: fs.createReadStream(file) }
});

console.log("Uploaded:", `https://www.youtube.com/watch?v=${res.data.id}`);
