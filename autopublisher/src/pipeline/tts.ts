import fs from 'fs-extra';
import path from 'path';
import axios from 'axios';
import { logger } from '../utils/logger.js';

interface TtsOpts {
  id: string;
  dir: string;
  scriptPath: string;
}

// Şimdilik ElevenLabs API entegrasyonu
// İleride: Coqui TTS / yerel TTS opsiyonu
export async function generateVoice(opts: TtsOpts): Promise<string> {
  const { id, dir, scriptPath } = opts;
  const voiceId = process.env.ELEVEN_VOICE_ID || '21m00Tcm4TlvDq8ikWAM'; // default voice
  const apiKey  = process.env.ELEVEN_API_KEY;

  const outFile = path.join(dir, `${id}-voice.mp3`);

  if (!apiKey) {
    logger.warn('ELEVEN_API_KEY yok, dummy ses dosyası yazılıyor.');
    await fs.outputFile(outFile, 'dummy-audio-data');
    return outFile;
  }

  const text = await fs.readFile(scriptPath, 'utf-8');

  try {
    const url = `https://api.elevenlabs.io/v1/text-to-speech/${voiceId}`;
    const resp = await axios.post(url,
      { text, model_id: 'eleven_multilingual_v2' },
      {
        headers: {
          'xi-api-key': apiKey,
          'Content-Type': 'application/json',
        },
        responseType: 'arraybuffer',
      }
    );
    await fs.outputFile(outFile, resp.data);
    return outFile;
  } catch (err) {
    logger.error('TTS API hata', { err });
    throw err;
  }
}
