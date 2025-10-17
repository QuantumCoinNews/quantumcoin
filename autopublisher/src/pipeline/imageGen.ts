import fs from 'fs-extra';
import path from 'path';
import axios from 'axios';
import { logger } from '../utils/logger.js';

interface ImgOpts {
  id: string;
  dir: string;
  topic: string;
  count: number;
}

// Basit Stable Diffusion (Automatic1111 WebUI API) entegrasyonu
// Eğer API yoksa placeholder görsel yazar
export async function generateImages(opts: ImgOpts): Promise<string[]> {
  const { id, dir, topic, count } = opts;
  const outDir = path.join(dir, 'images');
  await fs.ensureDir(outDir);

  const baseUrl = process.env.SD_BASE_URL || 'http://127.0.0.1:7860';
  const model   = process.env.SD_MODEL || 'sdxl';

  const outFiles: string[] = [];

  if (!baseUrl) {
    logger.warn('SD_BASE_URL yok, placeholder görseller yazılıyor.');
    for (let i = 0; i < count; i++) {
      const f = path.join(outDir, `${id}-${i}.jpg`);
      await fs.outputFile(f, 'dummy-image-data');
      outFiles.push(f);
    }
    return outFiles;
  }

  for (let i = 0; i < count; i++) {
    const prompt = `${topic}, cinematic, documentary style, 4k, detailed`;
    const file = path.join(outDir, `${id}-${i}.png`);

    try {
      const resp = await axios.post(`${baseUrl}/sdapi/v1/txt2img`, {
        prompt,
        steps: 20,
        width: 768,
        height: 512,
        sampler_index: 'Euler a',
        cfg_scale: 7,
        batch_size: 1,
        override_settings: { sd_model_checkpoint: model }
      }, { responseType: 'json' });

      if (resp.data?.images?.[0]) {
        const imgB64 = resp.data.images[0];
        const buf = Buffer.from(imgB64, 'base64');
        await fs.writeFile(file, buf);
        outFiles.push(file);
      }
    } catch (err) {
      logger.error('SD API hata', { err });
    }
  }

  return outFiles;
}
