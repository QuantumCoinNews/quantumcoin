# quantumcoin/telegram_game/blockchain/qc_chain_api.py

import requests
from config import BLOCKCHAIN_API_URL
from database.redis_store import get_wallet_address, set_wallet_address

# Kullanıcıya wallet oluştur (ya da varsa döndür)
def create_wallet_if_needed(user_id: str) -> str:
    current = get_wallet_address(user_id)
    if current != "Tanımsız":
        return current

    try:
        response = requests.post(f"{BLOCKCHAIN_API_URL}/wallet/new")
        if response.status_code == 200:
            address = response.json().get("address")
            if address:
                set_wallet_address(user_id, address)
                return address
    except Exception as e:
        print(f"❌ Wallet oluşturulamadı: {e}")

    return "Bilinmiyor"

# Kullanıcının QC bakiyesini al
def get_balance(address: str) -> float:
    try:
        response = requests.get(f"{BLOCKCHAIN_API_URL}/wallet/balance/{address}")
        if response.status_code == 200:
            return response.json().get("balance", 0)
    except Exception as e:
        print(f"❌ Bakiye alınamadı: {e}")
    return 0

# Kullanıcı için madencilik (PoW) işlemi başlat
def mine_block(address: str) -> dict:
    try:
        response = requests.post(f"{BLOCKCHAIN_API_URL}/mine", json={"address": address})
        if response.status_code == 200:
            return response.json()
    except Exception as e:
        print(f"❌ Kazım işlemi başarısız: {e}")
    return {"success": False, "message": "Zincire bağlanılamadı."}
