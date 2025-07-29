# quantumcoin/telegram_game/handlers/wallet.py

from aiogram import Router, types
from aiogram.filters import Command
from blockchain.qc_chain_api import create_wallet_if_needed, get_balance
from database.redis_store import save_user_if_not_exists

router = Router()

@router.message(Command("wallet"))
async def handle_wallet(message: types.Message):
    user_id = str(message.from_user.id)
    name = message.from_user.first_name or "Madenci"

    # Kullanıcıyı güncelle
    save_user_if_not_exists(user_id, name)

    # Cüzdan adresini al
    address = create_wallet_if_needed(user_id)

    # Bakiye sorgula
    balance = get_balance(address)

    # Mesaj oluştur
    reply = (
        f"👛 <b>Cüzdan Bilgilerin</b>\n"
        f"🪙 Adres: <code>{address}</code>\n"
        f"💰 Bakiye: <b>{balance:.2f} QC</b>\n\n"
        f"⛏ Kazım yapmak için /mine komutunu kullanabilirsin!"
    )

    await message.answer(reply)
