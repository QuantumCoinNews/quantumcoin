# api_server.py
# QuantumCoin Space Miner - Option B backend (COMPLETED)
# TGWT 4-bucket ledger (earned/pending/verified/withdrawn) + rev-based sync
# + Global pool cap (default 3,000,000 TGWT) + audit events + withdraw queue worker
#
# Run:
#   pip install fastapi uvicorn pydantic
#   uvicorn api_server:app --host 0.0.0.0 --port 8081 --reload
#
# Notes:
# - DEV-safe: no private keys, no chain calls. Withdraw is simulated by a background worker
#   that "CONFIRMS" queued withdraws after a short delay and moves Pending -> Withdrawn.
# - Prod:
#   (1) verify Telegram initData server-side
#   (2) auth + rate limit
#   (3) disable open /earn OR protect it with admin key
#   (4) replace simulated withdraw with real BSC send (treasury key stays server-side)
#
# Env:
#   QC_TGWT_DB                 path to sqlite db
#   QC_TGWT_POOL_CAP_TGWT      default 3000000 (TGWT units, not micro)
#   QC_ADMIN_KEY               optional admin key (X-Admin-Key)
#   QC_EARN_MODE               "open" (default) or "admin"  (prod -> admin)
#   QC_WITHDRAW_CONFIRM_DELAY_SEC  default 2.0
#   QC_WORKER_POLL_SEC             default 0.6

from __future__ import annotations

import os
import re
import time
import uuid
import sqlite3
import threading
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation, ROUND_DOWN
from typing import Optional, Dict, Any, List

from fastapi import FastAPI, Query, Header
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

APP_NAME = "QuantumCoin TGWT API"
UNIT = 1_000_000  # micro-units

DB_PATH = os.getenv("QC_TGWT_DB", os.path.join(os.path.dirname(__file__), "qc_tgwt.sqlite3"))

POOL_CAP_TGWT = int(os.getenv("QC_TGWT_POOL_CAP_TGWT", "3000000"))  # TGWT (human units)
POOL_CAP_U = POOL_CAP_TGWT * UNIT

ADMIN_KEY = os.getenv("QC_ADMIN_KEY", "").strip()
EARN_MODE = os.getenv("QC_EARN_MODE", "open").strip().lower()  # open | admin

WITHDRAW_CONFIRM_DELAY_SEC = float(os.getenv("QC_WITHDRAW_CONFIRM_DELAY_SEC", "2.0"))
WORKER_POLL_SEC = float(os.getenv("QC_WORKER_POLL_SEC", "0.6"))

EVM_ADDR_RE = re.compile(r"^0x[a-fA-F0-9]{40}$")

def utc_now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()

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
    d = (Decimal(u) / Decimal(UNIT)).quantize(Decimal("0.000001"), rounding=ROUND_DOWN)
    s = format(d, "f")
    s = s.rstrip("0").rstrip(".")
    return s if s else "0"

def is_admin(x_admin_key: Optional[str]) -> bool:
    if not ADMIN_KEY:
        # dev-friendly: if no admin key set, treat as admin allowed
        return True
    return (x_admin_key or "").strip() == ADMIN_KEY

def safe_user_id(user_id: str) -> str:
    user_id = (user_id or "").strip()
    if not user_id or len(user_id) > 128:
        raise ValueError("user_id_invalid")
    # Keep it simple: allow letters, digits, underscore, dash
    # (Telegram ids are numeric; we prefix tg_ anyway.)
    if not re.fullmatch(r"[a-zA-Z0-9_\-\.]+", user_id):
        raise ValueError("user_id_invalid_chars")
    return user_id


class TGWTEarnReq(BaseModel):
    userId: str = Field(..., min_length=2, max_length=128)
    amount: Any
    nonce: str = Field(..., min_length=6, max_length=128)
    meta: Optional[Dict[str, Any]] = None  # optional extra info

class TGWTVerifyReq(BaseModel):
    userId: str = Field(..., min_length=2, max_length=128)
    # amount optional: if omitted, verify ALL earned
    amount: Optional[Any] = None
    nonce: str = Field(..., min_length=6, max_length=128)
    source: Optional[str] = None

class TGWTGrantReq(BaseModel):
    """
    Server-side grant from global pool. Use in production for mission rewards, watch rewards, etc.
    """
    userId: str = Field(..., min_length=2, max_length=128)
    amount: Any
    nonce: str = Field(..., min_length=6, max_length=128)
    reason: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None

class WithdrawReq(BaseModel):
    userId: str = Field(..., min_length=2, max_length=128)
    asset: str = Field(..., min_length=2, max_length=16)  # "TGWT"
    amount: Any
    toAddress: str = Field(..., min_length=8, max_length=128)
    nonce: str = Field(..., min_length=6, max_length=128)


class Store:
    def __init__(self, db_path: str):
        self.db_path = db_path
        os.makedirs(os.path.dirname(db_path), exist_ok=True) if os.path.dirname(db_path) else None
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

                    CREATE TABLE IF NOT EXISTS pool (
                        id INTEGER PRIMARY KEY CHECK (id = 1),
                        cap_u INTEGER NOT NULL,
                        distributed_u INTEGER NOT NULL DEFAULT 0,
                        created_at TEXT NOT NULL,
                        updated_at TEXT NOT NULL
                    );

                    CREATE TABLE IF NOT EXISTS events (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        user_id TEXT,
                        kind TEXT NOT NULL,
                        amount_u INTEGER NOT NULL DEFAULT 0,
                        ref_id TEXT,
                        meta_json TEXT,
                        created_at TEXT NOT NULL
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

                    CREATE INDEX IF NOT EXISTS idx_events_user_created
                    ON events(user_id, created_at);
                    """
                )

                # Ensure pool row exists
                now = utc_now_iso()
                conn.execute(
                    """
                    INSERT INTO pool(id, cap_u, distributed_u, created_at, updated_at)
                    VALUES(1, ?, 0, ?, ?)
                    ON CONFLICT(id) DO NOTHING;
                    """,
                    (POOL_CAP_U, now, now),
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

    def _event(self, conn: sqlite3.Connection, kind: str, amount_u: int = 0, user_id: Optional[str] = None,
               ref_id: Optional[str] = None, meta: Optional[Dict[str, Any]] = None) -> None:
        meta_json = None
        if meta is not None:
            try:
                import json
                meta_json = json.dumps(meta, ensure_ascii=False)
            except Exception:
                meta_json = None
        conn.execute(
            """
            INSERT INTO events(user_id, kind, amount_u, ref_id, meta_json, created_at)
            VALUES(?,?,?,?,?,?);
            """,
            (user_id, kind, int(amount_u), ref_id, meta_json, utc_now_iso()),
        )

    def get_pool(self) -> Dict[str, Any]:
        with self._lock:
            conn = self._conn()
            try:
                row = conn.execute("SELECT * FROM pool WHERE id=1;").fetchone()
                cap_u = int(row["cap_u"])
                dist_u = int(row["distributed_u"])
                return {
                    "ok": True,
                    "cap_u": cap_u,
                    "distributed_u": dist_u,
                    "remaining_u": max(cap_u - dist_u, 0),
                    "cap": micro_to_str(cap_u),
                    "distributed": micro_to_str(dist_u),
                    "remaining": micro_to_str(max(cap_u - dist_u, 0)),
                    "ts": utc_now_iso(),
                }
            finally:
                conn.close()

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

    def earn_open(self, user_id: str, amount_u: int, nonce: str, meta: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        DEV-friendly: adds to user.earned without touching pool cap.
        In production: use grant_from_pool() instead (server-side only).
        """
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
                    (amount_u, utc_now_iso(), user_id),
                )
                self._save_nonce(conn, user_id, "earn", nonce)
                self._event(conn, "EARN_OPEN", amount_u=amount_u, user_id=user_id, ref_id=nonce, meta=meta)

                row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                conn.commit()
                out = self._state_row(row)
                out["idempotent"] = False
                return out
            finally:
                conn.close()

    def grant_from_pool(self, user_id: str, amount_u: int, nonce: str, reason: Optional[str] = None,
                        meta: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Production-safe mint: consumes from global pool cap and adds to user.earned.
        """
        user_id = safe_user_id(user_id)
        with self._lock:
            conn = self._conn()
            try:
                conn.execute("BEGIN IMMEDIATE;")
                self._ensure_user(conn, user_id)

                if self._nonce_seen(conn, user_id, "grant", nonce):
                    row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                    conn.commit()
                    out = self._state_row(row)
                    out["idempotent"] = True
                    return out

                prow = conn.execute("SELECT * FROM pool WHERE id=1;").fetchone()
                cap_u = int(prow["cap_u"])
                dist_u = int(prow["distributed_u"])
                remaining = cap_u - dist_u
                if amount_u > remaining:
                    raise ValueError("pool_insufficient")

                # pool distributed += amount
                conn.execute(
                    "UPDATE pool SET distributed_u = distributed_u + ?, updated_at=? WHERE id=1;",
                    (amount_u, utc_now_iso()),
                )

                # user earned += amount
                conn.execute(
                    """
                    UPDATE users
                    SET earned_u = earned_u + ?,
                        rev = rev + 1,
                        updated_at = ?
                    WHERE user_id = ?;
                    """,
                    (amount_u, utc_now_iso(), user_id),
                )

                self._save_nonce(conn, user_id, "grant", nonce)
                meta2 = dict(meta or {})
                if reason:
                    meta2["reason"] = reason
                self._event(conn, "GRANT_POOL", amount_u=amount_u, user_id=user_id, ref_id=nonce, meta=meta2)

                row = conn.execute("SELECT * FROM users WHERE user_id=?;", (user_id,)).fetchone()
                conn.commit()
                out = self._state_row(row)
                out["idempotent"] = False
                out["pool_remaining_u"] = remaining - amount_u
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
                move = earned if amount_u is None else amount_u

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
                self._event(conn, "VERIFY", amount_u=move, user_id=user_id, ref_id=nonce, meta={"source": source})

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

        to_address = (to_address or "").strip()
        if not EVM_ADDR_RE.match(to_address):
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
                if amount_u > verified:
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
                    (amount_u, amount_u, utc_now_iso(), user_id),
                )

                now = utc_now_iso()
                conn.execute(
                    """
                    INSERT INTO withdraw_requests(id, user_id, asset, amount_u, to_address, status, tx_hash, error, created_at, updated_at)
                    VALUES(?,?,?,?,?,'QUEUED',NULL,NULL,?,?);
                    """,
                    (req_id, user_id, asset, amount_u, to_address, now, now),
                )

                self._save_nonce(conn, user_id, "withdraw_request", nonce)
                self._event(conn, "WITHDRAW_REQUEST", amount_u=amount_u, user_id=user_id, ref_id=req_id,
                            meta={"to": to_address})

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
        with self._lock:
            conn = self._conn()
            try:
                row = conn.execute(
                    "SELECT * FROM withdraw_requests WHERE id=?;",
                    (req_id,),
                ).fetchone()

                if not row:
                    return {"ok": False, "error": "not_found"}

                amt_u = int(row["amount_u"])
                return {
                    "ok": True,
                    "request": {
                        "id": row["id"],
                        "userId": row["user_id"],
                        "asset": row["asset"],
                        "amount_u": amt_u,
                        "amount": micro_to_str(amt_u),
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
                            "status": r["status"],
                            "amount": micro_to_str(amt_u),
                            "amount_u": amt_u,
                            "toAddress": r["to_address"],
                            "txHash": r["tx_hash"],
                            "error": r["error"],
                            "createdAt": r["created_at"],
                            "updatedAt": r["updated_at"],
                        }
                    )

                return {
                    "ok": True,
                    "userId": user_id,
                    "limit": limit,
                    "items": items,
                    "ts": utc_now_iso(),
                }
            finally:
                conn.close()

    def list_events(self, user_id: Optional[str], limit: int = 50) -> Dict[str, Any]:
        limit = max(1, min(int(limit), 200))
        with self._lock:
            conn = self._conn()
            try:
                if user_id:
                    user_id = safe_user_id(user_id)
                    rows = conn.execute(
                        """
                        SELECT * FROM events
                        WHERE user_id=?
                        ORDER BY created_at DESC
                        LIMIT ?;
                        """,
                        (user_id, limit),
                    ).fetchall()
                else:
                    rows = conn.execute(
                        """
                        SELECT * FROM events
                        ORDER BY created_at DESC
                        LIMIT ?;
                        """,
                        (limit,),
                    ).fetchall()

                out = []
                for r in rows:
                    out.append({
                        "id": int(r["id"]),
                        "userId": r["user_id"],
                        "kind": r["kind"],
                        "amount_u": int(r["amount_u"]),
                        "amount": micro_to_str(int(r["amount_u"])),
                        "refId": r["ref_id"],
                        "meta": r["meta_json"],
                        "createdAt": r["created_at"],
                    })
                return {"ok": True, "items": out, "ts": utc_now_iso()}
            finally:
                conn.close()

    # -------- worker: simulate CONFIRMED --------
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
            self._event(conn, "WITHDRAW_REJECT", amount_u=amount_u, user_id=user_id, ref_id=req_id,
                        meta={"error": "user_missing"})
            return

        pending = int(urow["pending_u"])
        if pending < amount_u:
            conn.execute(
                "UPDATE withdraw_requests SET status='REJECTED', error=?, updated_at=? WHERE id=?;",
                ("insufficient_pending", utc_now_iso(), req_id),
            )
            # revert best-effort: pending -> verified
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
            self._event(conn, "WITHDRAW_REJECT", amount_u=amount_u, user_id=user_id, ref_id=req_id,
                        meta={"error": "insufficient_pending"})
            return

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
        self._event(conn, "WITHDRAW_CONFIRMED", amount_u=amount_u, user_id=user_id, ref_id=req_id,
                    meta={"txHash": tx_hash})

    def worker_tick(self) -> None:
        now = time.time()
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
                        created_ts = now
                    if (now - created_ts) >= WITHDRAW_CONFIRM_DELAY_SEC:
                        self._confirm_withdraw(conn, r["id"])
                conn.commit()
            finally:
                conn.close()


store = Store(DB_PATH)

_worker_stop = threading.Event()

def worker_loop():
    while not _worker_stop.is_set():
        try:
            store.worker_tick()
        except Exception:
            pass
        time.sleep(WORKER_POLL_SEC)


app = FastAPI(title=APP_NAME)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # dev
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.on_event("startup")
def on_startup():
    t = threading.Thread(target=worker_loop, daemon=True)
    t.start()

@app.on_event("shutdown")
def on_shutdown():
    _worker_stop.set()


@app.get("/health")
def health():
    return {"ok": True, "ts": utc_now_iso(), "name": APP_NAME}


# -------- TGWT endpoints --------

@app.get("/api/v1/tgwt/pool")
def tgwt_pool():
    try:
        return store.get_pool()
    except Exception as e:
        return {"ok": False, "error": "pool_failed", "detail": str(e)}


@app.get("/api/v1/tgwt/state")
def tgwt_state(user_id: str = Query(..., alias="user_id")):
    try:
        return store.get_state(user_id)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "state_failed", "detail": str(e)}


@app.get("/api/v1/tgwt/events")
def tgwt_events(user_id: Optional[str] = Query(None, alias="user_id"), limit: int = 50):
    try:
        return store.list_events(user_id, limit=limit)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "events_failed", "detail": str(e)}


@app.post("/api/v1/tgwt/earn")
def tgwt_earn(req: TGWTEarnReq, x_admin_key: Optional[str] = Header(None, alias="X-Admin-Key")):
    """
    DEV endpoint:
      - If QC_EARN_MODE=open (default): anyone can call earn (demo).
      - If QC_EARN_MODE=admin: requires X-Admin-Key (prod).
    Production should prefer /tgwt/grant which consumes from pool cap.
    """
    try:
        if EARN_MODE == "admin" and not is_admin(x_admin_key):
            return {"ok": False, "error": "admin_required"}

        user_id = safe_user_id(req.userId)
        amt_u = to_micro_units(req.amount)
        return store.earn_open(user_id, amt_u, req.nonce, req.meta)

    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "earn_failed", "detail": str(e)}


@app.post("/api/v1/tgwt/grant")
def tgwt_grant(req: TGWTGrantReq, x_admin_key: Optional[str] = Header(None, alias="X-Admin-Key")):
    """
    Production-safe minting:
    Consumes from global pool cap and adds to user.earned.
    Should be called ONLY from your backend logic (mission finish / watch earn / social follow).
    """
    try:
        if not is_admin(x_admin_key):
            return {"ok": False, "error": "admin_required"}

        user_id = safe_user_id(req.userId)
        amt_u = to_micro_units(req.amount)
        return store.grant_from_pool(user_id, amt_u, req.nonce, reason=req.reason, meta=req.meta)

    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "grant_failed", "detail": str(e)}


@app.post("/api/v1/tgwt/verify")
def tgwt_verify(req: TGWTVerifyReq):
    try:
        user_id = safe_user_id(req.userId)
        amt_u = None if req.amount is None else to_micro_units(req.amount)
        return store.verify(user_id, amt_u, req.nonce, req.source)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "verify_failed", "detail": str(e)}

# -------- Withdraw endpoints --------

_USER_ID_RE_W = re.compile(r"^[a-zA-Z0-9_\-:]{2,128}$")
_REQ_ID_RE_W = re.compile(r"^[a-f0-9]{12}$")


def safe_user_id_w(raw: str) -> str:
    """
    Withdraw için userId validator.
    Not: safe_user_id() ismini KULLANMIYORUZ -> TGWT tarafındaki safe_user_id ile çakışmasın.
    """
    s = (raw or "").strip()
    if not _USER_ID_RE_W.match(s):
        raise ValueError("user_id_invalid")
    return s


def safe_req_id_w(raw: str) -> str:
    s = (raw or "").strip().lower()
    if not _REQ_ID_RE_W.match(s):
        raise ValueError("request_id_invalid")
    return s


@app.post("/api/v1/withdraw/request")
def withdraw_request(req: WithdrawReq):
    try:
        user_id = safe_user_id_w(req.userId)
        amt_u = to_micro_units(req.amount)
        return store.create_withdraw_request(user_id, req.asset, amt_u, req.toAddress, req.nonce)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "withdraw_request_failed", "detail": str(e)}


@app.get("/api/v1/withdraw/status")
def withdraw_status(id: str = Query(...)):
    try:
        req_id = safe_req_id_w(id)
        return store.get_withdraw_status(req_id)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "status_failed", "detail": str(e)}


@app.get("/api/v1/withdraw/list")
def withdraw_list(user_id: str = Query(..., alias="user_id"), limit: int = 20):
    try:
        uid = safe_user_id_w(user_id)
        return store.list_withdraws(uid, limit=limit)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "list_failed", "detail": str(e)}


# ---- Optional: allow running via `python api_server.py` ----
if __name__ == "__main__":
    import uvicorn
    uvicorn.run("api_server:app", host="0.0.0.0", port=8081, reload=True)
