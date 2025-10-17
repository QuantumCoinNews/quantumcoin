import { DailyArtifacts } from '../pipeline/runDaily.js';
import { logger } from '../utils/logger.js';

// Şimdilik stub — ileride Instagram Graph API ile gerçek gönderim yapılacak.
export async function publishInstagram(artifacts: DailyArtifacts) {
  const caption = artifacts.captions?.ig || '';

  // Medya tercih sırası: video > ilk görsel
  const video = artifacts.videoPath || '';
  const image = artifacts.imagePaths?.[0] || '';

  logger.info('[Instagram] paylaşım simüle edildi', {
    caption,
    hasVideo: !!video,
    hasImage: !!image,
  });

  // TODO:
  // - Instagram Graph API (Business hesabı + Facebook Page)
  // - POST /{ig-user-id}/media
  // - POST /{ig-user-id}/media_publish
}
