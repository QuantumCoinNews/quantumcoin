# quantumcoin/telegram_game/handlers/referral.py

from aiogram import Router, types
from aiogram.filters import Command
from config import BOT_TOKEN
from database.redis_store import save_user_if_not_exists, get_total_users
from aiogram.utils.markdown import hbold

router = Router()

@router.message(Command("referral"))
async def handle_referral(message: types.Message):
    user_id = str(message.from_user.id)
    name = message.from_user.first_name or "Madenci"

    save_user_if_not_exists(user_id, name)

    # Davet linki oluştur (start parametreli)
    username = await message.bot.get_me()
    link = f"https://t.me/{username.username}?start={user_id}"

    reply = (
        f"🧑‍🚀 <b>Referans Sistemi</b>\n"
        f"{hbold('1.')} Bu bağlantıyı arkadaşlarınla paylaş:\n<code>{link}</code>\n\n"
        f"{hbold('2.')} Her gelen aktif kullanıcı için ödül kazanırsın.\n"
        f"{hbold('3.')} Ne kadar çok davet, o kadar çok ödül!\n\n"
        f"🪐 Toplam kayıtlı kullanıcı: {get_total_users()}"
    )

    await message.answer(reply)
