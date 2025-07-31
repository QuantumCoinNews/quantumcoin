import random
from config import BASE_REWARD_QC, BONUS_REWARD_QC, NFT_DROP_CHANCE

def calculate_reward(user_id: str) -> dict:
    reward = BASE_REWARD_QC
    bonus = 0
    nft_won = False
    nft_name = None

    # %5 ihtimalle bonus ödül
    if random.random() < 0.05:
        bonus = BONUS_REWARD_QC
        reward += bonus

    # %10 ihtimalle NFT kazanır
    if random.random() < NFT_DROP_CHANCE:
        nft_won = True
        nft_name = random.choice([
            "🌌 Galaksi Haritası",
            "🛸 Uzay Gemisi Parçası",
            "🔭 Kuantum Sonda NFT",
            "🚀 Ender Motor Parçası"
        ])

    return {
        "total_reward": reward,
        "base": BASE_REWARD_QC,
        "bonus": bonus,
        "nft_won": nft_won,
        "nft_name": nft_name
    }
