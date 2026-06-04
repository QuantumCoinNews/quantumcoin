from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import HTMLResponse
from pydantic import BaseModel
import os
import asyncio
import urllib.request
import sqlite3, time, threading, secrets, json, random
import json
from typing import Optional, Dict, Any
from telegram_game.utils.qc_api import (
    fetch_health as qc_fetch_health,
    fetch_address_balance as qc_fetch_address_balance,
    fetch_tx_status as qc_fetch_tx_status,
    fetch_mine_block as qc_fetch_mine_block,
    fetch_send_tx as qc_fetch_send_tx,
)

APP_DB = "dev_api_state.db"
ONLINE_TTL_SEC = 90

UNIT = 1_000_000  # TGWT micro-unit
TGWT_POOL_TOTAL_U = 3_000_000 * UNIT

WITHDRAW_FEE_BPS = 20   # 0.20%
DROP_CHANCE = 0.002     # 0.2%
DROP_AMOUNT_U = 1 * UNIT

app = FastAPI(title="Space Miner Dev API", version="0.4")

# DEV CORS (en sorunsuz)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

_lock = threading.Lock()


def _now() -> int:
    return int(time.time())


def _db():
    conn = sqlite3.connect(APP_DB, check_same_thread=False)
    conn.execute("PRAGMA journal_mode=WAL;")
    conn.execute("PRAGMA synchronous=NORMAL;")
    return conn


def _init():
    with _lock:
        conn = _db()

        # Players
        conn.execute("""
        CREATE TABLE IF NOT EXISTS players (
            player_id    TEXT PRIMARY KEY,
            display_name TEXT,
            total_qc     REAL NOT NULL DEFAULT 0,
            total_xp     REAL NOT NULL DEFAULT 0,
            sessions     INTEGER NOT NULL DEFAULT 0,
            first_seen   INTEGER NOT NULL DEFAULT 0,
            last_seen    INTEGER NOT NULL DEFAULT 0
        )
        """)
        conn.execute("CREATE INDEX IF NOT EXISTS idx_players_last_seen ON players(last_seen)")
        conn.execute("CREATE INDEX IF NOT EXISTS idx_players_total_qc ON players(total_qc)")

        # Settings
        conn.execute("""
        CREATE TABLE IF NOT EXISTS settings (
            player_id TEXT PRIMARY KEY,
            lang      TEXT,
            theme     TEXT,
            updated_at INTEGER NOT NULL DEFAULT 0
        )
        """)

        # Mining session state (dev)
        conn.execute("""
        CREATE TABLE IF NOT EXISTS mining_sessions (
            player_id  TEXT PRIMARY KEY,
            site_id    INTEGER NOT NULL DEFAULT 1,
            started_at INTEGER NOT NULL DEFAULT 0
        )
        """)

        # QC_REWARD_RECORDS_V1
        # Blockchain-ready reward ledger.
        # DEV_CONFIRMED now; later this becomes PENDING_CHAIN / CONFIRMED_CHAIN with real tx_hash.
        conn.execute("""
        CREATE TABLE IF NOT EXISTS reward_records (
            reward_id         TEXT PRIMARY KEY,
            player_id         TEXT NOT NULL,
            source            TEXT NOT NULL,
            zone_id           TEXT,
            site_id           INTEGER NOT NULL DEFAULT 1,
            asset             TEXT NOT NULL DEFAULT 'QC',
            amount            REAL NOT NULL DEFAULT 0,
            xp_amount         REAL NOT NULL DEFAULT 0,
            status            TEXT NOT NULL DEFAULT 'DEV_CONFIRMED',
            settlement_status TEXT NOT NULL DEFAULT 'DEV_ONLY',
            tx_hash           TEXT,
            meta_json         TEXT,
            created_at        INTEGER NOT NULL DEFAULT 0,
            confirmed_at      INTEGER NOT NULL DEFAULT 0
        )
        """)
        conn.execute("CREATE INDEX IF NOT EXISTS idx_reward_records_player ON reward_records(player_id, created_at)")
        conn.execute("CREATE INDEX IF NOT EXISTS idx_reward_records_status ON reward_records(status, settlement_status)")

        # TGWT state (micro-unit + rev)
        conn.execute("""
        CREATE TABLE IF NOT EXISTS tgwt_state (
            user_id     TEXT PRIMARY KEY,
            earned_u    INTEGER NOT NULL DEFAULT 0,
            pending_u   INTEGER NOT NULL DEFAULT 0,
            verified_u  INTEGER NOT NULL DEFAULT 0,
            withdrawn_u INTEGER NOT NULL DEFAULT 0,
            rev         INTEGER NOT NULL DEFAULT 0,
            updated_at  INTEGER NOT NULL DEFAULT 0
        )
        """)

        # TGWT Pool (single row id=1)
        conn.execute("""
        CREATE TABLE IF NOT EXISTS tgwt_pool (
            id            INTEGER PRIMARY KEY CHECK(id = 1),
            total_u       INTEGER NOT NULL,
            distributed_u INTEGER NOT NULL DEFAULT 0,
            updated_at    INTEGER NOT NULL DEFAULT 0
        )
        """)
        conn.execute("""
        INSERT OR IGNORE INTO tgwt_pool(id, total_u, distributed_u, updated_at)
        VALUES(1, ?, 0, ?)
        """, (TGWT_POOL_TOTAL_U, _now()))

        # TGWT events (for UI ledger cards/history)
        conn.execute("""
        CREATE TABLE IF NOT EXISTS tgwt_events (
            id         TEXT PRIMARY KEY,
            user_id    TEXT NOT NULL,
            kind       TEXT NOT NULL,
            amount_u   INTEGER NOT NULL DEFAULT 0,
            meta_json  TEXT,
            created_at INTEGER NOT NULL DEFAULT 0
        )
        """)
        conn.execute("CREATE INDEX IF NOT EXISTS idx_tgwt_events_user_created ON tgwt_events(user_id, created_at DESC)")

        # Nonce dedupe (anti double-submit for dev)
        conn.execute("""
        CREATE TABLE IF NOT EXISTS tgwt_nonces (
            user_id    TEXT NOT NULL,
            nonce      TEXT NOT NULL,
            created_at INTEGER NOT NULL DEFAULT 0,
            PRIMARY KEY(user_id, nonce)
        )
        """)

        # Withdraw requests (DEV queue)
        conn.execute("""
        CREATE TABLE IF NOT EXISTS withdraw_requests (
            id         TEXT PRIMARY KEY,
            user_id    TEXT NOT NULL,
            asset      TEXT NOT NULL,
            amount_u   INTEGER NOT NULL DEFAULT 0,
            to_address TEXT NOT NULL,
            status     TEXT NOT NULL,
            tx_hash    TEXT,
            created_at INTEGER NOT NULL DEFAULT 0,
            updated_at INTEGER NOT NULL DEFAULT 0
        )
        """)
        conn.execute("CREATE INDEX IF NOT EXISTS idx_withdraw_user_created ON withdraw_requests(user_id, created_at DESC)")

        conn.commit()
        conn.close()


_init()

# ------------------- Models -------------------

class HeartbeatIn(BaseModel):
    playerId: str
    displayName: Optional[str] = None


class SessionCompleteIn(BaseModel):
    playerId: str
    qcEarned: float = 0.0
    xpEarned: float = 0.0
    durationSec: float = 0.0
    displayName: Optional[str] = None


class SettingsIn(BaseModel):
    playerId: str
    lang: Optional[str] = None     # "en" | "tr" | "es" | "zh"
    theme: Optional[str] = None    # "auto" | "dark" | "light"


class MineStartIn(BaseModel):
    playerId: str
    siteId: int = 1
    displayName: Optional[str] = None


class MineFinishIn(BaseModel):
    playerId: str
    siteId: int = 1
    durationSec: float = 0.0
    qcEarned: Optional[float] = None
    xpEarned: Optional[float] = None
    displayName: Optional[str] = None


class TGWTEarnIn(BaseModel):
    userId: str
    amount: str  # decimal string
    nonce: str
    meta: Optional[Dict[str, Any]] = None


class TGWTVerifyIn(BaseModel):
    userId: str
    amount: Optional[str] = None  # null => verify all pending
    nonce: str
    source: Optional[str] = None


class WithdrawRequestIn(BaseModel):
    userId: str
    asset: str
    amount: float
    toAddress: str
    nonce: str


class SocialClaimIn(BaseModel):
    playerId: str
    platform: str
    nonce: str


class StoreBuyIn(BaseModel):
    playerId: str
    shipId: str
    nonce: str


# ------------------- Helpers -------------------

def _touch_player(conn, pid: str, name: Optional[str]):
    ts = _now()
    conn.execute(
        """
        INSERT OR IGNORE INTO players(player_id, display_name, first_seen, last_seen)
        VALUES(?,?,?,?)
        """,
        (pid, (name or None), ts, ts),
    )
    if name:
        conn.execute("UPDATE players SET display_name=?, last_seen=? WHERE player_id=?", (name, ts, pid))
    else:
        conn.execute("UPDATE players SET last_seen=? WHERE player_id=?", (ts, pid))


_rl = {}  # (user_id, key) -> (win_start, count)

def _rl_hit(user_id: str, key: str, limit: int, window_sec: int) -> bool:
    now = _now()
    k = (user_id, key)
    win, cnt = _rl.get(k, (now, 0))
    if now - win >= window_sec:
        win, cnt = now, 0
    cnt += 1
    _rl[k] = (win, cnt)
    return cnt <= limit


def _fee_u(amount_u: int) -> int:
    # round(amount_u * bps / 10000)
    return int((amount_u * WITHDRAW_FEE_BPS + 5000) // 10000)


def _pool_get(conn) -> Dict[str, int]:
    cur = conn.execute("SELECT total_u, distributed_u FROM tgwt_pool WHERE id=1")
    row = cur.fetchone()
    if not row:
        conn.execute(
            "INSERT INTO tgwt_pool(id, total_u, distributed_u, updated_at) VALUES(1, ?, 0, ?)",
            (TGWT_POOL_TOTAL_U, _now()),
        )
        return {"total_u": TGWT_POOL_TOTAL_U, "distributed_u": 0}
    return {"total_u": int(row[0] or 0), "distributed_u": int(row[1] or 0)}


def _pool_can_spend(conn, amount_u: int) -> bool:
    p = _pool_get(conn)
    return (p["distributed_u"] + int(amount_u)) <= p["total_u"]


def _pool_spend(conn, amount_u: int):
    conn.execute(
        "UPDATE tgwt_pool SET distributed_u = distributed_u + ?, updated_at=? WHERE id=1",
        (int(amount_u), _now()),
    )


def _tgwt_event(conn, user_id: str, kind: str, amount_u: int, meta: Optional[Dict[str, Any]] = None):
    eid = secrets.token_hex(12)
    meta_json = json.dumps(meta or {}, ensure_ascii=False)
    conn.execute(
        "INSERT INTO tgwt_events(id, user_id, kind, amount_u, meta_json, created_at) VALUES(?,?,?,?,?,?)",
        (eid, user_id, kind, int(amount_u), meta_json, _now()),
    )
    return eid


def _tgwt_touch(conn, user_id: str):
    ts = _now()
    conn.execute(
        "INSERT OR IGNORE INTO tgwt_state(user_id, updated_at) VALUES(?,?)",
        (user_id, ts),
    )
    conn.execute("UPDATE tgwt_state SET updated_at=? WHERE user_id=?", (ts, user_id))


def _tgwt_get(conn, user_id: str) -> Dict[str, int]:
    _tgwt_touch(conn, user_id)
    cur = conn.execute(
        "SELECT earned_u, pending_u, verified_u, withdrawn_u, rev FROM tgwt_state WHERE user_id=?",
        (user_id,),
    )
    row = cur.fetchone()
    if not row:
        return {"earned_u": 0, "pending_u": 0, "verified_u": 0, "withdrawn_u": 0, "rev": 0}
    return {
        "earned_u": int(row[0] or 0),
        "pending_u": int(row[1] or 0),
        "verified_u": int(row[2] or 0),
        "withdrawn_u": int(row[3] or 0),
        "rev": int(row[4] or 0),
    }


def _tgwt_bump_rev(conn, user_id: str):
    conn.execute("UPDATE tgwt_state SET rev = rev + 1, updated_at=? WHERE user_id=?", (_now(), user_id))


def _nonce_once(conn, user_id: str, nonce: str) -> bool:
    try:
        conn.execute(
            "INSERT INTO tgwt_nonces(user_id, nonce, created_at) VALUES(?,?,?)",
            (user_id, nonce, _now()),
        )
        return True
    except sqlite3.IntegrityError:
        return False


def _parse_amount_to_micro(amount_str: str) -> int:
    s = (amount_str or "").strip().replace(",", ".")
    if not s:
        return 0
    if s.count(".") > 1:
        return 0
    parts = s.split(".")
    whole = parts[0] if parts[0] else "0"
    frac = parts[1] if len(parts) > 1 else ""
    if len(frac) > 6:
        frac = frac[:6]
    frac = frac.ljust(6, "0")
    if not whole.isdigit() or not frac.isdigit():
        return 0
    return int(whole) * UNIT + int(frac)


def _tgwt_award_pending_from_pool(conn, user_id: str, amount_u: int, kind: str, meta: Optional[Dict[str, Any]] = None) -> bool:
    """Pool'dan harcayarak kullanıcıya pending TGWT ekler (earned+pending)."""
    amount_u = int(amount_u)
    if amount_u <= 0:
        return False
    _tgwt_touch(conn, user_id)
    if not _pool_can_spend(conn, amount_u):
        return False

    # Pool dağıtım sayacı: token ödülü verildiği anda artar (withdraw sırasında tekrar saymayız)
    _pool_spend(conn, amount_u)

    conn.execute(
        """
        UPDATE tgwt_state
        SET earned_u = earned_u + ?,
            pending_u = pending_u + ?
        WHERE user_id=?
        """,
        (amount_u, amount_u, user_id),
    )
    _tgwt_bump_rev(conn, user_id)
    _tgwt_event(conn, user_id, kind, amount_u, meta)
    return True


def _tgwt_move_pending_to_verified(conn, user_id: str, move_u: int) -> int:
    move_u = int(move_u)
    if move_u <= 0:
        return 0
    st = _tgwt_get(conn, user_id)
    move_u = min(move_u, int(st["pending_u"]))
    if move_u <= 0:
        return 0
    conn.execute(
        """
        UPDATE tgwt_state
        SET pending_u = pending_u - ?,
            verified_u = verified_u + ?
        WHERE user_id=?
        """,
        (move_u, move_u, user_id),
    )
    _tgwt_bump_rev(conn, user_id)
    _tgwt_event(conn, user_id, "VERIFY", move_u, {})
    return move_u


def _online_now(conn) -> int:
    ts = _now()
    cutoff = ts - ONLINE_TTL_SEC
    cur = conn.execute("SELECT COUNT(*) FROM players WHERE last_seen >= ?", (cutoff,))
    return int(cur.fetchone()[0])


def _total_players(conn) -> int:
    cur = conn.execute("SELECT COUNT(*) FROM players")
    return int(cur.fetchone()[0])


def _total_qc_all(conn) -> float:
    cur = conn.execute("SELECT COALESCE(SUM(total_qc), 0) FROM players")
    return float(cur.fetchone()[0] or 0.0)


def _get_player_totals(conn, pid: str) -> Dict[str, float]:
    cur = conn.execute("SELECT total_qc, total_xp FROM players WHERE player_id=?", (pid,))
    row = cur.fetchone()
    if not row:
        return {"qc": 0.0, "xp": 0.0}
    return {"qc": float(row[0] or 0.0), "xp": float(row[1] or 0.0)}


def _rank_for(conn, pid: str) -> int:
    my_qc = _get_player_totals(conn, pid)["qc"]
    cur = conn.execute("SELECT COUNT(*) FROM players WHERE total_qc > ?", (my_qc,))
    better = int(cur.fetchone()[0])
    return better + 1


def _xp_to_level(total_xp: float) -> Dict[str, Any]:
    if total_xp < 0:
        total_xp = 0
    level = int(total_xp // 100) + 1
    xp_in_level = int(total_xp % 100)
    xp_next = 100
    return {"level": level, "xp": xp_in_level, "xpNext": xp_next}




# QC_ADDRESS_LINK_V1
def _ensure_qc_address_column(conn):
    cols = [row[1] for row in conn.execute("PRAGMA table_info(players)").fetchall()]
    if "qc_address" not in cols:
        conn.execute("ALTER TABLE players ADD COLUMN qc_address TEXT DEFAULT ''")


def _wallet_payload(conn, pid: str) -> Dict[str, Any]:
    totals = _get_player_totals(conn, pid)
    xpinfo = _xp_to_level(totals["xp"])
    return {
        "playerId": pid,
        "address": f"QC_{pid[:10]}_DEV",
        "qc_balance": totals["qc"],
        "tgwt_balance": 0,
        "level": xpinfo["level"],
        "xp": xpinfo["xp"],
        "xp_next": xpinfo["xpNext"],
    }


def _compute_rewards(duration_sec: float) -> Dict[str, float]:
    if duration_sec < 0:
        duration_sec = 0
    qc = min(50.0, round(duration_sec * 0.5, 2))
    xp = min(100.0, round(duration_sec * 1.0, 2))
    return {"qc": qc, "xp": xp}


def _txhash_dummy() -> str:
    return "0x" + secrets.token_hex(32)










# QC_WITHDRAW_PAYOUT_META_V1
def _ensure_withdraw_payout_meta_columns(conn):
    cols = [row[1] for row in conn.execute("PRAGMA table_info(withdraw_batches)").fetchall()]
    patches = [
        ("payout_mode", "ALTER TABLE withdraw_batches ADD COLUMN payout_mode TEXT DEFAULT 'MOCK_PAYOUT'"),
        ("payout_type", "ALTER TABLE withdraw_batches ADD COLUMN payout_type TEXT DEFAULT 'DEV_MOCK'"),
        ("tx_confirmed_height", "ALTER TABLE withdraw_batches ADD COLUMN tx_confirmed_height INTEGER DEFAULT 0"),
        ("tx_checked_at", "ALTER TABLE withdraw_batches ADD COLUMN tx_checked_at INTEGER DEFAULT 0"),
        ("tx_check_note", "ALTER TABLE withdraw_batches ADD COLUMN tx_check_note TEXT DEFAULT ''"),
    ]

    for col, sql in patches:
        if col not in cols:
            conn.execute(sql)


# QC_WITHDRAW_BATCHES_V1
QC_MIN_WITHDRAW_AMOUNT = 50.0


def _ensure_withdraw_batch_tables(conn):
    # reward_records -> withdraw batch link
    reward_cols = [row[1] for row in conn.execute("PRAGMA table_info(reward_records)").fetchall()]

    if "withdraw_batch_id" not in reward_cols:
        conn.execute("ALTER TABLE reward_records ADD COLUMN withdraw_batch_id TEXT DEFAULT ''")

    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS withdraw_batches (
            batch_id TEXT PRIMARY KEY,
            player_id TEXT NOT NULL,
            target_qc_address TEXT NOT NULL DEFAULT '',
            amount REAL NOT NULL DEFAULT 0,
            reward_count INTEGER NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'REQUESTED',
            tx_hash TEXT DEFAULT NULL,
            reward_ids_json TEXT DEFAULT '[]',
            note TEXT DEFAULT '',
            min_chain_height INTEGER NOT NULL DEFAULT 0,
            max_chain_height INTEGER NOT NULL DEFAULT 0,
            created_at INTEGER NOT NULL DEFAULT 0,
            updated_at INTEGER NOT NULL DEFAULT 0
        )
        """
    )


# QC_GAME_MINER_50QC_CHAIN_REWARD_V1
QC_GAME_MINER_BLOCK_REWARD = 50.0
QC_GAME_MINER_CLIENT_TYPE = "GAME_MINER"
QC_GAME_MINER_REWARD_MODE = "NETWORK_BLOCK_REWARD"


def _fetch_qc_node_health_sync(timeout: float = 2.5) -> Dict[str, Any]:
    """
    QuantumCoin node health bilgisini sync okur.
    mine_finish sync endpoint olduğu için burada async helper kullanmıyoruz.
    """
    try:
        from telegram_game.config import get_api_base
        base = get_api_base().rstrip("/")
        url = f"{base}/api/health"
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
        data = json.loads(raw)
        if isinstance(data, dict):
            if "ok" not in data:
                data["ok"] = True
            return data
    except Exception as exc:
        return {"ok": False, "error": str(exc)}
    return {"ok": False, "error": "unexpected qc health response"}


# QC_REWARD_CHAIN_FIELDS_V1
def _ensure_reward_chain_columns(conn):
    cols = [row[1] for row in conn.execute("PRAGMA table_info(reward_records)").fetchall()]
    patches = [
        ("chain_height", "ALTER TABLE reward_records ADD COLUMN chain_height INTEGER DEFAULT 0"),
        ("block_reward", "ALTER TABLE reward_records ADD COLUMN block_reward REAL DEFAULT 0"),
        ("mining_client_type", "ALTER TABLE reward_records ADD COLUMN mining_client_type TEXT DEFAULT ''"),
        ("network_reward_mode", "ALTER TABLE reward_records ADD COLUMN network_reward_mode TEXT DEFAULT ''"),
    ]

    for col, sql in patches:
        if col not in cols:
            conn.execute(sql)


def _current_chain_height_from_qc_health(qc_health):
    try:
        return int((qc_health or {}).get("height") or 0)
    except Exception:
        return 0


# QC_REWARD_SETTLEMENT_READY_V1
def _ensure_reward_settlement_columns(conn):
    cols = [row[1] for row in conn.execute("PRAGMA table_info(reward_records)").fetchall()]

    needed = [
        ("target_qc_address", "ALTER TABLE reward_records ADD COLUMN target_qc_address TEXT DEFAULT ''"),
        ("settlement_note", "ALTER TABLE reward_records ADD COLUMN settlement_note TEXT DEFAULT ''"),
        ("settlement_updated_at", "ALTER TABLE reward_records ADD COLUMN settlement_updated_at INTEGER DEFAULT 0"),
    ]

    for name, sql in needed:
        if name not in cols:
            conn.execute(sql)


def _get_player_qc_address(conn, player_id: str) -> str:
    try:
        _ensure_qc_address_column(conn)
        row = conn.execute(
            "SELECT qc_address FROM players WHERE player_id=?",
            (player_id,),
        ).fetchone()
        return ((row[0] if row else "") or "").strip()
    except Exception:
        return ""


def _reward_record_insert(
    conn,
    *,
    player_id: str,
    source: str,
    zone_id: str,
    site_id: int,
    amount: float,
    xp_amount: float,
    meta: Optional[Dict[str, Any]] = None,
    chain_height: int = 0,
    block_reward: float = 0.0,
    mining_client_type: str = "",
    network_reward_mode: str = "",
) -> str:
    # QC_REWARD_RECORDS_V1 + QC_REWARD_SETTLEMENT_READY_V1 + QC_REWARD_CHAIN_FIELDS_V1
    _ensure_reward_settlement_columns(conn)
    _ensure_reward_chain_columns(conn)

    reward_id = "rw_" + secrets.token_hex(16)
    ts = _now()

    target_qc_address = _get_player_qc_address(conn, player_id)

    if target_qc_address:
        settlement_status = "READY_FOR_CHAIN"
        settlement_note = "Linked QC address found; reward is ready for future chain settlement."
    else:
        settlement_status = "DEV_ONLY"
        settlement_note = "No linked QC address; reward remains dev/game-only."

    conn.execute(
        """
        INSERT INTO reward_records(
            reward_id, player_id, source, zone_id, site_id,
            asset, amount, xp_amount, status, settlement_status,
            tx_hash, meta_json, created_at, confirmed_at,
            target_qc_address, settlement_note, settlement_updated_at,
            chain_height, block_reward, mining_client_type, network_reward_mode
        )
        VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
        """,
        (
            reward_id,
            player_id,
            source,
            zone_id,
            int(site_id or 1),
            "QC",
            float(amount or 0.0),
            float(xp_amount or 0.0),
            "DEV_CONFIRMED",
            settlement_status,
            None,
            json.dumps(meta or {}, ensure_ascii=False),
            ts,
            ts,
            target_qc_address,
            settlement_note,
            ts,
            int(chain_height or 0),
            float(block_reward or 0.0),
            (mining_client_type or "").strip(),
            (network_reward_mode or "").strip(),
        ),
    )
    return reward_id


def _dev_autoconfirm_withdraw(req_id: str, delay_sec: float = 1.5):
    def _worker():
        time.sleep(delay_sec)
        with _lock:
            conn = _db()
            cur = conn.execute("SELECT status FROM withdraw_requests WHERE id=?", (req_id,))
            row = cur.fetchone()
            if not row:
                conn.close()
                return
            if row[0] != "QUEUED":
                conn.close()
                return

            conn.execute(
                """
                UPDATE withdraw_requests
                SET status=?, tx_hash=?, updated_at=?
                WHERE id=?
                """,
                ("CONFIRMED", _txhash_dummy(), _now(), req_id),
            )
            conn.commit()
            conn.close()

    threading.Thread(target=_worker, daemon=True).start()


def _store_bonus_u(ship_id: str) -> int:
    # UI tarafında ship_01..05 kullanıyorsun; eski stub: starter/miner_mk1/miner_mk2
    table = {
        "ship_01": 5 * UNIT,
        "ship_02": 10 * UNIT,
        "ship_03": 15 * UNIT,
        "ship_04": 20 * UNIT,
        "ship_05": 25 * UNIT,
        "starter": 5 * UNIT,
        "miner_mk1": 10 * UNIT,
        "miner_mk2": 15 * UNIT,
    }
    return int(table.get(ship_id, 0))


# ------------------- Core endpoints -------------------

@app.get("/health")
def health():
    return {"ok": True, "ts": _now()}


@app.post("/api/presence/heartbeat")
def presence_heartbeat(inp: HeartbeatIn):
    with _lock:
        conn = _db()
        _touch_player(conn, inp.playerId, inp.displayName)
        conn.commit()

        online = _online_now(conn)
        total = _total_players(conn)
        rank = _rank_for(conn, inp.playerId)
        my_total = _get_player_totals(conn, inp.playerId)["qc"]
        conn.close()

    return {
        "ok": True,
        "totalPlayers": total,
        "onlineNow": online,
        "yourRank": rank,
        "yourTotalEarnedQC": my_total,
    }


@app.post("/api/session/complete")
def session_complete(inp: SessionCompleteIn):
    with _lock:
        conn = _db()
        _touch_player(conn, inp.playerId, inp.displayName)

        conn.execute(
            """
            UPDATE players
            SET total_qc = total_qc + ?,
                total_xp = total_xp + ?,
                sessions = sessions + 1
            WHERE player_id = ?
            """,
            (float(inp.qcEarned or 0), float(inp.xpEarned or 0), inp.playerId),
        )

        conn.commit()

        online = _online_now(conn)
        total = _total_players(conn)
        rank = _rank_for(conn, inp.playerId)
        my_total = _get_player_totals(conn, inp.playerId)["qc"]
        conn.close()

    return {
        "ok": True,
        "totalPlayers": total,
        "onlineNow": online,
        "yourRank": rank,
        "yourTotalEarnedQC": my_total,
    }


@app.post("/api/social/claim")
def social_claim(inp: SocialClaimIn):
    user_id = inp.playerId
    if not _rl_hit(user_id, "social_claim", limit=10, window_sec=60):
        return {"ok": False, "error": "rate_limited"}

    with _lock:
        conn = _db()
        _tgwt_touch(conn, user_id)
        if not _nonce_once(conn, user_id, inp.nonce):
            conn.close()
            return {"ok": False, "error": "dup_nonce"}

        ok = _tgwt_award_pending_from_pool(conn, user_id, 1 * UNIT, "SOCIAL_FOLLOW", {"platform": inp.platform})
        if not ok:
            conn.close()
            return {"ok": False, "error": "pool_exhausted"}

        st = _tgwt_get(conn, user_id)
        conn.commit()
        conn.close()

    return {"ok": True, "userId": user_id, "rev": st["rev"], "balances": st}


@app.get("/api/stats/global")
def stats_global(playerId: str):
    with _lock:
        conn = _db()
        _touch_player(conn, playerId, None)
        conn.commit()

        online = _online_now(conn)
        total = _total_players(conn)
        rank = _rank_for(conn, playerId)
        my_total = _get_player_totals(conn, playerId)["qc"]
        total_qc_all = _total_qc_all(conn)
        conn.close()

    return {
        "totalPlayers": total,
        "onlineNow": online,
        "yourRank": rank,
        "yourTotalEarnedQC": my_total,
        "totalQCEarned": total_qc_all,
    }


@app.get("/api/leaderboard/top")
def leaderboard_top(limit: int = 10):
    limit = max(1, min(int(limit), 50))
    now_ts = _now()

    with _lock:
        conn = _db()
        cur = conn.execute(
            """
            SELECT player_id, COALESCE(display_name, '') as name, total_qc, sessions, last_seen
            FROM players
            ORDER BY total_qc DESC, sessions DESC, last_seen DESC
            LIMIT ?
            """,
            (limit,),
        )
        rows = cur.fetchall()
        conn.close()

    out = []
    for i, (pid, name, total_qc, sessions, last_seen) in enumerate(rows, start=1):
        safe_name = name or f"Player {pid[:6]}"
        out.append({
            "rank": i,
            "playerId": pid,
            "name": safe_name,
            "displayName": safe_name,
            "qc": float(total_qc),
            "sessions": int(sessions),
            "lastSeen": int(last_seen),
            "lastSeenSecAgo": int(max(0, now_ts - int(last_seen))),
        })

    return {"rows": out}


# ------------------- Panels (stub/real-ish) -------------------

@app.get("/api/wallet")
async def wallet(playerId: str):
    # QC_WALLET_REAL_BALANCE_V1
    # Returns game wallet values plus linked real QuantumCoin chain balance.
    with _lock:
        conn = _db()
        _touch_player(conn, playerId, None)
        _ensure_qc_address_column(conn)
        payload = _wallet_payload(conn, playerId)

        row = conn.execute(
            "SELECT qc_address FROM players WHERE player_id=?",
            (playerId,),
        ).fetchone()

        # QC_WALLET_WITHDRAW_SUMMARY_V1
        _ensure_reward_settlement_columns(conn)
        _ensure_reward_chain_columns(conn)
        _ensure_withdraw_batch_tables(conn)
        _ensure_withdraw_payout_meta_columns(conn)

        withdrawable_row = conn.execute(
            """
            SELECT COUNT(*), COALESCE(SUM(amount), 0)
            FROM reward_records
            WHERE player_id=?
              AND settlement_status='READY_FOR_CHAIN'
              AND tx_hash IS NULL
              AND COALESCE(target_qc_address, '') != ''
              AND COALESCE(withdraw_batch_id, '') = ''
            """,
            (playerId,),
        ).fetchone()

        requested_row = conn.execute(
            """
            SELECT COUNT(*), COALESCE(SUM(amount), 0)
            FROM withdraw_batches
            WHERE player_id=?
              AND status IN ('REQUESTED', 'PENDING_CHAIN')
            """,
            (playerId,),
        ).fetchone()

        latest_batch = conn.execute(
            """
            SELECT batch_id, amount, reward_count, status, tx_hash,
                   target_qc_address, min_chain_height, max_chain_height,
                   created_at, updated_at
            FROM withdraw_batches
            WHERE player_id=?
            ORDER BY created_at DESC
            LIMIT 1
            """,
            (playerId,),
        ).fetchone()

        conn.commit()
        conn.close()

    qc_address = (row[0] if row else "") or ""

    payload["linked_qc_address"] = qc_address
    payload["real_qc_balance"] = None
    payload["real_qc_spendable"] = None
    payload["real_qc_height"] = None
    payload["real_qc_ok"] = False

    payload["withdrawableQc"] = float((withdrawable_row[1] if withdrawable_row else 0) or 0)
    payload["withdrawableRewardCount"] = int((withdrawable_row[0] if withdrawable_row else 0) or 0)
    payload["requestedWithdrawQc"] = float((requested_row[1] if requested_row else 0) or 0)
    payload["requestedWithdrawCount"] = int((requested_row[0] if requested_row else 0) or 0)
    payload["minWithdrawAmount"] = QC_MIN_WITHDRAW_AMOUNT
    payload["latestWithdrawBatch"] = None

    if latest_batch:
        payload["latestWithdrawBatch"] = {
            "batchId": latest_batch[0],
            "amount": float(latest_batch[1] or 0),
            "rewardCount": int(latest_batch[2] or 0),
            "status": latest_batch[3],
            "txHash": latest_batch[4],
            "targetQcAddress": latest_batch[5],
            "minChainHeight": int(latest_batch[6] or 0),
            "maxChainHeight": int(latest_batch[7] or 0),
            "createdAt": int(latest_batch[8] or 0),
            "updatedAt": int(latest_batch[9] or 0),
        }

    if qc_address:
        real = await qc_fetch_address_balance(qc_address)
        payload["real_qc"] = real
        payload["real_qc_ok"] = bool(real.get("ok"))
        payload["real_qc_balance"] = real.get("balance")
        payload["real_qc_spendable"] = real.get("spendable")
        payload["real_qc_height"] = real.get("height")
    else:
        payload["real_qc"] = {
            "ok": False,
            "linked": False,
            "error": "qc_address_not_linked",
        }

    return payload

@app.get("/api/store")
def store():
    return {"ships": [
        {"id": "starter", "name": "Starter Shuttle", "price_qc": 0},
        {"id": "miner_mk1", "name": "Miner MK-1", "price_qc": 250},
        {"id": "miner_mk2", "name": "Miner MK-2", "price_qc": 750},
    ]}


@app.post("/api/store/buy")
def store_buy(inp: StoreBuyIn):
    user_id = inp.playerId
    if not _rl_hit(user_id, "store_buy", limit=20, window_sec=60):
        return {"ok": False, "error": "rate_limited"}

    bonus_u = _store_bonus_u(inp.shipId)
    if bonus_u <= 0:
        return {"ok": True, "bonus": "0", "bonus_u": 0}

    with _lock:
        conn = _db()
        _tgwt_touch(conn, user_id)

        if not _nonce_once(conn, user_id, inp.nonce):
            conn.close()
            return {"ok": False, "error": "dup_nonce"}

        ok = _tgwt_award_pending_from_pool(conn, user_id, bonus_u, "STORE_BONUS", {"shipId": inp.shipId})
        if not ok:
            conn.close()
            return {"ok": False, "error": "pool_exhausted"}

        st = _tgwt_get(conn, user_id)
        conn.commit()
        conn.close()

    return {
        "ok": True,
        "bonus": str(bonus_u / UNIT),
        "bonus_u": bonus_u,
        "rev": st["rev"],
        "balances": st,
    }


@app.get("/api/social/links")
def social_links():
    return {"links": {
        "website": "https://qcnetwork.ai/",
        "x": "https://x.com/QuantumCoinQC",
        "telegram": "https://web.telegram.org/a/#-1002870924021",
        "tiktok": "https://www.tiktok.com/@quantumcoin21",
        "youtube": "https://www.youtube.com/@QuantumCoinHQ",
    }}


@app.get("/api/watch")
def watch():
    return {"enabled": True, "daily_limit": 5, "reward_tgwt": 1}


@app.get("/api/settings")
def get_settings(playerId: str):
    with _lock:
        conn = _db()
        cur = conn.execute("SELECT lang, theme FROM settings WHERE player_id=?", (playerId,))
        row = cur.fetchone()
        conn.close()
    if not row:
        return {"playerId": playerId, "lang": None, "theme": None}
    return {"playerId": playerId, "lang": row[0], "theme": row[1]}


@app.post("/api/settings")
def save_settings(inp: SettingsIn):
    with _lock:
        conn = _db()
        conn.execute(
            """
            INSERT INTO settings(player_id, lang, theme, updated_at)
            VALUES(?,?,?,?)
            ON CONFLICT(player_id) DO UPDATE SET
                lang=COALESCE(excluded.lang, lang),
                theme=COALESCE(excluded.theme, theme),
                updated_at=excluded.updated_at
            """,
            (inp.playerId, inp.lang, inp.theme, _now()),
        )
        conn.commit()
        cur = conn.execute("SELECT lang, theme FROM settings WHERE player_id=?", (inp.playerId,))
        row = cur.fetchone()
        conn.close()
    return {"ok": True, "playerId": inp.playerId, "lang": row[0], "theme": row[1]}


# ------------------- Mining dev endpoints -------------------

@app.post("/api/mine/start")
def mine_start(inp: MineStartIn):
    with _lock:
        conn = _db()
        _touch_player(conn, inp.playerId, inp.displayName)

        conn.execute(
            """
            INSERT INTO mining_sessions(player_id, site_id, started_at)
            VALUES(?,?,?)
            ON CONFLICT(player_id) DO UPDATE SET
                site_id=excluded.site_id,
                started_at=excluded.started_at
            """,
            (inp.playerId, int(inp.siteId), _now()),
        )
        conn.commit()
        payload = _wallet_payload(conn, inp.playerId)
        conn.close()

    return {"ok": True, "started": True, "siteId": int(inp.siteId), "wallet": payload}


@app.post("/api/mine/finish")
def mine_finish(inp: MineFinishIn):
    with _lock:
        conn = _db()
        _touch_player(conn, inp.playerId, inp.displayName)

        cur = conn.execute("SELECT started_at FROM mining_sessions WHERE player_id=?", (inp.playerId,))
        row = cur.fetchone()
        started_at = int(row[0]) if row and row[0] else 0

        duration = float(inp.durationSec or 0.0)
        if duration <= 0 and started_at > 0:
            duration = float(max(0, _now() - started_at))

        target_qc_address = _get_player_qc_address(conn, inp.playerId)

        if not target_qc_address:
            conn.close()
            return {
                "ok": False,
                "error": "qc_address_not_linked",
                "message": "Link or create a QC wallet address before real game mining.",
                "playerId": inp.playerId,
            }

        # QC_REAL_GAME_MINE_BRIDGE_V1
        # Telegram Game artık oyun ödülünü sahte yazmıyor:
        # bağlı QC adresi ile gerçek QuantumCoin node /api/mine çağrılır.
        try:
            qc_mine_result = asyncio.run(qc_fetch_mine_block(target_qc_address))
        except RuntimeError:
            loop = asyncio.new_event_loop()
            try:
                qc_mine_result = loop.run_until_complete(qc_fetch_mine_block(target_qc_address))
            finally:
                loop.close()

        if not bool(qc_mine_result.get("success") or qc_mine_result.get("ok")):
            conn.close()
            return {
                "ok": False,
                "error": "qc_real_mine_failed",
                "playerId": inp.playerId,
                "targetQcAddress": target_qc_address,
                "qcMineResult": qc_mine_result,
            }

        qc_health = _fetch_qc_node_health_sync()
        chain_height = int(qc_mine_result.get("height") or _current_chain_height_from_qc_health(qc_health) or 0)
        block_hash = str(qc_mine_result.get("block_hash") or qc_mine_result.get("blockHash") or "")
        qc = float(qc_mine_result.get("reward") or QC_GAME_MINER_BLOCK_REWARD)

        if inp.xpEarned is None:
            rw = _compute_rewards(duration)
            xp = float(rw["xp"])
        else:
            xp = float(inp.xpEarned or 0.0)

        block_reward = QC_GAME_MINER_BLOCK_REWARD
        mining_client_type = QC_GAME_MINER_CLIENT_TYPE
        network_reward_mode = QC_GAME_MINER_REWARD_MODE

        conn.execute(
            """
            UPDATE players
            SET total_qc = total_qc + ?,
                total_xp = total_xp + ?,
                sessions = sessions + 1
            WHERE player_id = ?
            """,
            (qc, xp, inp.playerId),
        )

        reward_id = _reward_record_insert(
            conn,
            player_id=inp.playerId,
            source="MINING_SESSION",
            zone_id=f"LUNA-{int(inp.siteId):02d}",
            site_id=int(inp.siteId),
            amount=qc,
            xp_amount=xp,
            meta={
                "durationSec": duration,
                "startedAt": started_at,
                "mode": "GAME_MINER_NETWORK_REWARD",
                "qcNodeHealth": qc_health,
                "qcMineResult": qc_mine_result,
                "blockHash": block_hash,
                "targetQcAddress": target_qc_address,
            },
            chain_height=chain_height,
            block_reward=block_reward,
            mining_client_type=mining_client_type,
            network_reward_mode=network_reward_mode,
        )

        conn.execute("DELETE FROM mining_sessions WHERE player_id=?", (inp.playerId,))

        settlement_row = conn.execute(
            "SELECT settlement_status FROM reward_records WHERE reward_id=?",
            (reward_id,),
        ).fetchone()
        settlement_status = settlement_row[0] if settlement_row else "UNKNOWN"

        # TGWT random drop (backend decides)
        if random.random() < DROP_CHANCE:
            _tgwt_award_pending_from_pool(conn, inp.playerId, DROP_AMOUNT_U, "RANDOM_DROP", {"siteId": int(inp.siteId)})

        conn.commit()

        online = _online_now(conn)
        total = _total_players(conn)
        rank = _rank_for(conn, inp.playerId)
        my_total = _get_player_totals(conn, inp.playerId)["qc"]
        wallet_payload = _wallet_payload(conn, inp.playerId)
        conn.close()

    return {
        "ok": True,
        "durationSec": duration,
        "qcEarned": qc,
        "xpEarned": xp,
        "totalPlayers": total,
        "onlineNow": online,
        "yourRank": rank,
        "yourTotalEarnedQC": my_total,
        "wallet": wallet_payload,
        "rewardId": reward_id,
        "settlementStatus": settlement_status,
        "chainHeight": chain_height,
        "blockReward": block_reward,
        "blockHash": block_hash,
        "miningClientType": mining_client_type,
        "networkRewardMode": network_reward_mode,
    }


# TGWT_WITHDRAW_ADMIN_REVIEW_V1
# ------------------- TGWT v1 endpoints -------------------

@app.get("/api/v1/tgwt/state")
def tgwt_state(user_id: str):
    with _lock:
        conn = _db()
        st = _tgwt_get(conn, user_id)
        conn.close()
    return {"ok": True, "userId": user_id, "rev": st["rev"], "balances": st, "ts": _now()}


@app.get("/api/v1/tgwt/ledger")
def tgwt_ledger(user_id: str, limit: int = 20):
    limit = max(1, min(int(limit), 50))
    with _lock:
        conn = _db()
        st = _tgwt_get(conn, user_id)
        pool = _pool_get(conn)
        cur = conn.execute(
            "SELECT id, kind, amount_u, meta_json, created_at FROM tgwt_events WHERE user_id=? ORDER BY created_at DESC LIMIT ?",
            (user_id, limit),
        )
        ev = cur.fetchall()
        conn.close()

    events = []
    for r in ev:
        events.append({
            "id": r[0],
            "kind": r[1],
            "amount": str((int(r[2] or 0) / UNIT)),
            "amount_u": int(r[2] or 0),
            "meta": json.loads(r[3] or "{}"),
            "createdAt": int(r[4] or 0),
        })

    return {
        "ok": True,
        "userId": user_id,
        "pool": {"total": str(pool["total_u"] / UNIT), "distributed": str(pool["distributed_u"] / UNIT)},
        "rev": st["rev"],
        "balances": st,
        "events": events,
        "ts": _now(),
    }


@app.post("/api/v1/tgwt/earn")
def tgwt_earn(inp: TGWTEarnIn):
    amt_u = _parse_amount_to_micro(inp.amount)
    if amt_u <= 0:
        return {"ok": False, "error": "invalid_amount"}

    user_id = inp.userId
    if not _rl_hit(user_id, "tgwt_earn", limit=120, window_sec=60):
        return {"ok": False, "error": "rate_limited"}

    with _lock:
        conn = _db()
        _tgwt_touch(conn, user_id)
        if not _nonce_once(conn, user_id, inp.nonce):
            conn.close()
            return {"ok": False, "error": "dup_nonce"}

        ok = _tgwt_award_pending_from_pool(conn, user_id, amt_u, "EARN", inp.meta or {})
        if not ok:
            conn.close()
            return {"ok": False, "error": "pool_exhausted"}

        st = _tgwt_get(conn, user_id)
        conn.commit()
        conn.close()

    return {"ok": True, "rev": st["rev"], "balances": st}


@app.post("/api/v1/tgwt/verify")
def tgwt_verify(inp: TGWTVerifyIn):
    user_id = inp.userId
    if not _rl_hit(user_id, "tgwt_verify", limit=20, window_sec=60):
        return {"ok": False, "error": "rate_limited"}

    with _lock:
        conn = _db()
        _tgwt_touch(conn, user_id)
        if not _nonce_once(conn, user_id, inp.nonce):
            conn.close()
            return {"ok": False, "error": "dup_nonce"}

        st = _tgwt_get(conn, user_id)
        if inp.amount is None:
            want_u = int(st["pending_u"])
        else:
            want_u = _parse_amount_to_micro(inp.amount)
            if want_u <= 0:
                conn.close()
                return {"ok": False, "error": "invalid_amount"}

        moved_u = _tgwt_move_pending_to_verified(conn, user_id, want_u)
        st2 = _tgwt_get(conn, user_id)
        conn.commit()
        conn.close()

    return {"ok": True, "moved_u": moved_u, "rev": st2["rev"], "balances": st2}


@app.post("/api/v1/withdraw/request")
def withdraw_request(inp: WithdrawRequestIn):
    asset = (inp.asset or "").upper().strip()
    if asset != "TGWT":
        return {"ok": False, "error": "unsupported_asset"}

    user_id = inp.userId
    if not _rl_hit(user_id, "withdraw_request", limit=10, window_sec=60):
        return {"ok": False, "error": "rate_limited"}

    to = (inp.toAddress or "").strip()
    if not (len(to) == 42 and to.startswith("0x")):
        return {"ok": False, "error": "invalid_to_address"}

    if inp.amount is None or inp.amount <= 0:
        return {"ok": False, "error": "invalid_amount"}

    amount_u = int(round(float(inp.amount) * UNIT))
    if amount_u <= 0:
        return {"ok": False, "error": "invalid_amount"}

    fee_u = _fee_u(amount_u)
    total_u = amount_u + fee_u
    req_id = secrets.token_hex(8)

    with _lock:
        conn = _db()
        _tgwt_touch(conn, user_id)

        if not _nonce_once(conn, user_id, inp.nonce):
            conn.close()
            return {"ok": False, "error": "dup_nonce"}

        st = _tgwt_get(conn, user_id)
        if total_u > st["verified_u"]:
            conn.close()
            return {"ok": False, "error": "insufficient_verified"}

        # verified -> withdrawn (fee düş)
        conn.execute(
            """
            UPDATE tgwt_state
            SET verified_u = verified_u - ?,
                withdrawn_u = withdrawn_u + ?
            WHERE user_id=?
            """,
            (total_u, amount_u, user_id),
        )
        _tgwt_bump_rev(conn, user_id)

        # request kaydı
        conn.execute(
            """
            INSERT INTO withdraw_requests(id, user_id, asset, amount_u, to_address, status, created_at, updated_at)
            VALUES(?,?,?,?,?,?,?,?)
            """,
            (req_id, user_id, asset, amount_u, to, "ADMIN_REVIEW", _now(), _now()),
        )

        _tgwt_event(conn, user_id, "WITHDRAW_REQUEST", amount_u, {"to": to, "fee_u": fee_u})

        st2 = _tgwt_get(conn, user_id)
        conn.commit()
        conn.close()

    # TGWT_WITHDRAW_ADMIN_REVIEW_V1:
    # No automatic BSC transfer and no dev auto-confirm here.
    # Final/testnet phase will process this request through treasury/BEP-20 payout.
    return {
        "ok": True,
        "requestId": req_id,
        "status": "ADMIN_REVIEW",
        "message": "TGWT withdrawal request created. Real BSC payout will be processed after admin review in the final/testnet payout module.",
        "fee": str(fee_u / UNIT),
        "rev": st2["rev"],
        "balances": st2,
    }


@app.get("/api/v1/withdraw/status")
def withdraw_status(id: str):
    with _lock:
        conn = _db()
        cur = conn.execute(
            """
            SELECT id, user_id, asset, amount_u, to_address, status, tx_hash, created_at, updated_at
            FROM withdraw_requests WHERE id=?
            """,
            (id,),
        )
        row = cur.fetchone()
        conn.close()
    if not row:
        return {"ok": False, "error": "not_found"}
    return {"ok": True, "request": {
        "id": row[0],
        "userId": row[1],
        "asset": row[2],
        "amount": str((int(row[3] or 0) / UNIT)),
        "amount_u": int(row[3] or 0),
        "toAddress": row[4],
        "status": row[5],
        "txHash": row[6],
        "createdAt": int(row[7] or 0),
        "updatedAt": int(row[8] or 0),
    }}


@app.get("/api/v1/withdraw/list")
def withdraw_list(user_id: str, limit: int = 10):
    limit = max(1, min(int(limit), 50))
    with _lock:
        conn = _db()
        cur = conn.execute(
            """
            SELECT id, asset, amount_u, to_address, status, tx_hash, created_at
            FROM withdraw_requests
            WHERE user_id=?
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (user_id, limit),
        )
        rows = cur.fetchall()
        conn.close()

    items = []
    for r in rows:
        items.append({
            "id": r[0],
            "asset": r[1],
            "amount": str((int(r[2] or 0) / UNIT)),
            "amount_u": int(r[2] or 0),
            "toAddress": r[3],
            "status": r[4],
            "txHash": r[5],
            "createdAt": int(r[6] or 0),
        })
    return {"ok": True, "items": items}


# ------------------- DEV ONLY reset -------------------



@app.get("/api/rewards/history")
def rewards_history(playerId: str, limit: int = 20):
    limit = max(1, min(int(limit), 100))
    with _lock:
        conn = _db()
        _touch_player(conn, playerId, None)
        cur = conn.execute(
            """
            SELECT reward_id, source, zone_id, site_id, asset, amount, xp_amount,
                   status, settlement_status, tx_hash, created_at, confirmed_at,
                   chain_height, block_reward, mining_client_type, network_reward_mode
            FROM reward_records
            WHERE player_id=?
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (playerId, limit),
        )
        rows = cur.fetchall()
        conn.commit()
        conn.close()

    return {
        "ok": True,
        "playerId": playerId,
        "rows": [
            {
                "rewardId": r[0],
                "source": r[1],
                "zoneId": r[2],
                "siteId": int(r[3] or 0),
                "asset": r[4],
                "amount": float(r[5] or 0),
                "xpAmount": float(r[6] or 0),
                "status": r[7],
                "settlementStatus": r[8],
                "txHash": r[9],
                "createdAt": int(r[10] or 0),
                "confirmedAt": int(r[11] or 0),
                "chainHeight": int(r[12] or 0),
                "blockReward": float(r[13] or 0),
                "miningClientType": r[14] or "",
                "networkRewardMode": r[15] or "",
            }
            for r in rows
        ],
    }




# QC_API_CHAIN_FIELDS_RESPONSE_V1

# QC_NODE_READ_ENDPOINTS_V1
@app.get("/api/qc/health")
async def qc_node_health():
    return await qc_fetch_health()


@app.get("/api/qc/address/balance")
async def qc_address_balance(addr: str):
    return await qc_fetch_address_balance(addr)


# QC_TX_LOCAL_LEDGER_V1
def _ensure_qc_tx_records_table(conn):
    conn.execute("""
    CREATE TABLE IF NOT EXISTS qc_tx_records (
        txid TEXT PRIMARY KEY,
        player_id TEXT NOT NULL DEFAULT '',
        from_address TEXT NOT NULL DEFAULT '',
        to_address TEXT NOT NULL DEFAULT '',
        amount REAL NOT NULL DEFAULT 0,
        status TEXT NOT NULL DEFAULT 'PENDING_CHAIN',
        created_height INTEGER NOT NULL DEFAULT 0,
        confirmed_height INTEGER NOT NULL DEFAULT 0,
        from_balance_before REAL NOT NULL DEFAULT 0,
        to_balance_before REAL NOT NULL DEFAULT 0,
        created_at INTEGER NOT NULL DEFAULT 0,
        checked_at INTEGER NOT NULL DEFAULT 0,
        note TEXT NOT NULL DEFAULT '',
        direct_json TEXT NOT NULL DEFAULT ''
    )
    """)
    conn.execute("CREATE INDEX IF NOT EXISTS idx_qc_tx_records_player_created ON qc_tx_records(player_id, created_at DESC)")
    conn.execute("CREATE INDEX IF NOT EXISTS idx_qc_tx_records_status ON qc_tx_records(status, created_at DESC)")


def _safe_float(v, default=0.0):
    try:
        return float(v)
    except Exception:
        return default


def _safe_int(v, default=0):
    try:
        return int(v)
    except Exception:
        return default


def _save_qc_tx_record(
    player_id,
    txid,
    from_address,
    to_address,
    amount,
    created_height,
    from_balance_before,
    to_balance_before,
    direct_json,
):
    if not txid:
        return

    with _lock:
        conn = _db()
        _ensure_qc_tx_records_table(conn)
        now_ts = _now()
        conn.execute(
            """
            INSERT OR REPLACE INTO qc_tx_records(
                txid, player_id, from_address, to_address, amount,
                status, created_height, confirmed_height,
                from_balance_before, to_balance_before,
                created_at, checked_at, note, direct_json
            )
            VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
            """,
            (
                txid,
                player_id or "",
                from_address or "",
                to_address or "",
                _safe_float(amount),
                "PENDING_CHAIN",
                _safe_int(created_height),
                0,
                _safe_float(from_balance_before),
                _safe_float(to_balance_before),
                now_ts,
                now_ts,
                "Created by Telegram Game /api/wallet/send-qc; waiting for block confirmation.",
                direct_json or "",
            ),
        )
        conn.commit()
        conn.close()


def _get_qc_tx_record(txid):
    with _lock:
        conn = _db()
        _ensure_qc_tx_records_table(conn)
        row = conn.execute(
            """
            SELECT txid, player_id, from_address, to_address, amount,
                   status, created_height, confirmed_height,
                   from_balance_before, to_balance_before,
                   created_at, checked_at, note, direct_json
            FROM qc_tx_records
            WHERE txid=?
            """,
            ((txid or "").strip(),),
        ).fetchone()
        conn.commit()
        conn.close()

    if not row:
        return None

    return {
        "txid": row[0],
        "playerId": row[1],
        "fromAddress": row[2],
        "toAddress": row[3],
        "amount": _safe_float(row[4]),
        "status": row[5],
        "createdHeight": _safe_int(row[6]),
        "confirmedHeight": _safe_int(row[7]),
        "fromBalanceBefore": _safe_float(row[8]),
        "toBalanceBefore": _safe_float(row[9]),
        "createdAt": _safe_int(row[10]),
        "checkedAt": _safe_int(row[11]),
        "note": row[12] or "",
        "directJson": row[13] or "",
    }


def _mark_qc_tx_record_confirmed(txid, confirmed_height, note):
    with _lock:
        conn = _db()
        _ensure_qc_tx_records_table(conn)
        conn.execute(
            """
            UPDATE qc_tx_records
            SET status='CONFIRMED_CHAIN_LOCAL',
                confirmed_height=?,
                checked_at=?,
                note=?
            WHERE txid=?
            """,
            (_safe_int(confirmed_height), _now(), note or "Confirmed by local balance/height fallback.", txid),
        )
        conn.commit()
        conn.close()


@app.get("/api/qc/tx/status")
async def qc_tx_status(txid: str):
    txid_clean = (txid or "").strip()
    if not txid_clean:
        return {"ok": False, "error": "empty_txid"}

    direct = await qc_fetch_tx_status(txid_clean)

    # If future QC node returns proper JSON confirmation, trust it.
    if isinstance(direct, dict) and bool(direct.get("ok")):
        return {
            "ok": True,
            "txid": txid_clean,
            "source": "qc_node",
            "status": direct.get("status") or direct.get("state") or "NODE_OK",
            "confirmed": bool(direct.get("confirmed", False)),
            "height": direct.get("height") or direct.get("blockHeight") or 0,
            "qcNode": direct,
        }

    # Fallback for old release quantumcoin.exe where /api/tx/status returns HTML.
    rec = _get_qc_tx_record(txid_clean)
    if not rec:
        return {
            "ok": False,
            "txid": txid_clean,
            "source": "local_fallback",
            "status": "NOT_TRACKED",
            "confirmed": False,
            "error": "tx_not_tracked_locally",
            "qcNode": direct,
        }

    to_balance = await qc_fetch_address_balance(rec["toAddress"])
    from_balance = await qc_fetch_address_balance(rec["fromAddress"])

    current_height = _safe_int(
        (to_balance or {}).get("height")
        or (from_balance or {}).get("height")
        or 0
    )

    current_to_balance = _safe_float((to_balance or {}).get("balance"), rec["toBalanceBefore"])
    expected_to_balance = _safe_float(rec["toBalanceBefore"]) + _safe_float(rec["amount"])

    confirmed_local = (
        bool((to_balance or {}).get("ok", True))
        and current_height > _safe_int(rec["createdHeight"])
        and current_to_balance + 0.00000001 >= expected_to_balance
    )

    if confirmed_local and rec["status"] != "CONFIRMED_CHAIN_LOCAL":
        _mark_qc_tx_record_confirmed(
            txid_clean,
            current_height,
            "Confirmed by fallback: recipient balance increased and chain height advanced.",
        )
        rec = _get_qc_tx_record(txid_clean) or rec

    return {
        "ok": True,
        "txid": txid_clean,
        "source": "local_fallback",
        "status": "CONFIRMED_CHAIN_LOCAL" if confirmed_local else rec["status"],
        "confirmed": bool(confirmed_local),
        "createdHeight": rec["createdHeight"],
        "confirmedHeight": current_height if confirmed_local else rec["confirmedHeight"],
        "fromAddress": rec["fromAddress"],
        "toAddress": rec["toAddress"],
        "amount": rec["amount"],
        "toBalanceBefore": rec["toBalanceBefore"],
        "toBalanceNow": current_to_balance,
        "expectedToBalance": expected_to_balance,
        "qcNode": direct,
        "note": "Fallback used because old release /api/tx/status did not return JSON.",
    }


@app.get("/api/wallet/tx-history")
def wallet_tx_history(playerId: str, limit: int = 20):
    player_id = (playerId or "").strip()
    limit = max(1, min(int(limit or 20), 50))

    with _lock:
        conn = _db()
        _ensure_qc_tx_records_table(conn)
        rows = conn.execute(
            """
            SELECT txid, player_id, from_address, to_address, amount,
                   status, created_height, confirmed_height,
                   created_at, checked_at, note
            FROM qc_tx_records
            WHERE player_id=?
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (player_id, limit),
        ).fetchall()
        conn.commit()
        conn.close()

    return {
        "ok": True,
        "playerId": player_id,
        "count": len(rows),
        "items": [
            {
                "txid": r[0],
                "playerId": r[1],
                "fromAddress": r[2],
                "toAddress": r[3],
                "amount": _safe_float(r[4]),
                "status": r[5],
                "createdHeight": _safe_int(r[6]),
                "confirmedHeight": _safe_int(r[7]),
                "createdAt": _safe_int(r[8]),
                "checkedAt": _safe_int(r[9]),
                "note": r[10] or "",
            }
            for r in rows
        ],
    }



# QC_ADDRESS_LINK_V1
class LinkQCAddressIn(BaseModel):
    playerId: str
    address: str


# QC_WALLET_SEND_QC_V1
class WalletSendQCIn(BaseModel):
    playerId: str
    toAddress: str
    amount: str
    privHex: str


@app.post("/api/wallet/link-qc-address")
async def link_qc_address(inp: LinkQCAddressIn):
    player_id = (inp.playerId or "").strip()
    address = (inp.address or "").strip()

    if not player_id:
        return {"ok": False, "error": "empty playerId"}

    if not address:
        return {"ok": False, "error": "empty address"}

    # Önce QC node üzerinden adres okunabiliyor mu kontrol et.
    balance = await qc_fetch_address_balance(address)

    if not balance.get("ok"):
        return {
            "ok": False,
            "playerId": player_id,
            "address": address,
            "error": "qc_address_not_readable",
            "qcNode": balance,
        }

    with _lock:
        conn = _db()
        _touch_player(conn, player_id, None)
        _ensure_qc_address_column(conn)
        conn.execute(
            "UPDATE players SET qc_address=?, last_seen=? WHERE player_id=?",
            (address, _now(), player_id),
        )
        conn.commit()
        conn.close()

    return {
        "ok": True,
        "playerId": player_id,
        "qcAddress": address,
        "realBalance": balance,
    }


@app.get("/api/wallet/real-balance")
async def wallet_real_balance(playerId: str):
    player_id = (playerId or "").strip()

    if not player_id:
        return {"ok": False, "error": "empty playerId"}

    with _lock:
        conn = _db()
        _touch_player(conn, player_id, None)
        _ensure_qc_address_column(conn)
        row = conn.execute(
            "SELECT qc_address FROM players WHERE player_id=?",
            (player_id,),
        ).fetchone()
        conn.commit()
        conn.close()

    qc_address = (row[0] if row else "") or ""

    if not qc_address:
        return {
            "ok": False,
            "playerId": player_id,
            "linked": False,
            "error": "qc_address_not_linked",
        }

    balance = await qc_fetch_address_balance(qc_address)

    return {
        "ok": bool(balance.get("ok")),
        "playerId": player_id,
        "linked": True,
        "qcAddress": qc_address,
        "realBalance": balance,
    }


# QC_WALLET_SEND_QC_V1
@app.post("/api/wallet/send-qc")
async def wallet_send_qc(inp: WalletSendQCIn):

    # PUBLIC_GIT_SAFE_PRIVHEX_API_DISABLED_V1
    # Private-key based DEV send is disabled by default in public GitHub builds.
    # To use locally during controlled development only, set:
    # QC_ENABLE_DEV_PRIVHEX_SEND=1
    if os.getenv("QC_ENABLE_DEV_PRIVHEX_SEND", "").strip().lower() not in ("1", "true", "yes"):
        return {
            "ok": False,
            "error": "disabled_public_build",
            "message": "Private-key based DEV send is disabled in the public GitHub build. Use secure wallet signing in final builds."
        }

    player_id = (inp.playerId or "").strip()
    to_address = (inp.toAddress or "").strip()
    amount_str = str(inp.amount or "").strip()
    priv_hex = (inp.privHex or "").strip()

    if not player_id:
        return {"ok": False, "error": "empty playerId"}
    if not to_address:
        return {"ok": False, "error": "empty toAddress"}
    if not amount_str:
        return {"ok": False, "error": "empty amount"}
    if not priv_hex:
        return {"ok": False, "error": "empty privHex"}

    try:
        amount_val = float(amount_str)
    except Exception:
        return {"ok": False, "error": "invalid_amount"}

    if amount_val <= 0:
        return {"ok": False, "error": "invalid_amount"}

    with _lock:
        conn = _db()
        _touch_player(conn, player_id, None)
        _ensure_qc_address_column(conn)
        qc_address = _get_player_qc_address(conn, player_id)
        conn.commit()
        conn.close()

    if not qc_address:
        return {
            "ok": False,
            "error": "qc_address_not_linked",
            "playerId": player_id,
        }

    from_before = await qc_fetch_address_balance(qc_address)
    to_before = await qc_fetch_address_balance(to_address)

    created_height = _safe_int(
        (from_before or {}).get("height")
        or (to_before or {}).get("height")
        or 0
    )
    from_balance_before = _safe_float((from_before or {}).get("balance"), 0.0)
    to_balance_before = _safe_float((to_before or {}).get("balance"), 0.0)

    send_result = await qc_fetch_send_tx(
        from_address=qc_address,
        to_address=to_address,
        amount=amount_str,
        priv_hex=priv_hex,
    )

    if not bool(send_result.get("success") or send_result.get("ok")):
        return {
            "ok": False,
            "error": "qc_send_failed",
            "playerId": player_id,
            "fromAddress": qc_address,
            "toAddress": to_address,
            "amount": amount_str,
            "qcSendResult": send_result,
        }

    txid = (
        send_result.get("txid")
        or send_result.get("tx_hash")
        or send_result.get("txHash")
        or ""
    )

    try:
        _save_qc_tx_record(
            player_id=player_id,
            txid=txid,
            from_address=qc_address,
            to_address=to_address,
            amount=amount_str,
            created_height=created_height,
            from_balance_before=from_balance_before,
            to_balance_before=to_balance_before,
            direct_json=json.dumps(send_result, ensure_ascii=False),
        )
    except Exception:
        pass

    real_balance = await qc_fetch_address_balance(qc_address)

    return {
        "ok": True,
        "playerId": player_id,
        "fromAddress": qc_address,
        "toAddress": to_address,
        "amount": amount_str,
        "txid": txid,
        "status": "PENDING_CHAIN",
        "message": "QC transaction created. Mine a block to include it in chain.",
        "qcSendResult": send_result,
        "realBalance": real_balance,
    }




# QC_SETTLEMENT_QUEUE_V1
@app.get("/api/settlement/queue")
def settlement_queue(limit: int = 20):
    limit = max(1, min(int(limit), 100))

    with _lock:
        conn = _db()
        _ensure_reward_settlement_columns(conn)
        _ensure_reward_chain_columns(conn)

        rows = conn.execute(
            """
            SELECT reward_id, player_id, source, zone_id, site_id, asset,
                   amount, xp_amount, status, settlement_status, tx_hash,
                   target_qc_address, settlement_note, created_at, confirmed_at,
                   settlement_updated_at, chain_height, block_reward,
                   mining_client_type, network_reward_mode
            FROM reward_records
            WHERE settlement_status = 'READY_FOR_CHAIN'
              AND tx_hash IS NULL
              AND COALESCE(target_qc_address, '') != ''
            ORDER BY created_at ASC
            LIMIT ?
            """,
            (limit,),
        ).fetchall()

        conn.commit()
        conn.close()

    return {
        "ok": True,
        "status": "READY_FOR_CHAIN",
        "count": len(rows),
        "rows": [
            {
                "rewardId": r[0],
                "playerId": r[1],
                "source": r[2],
                "zoneId": r[3],
                "siteId": int(r[4] or 0),
                "asset": r[5],
                "amount": float(r[6] or 0),
                "xpAmount": float(r[7] or 0),
                "status": r[8],
                "settlementStatus": r[9],
                "txHash": r[10],
                "targetQcAddress": r[11],
                "settlementNote": r[12],
                "createdAt": int(r[13] or 0),
                "confirmedAt": int(r[14] or 0),
                "settlementUpdatedAt": int(r[15] or 0),
                "chainHeight": int(r[16] or 0),
                "blockReward": float(r[17] or 0),
                "miningClientType": r[18] or "",
                "networkRewardMode": r[19] or "",
            }
            for r in rows
        ],
    }






# QC_SETTLEMENT_DEV_ACTION_GUARD_V1
def _settlement_dev_actions_enabled() -> bool:
    return os.getenv("QC_SETTLEMENT_DEV_ACTIONS", "0").strip().lower() in ("1", "true", "yes", "on")


def _settlement_dev_actions_error():
    return {
        "ok": False,
        "error": "settlement_dev_actions_disabled",
        "message": "Set QC_SETTLEMENT_DEV_ACTIONS=1 only in local/dev mode to enable settlement state-changing actions.",
    }


# QC_SETTLEMENT_MARK_PENDING_V1
class SettlementMarkPendingIn(BaseModel):
    rewardId: str


@app.post("/api/settlement/mark-pending")
def settlement_mark_pending(inp: SettlementMarkPendingIn):
    reward_id = (inp.rewardId or "").strip()

    if not reward_id:
        return {"ok": False, "error": "empty rewardId"}

    if not _settlement_dev_actions_enabled():
        return _settlement_dev_actions_error()

    with _lock:
        conn = _db()
        _ensure_reward_settlement_columns(conn)

        row = conn.execute(
            """
            SELECT reward_id, player_id, source, zone_id, site_id, asset,
                   amount, xp_amount, status, settlement_status, tx_hash,
                   target_qc_address, settlement_note, created_at, confirmed_at,
                   settlement_updated_at, chain_height, block_reward,
                   mining_client_type, network_reward_mode
            FROM reward_records
            WHERE reward_id=?
            """,
            (reward_id,),
        ).fetchone()

        if not row:
            conn.close()
            return {"ok": False, "error": "reward_not_found", "rewardId": reward_id}

        current_status = row[9]
        tx_hash = row[10]
        target_qc_address = (row[11] or "").strip()

        if current_status != "READY_FOR_CHAIN":
            conn.close()
            return {
                "ok": False,
                "error": "invalid_settlement_status",
                "rewardId": reward_id,
                "currentStatus": current_status,
                "expectedStatus": "READY_FOR_CHAIN",
            }

        if tx_hash:
            conn.close()
            return {
                "ok": False,
                "error": "tx_hash_already_exists",
                "rewardId": reward_id,
                "txHash": tx_hash,
            }

        if not target_qc_address:
            conn.close()
            return {
                "ok": False,
                "error": "missing_target_qc_address",
                "rewardId": reward_id,
            }

        now_ts = _now()

        conn.execute(
            """
            UPDATE reward_records
            SET settlement_status='PENDING_CHAIN',
                settlement_note=?,
                settlement_updated_at=?
            WHERE reward_id=?
            """,
            (
                "Marked as PENDING_CHAIN. Waiting for future QC node settlement transaction.",
                now_ts,
                reward_id,
            ),
        )

        conn.commit()

        updated = conn.execute(
            """
            SELECT reward_id, player_id, source, zone_id, site_id, asset,
                   amount, xp_amount, status, settlement_status, tx_hash,
                   target_qc_address, settlement_note, created_at, confirmed_at,
                   settlement_updated_at, chain_height, block_reward,
                   mining_client_type, network_reward_mode
            FROM reward_records
            WHERE reward_id=?
            """,
            (reward_id,),
        ).fetchone()

        conn.close()

    return {
        "ok": True,
        "reward": {
            "rewardId": updated[0],
            "playerId": updated[1],
            "source": updated[2],
            "zoneId": updated[3],
            "siteId": int(updated[4] or 0),
            "asset": updated[5],
            "amount": float(updated[6] or 0),
            "xpAmount": float(updated[7] or 0),
            "status": updated[8],
            "settlementStatus": updated[9],
            "txHash": updated[10],
            "targetQcAddress": updated[11],
            "settlementNote": updated[12],
            "createdAt": int(updated[13] or 0),
            "confirmedAt": int(updated[14] or 0),
            "settlementUpdatedAt": int(updated[15] or 0),
        }
    }




# QC_SETTLEMENT_PENDING_V1
@app.get("/api/settlement/pending")
def settlement_pending(limit: int = 20):
    limit = max(1, min(int(limit), 100))

    with _lock:
        conn = _db()
        _ensure_reward_settlement_columns(conn)
        _ensure_reward_chain_columns(conn)

        rows = conn.execute(
            """
            SELECT reward_id, player_id, source, zone_id, site_id, asset,
                   amount, xp_amount, status, settlement_status, tx_hash,
                   target_qc_address, settlement_note, created_at, confirmed_at,
                   settlement_updated_at
            FROM reward_records
            WHERE settlement_status = 'PENDING_CHAIN'
              AND tx_hash IS NULL
              AND COALESCE(target_qc_address, '') != ''
            ORDER BY settlement_updated_at ASC, created_at ASC
            LIMIT ?
            """,
            (limit,),
        ).fetchall()

        conn.commit()
        conn.close()

    return {
        "ok": True,
        "status": "PENDING_CHAIN",
        "count": len(rows),
        "rows": [
            {
                "rewardId": r[0],
                "playerId": r[1],
                "source": r[2],
                "zoneId": r[3],
                "siteId": int(r[4] or 0),
                "asset": r[5],
                "amount": float(r[6] or 0),
                "xpAmount": float(r[7] or 0),
                "status": r[8],
                "settlementStatus": r[9],
                "txHash": r[10],
                "targetQcAddress": r[11],
                "settlementNote": r[12],
                "createdAt": int(r[13] or 0),
                "confirmedAt": int(r[14] or 0),
                "settlementUpdatedAt": int(r[15] or 0),
                "chainHeight": int(r[16] or 0),
                "blockReward": float(r[17] or 0),
                "miningClientType": r[18] or "",
                "networkRewardMode": r[19] or "",
            }
            for r in rows
        ],
    }




# QC_SETTLEMENT_MOCK_CONFIRM_V1
class SettlementMockConfirmIn(BaseModel):
    rewardId: str
    txHash: Optional[str] = None


@app.post("/api/settlement/mock-confirm")
def settlement_mock_confirm(inp: SettlementMockConfirmIn):
    reward_id = (inp.rewardId or "").strip()
    provided_tx = (inp.txHash or "").strip()

    if not reward_id:
        return {"ok": False, "error": "empty rewardId"}

    if not _settlement_dev_actions_enabled():
        return _settlement_dev_actions_error()

    with _lock:
        conn = _db()
        _ensure_reward_settlement_columns(conn)

        row = conn.execute(
            """
            SELECT reward_id, player_id, source, zone_id, site_id, asset,
                   amount, xp_amount, status, settlement_status, tx_hash,
                   target_qc_address, settlement_note, created_at, confirmed_at,
                   settlement_updated_at
            FROM reward_records
            WHERE reward_id=?
            """,
            (reward_id,),
        ).fetchone()

        if not row:
            conn.close()
            return {"ok": False, "error": "reward_not_found", "rewardId": reward_id}

        current_status = row[9]
        current_tx = row[10]
        target_qc_address = (row[11] or "").strip()

        if current_status != "PENDING_CHAIN":
            conn.close()
            return {
                "ok": False,
                "error": "invalid_settlement_status",
                "rewardId": reward_id,
                "currentStatus": current_status,
                "expectedStatus": "PENDING_CHAIN",
            }

        if current_tx:
            conn.close()
            return {
                "ok": False,
                "error": "tx_hash_already_exists",
                "rewardId": reward_id,
                "txHash": current_tx,
            }

        if not target_qc_address:
            conn.close()
            return {
                "ok": False,
                "error": "missing_target_qc_address",
                "rewardId": reward_id,
            }

        now_ts = _now()
        tx_hash = provided_tx or ("mock_tx_" + secrets.token_hex(16))

        conn.execute(
            """
            UPDATE reward_records
            SET settlement_status='CONFIRMED_CHAIN',
                tx_hash=?,
                settlement_note=?,
                settlement_updated_at=?
            WHERE reward_id=?
            """,
            (
                tx_hash,
                "Mock confirmed. Replace this with real QC node transaction confirmation later.",
                now_ts,
                reward_id,
            ),
        )

        conn.commit()

        updated = conn.execute(
            """
            SELECT reward_id, player_id, source, zone_id, site_id, asset,
                   amount, xp_amount, status, settlement_status, tx_hash,
                   target_qc_address, settlement_note, created_at, confirmed_at,
                   settlement_updated_at
            FROM reward_records
            WHERE reward_id=?
            """,
            (reward_id,),
        ).fetchone()

        conn.close()

    return {
        "ok": True,
        "mock": True,
        "reward": {
            "rewardId": updated[0],
            "playerId": updated[1],
            "source": updated[2],
            "zoneId": updated[3],
            "siteId": int(updated[4] or 0),
            "asset": updated[5],
            "amount": float(updated[6] or 0),
            "xpAmount": float(updated[7] or 0),
            "status": updated[8],
            "settlementStatus": updated[9],
            "txHash": updated[10],
            "targetQcAddress": updated[11],
            "settlementNote": updated[12],
            "createdAt": int(updated[13] or 0),
            "confirmedAt": int(updated[14] or 0),
            "settlementUpdatedAt": int(updated[15] or 0),
        }
    }




# QC_SETTLEMENT_CONFIRMED_V1
@app.get("/api/settlement/confirmed")
def settlement_confirmed(limit: int = 20):
    limit = max(1, min(int(limit), 100))

    with _lock:
        conn = _db()
        _ensure_reward_settlement_columns(conn)
        _ensure_reward_chain_columns(conn)

        rows = conn.execute(
            """
            SELECT reward_id, player_id, source, zone_id, site_id, asset,
                   amount, xp_amount, status, settlement_status, tx_hash,
                   target_qc_address, settlement_note, created_at, confirmed_at,
                   settlement_updated_at
            FROM reward_records
            WHERE settlement_status = 'CONFIRMED_CHAIN'
              AND COALESCE(tx_hash, '') != ''
              AND COALESCE(target_qc_address, '') != ''
            ORDER BY settlement_updated_at DESC, created_at DESC
            LIMIT ?
            """,
            (limit,),
        ).fetchall()

        conn.commit()
        conn.close()

    return {
        "ok": True,
        "status": "CONFIRMED_CHAIN",
        "count": len(rows),
        "rows": [
            {
                "rewardId": r[0],
                "playerId": r[1],
                "source": r[2],
                "zoneId": r[3],
                "siteId": int(r[4] or 0),
                "asset": r[5],
                "amount": float(r[6] or 0),
                "xpAmount": float(r[7] or 0),
                "status": r[8],
                "settlementStatus": r[9],
                "txHash": r[10],
                "targetQcAddress": r[11],
                "settlementNote": r[12],
                "createdAt": int(r[13] or 0),
                "confirmedAt": int(r[14] or 0),
                "settlementUpdatedAt": int(r[15] or 0),
                "chainHeight": int(r[16] or 0),
                "blockReward": float(r[17] or 0),
                "miningClientType": r[18] or "",
                "networkRewardMode": r[19] or "",
            }
            for r in rows
        ],
    }






# QC_WITHDRAW_REAL_PREP_V1
# QC_WITHDRAW_REQUEST_V1
class QCWithdrawRequestIn(BaseModel):
    playerId: str


@app.post("/api/withdraw/request")
def qc_withdraw_request(inp: QCWithdrawRequestIn):
    player_id = (inp.playerId or "").strip()

    if not player_id:
        return {"ok": False, "error": "empty playerId"}

    with _lock:
        conn = _db()
        _touch_player(conn, player_id, None)
        _ensure_qc_address_column(conn)
        _ensure_reward_settlement_columns(conn)
        _ensure_reward_chain_columns(conn)
        _ensure_withdraw_batch_tables(conn)
        _ensure_withdraw_payout_meta_columns(conn)

        target_qc_address = _get_player_qc_address(conn, player_id)

        if not target_qc_address:
            conn.close()
            return {
                "ok": False,
                "error": "qc_address_not_linked",
                "message": "Link or create a QC wallet address before requesting withdraw.",
            }

        rows = conn.execute(
            """
            SELECT reward_id, amount, chain_height
            FROM reward_records
            WHERE player_id=?
              AND settlement_status='READY_FOR_CHAIN'
              AND tx_hash IS NULL
              AND COALESCE(target_qc_address, '') != ''
              AND COALESCE(withdraw_batch_id, '') = ''
            ORDER BY created_at ASC
            """,
            (player_id,),
        ).fetchall()

        if not rows:
            conn.close()
            return {
                "ok": False,
                "error": "no_withdrawable_rewards",
                "playerId": player_id,
                "minWithdrawAmount": QC_MIN_WITHDRAW_AMOUNT,
            }

        reward_ids = [r[0] for r in rows]
        amount = float(sum(float(r[1] or 0) for r in rows))
        reward_count = len(rows)

        chain_heights = [int(r[2] or 0) for r in rows if int(r[2] or 0) > 0]
        min_chain_height = min(chain_heights) if chain_heights else 0
        max_chain_height = max(chain_heights) if chain_heights else 0

        if amount < QC_MIN_WITHDRAW_AMOUNT:
            conn.close()
            return {
                "ok": False,
                "error": "below_min_withdraw_amount",
                "playerId": player_id,
                "withdrawableAmount": amount,
                "minWithdrawAmount": QC_MIN_WITHDRAW_AMOUNT,
                "rewardCount": reward_count,
            }

        batch_id = "wdb_" + secrets.token_hex(16)
        ts = _now()

        conn.execute(
            """
            INSERT INTO withdraw_batches(
                batch_id, player_id, target_qc_address, amount, reward_count,
                status, tx_hash, reward_ids_json, note,
                min_chain_height, max_chain_height, created_at, updated_at,
                payout_mode, payout_type, tx_check_note
            )
            VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
            """,
            (
                batch_id,
                player_id,
                target_qc_address,
                amount,
                reward_count,
                "REQUESTED",
                None,
                json.dumps(reward_ids, ensure_ascii=False),
                "User requested QC withdraw. Ready for admin/future payout processing.",
                min_chain_height,
                max_chain_height,
                ts,
                ts,
                "REAL_PAYOUT_PREP",
                "ADMIN_REVIEW",
                "QC_WITHDRAW_REAL_PREP_V1: Prepared for real payout precheck only. No private key accepted and no QC transfer sent.",
            ),
        )

        placeholders = ",".join(["?"] * len(reward_ids))
        conn.execute(
            f"""
            UPDATE reward_records
            SET
                withdraw_batch_id=?,
                settlement_note='Withdraw batch requested. Waiting for payout processing.',
                settlement_updated_at=?
            WHERE reward_id IN ({placeholders})
            """,
            [batch_id, ts] + reward_ids,
        )

        conn.commit()
        conn.close()

    return {
        "ok": True,
        "batch": {
            "batchId": batch_id,
            "playerId": player_id,
            "targetQcAddress": target_qc_address,
            "amount": amount,
            "rewardCount": reward_count,
            "status": "REQUESTED",
            "rewardIds": reward_ids,
            "minChainHeight": min_chain_height,
            "maxChainHeight": max_chain_height,
            "minWithdrawAmount": QC_MIN_WITHDRAW_AMOUNT,
        },
    }


# QC_WITHDRAW_STATUS_PAYOUT_META_V1
@app.get("/api/withdraw/status")
def qc_withdraw_status(playerId: str):
    player_id = (playerId or "").strip()

    if not player_id:
        return {"ok": False, "error": "empty playerId"}

    with _lock:
        conn = _db()
        _touch_player(conn, player_id, None)
        _ensure_withdraw_batch_tables(conn)
        _ensure_withdraw_payout_meta_columns(conn)

        rows = conn.execute(
            """
            SELECT batch_id, player_id, target_qc_address, amount, reward_count,
                   status, tx_hash, reward_ids_json, note,
                   min_chain_height, max_chain_height, created_at, updated_at,
                   payout_mode, payout_type, tx_confirmed_height, tx_checked_at, tx_check_note
            FROM withdraw_batches
            WHERE player_id=?
            ORDER BY created_at DESC
            LIMIT 20
            """,
            (player_id,),
        ).fetchall()

        conn.commit()
        conn.close()

    out = []
    for r in rows:
        try:
            reward_ids = json.loads(r[7] or "[]")
        except Exception:
            reward_ids = []

        out.append({
            "batchId": r[0],
            "playerId": r[1],
            "targetQcAddress": r[2],
            "amount": float(r[3] or 0),
            "rewardCount": int(r[4] or 0),
            "status": r[5],
            "txHash": r[6],
            "rewardIds": reward_ids,
            "note": r[8],
            "minChainHeight": int(r[9] or 0),
            "maxChainHeight": int(r[10] or 0),
            "createdAt": int(r[11] or 0),
            "updatedAt": int(r[12] or 0),
            "payoutMode": r[13] or "",
            "payoutType": r[14] or "",
            "txConfirmedHeight": int(r[15] or 0),
            "txCheckedAt": int(r[16] or 0),
            "txCheckNote": r[17] or "",
        })

    return {
        "ok": True,
        "playerId": player_id,
        "count": len(out),
        "rows": out,
    }




# QC_WITHDRAW_LIFECYCLE_V1
class QCWithdrawBatchIn(BaseModel):
    batchId: str
    txHash: Optional[str] = None


@app.post("/api/withdraw/mark-pending")
def qc_withdraw_mark_pending(inp: QCWithdrawBatchIn):
    batch_id = (inp.batchId or "").strip()

    if not batch_id:
        return {"ok": False, "error": "empty batchId"}

    if not _settlement_dev_actions_enabled():
        return _settlement_dev_actions_error()

    with _lock:
        conn = _db()
        _ensure_withdraw_batch_tables(conn)
        _ensure_withdraw_payout_meta_columns(conn)
        _ensure_reward_settlement_columns(conn)

        row = conn.execute(
            """
            SELECT batch_id, player_id, amount, reward_count, status, tx_hash,
                   target_qc_address, min_chain_height, max_chain_height
            FROM withdraw_batches
            WHERE batch_id=?
            """,
            (batch_id,),
        ).fetchone()

        if not row:
            conn.close()
            return {"ok": False, "error": "batch_not_found", "batchId": batch_id}

        status = row[4] or ""
        if status != "REQUESTED":
            conn.close()
            return {
                "ok": False,
                "error": "invalid_batch_status",
                "batchId": batch_id,
                "status": status,
                "expected": "REQUESTED",
            }

        ts = _now()

        conn.execute(
            """
            UPDATE withdraw_batches
            SET
                status='PENDING_CHAIN',
                note='Marked as PENDING_CHAIN. Waiting for QC node payout transaction.',
                updated_at=?
            WHERE batch_id=?
            """,
            (ts, batch_id),
        )

        conn.execute(
            """
            UPDATE reward_records
            SET
                settlement_status='PENDING_CHAIN',
                settlement_note='Withdraw batch marked as PENDING_CHAIN. Waiting for payout transaction.',
                settlement_updated_at=?
            WHERE withdraw_batch_id=?
              AND settlement_status='READY_FOR_CHAIN'
              AND tx_hash IS NULL
            """,
            (ts, batch_id),
        )

        updated = conn.execute(
            """
            SELECT batch_id, player_id, target_qc_address, amount, reward_count,
                   status, tx_hash, note, min_chain_height, max_chain_height,
                   created_at, updated_at, payout_mode, payout_type,
                   tx_confirmed_height, tx_checked_at, tx_check_note
            FROM withdraw_batches
            WHERE batch_id=?
            """,
            (batch_id,),
        ).fetchone()

        conn.commit()
        conn.close()

    return {
        "ok": True,
        "batch": {
            "batchId": updated[0],
            "playerId": updated[1],
            "targetQcAddress": updated[2],
            "amount": float(updated[3] or 0),
            "rewardCount": int(updated[4] or 0),
            "status": updated[5],
            "txHash": updated[6],
            "note": updated[7],
            "minChainHeight": int(updated[8] or 0),
            "maxChainHeight": int(updated[9] or 0),
            "createdAt": int(updated[10] or 0),
            "updatedAt": int(updated[11] or 0),
        },
    }


@app.post("/api/withdraw/mock-confirm")
def qc_withdraw_mock_confirm(inp: QCWithdrawBatchIn):
    batch_id = (inp.batchId or "").strip()
    provided_tx = (inp.txHash or "").strip()

    if not batch_id:
        return {"ok": False, "error": "empty batchId"}

    if not _settlement_dev_actions_enabled():
        return _settlement_dev_actions_error()

    with _lock:
        conn = _db()
        _ensure_withdraw_batch_tables(conn)
        _ensure_withdraw_payout_meta_columns(conn)
        _ensure_reward_settlement_columns(conn)

        row = conn.execute(
            """
            SELECT batch_id, player_id, amount, reward_count, status, tx_hash,
                   target_qc_address, min_chain_height, max_chain_height
            FROM withdraw_batches
            WHERE batch_id=?
            """,
            (batch_id,),
        ).fetchone()

        if not row:
            conn.close()
            return {"ok": False, "error": "batch_not_found", "batchId": batch_id}

        status = row[4] or ""
        if status != "PENDING_CHAIN":
            conn.close()
            return {
                "ok": False,
                "error": "invalid_batch_status",
                "batchId": batch_id,
                "status": status,
                "expected": "PENDING_CHAIN",
            }

        tx_hash = provided_tx or ("mock_wdb_tx_" + secrets.token_hex(16))
        ts = _now()

        conn.execute(
            """
            UPDATE withdraw_batches
            SET
                status='CONFIRMED_CHAIN',
                tx_hash=?,
                note='Mock confirmed withdraw batch. Replace with real QC payout tx confirmation later.',
                updated_at=?
            WHERE batch_id=?
            """,
            (tx_hash, ts, batch_id),
        )

        conn.execute(
            """
            UPDATE reward_records
            SET
                settlement_status='CONFIRMED_CHAIN',
                tx_hash=?,
                settlement_note='Withdraw batch mock confirmed. Replace with real QC payout tx confirmation later.',
                settlement_updated_at=?
            WHERE withdraw_batch_id=?
              AND settlement_status='PENDING_CHAIN'
            """,
            (tx_hash, ts, batch_id),
        )

        updated = conn.execute(
            """
            SELECT batch_id, player_id, target_qc_address, amount, reward_count,
                   status, tx_hash, note, min_chain_height, max_chain_height,
                   created_at, updated_at
            FROM withdraw_batches
            WHERE batch_id=?
            """,
            (batch_id,),
        ).fetchone()

        conn.commit()
        conn.close()

    return {
        "ok": True,
        "batch": {
            "batchId": updated[0],
            "playerId": updated[1],
            "targetQcAddress": updated[2],
            "amount": float(updated[3] or 0),
            "rewardCount": int(updated[4] or 0),
            "status": updated[5],
            "txHash": updated[6],
            "note": updated[7],
            "minChainHeight": int(updated[8] or 0),
            "maxChainHeight": int(updated[9] or 0),
            "createdAt": int(updated[10] or 0),
            "updatedAt": int(updated[11] or 0),
        },
    }






# QC_REAL_PAYOUT_TXSTATUS_READY_V1
async def qc_check_tx_status_endpoint_ready():
    base = (os.environ.get("QC_API_BASE") or "http://127.0.0.1:8082").rstrip("/")
    url = base + "/api/tx/status?id=__precheck__"

    try:
        import urllib.request
        import json as _json

        def _fetch():
            req = urllib.request.Request(
                url,
                headers={"Accept": "application/json"},
                method="GET",
            )
            with urllib.request.urlopen(req, timeout=5) as resp:
                raw = resp.read().decode("utf-8", errors="replace")
                return resp.status, raw

        status, raw = await asyncio.to_thread(_fetch)

        if status != 200:
            return {
                "ok": False,
                "ready": False,
                "status": status,
                "url": url,
                "error": "non_200_status",
            }

        data = _json.loads(raw)

        ready = (
            data.get("ok") is True
            and data.get("id") == "__precheck__"
            and "inBlock" in data
            and "inMempool" in data
            and "height" in data
        )

        return {
            "ok": True,
            "ready": bool(ready),
            "status": status,
            "url": url,
            "response": data,
        }

    except Exception as e:
        return {
            "ok": False,
            "ready": False,
            "url": url,
            "error": str(e),
        }


# QC_REAL_PAYOUT_PRECHECK_V1
class QCRealPayoutPrecheckIn(BaseModel):
    batchId: str


@app.post("/api/withdraw/real-payout-precheck")
async def qc_withdraw_real_payout_precheck(inp: QCRealPayoutPrecheckIn):
    batch_id = (inp.batchId or "").strip()

    if not batch_id:
        return {"ok": False, "ready": False, "error": "empty batchId"}

    treasury_address = (os.environ.get("QC_TREASURY_ADDRESS") or "").strip()

    checks = {
        "batchExists": False,
        "batchStatusAllowed": False,
        "targetAddressPresent": False,
        "treasuryAddressConfigured": bool(treasury_address),
        "qcNodeReachable": False,
        "treasuryBalanceReadable": False,
        "treasurySpendableEnough": False,
        "realSendEndpointKnown": True,
        "txStatusEndpointReady": False,
        "privateKeyProvided": False,
    }

    with _lock:
        conn = _db()
        _ensure_withdraw_batch_tables(conn)
        _ensure_withdraw_payout_meta_columns(conn)

        row = conn.execute(
            """
            SELECT batch_id, player_id, target_qc_address, amount, reward_count,
                   status, tx_hash, note, min_chain_height, max_chain_height,
                   created_at, updated_at, payout_mode, payout_type,
                   tx_confirmed_height, tx_checked_at, tx_check_note
            FROM withdraw_batches
            WHERE batch_id=?
            """,
            (batch_id,),
        ).fetchone()

        conn.commit()
        conn.close()

    if not row:
        return {
            "ok": False,
            "ready": False,
            "error": "batch_not_found",
            "batchId": batch_id,
            "checks": checks,
        }

    checks["batchExists"] = True

    batch = {
        "batchId": row[0],
        "playerId": row[1],
        "targetQcAddress": row[2],
        "amount": float(row[3] or 0),
        "rewardCount": int(row[4] or 0),
        "status": row[5],
        "txHash": row[6],
        "note": row[7],
        "minChainHeight": int(row[8] or 0),
        "maxChainHeight": int(row[9] or 0),
        "createdAt": int(row[10] or 0),
        "updatedAt": int(row[11] or 0),
        "payoutMode": row[12] or "",
        "payoutType": row[13] or "",
        "txConfirmedHeight": int(row[14] or 0),
        "txCheckedAt": int(row[15] or 0),
        "txCheckNote": row[16] or "",
    }

    # For real payout, a batch should normally be REQUESTED or PENDING_CHAIN.
    # CONFIRMED_CHAIN is already completed and should not be sent again.
    allowed_statuses = {"REQUESTED", "PENDING_CHAIN"}
    checks["batchStatusAllowed"] = batch["status"] in allowed_statuses
    checks["targetAddressPresent"] = bool(batch["targetQcAddress"])

    qc_health = await qc_fetch_health()
    checks["qcNodeReachable"] = bool(qc_health.get("ok"))

    treasury_balance = None
    if treasury_address:
        treasury_balance = await qc_fetch_address_balance(treasury_address)
        checks["treasuryBalanceReadable"] = bool(treasury_balance.get("ok"))
        spendable = float(treasury_balance.get("spendable") or 0)
        checks["treasurySpendableEnough"] = spendable >= float(batch["amount"] or 0)

    # Important: We intentionally do not accept or read private keys here.
    checks["privateKeyProvided"] = False

    tx_status_check = await qc_check_tx_status_endpoint_ready()
    checks["txStatusEndpointReady"] = bool(tx_status_check.get("ready"))

    ready = all([
        checks["batchExists"],
        checks["batchStatusAllowed"],
        checks["targetAddressPresent"],
        checks["treasuryAddressConfigured"],
        checks["qcNodeReachable"],
        checks["treasuryBalanceReadable"],
        checks["treasurySpendableEnough"],
        checks["realSendEndpointKnown"],
    ])

    blockers = [k for k, v in checks.items() if not v and k not in ("privateKeyProvided", "txStatusEndpointReady")]

    return {
        "ok": True,
        "ready": bool(ready),
        "batch": batch,
        "treasury": {
            "address": treasury_address,
            "balance": treasury_balance.get("balance") if treasury_balance else None,
            "spendable": treasury_balance.get("spendable") if treasury_balance else None,
            "height": treasury_balance.get("height") if treasury_balance else None,
            "ok": bool(treasury_balance.get("ok")) if treasury_balance else False,
        },
        "qcHealth": qc_health,
        "txStatusCheck": tx_status_check,
        "checks": checks,
        "blockers": blockers,
        "note": "Precheck only. No QC transfer is sent. Private key is not accepted here.",
    }


# QC_SETTLEMENT_SUMMARY_V1
@app.get("/api/settlement/summary")
def settlement_summary():
    with _lock:
        conn = _db()
        _ensure_reward_settlement_columns(conn)
        _ensure_reward_chain_columns(conn)

        rows = conn.execute(
            """
            SELECT
                settlement_status,
                COUNT(*) AS cnt,
                COALESCE(SUM(amount), 0) AS total_amount
            FROM reward_records
            GROUP BY settlement_status
            """
        ).fetchall()

        total_rows = conn.execute(
            """
            SELECT
                COUNT(*) AS cnt,
                COALESCE(SUM(amount), 0) AS total_amount
            FROM reward_records
            """
        ).fetchone()

        chain_ready_rows = conn.execute(
            """
            SELECT
                COUNT(*) AS cnt,
                COALESCE(SUM(amount), 0) AS total_amount
            FROM reward_records
            WHERE settlement_status = 'READY_FOR_CHAIN'
              AND tx_hash IS NULL
              AND COALESCE(target_qc_address, '') != ''
            """
        ).fetchone()

        pending_rows = conn.execute(
            """
            SELECT
                COUNT(*) AS cnt,
                COALESCE(SUM(amount), 0) AS total_amount
            FROM reward_records
            WHERE settlement_status = 'PENDING_CHAIN'
              AND tx_hash IS NULL
              AND COALESCE(target_qc_address, '') != ''
            """
        ).fetchone()

        confirmed_rows = conn.execute(
            """
            SELECT
                COUNT(*) AS cnt,
                COALESCE(SUM(amount), 0) AS total_amount
            FROM reward_records
            WHERE settlement_status = 'CONFIRMED_CHAIN'
              AND COALESCE(tx_hash, '') != ''
              AND COALESCE(target_qc_address, '') != ''
            """
        ).fetchone()

        failed_rows = conn.execute(
            """
            SELECT
                COUNT(*) AS cnt,
                COALESCE(SUM(amount), 0) AS total_amount
            FROM reward_records
            WHERE settlement_status = 'FAILED_CHAIN'
            """
        ).fetchone()

        conn.commit()
        conn.close()

    by_status = {}
    for status, cnt, amount in rows:
        key = status or "UNKNOWN"
        by_status[key] = {
            "count": int(cnt or 0),
            "amount": float(amount or 0),
        }

    return {
        "ok": True,
        "summary": {
            "totalCount": int(total_rows[0] or 0),
            "totalAmount": float(total_rows[1] or 0),

            "readyCount": int(chain_ready_rows[0] or 0),
            "readyAmount": float(chain_ready_rows[1] or 0),

            "pendingCount": int(pending_rows[0] or 0),
            "pendingAmount": float(pending_rows[1] or 0),

            "confirmedCount": int(confirmed_rows[0] or 0),
            "confirmedAmount": float(confirmed_rows[1] or 0),

            "failedCount": int(failed_rows[0] or 0),
            "failedAmount": float(failed_rows[1] or 0),

            "byStatus": by_status,
        },
    }






# QC_ADMIN_SETTLEMENT_PANEL_V1
@app.get("/admin/settlement")
def admin_settlement_panel():
    # QC_ADMIN_SETTLEMENT_BUTTONS_V1
    # QC_ADMIN_CHAIN_FIELDS_V1
    # QC_ADMIN_WITHDRAW_REQUESTS_V1
    # QC_ADMIN_WITHDRAW_PAYOUT_META_V1
    dev_actions_enabled = _settlement_dev_actions_enabled()

    with _lock:
        conn = _db()
        _ensure_reward_settlement_columns(conn)

        summary_rows = conn.execute(
            """
            SELECT settlement_status, COUNT(*), COALESCE(SUM(amount), 0)
            FROM reward_records
            GROUP BY settlement_status
            ORDER BY settlement_status
            """
        ).fetchall()

        def fetch_rows(status, limit=20):
            return conn.execute(
                """
                SELECT reward_id, player_id, source, zone_id, site_id, asset,
                       amount, xp_amount, status, settlement_status, tx_hash,
                       target_qc_address, settlement_note, created_at, confirmed_at,
                       settlement_updated_at, chain_height, block_reward,
                       mining_client_type, network_reward_mode, meta_json
                FROM reward_records
                WHERE settlement_status=?
                ORDER BY settlement_updated_at DESC, created_at DESC
                LIMIT ?
                """,
                (status, limit),
            ).fetchall()

        ready_rows = fetch_rows("READY_FOR_CHAIN", 20)
        pending_rows = fetch_rows("PENDING_CHAIN", 20)
        confirmed_rows = fetch_rows("CONFIRMED_CHAIN", 20)
        failed_rows = fetch_rows("FAILED_CHAIN", 20)

        _ensure_withdraw_batch_tables(conn)
        _ensure_withdraw_payout_meta_columns(conn)

        withdraw_rows = conn.execute(
            """
            SELECT batch_id, player_id, target_qc_address, amount, reward_count,
                   status, tx_hash, note, min_chain_height, max_chain_height,
                   created_at, updated_at, payout_mode, payout_type,
                   tx_confirmed_height, tx_checked_at, tx_check_note
            FROM withdraw_batches
            ORDER BY created_at DESC
            LIMIT 30
            """
        ).fetchall()

        conn.commit()
        conn.close()

    def esc(v):
        import html
        return html.escape("" if v is None else str(v), quote=True)

    def amount(v):
        try:
            return f"{float(v):,.2f}"
        except Exception:
            return "0.00"

    def render_rows(rows, action="none"):
        if not rows:
            return '<tr><td colspan="13" class="empty">No records</td></tr>'

        out = []
        for r in rows:
            reward_id = r[0]
            player_id = r[1]
            zone_id = r[3]
            asset = r[5]
            amt = r[6]
            st = r[9]
            tx_hash = r[10] or ""
            target = r[11] or ""
            note = r[12] or ""
            chain_height = int(r[16] or 0)
            block_reward = float(r[17] or 0)
            mining_client_type = r[18] or ""
            network_reward_mode = r[19] or ""

            meta_json = r[20] if len(r) > 20 else ""
            block_hash = ""
            try:
                meta = json.loads(meta_json or "{}")
                qmr = meta.get("qcMineResult") or {}
                block_hash = str(meta.get("blockHash") or qmr.get("block_hash") or "")
            except Exception:
                block_hash = ""

            if len(block_hash) > 32:
                short_block_hash = block_hash[:16] + "..." + block_hash[-8:]
            else:
                short_block_hash = block_hash

            action_html = "-"
            if dev_actions_enabled and action == "mark_pending":
                action_html = (
                    '<button class="action-btn pending" data-action="mark-pending" '
                    f'data-reward="{esc(reward_id)}">Mark Pending</button>'
                )
            elif dev_actions_enabled and action == "mock_confirm":
                action_html = (
                    '<button class="action-btn confirm" data-action="mock-confirm" '
                    f'data-reward="{esc(reward_id)}">Mock Confirm</button>'
                )
            elif not dev_actions_enabled and action != "none":
                action_html = '<span class="disabled-action">Dev actions off</span>'

            # QC_ADMIN_CLASSIFICATION_BADGES_V1
            try:
                amt_float = float(amt or 0)
            except Exception:
                amt_float = 0.0

            record_class = "NEEDS_REVIEW"
            record_style = "background:#334155;color:#e5e7eb;border:1px solid #64748b;"

            if (
                mining_client_type == "GAME_MINER"
                and network_reward_mode == "NETWORK_BLOCK_REWARD"
                and block_hash
            ):
                record_class = "REAL_GAME_MINING"
                record_style = "background:#064e3b;color:#d1fae5;border:1px solid #10b981;"
            elif "LEGACY" in network_reward_mode or amt_float < 50:
                record_class = "LEGACY_DEMO"
                record_style = "background:#78350f;color:#ffedd5;border:1px solid #f59e0b;"
            elif (
                str(tx_hash or "").startswith("mock_")
                or "mock" in str(note or "").lower()
                or "DEV" in network_reward_mode
                or "MOCK" in network_reward_mode
            ):
                record_class = "MOCK_OR_DEV"
                record_style = "background:#581c87;color:#f3e8ff;border:1px solid #a855f7;"

            record_badge = (
                f'<span style="display:inline-block;padding:3px 7px;border-radius:999px;'
                f'font-size:11px;font-weight:800;letter-spacing:.03em;{record_style}">'
                f'{esc(record_class)}</span>'
            )

            out.append(f"""
            <tr>
              <td><code>{esc(reward_id)}</code></td>
              <td>{esc(player_id)}</td>
              <td>{esc(zone_id)}</td>
              <td>{esc(asset)}</td>
              <td class="num">{amount(amt)}</td>
              <td><span class="pill {esc(st.lower())}">{esc(st)}</span></td>
              <td><code>{esc(target)}</code></td>
              <td><code>{esc(tx_hash) if tx_hash else '-'}</code></td>
              <td><code title="{esc(block_hash)}">{esc(short_block_hash) if short_block_hash else '-'}</code></td>
              <td>{chain_height}</td>
              <td class="num">{amount(block_reward)}</td>
              <td><code>{esc(network_reward_mode)}</code></td>
              <td class="note">{record_badge}<br>{esc(note)}</td>
              <td>{action_html}</td>
            </tr>
            """)
        return "\n".join(out)

    total_count = 0
    total_amount = 0.0
    summary_cards = []

    for st, cnt, amt in summary_rows:
        total_count += int(cnt or 0)
        total_amount += float(amt or 0)
        summary_cards.append(f"""
        <div class="card">
          <div class="label">{esc(st)}</div>
          <div class="value">{int(cnt or 0)}</div>
          <div class="sub">{amount(amt)} QC</div>
        </div>
        """)

    def render_withdraw_rows(rows):
        if not rows:
            return '<tr><td colspan="16" class="empty">No withdraw requests</td></tr>'

        out = []
        for r in rows:
            batch_id = r[0]
            player_id = r[1]
            target = r[2] or ""
            amt = r[3]
            reward_count = int(r[4] or 0)
            status = r[5] or ""
            tx_hash = r[6] or ""
            note = r[7] or ""
            min_chain = int(r[8] or 0)
            max_chain = int(r[9] or 0)
            created_at = int(r[10] or 0)
            # QC_ADMIN_WITHDRAW_PAYOUT_META_REPAIR_V1
            payout_mode = (r[12] if len(r) > 12 else "") or ""
            payout_type = (r[13] if len(r) > 13 else "") or ""
            tx_confirmed_height = int((r[14] if len(r) > 14 else 0) or 0)
            tx_checked_at = int((r[15] if len(r) > 15 else 0) or 0)
            tx_check_note = (r[16] if len(r) > 16 else "") or ""

            # QC_ADMIN_WITHDRAW_CLASSIFICATION_BADGES_V1
            payout_mode_u = str(payout_mode or "").upper()
            payout_type_u = str(payout_type or "").upper()
            tx_hash_u = str(tx_hash or "")
            note_l = str(note or "").lower()

            withdraw_class = "WITHDRAW_REVIEW"
            withdraw_style = "background:#334155;color:#e5e7eb;border:1px solid #64748b;"

            if "MOCK" in payout_mode_u or "DEV" in payout_type_u or tx_hash_u.startswith("mock_") or "mock" in note_l:
                withdraw_class = "MOCK_PAYOUT"
                withdraw_style = "background:#581c87;color:#f3e8ff;border:1px solid #a855f7;"
            elif status == "REQUESTED":
                withdraw_class = "REQUESTED_REAL_PREP"
                withdraw_style = "background:#1e3a8a;color:#dbeafe;border:1px solid #60a5fa;"
            elif status == "PENDING_CHAIN":
                withdraw_class = "PENDING_CHAIN"
                withdraw_style = "background:#78350f;color:#ffedd5;border:1px solid #f59e0b;"
            elif status == "CONFIRMED_CHAIN":
                withdraw_class = "CONFIRMED_CHAIN"
                withdraw_style = "background:#064e3b;color:#d1fae5;border:1px solid #10b981;"

            withdraw_badge = (
                f'<span style="display:inline-block;padding:3px 7px;border-radius:999px;'
                f'font-size:11px;font-weight:800;letter-spacing:.03em;{withdraw_style}">'
                f'{esc(withdraw_class)}</span>'
            )

            out.append(f"""
            <tr>
              <td><code>{esc(batch_id)}</code></td>
              <td>{esc(player_id)}</td>
              <td class="num">{amount(amt)}</td>
              <td class="num">{reward_count}</td>
              <td><span class="pill withdraw_{esc(status.lower())}">{esc(status)}</span></td>
              <td><code>{esc(target)}</code></td>
              <td><code>{esc(tx_hash) if tx_hash else '-'}</code></td>
              <td>{min_chain}</td>
              <td>{max_chain}</td>
              <td>{created_at}</td>
              <td><code>{esc(payout_mode)}</code></td>
              <td><code>{esc(payout_type)}</code></td>
              <td>{tx_confirmed_height}</td>
              <td>{tx_checked_at}</td>
              <td class="note">{esc(tx_check_note)}</td>
              <td class="note">{withdraw_badge}<br>{esc(note)}</td>
            </tr>
            """)

        return "\n".join(out)


    def withdraw_table():
        return f"""
        <div class="section">
          <h2>WITHDRAW REQUESTS</h2>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Batch ID</th><th>Player</th><th>Amount</th><th>Rewards</th><th>Status</th>
                  <th>Target QC Address</th><th>TxHash</th><th>Min Chain</th><th>Max Chain</th><th>Created</th>
                  <th>Payout Mode</th><th>Payout Type</th><th>Tx Height</th><th>Tx Checked</th><th>Tx Check Note</th><th>Note</th>
                </tr>
              </thead>
              <tbody>{render_withdraw_rows(withdraw_rows)}</tbody>
            </table>
          </div>
        </div>
        """


    def table(title, rows, action):
        return f"""
        <div class="section">
          <h2>{esc(title)}</h2>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Reward ID</th><th>Player</th><th>Zone</th><th>Asset</th><th>Amount</th>
                  <th>Status</th><th>Target QC Address</th><th>TxHash</th><th>Block Hash</th><th>Chain</th><th>Block Reward</th><th>Mode</th><th>Note</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>{render_rows(rows, action)}</tbody>
            </table>
          </div>
        </div>
        """

    dev_label = "ENABLED" if dev_actions_enabled else "DISABLED"

    html_doc = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>QuantumCoin Settlement Admin</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #06111f;
      --panel: rgba(12, 25, 45, .92);
      --panel2: rgba(10, 42, 62, .78);
      --line: rgba(124, 231, 255, .22);
      --text: #eafcff;
      --muted: rgba(234,252,255,.66);
      --cyan: #7eeaff;
      --green: #77ffb0;
      --yellow: #ffd166;
      --red: #ff6b6b;
      --blue: #8cb7ff;
    }
    body {
      margin: 0;
      font-family: system-ui, -apple-system, Segoe UI, Arial, sans-serif;
      background:
        radial-gradient(circle at 20% 0%, rgba(0, 208, 255, .16), transparent 30%),
        radial-gradient(circle at 80% 10%, rgba(84, 64, 255, .14), transparent 32%),
        var(--bg);
      color: var(--text);
    }
    .wrap { width: min(1500px, calc(100% - 32px)); margin: 24px auto 60px; }
    .top { display: flex; justify-content: space-between; gap: 16px; align-items: center; margin-bottom: 18px; }
    h1 { margin: 0; font-size: 28px; letter-spacing: .02em; }
    .small { color: var(--muted); font-size: 13px; margin-top: 6px; }
    .refresh {
      border: 1px solid var(--line);
      background: rgba(126,234,255,.08);
      color: var(--text);
      border-radius: 14px;
      padding: 10px 14px;
      cursor: pointer;
      font-weight: 700;
    }
    .dev-banner {
      border: 1px solid var(--line);
      background: rgba(255,209,102,.08);
      color: var(--yellow);
      border-radius: 14px;
      padding: 10px 12px;
      margin: 0 0 16px;
      font-size: 13px;
      font-weight: 800;
    }
    .grid { display: grid; grid-template-columns: repeat(4, minmax(160px, 1fr)); gap: 12px; margin-bottom: 18px; }
    .card {
      border: 1px solid var(--line);
      background: linear-gradient(135deg, var(--panel), var(--panel2));
      border-radius: 18px;
      padding: 16px;
      box-shadow: 0 0 22px rgba(0, 210, 255, .08);
    }
    .label { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .08em; }
    .value { font-size: 28px; font-weight: 900; margin-top: 6px; }
    .sub { color: var(--cyan); font-size: 13px; margin-top: 2px; }
    .section {
      border: 1px solid var(--line);
      background: rgba(3, 13, 26, .72);
      border-radius: 20px;
      margin: 18px 0;
      overflow: hidden;
    }
    .section h2 {
      margin: 0;
      padding: 16px;
      font-size: 17px;
      border-bottom: 1px solid var(--line);
      background: rgba(126,234,255,.05);
    }
    .table-wrap { overflow-x: auto; }
    table { width: 100%; border-collapse: collapse; min-width: 1780px; }
    th, td { padding: 11px 12px; border-bottom: 1px solid rgba(255,255,255,.07); vertical-align: top; font-size: 13px; }
    th {
      text-align: left;
      color: var(--muted);
      font-size: 11px;
      text-transform: uppercase;
      letter-spacing: .06em;
      background: rgba(255,255,255,.03);
    }
    code { color: #b8f6ff; word-break: break-all; }
    .num { text-align: right; font-weight: 800; }
    .note { color: var(--muted); max-width: 320px; }
    .pill {
      display: inline-flex;
      padding: 5px 8px;
      border-radius: 999px;
      font-size: 11px;
      font-weight: 800;
      border: 1px solid rgba(255,255,255,.18);
    }
    .ready_for_chain { color: var(--yellow); background: rgba(255,209,102,.09); }
    .pending_chain { color: var(--blue); background: rgba(140,183,255,.10); }
    .confirmed_chain { color: var(--green); background: rgba(119,255,176,.10); }
    .failed_chain { color: var(--red); background: rgba(255,107,107,.10); }
    .withdraw_requested { color: var(--yellow); background: rgba(255,209,102,.09); }
    .withdraw_pending_chain { color: var(--blue); background: rgba(140,183,255,.10); }
    .withdraw_confirmed_chain { color: var(--green); background: rgba(119,255,176,.10); }
    .withdraw_failed_chain { color: var(--red); background: rgba(255,107,107,.10); }
    .empty { color: var(--muted); text-align: center; padding: 22px; }
    .action-btn {
      border: 1px solid rgba(255,255,255,.18);
      border-radius: 12px;
      padding: 8px 10px;
      color: var(--text);
      cursor: pointer;
      font-weight: 800;
      white-space: nowrap;
    }
    .action-btn.pending { background: rgba(140,183,255,.16); color: var(--blue); }
    .action-btn.confirm { background: rgba(119,255,176,.14); color: var(--green); }
    .disabled-action { color: var(--muted); font-size: 12px; white-space: nowrap; }
  </style>
</head>
<body>
  <div class="wrap">
"""

    html_doc += f"""
    <div class="top">
      <div>
        <h1>QuantumCoin Settlement Admin</h1>
        <div class="small">Read-only/admin-dev view · Game rewards → chain settlement tracking</div>
      </div>
      <button class="refresh" onclick="location.reload()">Refresh</button>
    </div>

    <div class="dev-banner">
      Dev actions: {esc(dev_label)} · State-changing buttons are for local/dev testing only.
    </div>

    <div class="grid">
      <div class="card">
        <div class="label">Total Records</div>
        <div class="value">{total_count}</div>
        <div class="sub">{amount(total_amount)} QC total</div>
      </div>
      {''.join(summary_cards)}
    </div>

    {withdraw_table()}

    {table("READY_FOR_CHAIN", ready_rows, "mark_pending")}
    {table("PENDING_CHAIN", pending_rows, "mock_confirm")}
    {table("CONFIRMED_CHAIN", confirmed_rows, "none")}
    {table("FAILED_CHAIN", failed_rows, "none")}
  </div>

  <script>
    async function postJson(url, payload) {{
      const res = await fetch(url, {{
        method: "POST",
        headers: {{"Content-Type": "application/json", "Accept": "application/json"}},
        body: JSON.stringify(payload)
      }});

      const data = await res.json();

      if (!data.ok) {{
        alert("Action failed: " + (data.error || "unknown_error"));
        console.warn(data);
        return;
      }}

      location.reload();
    }}

    document.addEventListener("click", function(ev) {{
      const btn = ev.target.closest("button[data-action]");
      if (!btn) return;

      const rewardId = btn.dataset.reward || "";
      const action = btn.dataset.action || "";

      if (action === "mark-pending") {{
        if (!confirm("Mark this reward as PENDING_CHAIN?\\n" + rewardId)) return;
        postJson("/api/settlement/mark-pending", {{rewardId}});
      }}

      if (action === "mock-confirm") {{
        if (!confirm("Mock confirm this pending reward?\\nNo real QC transfer will be sent.\\n" + rewardId)) return;
        postJson("/api/settlement/mock-confirm", {{rewardId}});
      }}
    }});
  </script>
</body>
</html>
"""

    return HTMLResponse(content=html_doc)


@app.post("/api/dev/reset")
def dev_reset():
    with _lock:
        conn = _db()
        conn.execute("DELETE FROM players")
        conn.execute("DELETE FROM settings")
        conn.execute("DELETE FROM mining_sessions")
        conn.execute("DELETE FROM tgwt_state")
        conn.execute("DELETE FROM tgwt_nonces")
        conn.execute("DELETE FROM withdraw_requests")
        conn.execute("DELETE FROM tgwt_events")
        conn.execute("UPDATE tgwt_pool SET distributed_u=0, updated_at=? WHERE id=1", (_now(),))
        conn.commit()
        conn.close()
    return {"ok": True}



# WATCH_REWARDED_AD_POLICY_V1
# WATCH_CLAIM_TGWT_V1
# DEV TEST ONLY: final Watch & Earn must use verified Google Rewarded Ads completion before TGWT reward.
class WatchClaimIn(BaseModel):
    playerId: str
    videoId: str = "watch_demo"
    nonce: str


@app.post("/api/watch/claim")
def watch_claim(inp: WatchClaimIn):
    user_id = (inp.playerId or "").strip()
    video_id = (inp.videoId or "watch_demo").strip()
    nonce = (inp.nonce or "").strip()

    if not user_id:
        return {"ok": False, "error": "empty_playerId"}
    if not nonce:
        return {"ok": False, "error": "empty_nonce"}

    if not _rl_hit(user_id, "watch_claim", limit=10, window_sec=60):
        return {"ok": False, "error": "rate_limited"}

    with _lock:
        conn = _db()
        _tgwt_touch(conn, user_id)

        if not _nonce_once(conn, user_id, nonce):
            conn.close()
            return {"ok": False, "error": "dup_nonce"}

        ok = _tgwt_award_pending_from_pool(
            conn,
            user_id,
            1 * UNIT,
            "WATCH_DEV_TEST_REWARD",
            {"videoId": video_id, "source": "watch_claim_v1"}
        )

        if not ok:
            conn.close()
            return {"ok": False, "error": "pool_exhausted"}

        st = _tgwt_get(conn, user_id)
        conn.commit()
        conn.close()

    return {
        "ok": True,
        "userId": user_id,
        "videoId": video_id,
        "rewardTGWT": 1,
        "status": "DEV_TEST_PENDING_VERIFICATION",
        "mode": "DEV_TEST_ONLY",
        "finalMode": "GOOGLE_REWARDED_ADS_VERIFIED_COMPLETION",
        "rev": st["rev"],
        "balances": st,
        "message": "DEV TEST ONLY: +1 TGWT added as pending. Final Watch & Earn will require verified Google Rewarded Ads completion before TGWT reward."
    }

