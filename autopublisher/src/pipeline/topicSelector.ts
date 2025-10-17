import fs from 'fs-extra';
import path from 'path';
import { logger } from '../utils/logger.js';

// Basit konu seçici – şimdilik sabit/rasgele
// İleride: AI API (OpenAI, Llama, vs.) ile gerçek konu seçimi
export async function selectTopic(opts: { id: string, promptDir: string }): Promise<string> {
  const { id, promptDir } = opts;

  // Eğer prompt klasöründe "topics.txt" varsa oradan seç
  const topicFile = path.join(promptDir, 'topics.txt');
  if (await fs.pathExists(topicFile)) {
    const lines = (await fs.readFile(topicFile, 'utf-8'))
      .split('\n')
      .map(l => l.trim())
      .filter(Boolean);
    if (lines.length > 0) {
      // deterministik: gün ID’sine göre seçim
      const idx = Math.abs(hashCode(id)) % lines.length;
      return lines[idx];
    }
  }

  // fallback: sabit örnekler
  const fallback = [
    'Bitcoin’in doğuşu ve Satoshi Nakamoto',
    'Blockchain teknolojisinin geleceği',
    'Yapay zekâ ve finansın kesişimi',
    'Kripto para madenciliğinin evrimi',
    'Tarihten büyük ekonomik balonlar'
  ];
  const idx = Math.abs(hashCode(id)) % fallback.length;
  return fallback[idx];
}

// küçük hash helper
function hashCode(str: string): number {
  let h = 0;
  for (let i = 0; i < str.length; i++) {
    h = (h << 5) - h + str.charCodeAt(i);
    h |= 0;
  }
  return h;
}
