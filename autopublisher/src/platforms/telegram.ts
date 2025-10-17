import TelegramBot from 'node-telegram-bot-api';
import { DailyArtifacts } from '../pipeline/runDaily.js';
import { logger } from '../utils/logger.js';

let bot: TelegramBot | null = null;

function getBot(): TelegramBot {
  if (!bot) {
    const token = process.env.TELEGRAM_BOT_TOKEN;
    if (!token) throw new Error('TELEGRAM_BOT_TOKEN .env dosyasında eksik');
    bot = new TelegramBot(token, { polling: false });
  }
  return bot;
}

// Telegram gönderici
export async function publishTelegram(artifacts: DailyArtifacts) {
  const chatId = process.env.TELEGRAM_CHAT_ID;
  if (!chatId) {
    logger.warn('TELEGRAM_CHAT_ID yok, Telegram paylaşımı atlanıyor.');
    return;
  }

  const caption = artifacts.captions?.tg || '';

  const b = getBot();

  try {
    if (artifacts.videoPath) {
      await b.sendVideo(chatId, artifacts.videoPath, { caption });
      logger.info('[Telegram] video gönderildi.');
    } else if (artifacts.imagePaths && artifacts.imagePaths.length > 0) {
      await b.sendPhoto(chatId, artifacts.imagePaths[0], { caption });
      logger.info('[Telegram] görsel gönderildi.');
    } else {
      await b.sendMessage(chatId, caption || 'Yeni paylaşım');
      logger.info('[Telegram] metin gönderildi.');
    }
  } catch (err) {
    logger.error('[Telegram] gönderim hatası', { err });
  }
}
