import { DailyArtifacts } from '../pipeline/runDaily.js';
import { logger } from '../utils/logger.js';

// Şimdilik stub — TikTok için resmi API kısıtlıdır.
// Genellikle Business hesap + partner SDK gerekir.
// Geçici olarak sadece simülasyon log basıyor.
export async function publishTikTok(artifacts: DailyArtifacts) {
  const caption = artifacts.captions?.tt || '';
  const video = artifacts.videoPath || '';

  logger.info('[TikTok] paylaşım simüle edildi', {
    caption,
    hasVideo: !!video,
  });

  // TODO:
  // - TikTok Business API entegrasyonu
  //   (https://developers.tiktokglobalshop.com/)
  // - veya otomasyon (Playwright/Selenium) ile yükleme
}
