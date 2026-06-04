"""
database/store.py
Option B TGWT ledger store (SQLite).

DEV-safe:
- No private keys
- No chain calls
- Withdraw is simulated by moving balances Pending -> Withdrawn via a worker tick

Buckets:
- earned_u
- pending_u
- verified_u
- withdrawn_u

All amounts are micro-units (UNIT = 1_000_000).
"""
from __future__ import annotations

import os
import re
import time
import uuid
import sqlite3
import threading
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation, ROUND_DOWN
from typing import Optional, Dict, Any


UNIT = 1_000_000  # micro-units
EVM_ADDR_RE = re.compile(r"^0x[a-fA-F0-9]{40}$")

# userId safety (avoid weird chars / injection-ish ids)
_USER_ID_RE = re.compile(r"^[a-zA-Z0-9][a-zA-Z0-9_\-:]{1,127}$")


def default_db_path() -> str:
    """
    Default DB file under project root: <root>/qc_tgwt.sqlite3
    Can be overridden with env QC_TGWT_DB.
    """
    here = os.path.abspath(os.path.dirname(__file__))
    root = os.path.abspath(os.path.join(here, ".."))
    return os.getenv("QC_TGWT_DB", os.path.join(root, "qc_tgwt.sqlite3"))


def utc_now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def safe_user_id(user_id: str) -> str:
    """
    Validate and normalize user id used in DB keys.
    Expect forms like: tg_123456 or dev_abcdef01 etc.
    """
    u = (user_id or "").strip()
    if not _USER_ID_RE.match(u):
        raise ValueError("user_id_invalid")
    return u


def to_micro_units(amount: Any) -> int:
    """
    Accepts float/str/Decimal representing TGWT amount and converts to integer micro units.
    Rounds DOWN to 6 decimals for safety.
    """
    try:
        d = Decimal(str(amount))
    except (InvalidOperation, ValueError, TypeError):
        raise ValueError("amount_invalid")

    if d <= 0:
        raise ValueError("amount_must_be_positive")

    d = d.quantize(Decimal("0.000001"), rounding=ROUND_DOWN)
    u = int(d * UNIT)
    if u <= 0:
        raise ValueError("amount_too_small")
    return u


def micro_to_str(u: int) -> str:
    d = (Decimal(int(u)) / Decimal(UNIT)).quantize(Decimal("0.000001"), rounding=ROUND_DOWN)
    s = format(d, "f")
    s = s.rstrip("0").rstrip(".")
    return s if s else "0"


class Store:
    def __init__(self, db_path: str, withdraw_confirm_delay_sec: float = 2.0):
        self.db_path = db_path
        self.withdraw_confirm_delay_sec = float(withdraw_confirm_delay_sec)

        # ensure directory exists
        d = os.path.dirname(db_path)
        if d:
            os.makedirs(d, exist_ok=True)

        self._lock = threading.Lock()
        self._init_db()

    def _conn(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self.db_path, check_same_thread=False)
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA journal_mode=WAL;")
        conn.execute("PRAGMA synchronous=NORMAL;")
        conn.execute("PRAGMA foreign_keys=ON;")
        return conn

    def _init_db(self) -> None:
        with self._lock:
            conn = self._conn()
            try:
                conn.executescript(
                    """
                    CREATE TABLE IF NOT EXISTS users (
                        user_id TEXT PRIMARY KEY,
                        rev INTEGER NOT NULL DEFAULT 0,
                        earned_u INTEGER NOT NULL DEFAULT 0,
                        pending_u INTEGER NOT NULL DEFAULT 0,
                        verified_u INTEGER NOT NULL DEFAULT 0,
                        withdrawn_u INTEGER NOT NULL DEFAULT 0,
                        created_at TEXT NOT NULL,
                        updated_at TEXT NOT NULL
                    );

                    CREATE TABLE IF NOT EXISTS nonces (
                        user_id TEXT NOT NULL,
                        kind TEXT NOT NULL,
                        nonce TEXT NOT NULL,
                        created_at TEXT NOT NULL,
                        PRIMARY KEY (user_id, kind, nonce)
                    );

                    CREATE TABLE IF NOT EXISTS withdraw_requests (
                        id TEXT PRIMARY KEY,
                        user_id TEXT NOT NULL,
                        asset TEXT NOT NULL,
                        amount_u INTEGER NOT NULL,
                        to_address TEXT NOT NULL,
                        status TEXT NOT NULL,
                        tx_hash TEXT,
                        error TEXT,
                        created_at TEXT NOT NULL,
                        updated_at TEXT NOT NULL,
                        FOREIGN KEY (user_id) REFERENCES users(user_id)
                    );

                    CREATE INDEX IF NOT EXISTS idx_withdraw_status_created
                    ON withdraw_requests(status, created_at);

                    CREATE INDEX IF NOT EXISTS idx_withdraw_user_created
                    ON withdraw_requests(user_id, created_at);
                    """
                )
                conn.commit()
            finally:
                conn.close()

    def _ensure_user(self, conn: sqlite3.Connection, user_id: str) -> None:
        now = utc_now_iso()
        conn.execute(
            """
            INSERT INTO users(user_id, rev, earned_u, pending_u, verified_u, withdrawn_u, created_at, updated_at)
            VALUES(?, 0, 0, 0, 0, 0, ?, ?)
            ON CONFLICT(user_id) DO NOTHING;
            """,
            (user_id, now, now),
        )

    def _nonce_seen(self, conn: sqlite3.Connection, user_id: str, kind: str, nonce: str) -> bool:
        row = conn.execute(
            "SELECT 1 FROM nonces WHERE user_id=? AND kind=? AND nonce=? LIMIT 1;",
            (user_id, kind, nonce),
        ).fetchone()
        return row is not None

    def _save_nonce(self, conn: sqlite3.Connection, user_id: str, kind: str, nonce: str) -> None:
        conn.execute(
            "INSERT OR IGNORE INTO nonces(user_id, kind, nonce, created_at) VALUES(?,?,?,?);",
            (user_id, kind, nonce, utc_now_iso()),
        )

    def _state_row(self, row: sqlite3.Row) -> Dict[str, Any]:
        return {
            "ok": True,
            "user_id": row["user_id"],
            "rev": int(row["rev"]),
            "balances": {
                "earned_u": int(row["earned_u"]),
                "pending_u": int(row["pending_u"]),
                "verified_u": int(row["verified_u"]),
                "withdrawn_u": int(row["withdrawn_u"]),
            },
            "balances_h": {
                "earned": micro_to_str(int(row["earned_u"])),
                "pending": micro_to_str(int(row["pending_u"])),
                "verified": micro_to_str(int(row["verified_u"])),
                "withdrawn": micro_to_str(int(row["withdrawn_u"])),
            },
            "ts": utc_now_iso(),
        }

    def get_state(self, user_id: str) -> Dict[str, Any]:
        user_id = safe_user_id(user_id)
        with self._lock:
            conn = self._conn()
            try:
                conn.execute("BEGIN IMMEDIATE;")
                self._ensure_user(conn, user_id)
                row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                conn.commit()
                return self._state_row(row)
            finally:
                conn.close()

    def earn(self, user_id: str, amount_u: int, nonce: str, meta: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        user_id = safe_user_id(user_id)
        with self._lock:
            conn = self._conn()
            try:
                conn.execute("BEGIN IMMEDIATE;")
                self._ensure_user(conn, user_id)

                if self._nonce_seen(conn, user_id, "earn", nonce):
                    row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                    conn.commit()
                    out = self._state_row(row)
                    out["idempotent"] = True
                    return out

                conn.execute(
                    """
                    UPDATE users
                    SET earned_u = earned_u + ?,
                        rev = rev + 1,
                        updated_at = ?
                    WHERE user_id = ?;
                    """,
                    (int(amount_u), utc_now_iso(), user_id),
                )
                self._save_nonce(conn, user_id, "earn", nonce)

                row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                conn.commit()
                out = self._state_row(row)
                out["idempotent"] = False
                return out
            finally:
                conn.close()

    def verify(self, user_id: str, amount_u: Optional[int], nonce: str, source: Optional[str] = None) -> Dict[str, Any]:
        user_id = safe_user_id(user_id)
        with self._lock:
            conn = self._conn()
            try:
                conn.execute("BEGIN IMMEDIATE;")
                self._ensure_user(conn, user_id)

                if self._nonce_seen(conn, user_id, "verify", nonce):
                    row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                    conn.commit()
                    out = self._state_row(row)
                    out["idempotent"] = True
                    return out

                row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                earned = int(row["earned_u"])
                move = earned if amount_u is None else int(amount_u)

                if move <= 0:
                    raise ValueError("verify_amount_invalid")
                if move > earned:
                    raise ValueError("insufficient_earned")

                conn.execute(
                    """
                    UPDATE users
                    SET earned_u = earned_u - ?,
                        verified_u = verified_u + ?,
                        rev = rev + 1,
                        updated_at = ?
                    WHERE user_id = ?;
                    """,
                    (move, move, utc_now_iso(), user_id),
                )
                self._save_nonce(conn, user_id, "verify", nonce)

                row2 = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                conn.commit()
                out = self._state_row(row2)
                out["idempotent"] = False
                out["moved_u"] = move
                out["source"] = source
                return out
            finally:
                conn.close()

    def create_withdraw_request(self, user_id: str, asset: str, amount_u: int, to_address: str, nonce: str) -> Dict[str, Any]:
        user_id = safe_user_id(user_id)
        asset = (asset or "").upper().strip()
        if asset != "TGWT":
            raise ValueError("asset_not_supported")

        if not EVM_ADDR_RE.match((to_address or "").strip()):
            raise ValueError("to_address_invalid")

        req_id = uuid.uuid4().hex[:12]

        with self._lock:
            conn = self._conn()
            try:
                conn.execute("BEGIN IMMEDIATE;")
                self._ensure_user(conn, user_id)

                if self._nonce_seen(conn, user_id, "withdraw_request", nonce):
                    row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                    conn.commit()
                    out = self._state_row(row)
                    out["idempotent"] = True
                    out["note"] = "withdraw_request_nonce_seen"
                    return out

                row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                verified = int(row["verified_u"])
                if int(amount_u) > verified:
                    raise ValueError("insufficient_verified")

                # verified -> pending
                conn.execute(
                    """
                    UPDATE users
                    SET verified_u = verified_u - ?,
                        pending_u = pending_u + ?,
                        rev = rev + 1,
                        updated_at = ?
                    WHERE user_id = ?;
                    """,
                    (int(amount_u), int(amount_u), utc_now_iso(), user_id),
                )

                now = utc_now_iso()
                conn.execute(
                    """
                    INSERT INTO withdraw_requests(id, user_id, asset, amount_u, to_address, status, tx_hash, error, created_at, updated_at)
                    VALUES(?,?,?,?,?,'QUEUED',NULL,NULL,?,?);
                    """,
                    (req_id, user_id, asset, int(amount_u), to_address.strip(), now, now),
                )

                self._save_nonce(conn, user_id, "withdraw_request", nonce)

                row2 = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                conn.commit()

                out = self._state_row(row2)
                out["idempotent"] = False
                out["requestId"] = req_id
                out["status"] = "QUEUED"
                return out
            finally:
                conn.close()

    def get_withdraw_status(self, req_id: str) -> Dict[str, Any]:
        rid = (req_id or "").strip()
        if not rid:
            return {"ok": False, "error": "not_found"}

        with self._lock:
            conn = self._conn()
            try:
                row = conn.execute("SELECT * FROM withdraw_requests WHERE id=?;", (rid,)).fetchone()
                if not row:
                    return {"ok": False, "error": "not_found"}
                return {
                    "ok": True,
                    "request": {
                        "id": row["id"],
                        "userId": row["user_id"],
                        "asset": row["asset"],
                        "amount_u": int(row["amount_u"]),
                        "amount": micro_to_str(int(row["amount_u"])),
                        "toAddress": row["to_address"],
                        "status": row["status"],
                        "txHash": row["tx_hash"],
                        "error": row["error"],
                        "createdAt": row["created_at"],
                        "updatedAt": row["updated_at"],
                    },
                }
            finally:
                conn.close()

    def list_withdraws(self, user_id: str, limit: int = 20) -> Dict[str, Any]:
        user_id = safe_user_id(user_id)

        try:
            limit = int(limit)
        except Exception:
            limit = 20
        limit = max(1, min(limit, 100))

        with self._lock:
            conn = self._conn()
            try:
                rows = conn.execute(
                    """
                    SELECT id, user_id, asset, amount_u, to_address, status, tx_hash, error, created_at, updated_at
                    FROM withdraw_requests
                    WHERE user_id=?
                    ORDER BY created_at DESC
                    LIMIT ?;
                    """,
                    (user_id, limit),
                ).fetchall()

                items = []
                for r in rows:
                    amt_u = int(r["amount_u"])
                    items.append(
                        {
                            "id": r["id"],
                            "userId": r["user_id"],
                            "asset": r["asset"],
                            "amount_u": amt_u,
                            "amount": micro_to_str(amt_u),
                            "toAddress": r["to_address"],
                            "status": r["status"],
                            "txHash": r["tx_hash"],
                            "error": r["error"],
                            "createdAt": r["created_at"],
                            "updatedAt": r["updated_at"],
                        }
                    )

                return {"ok": True, "user_id": user_id, "limit": limit, "items": items, "ts": utc_now_iso()}
            finally:
                conn.close()

    def _confirm_withdraw(self, conn: sqlite3.Connection, req_id: str) -> None:
        row = conn.execute("SELECT * FROM withdraw_requests WHERE id=?;", (req_id,)).fetchone()
        if not row or row["status"] != "QUEUED":
            return

        user_id = row["user_id"]
        amount_u = int(row["amount_u"])

        urow = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
        if not urow:
            conn.execute(
                "UPDATE withdraw_requests SET status='REJECTED', error=?, updated_at=? WHERE id=?;",
                ("user_missing", utc_now_iso(), req_id),
            )
            return

        pending = int(urow["pending_u"])
        if pending < amount_u:
            conn.execute(
                "UPDATE withdraw_requests SET status='REJECTED', error=?, updated_at=? WHERE id=?;",
                ("insufficient_pending", utc_now_iso(), req_id),
            )
            # best-effort revert pending -> verified
            conn.execute(
                """
                UPDATE users
                SET verified_u = verified_u + ?,
                    pending_u = CASE WHEN pending_u >= ? THEN pending_u - ? ELSE pending_u END,
                    rev = rev + 1,
                    updated_at = ?
                WHERE user_id = ?;
                """,
                (amount_u, amount_u, amount_u, utc_now_iso(), user_id),
            )
            return

        # simulate tx hash
        tx_hash = "0x" + uuid.uuid4().hex + uuid.uuid4().hex[:8]

        conn.execute(
            """
            UPDATE users
            SET pending_u = pending_u - ?,
                withdrawn_u = withdrawn_u + ?,
                rev = rev + 1,
                updated_at = ?
            WHERE user_id = ?;
            """,
            (amount_u, amount_u, utc_now_iso(), user_id),
        )
        conn.execute(
            """
            UPDATE withdraw_requests
            SET status='CONFIRMED', tx_hash=?, updated_at=?
            WHERE id=?;
            """,
            (tx_hash, utc_now_iso(), req_id),
        )

    def worker_tick(self) -> None:
        """
        Confirm queued withdraws after self.withdraw_confirm_delay_sec.
        Call this periodically from a background thread.
        """
        now_ts = time.time()
        with self._lock:
            conn = self._conn()
            try:
                rows = conn.execute(
                    """
                    SELECT id, created_at
                    FROM withdraw_requests
                    WHERE status='QUEUED'
                    ORDER BY created_at ASC
                    LIMIT 20;
                    """
                ).fetchall()

                if not rows:
                    return

                conn.execute("BEGIN IMMEDIATE;")
                for r in rows:
                    created_iso = r["created_at"]
                    try:
                        created_ts = datetime.fromisoformat(created_iso).timestamp()
                    except Exception:
                        created_ts = now_ts  # if parse fails, skip delay
                    if (now_ts - created_ts) >= float(self.withdraw_confirm_delay_sec):
                        self._confirm_withdraw(conn, r["id"])
                conn.commit()
            finally:
                conn.close()
