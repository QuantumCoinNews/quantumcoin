import os
import random

NFT_FOLDER = os.path.join("assets", "nft_templates")

def get_random_nft_image() -> str:
    try:
        files = [f for f in os.listdir(NFT_FOLDER) if f.endswith(('.png', '.jpg', '.jpeg', '.gif'))]
        if not files:
            return ""
        return os.path.join(NFT_FOLDER, random.choice(files))
    except Exception as e:
        print(f"[NFT] Dosya okunamadı: {e}")
        return ""
