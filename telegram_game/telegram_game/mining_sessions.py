"""Server-owned, single-use mining runs shared by every process and entry point.

PROCESSING is committed before calling the node. A crash/failure is deliberately
not retried automatically: a possibly produced block requires reconciliation.
"""
import secrets
import json


def ensure_schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS game_mining_runs_v1 (
        session_id TEXT PRIMARY KEY, player_id TEXT NOT NULL, site_id INTEGER NOT NULL,
        started_at INTEGER NOT NULL, expires_at INTEGER NOT NULL,
        status TEXT NOT NULL CHECK(status IN ('STARTED','PROCESSING','COMPLETED','EXPIRED')),
        reward_id TEXT)""")
    conn.execute("""CREATE UNIQUE INDEX IF NOT EXISTS game_mining_active_v1
        ON game_mining_runs_v1(player_id) WHERE status IN ('STARTED','PROCESSING')""")


def open_run(conn, player_id, site_id, now):
    conn.execute("UPDATE game_mining_runs_v1 SET status='EXPIRED' WHERE player_id=? AND status='STARTED' AND expires_at<?", (player_id, now))
    row = conn.execute("""SELECT session_id,site_id,started_at,expires_at,status
        FROM game_mining_runs_v1 WHERE player_id=? AND status IN ('STARTED','PROCESSING')""", (player_id,)).fetchone()
    if row:
        if row[4] != "STARTED" or row[1] != site_id:
            return {"ok": False, "error": "mining_run_already_active"}
        session_id, _, started_at, expires_at, _ = row
    else:
        session_id, started_at, expires_at = "game_" + secrets.token_hex(24), now, now + 3600
        conn.execute("INSERT INTO game_mining_runs_v1 VALUES (?,?,?,?,?,'STARTED',NULL)",
                     (session_id, player_id, site_id, started_at, expires_at))
    conn.execute("""INSERT INTO mining_sessions(player_id,site_id,started_at) VALUES (?,?,?)
        ON CONFLICT(player_id) DO UPDATE SET site_id=excluded.site_id,started_at=excluded.started_at""",
        (player_id, site_id, started_at))
    return {"ok": True, "sessionId": session_id, "startedAt": started_at, "expiresAt": expires_at}


def reserve_run(db, player_id, site_id, session_id, now, require_pow=False):
    if not session_id:
        return {"ok": False, "error": "server_session_required"}
    conn = db()
    try:
        conn.execute("BEGIN IMMEDIATE")
        row = conn.execute("""SELECT started_at,expires_at,status FROM game_mining_runs_v1
            WHERE session_id=? AND player_id=? AND site_id=?""", (session_id, player_id, site_id)).fetchone()
        if not row or row[2] != "STARTED":
            return {"ok": False, "error": "mining_session_invalid_or_consumed"}
        duration = now - row[0]
        if duration < 60:
            return {"ok": False, "error": "mining_session_too_short"}
        if now > row[1] or duration > 3600:
            conn.execute("UPDATE game_mining_runs_v1 SET status='EXPIRED' WHERE session_id=?", (session_id,))
            conn.commit()
            return {"ok": False, "error": "mining_session_expired"}
        proof = None
        if require_pow:
            work = conn.execute("SELECT receipt_json FROM game_pow_work_v1 WHERE session_id=? AND player_id=? AND status='ACCEPTED'",
                                (session_id, player_id)).fetchone()
            if not work:
                work = conn.execute("SELECT receipt_json FROM game_pow_receipts_v1 WHERE session_id=? AND player_id=? ORDER BY rowid DESC LIMIT 1",(session_id,player_id)).fetchone()
            if not work:
                return {"ok": False, "error": "accepted_client_pow_required"}
            proof = json.loads(work[0])
        conn.execute("UPDATE game_mining_runs_v1 SET status='PROCESSING' WHERE session_id=? AND status='STARTED'", (session_id,))
        conn.commit()
        return {"ok": True, "startedAt": row[0], "durationSec": duration, "powReceipt": proof}
    finally:
        conn.close()
