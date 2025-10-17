import { DailyArtifacts } from '../pipeline/runDaily.js';
import { logger } from '../utils/logger.js';

// Şimdilik stub – ileride YouTube Data API v3 entegrasyonu yapılacak
// Gereken: OAuth2 client_secret.json + token.json
export async function publishYouTube(artifacts: DailyArtifacts) {
  if (!artifacts.videoPath) {
    logger.warn('YouTube için video yok, atlanıyor.');
    return;
  }

  logger.info('[YouTube] yükleme simüle edildi', {
    title: artifacts.captions?.yt,
    video: artifacts.videoPath,
  });
}
