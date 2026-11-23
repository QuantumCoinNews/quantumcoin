import sqlite3, json, os

db = "dev_api_state.db"
conn = sqlite3.connect(db)
conn.row_factory = sqlite3.Row

rows = conn.execute("""
SELECT qc_address, COUNT(*) AS count, GROUP_CONCAT(player_id) AS players
FROM players
WHERE COALESCE(qc_address, '') != ''
GROUP BY qc_address
HAVING COUNT(*) > 1
""").fetchall()

print("\n===== DUPLICATE QC ADDRESS CHECK =====")
print(json.dumps([dict(r) for r in rows], indent=2, ensure_ascii=False))

all_rows = conn.execute("""
SELECT player_id, qc_address, total_qc, sessions
FROM players
ORDER BY player_id
""").fetchall()

print("\n===== PLAYERS ADDRESS LIST =====")
print(json.dumps([dict(r) for r in all_rows], indent=2, ensure_ascii=False))

conn.close()
