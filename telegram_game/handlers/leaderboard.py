# quantumcoin/telegram_game/handlers/leaderboard.py

from aiogram import Router, types
from aiogram.filters import Command
from database.redis_store import get_top_miners
from aiogram.utils.markdown import hbold

router = Router()

@router.message(Command("leaderboard"))
async def handle_leaderboard(message: types.Message):
    top_users = get_top_miners(limit=10)

    if not top_users:
        await message.answer("🏆 Henüz sıralama verisi bulunamadı.")
        return

    reply = "🏆 <b>Quantum Madenci Liderliği</b> (En çok kazım yapanlar)\n\n"
    for i, user in enumerate(top_users, start=1):
        reply += f"{i}. {hbold(user['name'])} – {user['count']} kazım\n"

    await message.answer(reply)
