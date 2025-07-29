# quantumcoin/telegram_game/main_bot.py

import asyncio
from aiogram import Bot, Dispatcher
from aiogram.types import BotCommand
from aiogram.fsm.storage.memory import MemoryStorage
from config import BOT_TOKEN
from handlers import start, mining, wallet, referral, leaderboard

# Botu ve dispatcher'ı başlat
bot = Bot(token=BOT_TOKEN, parse_mode="HTML")
dp = Dispatcher(storage=MemoryStorage())

# Komut listesini kullanıcıya göstermek için
async def set_bot_commands():
    commands = [
        BotCommand(command="/start", description="🚀 Oyuna Başla"),
        BotCommand(command="/mine", description="⛏ Uzayda Kazım Yap"),
        BotCommand(command="/wallet", description="👛 Cüzdanını Görüntüle"),
        BotCommand(command="/claim", description="🎁 Ödülünü Talep Et"),
        BotCommand(command="/referral", description="🧑‍🚀 Davet Sistemi"),
        BotCommand(command="/leaderboard", description="🏆 En İyiler Tablosu"),
    ]
    await bot.set_my_commands(commands)

# Botu başlatan ana fonksiyon
async def main():
    print("🚀 QuantumCoin Uzay Madenciliği Botu Başlatılıyor...")

    # Komutları tanımla
    await set_bot_commands()

    # Yönlendirici kayıtları (handlers)
    dp.include_router(start.router)
    dp.include_router(mining.router)
    dp.include_router(wallet.router)
    dp.include_router(referral.router)
    dp.include_router(leaderboard.router)
    # claim.py ve diğer handler'lar eklenebilir

    # Polling başlat
    await dp.start_polling(bot)

if __name__ == "__main__":
    asyncio.run(main())
