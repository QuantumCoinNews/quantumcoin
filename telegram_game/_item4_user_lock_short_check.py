import sqlite3, json, os

db = "dev_api_state.db"
print("DB:", os.path.abspath(db), "exists:", os.path.exists(db))

conn = sqlite3.connect(db)
conn.row_factory = sqlite3.Row

def q(sql, params=()):
    try:
        return [dict(r) for r in conn.execute(sql, params).fetchall()]
    except Exception as e:
        return [{"error": str(e), "sql": sql}]

print("\n--- players ---")
print(json.dumps(q("""
SELECT player_id, display_name, total_qc, total_xp, sessions, qc_address, last_seen
FROM players
ORDER BY last_seen DESC
LIMIT 10
"""), indent=2, ensure_ascii=False))

print("\n--- reward_records by player ---")
print(json.dumps(q("""
SELECT player_id, COUNT(*) AS reward_count, COALESCE(SUM(amount),0) AS reward_amount
FROM reward_records
GROUP BY player_id
ORDER BY reward_count DESC
LIMIT 10
"""), indent=2, ensure_ascii=False))

print("\n--- latest rewards ---")
print(json.dumps(q("""
SELECT reward_id, player_id, amount, xp_amount, settlement_status,
       chain_height, block_reward, mining_client_type, network_reward_mode,
       target_qc_address
FROM reward_records
ORDER BY created_at DESC
LIMIT 10
"""), indent=2, ensure_ascii=False))

print("\n--- withdraw_batches by player ---")
print(json.dumps(q("""
SELECT player_id, COUNT(*) AS batch_count, COALESCE(SUM(amount),0) AS batch_amount
FROM withdraw_batches
GROUP BY player_id
ORDER BY batch_count DESC
LIMIT 10
"""), indent=2, ensure_ascii=False))

print("\n--- tgwt_state ---")
print(json.dumps(q("""
SELECT user_id, earned_u, pending_u, verified_u, withdrawn_u, rev
FROM tgwt_state
ORDER BY updated_at DESC
LIMIT 10
"""), indent=2, ensure_ascii=False))

conn.close()
