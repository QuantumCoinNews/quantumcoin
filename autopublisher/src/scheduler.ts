import cron from 'node-cron';
import { logger } from './utils/logger.js';
import { runDaily } from './pipeline/runDaily.js';

// Günlük job planlayıcı
export async function scheduleDaily(spec: string) {
  if (!cron.validate(spec)) {
    throw new Error(`Geçersiz cron ifadesi: ${spec}`);
  }

  logger.info(`Scheduler başlatıldı. Cron ifadesi: "${spec}"`);

  cron.schedule(spec, async () => {
    try {
      logger.info('⏰ Günlük otomasyon tetiklendi');
      await runDaily();
    } catch (err) {
      logger.error('runDaily hata', { err });
    }
  }, {
    timezone: process.env.TZ || 'UTC'
  });
}
