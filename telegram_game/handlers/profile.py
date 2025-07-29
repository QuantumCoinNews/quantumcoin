# quantumcoin/telegram_game/handlers/profile.py

from aiogram import Router, types
from aiogram.filters import Command
from utils.stats import get_user_stats
from aiogram.utils.markdown import hbold
import datetime

router = Router()

def format_timestamp(unix_ts: int) -> str:
    try:
        dt = datetime.datetime.fromtimestamp(unix_ts)
        return dt.strftime("%d %b %Y %H:%M")
    except:
        return "Bilinmiyor"

@router.message(Command("profile", "stats"))
async def handle_profile(message: types.Message):
    user_id = str(message.from_user.id)
    stats = get_user_stats(user_id)

    if not stats:
        await message.answer("❌ Kayıt bulunamadı. Önce /start komutunu kullanmalısın.")
        return

    reply = (
        f"🪐 <b>Profil Bilgilerin</b>\n\n"
        f"{hbold('👤 Kullanıcı:')} {stats['name']}\n"
        f"{hbold('⛏ Toplam Kazım:')} {stats['mining_count']}\n"
        f"{hbold('🎁 Toplam Ödül:')} {stats['total_rewards']} QC\n"
        f"{hbold('👥 Referanslar:')} {stats['referrals']}\n"
        f"{hbold('🕒 Son Aktif:')} {format_timestamp(stats['last_active'])}\n"
    )

    await message.answer(reply)
