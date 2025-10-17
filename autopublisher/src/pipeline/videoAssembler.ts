import fs from 'fs-extra';
import path from 'path';
import ffmpeg from 'fluent-ffmpeg';
import { logger } from '../utils/logger.js';

interface VideoOpts {
  id: string;
  dir: string;
  audioPath: string;
  imagePaths: string[];
  fps: number;
  width: number;
  height: number;
}

// Basit video montaj: görselleri slayt olarak sırala, ses ile birleştir
export async function assembleVideo(opts: VideoOpts): Promise<string> {
  const { id, dir, audioPath, imagePaths, fps, width, height } = opts;
  const outFile = path.join(dir, `${id}-video.mp4`);

  if (!imagePaths || imagePaths.length === 0) {
    logger.warn('Hiç görsel yok, video oluşturulamadı.');
    return '';
  }

  // Görselleri ffmpeg concat için txt listesi yap
  const listFile = path.join(dir, `${id}-images.txt`);
  let listContent = '';
  for (const img of imagePaths) {
    listContent += `file '${path.resolve(img)}'\n`;
    listContent += `duration 3\n`; // her görsel 3 saniye ekranda
  }
  // son görsel en az 1s kalsın
  listContent += `file '${path.resolve(imagePaths[imagePaths.length - 1])}'\n`;
  await fs.outputFile(listFile, listContent, 'utf-8');

  logger.info('ffmpeg ile video oluşturuluyor…', { outFile });

  return new Promise((resolve, reject) => {
    ffmpeg()
      .input(listFile)
      .inputOptions(['-f concat', '-safe 0'])
      .input(audioPath)
      .outputOptions([
        `-r ${fps}`,
        `-s ${width}x${height}`,
        '-c:v libx264',
        '-pix_fmt yuv420p',
        '-c:a aac',
      ])
      .save(outFile)
      .on('end', () => {
        logger.info('Video başarıyla oluşturuldu.', { outFile });
        resolve(outFile);
      })
      .on('error', (err) => {
        logger.error('ffmpeg hata', { err });
        reject(err);
      });
  });
}
