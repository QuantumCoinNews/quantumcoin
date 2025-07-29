# quantumcoin/telegram_game/handlers/start.py

from aiogram import Router, types
from aiogram.filters import CommandStart
from aiogram.utils.markdown import hbold
from config import START_MESSAGE
from database.redis_store import save_user_if_not_exists
from blockchain.qc_chain_api import create_wallet_if_needed

router = Router()

@router.message(CommandStart())
async def handle_start(message: types.Message):
    user_id = str(message.from_user.id)
    first_name = message.from_user.first_name or "Uzaylı"

    # Kullanıcıyı Redis'e kaydet
    user_created = save_user_if_not_exists(user_id, first_name)

    # Cüzdan oluştur veya mevcutsa getir
    wallet_address = create_wallet_if_needed(user_id)

    # Hoş geldin mesajı
    reply = f"{START_MESSAGE}\n\n"
    reply += f"🪐 Adın: {hbold(first_name)}\n"
    reply += f"🪙 Wallet: <code>{wallet_address}</code>\n\n"
    reply += "⛏ Şimdi /mine yazarak ilk kazımını yapabilirsin!"

    await message.answer(reply)
