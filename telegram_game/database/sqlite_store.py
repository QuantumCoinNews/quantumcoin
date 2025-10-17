# database/sqlite_store.py  (ASCII-only, fixed)
import sqlite3
import os
from datetime import datetime
from typing import Optional

# Stable Windows path: %LOCALAPPDATA%\QuantumCoin\game.db
_BASE = os.environ.get("LOCALAPPDATA") or os.getcwd()
_DATA_DIR = os.path.join(_BASE, "QuantumCoin")
os.makedirs(_DATA_DIR, exist_ok=True)
_DB_PATH = os.path.join(_DATA_DIR, "game.db")


def get_conn():
    conn = sqlite3.connect(_DB_PATH, check_same_thread=False)
    conn.row_factory = sqlite3.Row
    return conn


def _now() -> str:
    return datetime.utcnow().isoformat()


def init_db() -> None:
    with get_conn() as c:
        c.executescript("""
        PRAGMA journal_mode=WAL;
        PRAGMA foreign_keys=ON;

        CREATE TABLE IF NOT EXISTS users(
          id INTEGER PRIMARY KEY,
          username TEXT,
          qc REAL NOT NULL DEFAULT 0,
          tgwt INTEGER NOT NULL DEFAULT 0,
          energy INTEGER NOT NULL DEFAULT 100,
          level INTEGER NOT NULL DEFAULT 1,
          ship TEXT NOT NULL DEFAULT 'Basic Starter Ship',
          ship_mul REAL NOT NULL DEFAULT 1.0,
          verified INTEGER NOT NULL DEFAULT 0,
          created_at TEXT NOT NULL,
          updated_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS sessions(
          id INTEGER PRIMARY KEY,
          user_id INTEGER NOT NULL,
          sector TEXT NOT NULL,
          start_ts TEXT NOT NULL,
          end_ts TEXT,
          reward_qc REAL,
          FOREIGN KEY(user_id) REFERENCES users(id)
        );

        CREATE TABLE IF NOT EXISTS tgwt_claims(
          id INTEGER PRIMARY KEY,
          user_id INTEGER NOT NULL,
          task TEXT NOT NULL,
          amount INTEGER NOT NULL,
          day_key TEXT NOT NULL,
          created_at TEXT NOT NULL,
          UNIQUE(user_id, task, day_key)
        );

        CREATE TABLE IF NOT EXISTS daily_pool(
          day_key TEXT PRIMARY KEY,
          remaining INTEGER NOT NULL
        );

        CREATE TABLE IF NOT EXISTS referrals(
          id INTEGER PRIMARY KEY,
          referrer_id INTEGER NOT NULL,
          invitee_id INTEGER NOT NULL,
          created_at TEXT NOT NULL,
          UNIQUE(referrer_id, invitee_id)
        );

        CREATE TABLE IF NOT EXISTS tx_log(
          id INTEGER PRIMARY KEY,
          user_id INTEGER,
          kind TEXT NOT NULL,
          amount REAL NOT NULL,
          meta TEXT,
          created_at TEXT NOT NULL
        );
        """)


# ----- Users -----

def get_or_create_user(user_id: int, username: Optional[str]):
    with get_conn() as c:
        r = c.execute("SELECT * FROM users WHERE id=?", (user_id,)).fetchone()
        if r:
            if username is not None and r["username"] != username:
                c.execute(
                    "UPDATE users SET username=?, updated_at=? WHERE id=?",
                    (username, _now(), user_id),
                )
            return dict(r)

        c.execute(
            """INSERT INTO users(id, username, qc, tgwt, energy, level, ship, ship_mul, verified, created_at, updated_at)
               VALUES(?,?,?,?,?,?,?,?,?,?,?)""",
            (user_id, username or "", 0.0, 0, 100, 1, "Basic Starter Ship", 1.0, 0, _now(), _now()),
        )
        row = c.execute("SELECT * FROM users WHERE id=?", (user_id,)).fetchone()
        return dict(row)


def update_user_balances(user_id: int, *, qc=None, tgwt=None, energy=None, level=None, verified=None, ship=None, ship_mul=None):
    fields, vals = [], []
    if qc       is not None: fields.append("qc=?");        vals.append(qc)
    if tgwt     is not None: fields.append("tgwt=?");      vals.append(tgwt)
    if energy   is not None: fields.append("energy=?");    vals.append(energy)
    if level    is not None: fields.append("level=?");     vals.append(level)
    if verified is not None: fields.append("verified=?");  vals.append(verified)
    if ship     is not None: fields.append("ship=?");      vals.append(ship)
    if ship_mul is not None: fields.append("ship_mul=?");  vals.append(ship_mul)
    if not fields:
        return
    fields.append("updated_at=?"); vals.append(_now()); vals.append(user_id)
    with get_conn() as c:
        c.execute(f"UPDATE users SET {', '.join(fields)} WHERE id=?", vals)


# ----- Sessions -----

def session_start(user_id: int, sector: str, start_ts: str):
    with get_conn() as c:
        c.execute(
            "INSERT INTO sessions(user_id, sector, start_ts) VALUES(?,?,?)",
            (user_id, sector, start_ts)
        )


def session_finish(user_id: int, sector: str, end_ts: str, reward_qc: float):
    with get_conn() as c:
        r = c.execute(
            """SELECT id FROM sessions
               WHERE user_id=? AND sector=? AND end_ts IS NULL
               ORDER BY id DESC LIMIT 1""",
            (user_id, sector)
        ).fetchone()
        if r:
            c.execute(
                "UPDATE sessions SET end_ts=?, reward_qc=? WHERE id=?",
                (end_ts, reward_qc, r["id"])
            )
        c.execute(
            "INSERT INTO tx_log(user_id, kind, amount, meta, created_at) VALUES(?,?,?,?,?)",
            (user_id, "qc_mining", reward_qc, sector, _now())
        )


# ----- TGWT pool & claims -----

def get_or_init_daily_pool(day_key: str, init_amount: int) -> int:
    with get_conn() as c:
        res = c.execute("SELECT remaining FROM daily_pool WHERE day_key=?", (day_key,)).fetchone()
        if res:
            return int(res["remaining"])
        c.execute("INSERT INTO daily_pool(day_key, remaining) VALUES(?,?)", (day_key, init_amount))
        return init_amount


def dec_daily_pool(day_key: str, amount: int):
    with get_conn() as c:
        c.execute("UPDATE daily_pool SET remaining = remaining - ? WHERE day_key=?", (amount, day_key))


def insert_tgwt_claim(user_id: int, task: str, amount: int, day_key: str):
    with get_conn() as c:
        c.execute(
            """INSERT INTO tgwt_claims(user_id, task, amount, day_key, created_at)
               VALUES(?,?,?,?,?)""",
            (user_id, task, amount, day_key, _now())
        )
        c.execute(
            "INSERT INTO tx_log(user_id, kind, amount, meta, created_at) VALUES(?,?,?,?,?)",
            (user_id, "tgwt_claim", amount, task, _now())
        )


# ----- Referral -----

def insert_referral(referrer_id: int, invitee_id: int):
    with get_conn() as c:
        c.execute(
            "INSERT OR IGNORE INTO referrals(referrer_id, invitee_id, created_at) VALUES(?,?,?)",
            (referrer_id, invitee_id, _now())
        )
