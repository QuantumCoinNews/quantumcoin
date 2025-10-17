import path from 'path';
import { ensureDir } from '../utils/files.js';
import { logger } from '../utils/logger.js';

import { selectTopic } from './topicSelector.js';
import { writeScript } from './scriptWriter.js';
import { generateVoice } from './tts.js';
import { generateImages } from './imageGen.js';
import { assembleVideo } from './videoAssembler.js';

import { publishYouTube } from '../platforms/youtube.js';
import { publishX } from '../platforms/x.js';
import { publishInstagram } from '../platforms/instagram.js';
import { publishTelegram } from '../platforms/telegram.js';
import { publishTikTok } from '../platforms/tiktok.js';

export interface DailyArtifacts {
  id: string;                 // örn: 2025-10-02
  topic: string;              // seçilen konu
  scriptPath: string;         // .txt / .md
  audioPath: string;          // .mp3 / .wav
  imagePaths: string[];       // oluşturulan görseller
  videoPath?: string;         // render edilen video (varsa)
  captions?: {
    yt?: string;
    x?: string;
    ig?: string;
    tg?: string;
    tt?: string;
  };
}

export async function runDaily() {
  const CONTENT_DIR = process.env.CONTENT_DIR || 'src/content';
  const OUT_DIR     = process.env.OUT_DIR     || path.join(CONTENT_DIR, 'out');
  const PROMPT_DIR  = process.env.PROMPT_DIR  || path.join(CONTENT_DIR, 'prompts');

  // Gün klasörü (deterministik)
  const id = new Date().toISOString().slice(0, 10); // YYYY-MM-DD
  const dayDir = path.join(OUT_DIR, id);
  await ensureDir(dayDir);

  logger.info(`runDaily started → ${id}`);

  // 1) Konu seç
  const topic = await selectTopic({ id, promptDir: PROMPT_DIR });
  logger.info('topic selected', { topic });

  // 2) Senaryo yaz
  const scriptPath = await writeScript({ id, dir: dayDir, topic });
  logger.info('script written', { scriptPath });

  // 3) TTS (seslendirme)
  const audioPath = await generateVoice({ id, dir: dayDir, scriptPath });
  logger.info('voice generated', { audioPath });

  // 4) Görseller
  const imagePaths = await generateImages({ id, dir: dayDir, topic, count: 8 });
  logger.info('images generated', { count: imagePaths.length });

  // 5) Video montaj (opsiyonel – kısa video üret)
  const videoPath = await assembleVideo({
    id, dir: dayDir, audioPath, imagePaths, fps: 30, width: 1080, height: 1920
  });
  logger.info('video assembled', { videoPath });

  const artifacts: DailyArtifacts = {
    id,
    topic,
    scriptPath,
    audioPath,
    imagePaths,
    videoPath,
    captions: {
      yt: `Belgesel klibi – ${topic}\n#Bitcoin #Crypto #Belgesel`,
      x:  `Bugün: ${topic} #Bitcoin #Crypto`,
      ig: `${topic} – kısa kesit\n#bitcoin #crypto #reels`,
      tg: `Günün konusu: ${topic}\nDaha fazlası için takipte kalın.`,
      tt: `${topic} | Daha fazlası için profili takip et!`
    }
  };

  // 6) Yayınla (platformlara göre; hata olursa logla ve devam et)
  try { await publishYouTube(artifacts); } catch (e) { logger.error('YouTube publish error', e); }
  try { await publishX(artifacts); }       catch (e) { logger.error('X publish error', e); }
  try { await publishInstagram(artifacts);}catch (e) { logger.error('Instagram publish error', e); }
  try { await publishTelegram(artifacts);} catch (e) { logger.error('Telegram publish error', e); }
  try { await publishTikTok(artifacts); }  catch (e) { logger.error('TikTok publish error', e); }

  logger.info('runDaily finished', { id });
  return artifacts;
}
