# quantumcoin/telegram_game/handlers/mining.py

from aiogram import Router, types
from aiogram.filters import Command
from blockchain.qc_chain_api import create_wallet_if_needed, mine_block
from database.redis_store import save_user_if_not_exists
from config import BASE_REWARD_QC
import random
import os

router = Router()

@router.message(Command("mine"))
async def handle_mine(message: types.Message):
    user_id = str(message.from_user.id)
    name = message.from_user.first_name or "Madenci"

    save_user_if_not_exists(user_id, name)
    wallet_address = create_wallet_if_needed(user_id)

    # 🎬 GIF gönder
    gif_path = os.path.join("assets", "mining_gif.gif")
    try:
        with open(gif_path, "rb") as gif:
            await message.answer_animation(gif, caption="⛏ Quantum Miner devreye giriyor...\nUzay boşluğunda blok aranıyor...")
    except:
        await message.answer("⛏ Kazım başlatılıyor...")

    # Zincirde kazım işlemi başlat
    result = mine_block(wallet_address)

    if result.get("success"):
        reward = result.get("reward", BASE_REWARD_QC)
        block_hash = result.get("block_hash", "bilinmiyor")

        nft_message = ""
        if random.random() < 0.1:
            nft_message = "\n🎉 <b>Nadir bir NFT kazandınız!</b> (Uzay Gemisi Parçası)"

        await message.answer(
            f"☄️ <b>Blok bulundu!</b>\n"
            f"🧱 Hash: <code>{block_hash[:12]}...</code>\n"
            f"🎁 Ödül: {reward} QC\n"
            f"{nft_message}"
        )
    else:
        await message.answer("🚫 Kazım sırasında bir hata oluştu. Lütfen daha sonra tekrar deneyin.")
