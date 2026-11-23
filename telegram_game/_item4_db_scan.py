import sqlite3, json, os

db = "dev_api_state.db"

print("\n===== ITEM 4 DB SCAN =====")
print("db_exists:", os.path.exists(db), db)

conn = sqlite3.connect(db)
conn.row_factory = sqlite3.Row

def q(sql, params=()):
    try:
        rows = conn.execute(sql, params).fetchall()
        return [dict(r) for r in rows]
    except Exception as e:
        return {"error": str(e), "sql": sql}

tables = q("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
print("\n--- tables ---")
print(json.dumps(tables, indent=2, ensure_ascii=False))

for table in ["players", "mining_sessions", "reward_records", "withdraw_batches", "tgwt_balances", "tgwt_ledger"]:
    print(f"\n--- {table} columns ---")
    print(json.dumps(q(f"PRAGMA table_info({table})"), indent=2, ensure_ascii=False))

print("\n--- players sample ---")
print(json.dumps(q("""
SELECT player_id, display_name, total_qc, total_xp, sessions, qc_address, last_seen
FROM players
ORDER BY last_seen DESC
LIMIT 20
"""), indent=2, ensure_ascii=False))

print("\n--- reward_records by player ---")
print(json.dumps(q("""
SELECT player_id, COUNT(*) AS count, COALESCE(SUM(amount),0) AS amount
FROM reward_records
GROUP BY player_id
ORDER BY count DESC
LIMIT 20
"""), indent=2, ensure_ascii=False))

print("\n--- withdraw_batches by player ---")
print(json.dumps(q("""
SELECT player_id, COUNT(*) AS count, COALESCE(SUM(amount),0) AS amount
FROM withdraw_batches
GROUP BY player_id
ORDER BY count DESC
LIMIT 20
"""), indent=2, ensure_ascii=False))

conn.close()
