# quantumcoin/telegram_game/database/redis_store.py

import redis
import time
from config import REDIS_HOST, REDIS_PORT, REDIS_DB

# Redis bağlantısı
r = redis.StrictRedis(host=REDIS_HOST, port=REDIS_PORT, db=REDIS_DB, decode_responses=True)

def save_user_if_not_exists(user_id: str, name: str) -> bool:
    key = f"user:{user_id}"
    if not r.exists(key):
        r.hset(key, mapping={
            "name": name,
            "joined_at": int(time.time()),
            "mining_count": 0,
            "last_active": int(time.time())
        })
        r.sadd("users", user_id)
        return True
    else:
        r.hset(key, "last_active", int(time.time()))
        return False

def set_wallet_address(user_id: str, address: str):
    r.hset(f"user:{user_id}", "wallet", address)

def get_wallet_address(user_id: str) -> str:
    return r.hget(f"user:{user_id}", "wallet") or "Tanımsız"

def get_total_users() -> int:
    return r.scard("users")

def get_active_users(hours: int = 24) -> int:
    now = int(time.time())
    threshold = now - hours * 3600
    count = 0
    for user_id in r.smembers("users"):
        last_active = int(r.hget(f"user:{user_id}", "last_active") or 0)
        if last_active > threshold:
            count += 1
    return count

# ⬇️ Bunu BURAYA ekle ⬇️
def get_top_miners(limit=10):
    top = []
    for user_id in r.smembers("users"):
        data = r.hgetall(f"user:{user_id}")
        if not data:
            continue
        name = data.get("name", "Bilinmeyen")
        count = int(data.get("mining_count", 0))
        top.append({"name": name, "count": count})
    top.sort(key=lambda x: x["count"], reverse=True)
    return top[:limit]
