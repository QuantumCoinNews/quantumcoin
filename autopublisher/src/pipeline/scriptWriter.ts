import fs from 'fs-extra';
import path from 'path';
import { logger } from '../utils/logger.js';

// Basit senaryo yazıcı – şimdilik sabit şablon + konu
// İleride: AI API (OpenAI vb.) ile uzun metin üretimi
export async function writeScript(opts: { id: string, dir: string, topic: string }): Promise<string> {
  const { id, dir, topic } = opts;
  const file = path.join(dir, `${id}-script.txt`);

  // Eğer dosya daha önce yazılmışsa tekrar üretme
  if (await fs.pathExists(file)) {
    logger.info('script already exists, skipping', { file });
    return file;
  }

  const text = `
=== ${topic} ===

Merhaba! Bugünün konusu: ${topic}.

Bu video/konuşma otomatik oluşturulmuş bir senaryodur.
Günlük yayın akışımızda kripto para, blockchain, bilim, tarih ve teknolojiden konuları ele alıyoruz.

(Bu kısımda ileride AI tarafından detaylı belgesel metni üretilecek.)
`;

  await fs.outputFile(file, text.trim() + '\n', 'utf-8');
  return file;
}
