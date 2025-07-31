from database.redis_store import r

def get_user_stats(user_id: str) -> dict:
    key = f"user:{user_id}"
    if not r.exists(key):
        return {}

    name = r.hget(key, "name") or "Bilinmeyen"
    mining_count = int(r.hget(key, "mining_count") or 0)
    referrals = int(r.hget(key, "referral_count") or 0)
    total_rewards = float(r.hget(key, "total_qc_earned") or 0)
    last_active = int(r.hget(key, "last_active") or 0)

    return {
        "name": name,
        "mining_count": mining_count,
        "referrals": referrals,
        "total_rewards": total_rewards,
        "last_active": last_active
    }

def get_global_stats() -> dict:
    users = r.smembers("users")
    total_mining = 0
    total_qc = 0.0
    total_referrals = 0

    for uid in users:
        key = f"user:{uid}"
        total_mining += int(r.hget(key, "mining_count") or 0)
        total_qc += float(r.hget(key, "total_qc_earned") or 0)
        total_referrals += int(r.hget(key, "referral_count") or 0)

    return {
        "total_users": len(users),
        "total_mining": total_mining,
        "total_qc_distributed": round(total_qc, 2),
        "total_referrals": total_referrals
    }
