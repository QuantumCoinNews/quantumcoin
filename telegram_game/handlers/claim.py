# quantumcoin/telegram_game/handlers/claim.py

from aiogram import Router, types
from aiogram.filters import Command
from database.redis_store import save_user_if_not_exists
from blockchain.qc_chain_api import create_wallet_if_needed
from utils.reward_engine import calculate_reward
import requests
from config import BLOCKCHAIN_API_URL

router = Router()

@router.message(Command("claim"))
async def handle_claim(message: types.Message):
    user_id = str(message.from_user.id)
    name = message.from_user.first_name or "Madenci"

    # Kullanıcıyı ve cüzdanı hazırla
    save_user_if_not_exists(user_id, name)
    wallet_address = create_wallet_if_needed(user_id)

    # Ödülü hesapla
    reward_data = calculate_reward(user_id)
    total_reward = reward_data["total_reward"]
    nft_msg = ""

    # Blockchain'e gönder
    payload = {
        "address": wallet_address,
        "amount": total_reward,
        "note": "Telegram ödül transferi"
    }

    try:
        res = requests.post(f"{BLOCKCHAIN_API_URL}/wallet/claim", json=payload)
        if res.status_code == 200 and res.json().get("success"):
            msg = f"🎁 <b>Ödül Başarıyla Gönderildi!</b>\n💰 Miktar: {total_reward} QC\n📬 Cüzdan: <code>{wallet_address}</code>"
        else:
            msg = "🚫 Ödül gönderimi başarısız oldu. Lütfen daha sonra tekrar dene."
    except Exception as e:
        msg = f"❌ Zincire bağlanılamadı: {e}"

    # NFT mesajı ekle
    if reward_data["nft_won"]:
        nft_msg = f"\n\n✨ <b>Tebrikler!</b> Ayrıca bir NFT kazandınız: {reward_data['nft_name']}"

    await message.answer(msg + nft_msg)
