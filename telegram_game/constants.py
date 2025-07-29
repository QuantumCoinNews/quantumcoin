# quantumcoin/telegram_game/constants.py

# === Tema ve Oyunsal Bilgiler ===

THEME_NAME = "🚀 Quantum Mining in Space"
NFT_COLLECTION_NAME = "🪐 Quantum Galaxy Artifacts"
GAME_DESCRIPTION = (
    "🚀 Uzayın derinliklerinde Quantum bloklarını kaz ve ödülleri topla!\n"
    "Her kazımda 50 QC kazanma şansı seni bekliyor. Bonuslar ve nadir NFT'lerle dolu bu yolculukta zirveye oyna!"
)

# === Emoji ve Görsel Sabitler ===

EMOJI_MINE = "⛏"
EMOJI_QC = "💰"
EMOJI_NFT = "✨"
EMOJI_BLOCK = "🧱"
EMOJI_USER = "👤"
EMOJI_WALLET = "👛"
EMOJI_STAR = "🌟"
EMOJI_FIRE = "🔥"
EMOJI_TROPHY = "🏆"

# === Kazım Ödül Sabitleri (config.py'ye paralel) ===

QC_BASE_REWARD = 50
QC_BONUS_REWARD = 100
NFT_DROP_CHANCE = 0.1  # %10 ihtimal
BONUS_QC_CHANCE = 0.05  # %5 ihtimal

# === Sistem Süreleri ve Limitleri ===

CLAIM_COOLDOWN_SECONDS = 3600  # 1 saat bekleme süresi (gelecek için)

# === NFT Etiketleri ===

NFT_NAMES = [
    "🌌 Galaksi Haritası",
    "🛸 Uzay Gemisi Parçası",
    "🔭 Kuantum Sonda NFT",
    "🚀 Ender Motor Parçası",
    "🛰️ Orbital Kod Fragmanı"
]
