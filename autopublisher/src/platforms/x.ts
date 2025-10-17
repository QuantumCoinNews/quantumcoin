import { DailyArtifacts } from '../pipeline/runDaily.js';
import { logger } from '../utils/logger.js';
import { enforceLimit, Limits } from '../utils/limits.js';

// Şimdilik stub — ileride twitter-api-v2 ile gerçek tweet atacağız.
export async function publishX(artifacts: DailyArtifacts) {
  const text = enforceLimit(artifacts.captions?.x ?? '', Limits.x.text);

  // Medya karar mantığı (ileride gerçek API çağrısına dönecek)
  const media = artifacts.imagePaths?.slice(0, 4) ?? []; // X max 4 görsel
  const video = artifacts.videoPath || '';

  logger.info('[X] paylaşım simüle edildi', {
    text,
    mediaCount: media.length,
    hasVideo: !!video,
  });

  // TODO: twitter-api-v2 entegrasyonu
  // const client = new TwitterApi({...});
  // await client.v2.tweet({ text, media: { media_ids: [...] } });
}
