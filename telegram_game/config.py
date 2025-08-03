import os
from datetime import datetime, timedelta

BOT_TOKEN = "YOUR_BOT_TOKEN"
# veya
BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN")


# Blockchain API bağlantı noktası (Go node ile konuşur)
BLOCKCHAIN_API_URL = "http://localhost:8081/api"  # Gerekirse değiştir

# Redis bağlantısı
REDIS_HOST = "localhost"
REDIS_PORT = 6379
REDIS_DB = 0

# Uzay temalı metinler
THEME_NAME = "🚀 Quantum Mining in Space"
START_MESSAGE = (
    "👨‍🚀 <b>Uzay Madenciliğine Hoş Geldin!</b>\n"
    "Hazırsan Quantum Miner’ını başlat ve galaksinin derinliklerinden QC kazanmaya başla!"
)

# Oyun süresi (1 yıl)
GAME_START_DATE = datetime(2025, 7, 27)
GAME_END_DATE = GAME_START_DATE + timedelta(days=365)

# Kazım ödülleri (QC)
BASE_REWARD_QC = 50
BONUS_REWARD_QC = 100  # Yıllık rastgele bonus
NFT_DROP_CHANCE = 0.1  # %10 ihtimalle NFT kazanma

# Ödül dağılım oranları (gelişmiş kontrol)
REWARD_DISTRIBUTION = {
    "miner": 0.70,
    "staker": 0.10,
    "dev_wallet": 0.10,
    "burn": 0.05,
    "system_fee": 0.05
}

# NFT koleksiyonu adı
NFT_COLLECTION_NAME = "Quantum Galaxy Artifacts"
