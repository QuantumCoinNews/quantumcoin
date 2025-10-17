import 'dotenv/config';
import { scheduleDaily } from './scheduler.js';
import { logger } from './utils/logger.js';

async function boot() {
  const tz = process.env.TZ || 'UTC';
  const cron = process.env.DAILY_CRON || '0 18 * * *'; // varsayılan: her gün 18:00 UTC
  logger.info(`autopublisher starting… (TZ=${tz}, CRON="${cron}")`);

  // Günlük otomasyon tetikleyicisini başlat
  await scheduleDaily(cron);

  logger.info('autopublisher is running. Press Ctrl+C to stop.');
}

boot().catch((err) => {
  logger.error('Fatal error on boot', { err });
  process.exit(1);
});
