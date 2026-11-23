from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import HTMLResponse, FileResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
import os
from pathlib import Path
import asyncio
import urllib.request
import sqlite3, time, threading, secrets, json, random
import json
import urllib.parse
import hashlib
import hmac
import math
from contextlib import closing
from typing import Optional, Dict, Any
from telegram_game.utils.qc_api import (
    fetch_health as qc_fetch_health,
    fetch_address_balance as qc_fetch_address_balance,
    fetch_tx_status as qc_fetch_tx_status,
    fetch_mine_block as qc_fetch_mine_block,
    fetch_send_tx as qc_fetch_send_tx,
)

from .persistence import database_path, connect_database
from . import exact_numbers
from .response_contracts import (
    HealthResponse, IdentityStatusResponse, IdentityVerificationResponse, RouteError,
    RouteOK, WalletChallengeResponse, WalletLinkResponse, WithdrawStatusResponse,
)
APP_DB = database_path()
_NFT_REPOSITORY_ROOT_V1 = Path(__file__).resolve().parents[2]
ONLINE_TTL_SEC = 90

from .tgwt_pool_policy import TGWT_UNIT as UNIT, TGWT_POOL_TOTAL_U

WITHDRAW_FEE_BPS = 0  # BNB gas is paid by the user wallet, never fabricated as a TGWT deduction.
DROP_CHANCE = 0.002     # 0.2%
DROP_AMOUNT_U = 1 * UNIT

app = FastAPI(title="QuantumCoin Telegram Game", version="0.5")
from .endpoint_response_contracts import ContractRoute
app.router.route_class = ContractRoute

from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

@app.exception_handler(RequestValidationError)
async def _redacted_request_validation_error(request, exc):
    return JSONResponse(status_code=422, content={"detail": [{"loc": item.get("loc", []), "type": item.get("type", "invalid"), "msg": "invalid request value"} for item in exc.errors()]})


# ACTION_BURST_GUARD_V1
# Purpose: lightweight player+endpoint+client burst guard for sensitive action endpoints.
# Boundary: this middleware does not mine, mint, transfer, change balances, change chain, or change supply.

_QC_ACTION_BURST_EVENTS_V1 = {}
_QC_ACTION_BURST_BLOCKS_V1 = []

_QC_ACTION_BURST_POLICY_V1 = {
    "/api/security/action-burst/test-ping": {"limit": 2, "windowSec": 10, "category": "safe_test"},
    "/api/mine/work": {"limit":60,"windowSec":60,"category":"client_pow"},
    "/api/mine/work/submit": {"limit":60,"windowSec":60,"category":"client_pow"},
    "/api/mine/work/status": {"limit":60,"windowSec":60,"category":"client_pow_status"},
    "/api/tgwt/rewards/claim": {"limit":8,"windowSec":60,"category":"verified_reward"},
    "/api/admin/tgwt/campaigns": {"limit":6,"windowSec":60,"category":"reward_policy"},
    "/api/social/x/start": {"limit":4,"windowSec":60,"category":"x_verification"},
    "/api/social/x/complete": {"limit":4,"windowSec":60,"category":"x_verification"},
    "/api/social/telegram/verify": {"limit":4,"windowSec":60,"category":"telegram_social_verification"},
    "/api/social/youtube/start": {"limit":4,"windowSec":60,"category":"youtube_social_verification"},
    "/api/social/youtube/complete": {"limit":4,"windowSec":60,"category":"youtube_social_verification"},
    "/api/mine/start": {"limit": 6, "windowSec": 20, "category": "mining_session"},
    "/api/mine/finish": {"limit": 4, "windowSec": 20, "category": "mining_reward"},
    "/api/wallet/send-qc": {"limit": 3, "windowSec": 30, "category": "chain_send"},
    "/api/wallet/link-qc-address": {"limit": 5, "windowSec": 60, "category": "wallet_link"},
    "/api/wallet/link-challenge": {"limit": 4, "windowSec": 60, "category": "wallet_link"},
    "/api/v1/withdraw/cancel": {"limit": 6, "windowSec": 60, "category": "tgwt_withdraw"},
    "/api/withdraw/request": {"limit": 3, "windowSec": 60, "category": "withdraw"},
    "/api/withdraw/mark-pending": {"limit": 4, "windowSec": 60, "category": "withdraw_admin"},
    "/api/withdraw/mock-confirm": {"limit": 4, "windowSec": 60, "category": "withdraw_admin"},
    "/api/withdraw/real-payout-precheck": {"limit": 3, "windowSec": 60, "category": "withdraw_admin"},
    "/api/v1/withdraw/request": {"limit": 3, "windowSec": 60, "category": "tgwt_withdraw"},
    "/api/settlement/mark-pending": {"limit": 4, "windowSec": 60, "category": "settlement_admin"},
    "/api/settlement/mock-confirm": {"limit": 4, "windowSec": 60, "category": "settlement_admin"},
    "/api/watch/claim": {"limit": 8, "windowSec": 60, "category": "claim"},
    "/api/social/claim": {"limit": 8, "windowSec": 60, "category": "claim"},
    "/api/store/buy": {"limit": 5, "windowSec": 60, "category": "store"},
    "/api/settings": {"limit": 10, "windowSec": 60, "category": "settings"},
    "/api/telegram/identity/verify": {"limit": 12, "windowSec": 60, "category": "identity"},
    "/api/nft/wallet/challenge": {"limit": 4, "windowSec": 60, "category": "nft_wallet"},
    "/api/nft/wallet/prove": {"limit": 8, "windowSec": 60, "category": "nft_wallet"},
    "/api/v1/nft/monthly/candidate": {"limit": 5, "windowSec": 60, "category": "nft_candidate"},
    "/api/admin/nft/monthly/review": {"limit": 8, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/ai-review": {"limit": 8, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/select-winner": {"limit": 4, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/freeze-pool": {"limit": 2, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/register-draw-commitment": {"limit": 4, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/draw-winner": {"limit": 4, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/publish-draw-commitment": {"limit": 2, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/draw-from-beacon": {"limit": 4, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/final-approval-statement": {"limit": 4, "windowSec": 60, "category": "nft_admin"},
    "/api/admin/nft/monthly/controlled-mint": {"limit": 2, "windowSec": 60, "category": "nft_admin"},
}


def _qc_action_burst_now_v1() -> int:
    import time
    return int(time.time())


def _runtime_limit_db_v1():
    # Separate short-lived connection, same canonical path. No economy hooks.
    return connect_database(APP_DB)


def _qc_action_burst_client_v1(request) -> str:
    return str(request.client.host)[:80] if request.client and request.client.host else "unknown-client"


def _qc_action_burst_extract_player_v1(body_bytes: bytes) -> str:
    try:
        import json
        if not body_bytes:
            return "unknown-player"
        data = json.loads(body_bytes.decode("utf-8", errors="replace"))
        for key in ("playerId", "player_id", "userId", "user_id", "id"):
            val = data.get(key)
            if val is not None and str(val).strip():
                return str(val).strip()[:120]
    except Exception:
        pass
    return "unknown-player"


def _qc_action_burst_key_v1(path: str, client: str, player: str) -> str:
    return "%s|%s|%s" % (path, client, player)


def _qc_action_burst_prune_v1(key: str, now: int, window_sec: int):
    events = _QC_ACTION_BURST_EVENTS_V1.get(key, [])
    events = [t for t in events if now - int(t) <= int(window_sec)]
    _QC_ACTION_BURST_EVENTS_V1[key] = events
    return events


def _qc_action_burst_block_response_v1(path: str, client: str, player: str, policy: dict, current_count: int):
    from fastapi.responses import JSONResponse

    now = _qc_action_burst_now_v1()
    row = {
        "ts": now,
        "path": path,
        "client": client,
        "player": player,
        "limit": int(policy.get("limit", 1)),
        "windowSec": int(policy.get("windowSec", 60)),
        "currentCount": int(current_count),
        "category": policy.get("category", "action"),
    }

    _QC_ACTION_BURST_BLOCKS_V1.append(row)
    del _QC_ACTION_BURST_BLOCKS_V1[:-50]

    return JSONResponse(
        status_code=429,
        content={
            "ok": False,
            "error": "action_burst_rate_limited",
            "message": "Action burst blocked before reaching endpoint logic.",
            "diagnosticVersion": "ACTION_BURST_GUARD_V1",
            "path": path,
            "playerId": player,
            "client": client,
            "limit": row["limit"],
            "windowSec": row["windowSec"],
            "currentCount": row["currentCount"],
            "safeBoundary": {
                "readOnly": False,
                "startsMining": False,
                "changesChain": False,
                "changesBalances": False,
                "mints": False,
                "changesSupply": False,
                "sendsTransactions": False,
                "adminAction": False,
                "reason": "Blocked before endpoint logic; no chain/reward/balance path reached."
            }
        }
    )


@app.middleware("http")
async def _qc_action_burst_guard_v1(request, call_next):
    path = request.url.path.rstrip("/")
    if request.method.upper() not in ("POST", "PUT", "PATCH", "DELETE") or path not in _QC_ACTION_BURST_POLICY_V1:
        return await call_next(request)
    from .player_auth import ACTOR
    from .durable_limits import admit
    policy = _QC_ACTION_BURST_POLICY_V1[path]
    now = _qc_action_burst_now_v1()
    # Authentication is outermost. Actor is independent of IP/body/requestId.
    client = _qc_action_burst_client_v1(request)
    actor = ACTOR.get() or ("bootstrap:"+client if path=="/api/telegram/identity/verify" else "operator")
    allowed = admit(_runtime_limit_db_v1, [(path+":actor",actor,int(policy["limit"]),int(policy["windowSec"])),
                          (path+":peer",client,max(120,int(policy["limit"])*10),int(policy["windowSec"]))],now)
    # Retire old unbounded maps. Preserve their names for historic diagnostics.
    _QC_ACTION_BURST_EVENTS_V1.clear()
    if not allowed:
        return _qc_action_burst_block_response_v1(path,client,actor,policy,int(policy["limit"]))
    return await call_next(request)


@app.get("/api/security/action-burst/status")
def _qc_security_action_burst_status_v1(limit: int = 50):
    import time

    safe_limit = max(1, min(int(limit or 50), 100))
    now = int(time.time())

    counters = []
    for key, events in list(_QC_ACTION_BURST_EVENTS_V1.items()):
        parts = key.split("|", 2)
        path = parts[0] if len(parts) > 0 else ""
        client = parts[1] if len(parts) > 1 else ""
        player = parts[2] if len(parts) > 2 else ""
        policy = _QC_ACTION_BURST_POLICY_V1.get(path, {})
        window_sec = int(policy.get("windowSec", 60))
        active = [int(t) for t in events if now - int(t) <= window_sec]
        _QC_ACTION_BURST_EVENTS_V1[key] = active
        counters.append({
            "path": path,
            "client": client,
            "player": player,
            "count": len(active),
            "limit": int(policy.get("limit", 0)),
            "windowSec": window_sec,
            "category": policy.get("category", "action"),
        })

    counters = sorted(counters, key=lambda r: (r["path"], r["player"]))[:safe_limit]

    return {
        "ok": True,
        "diagnosticVersion": "ACTION_BURST_GUARD_V1",
        "enabled": True,
        "protectedPathCount": len(_QC_ACTION_BURST_POLICY_V1),
        "protectedPaths": [
            {"path": p, **cfg}
            for p, cfg in sorted(_QC_ACTION_BURST_POLICY_V1.items())
        ],
        "activeCounters": counters,
        "recentBlocks": list(reversed(_QC_ACTION_BURST_BLOCKS_V1[-safe_limit:])),
        "summary": {
            "activeCounterCount": len(counters),
            "recentBlockCount": len(_QC_ACTION_BURST_BLOCKS_V1),
            "hasRecentBlocks": len(_QC_ACTION_BURST_BLOCKS_V1) > 0,
        },
        "safeBoundary": {
            "readOnly": True,
            "startsMining": False,
            "changesChain": False,
            "changesBalances": False,
            "mints": False,
            "changesSupply": False,
            "sendsTransactions": False,
            "adminAction": False
        }
    }


@app.post("/api/security/action-burst/test-ping")
def _qc_security_action_burst_test_ping_v1(payload: dict = None):
    payload = payload or {}
    return {
        "ok": True,
        "diagnosticVersion": "ACTION_BURST_GUARD_V1",
        "testOnly": True,
        "playerId": payload.get("playerId") or payload.get("player_id") or "unknown-player",
        "safeBoundary": {
            "readOnly": True,
            "startsMining": False,
            "changesChain": False,
            "changesBalances": False,
            "mints": False,
            "changesSupply": False,
            "sendsTransactions": False,
            "adminAction": False
        }
    }



# RATE_LIMIT_STATUS_V1
# Purpose: read-only rate-limit/burst visibility endpoint for Telegram Game API.
# Boundary: this endpoint does not mine, mint, transfer, change balances, change chain, or change supply.

def _qc_rate_limit_db_path_v1() -> str:
    from pathlib import Path
    return APP_DB


def _qc_rate_limit_source_text_v1() -> str:
    try:
        from pathlib import Path
        return Path(__file__).read_text(encoding="utf-8", errors="replace")
    except Exception:
        return ""


def _qc_rate_limit_count_v1(pattern: str, text: str) -> int:
    try:
        import re
        return len(re.findall(pattern, text, re.I))
    except Exception:
        return 0


def _qc_rate_limit_rows_v1(conn, sql: str, args=()):
    try:
        return [dict(r) for r in conn.execute(sql, args).fetchall()]
    except Exception as exc:
        return [{"error": str(exc), "sql": sql}]


def _qc_rate_limit_endpoint_audit_v1(text: str, limit: int = 50):
    import re

    endpoint_re = re.compile(
        r'@app\.(get|post|put|delete|patch)\(\"([^\"]+)\"[^\n]*\)\s*\n(?:async\s+)?def\s+([A-Za-z0-9_]+)\s*\(',
        re.M,
    )
    matches = list(endpoint_re.finditer(text))
    rows = []

    for i, m in enumerate(matches):
        start = m.start()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
        block = text[start:end]
        method = m.group(1).upper()
        path = m.group(2)
        function = m.group(3)
        low = (path + " " + function + " " + block).lower()

        is_action = method in ("POST", "PUT", "PATCH", "DELETE")
        is_publicish = any(
            k in low
            for k in [
                "wallet", "mine", "withdraw", "settlement", "watch",
                "social", "claim", "send", "store", "ship", "telegram",
                "identity", "admin", "nft", "dev/reset", "tx",
            ]
        )

        if not is_action:
            continue

        has_rate = bool(re.search(r"rate[_ -]?limit|ratelimit|limiter|too many|429|burst|throttle|cooldown", block, re.I))
        has_ip = bool(re.search(r"client\.host|request\.client|x-forwarded-for|remote_addr|ip[_ -]?address", block, re.I))
        has_idem = bool(re.search(r"idempot|requestId|request_id|nonce|replay", block, re.I))
        has_player = bool(re.search(r"playerId|player_id|get_player|require_player|telegram.*player|identity", block, re.I))
        has_state_effect = bool(re.search(r"mine|send-qc|tx/send|withdraw|settlement|reward_records|total_qc|qc_balance|real_payout", block, re.I))
        has_admin = bool(re.search(r"admin|ADMIN|dev guard|DEV", block, re.I))

        risk = []
        if is_publicish and not has_rate:
            risk.append("no_clear_rate_limit_or_burst_marker")
        if is_publicish and not has_ip:
            risk.append("no_clear_ip_client_marker")
        if has_state_effect and not has_idem:
            risk.append("state_changing_without_clear_idempotency_marker")
        if is_publicish and not has_player and not has_admin:
            risk.append("no_clear_player_or_admin_guard")

        if risk:
            rows.append(
                {
                    "method": method,
                    "path": path,
                    "function": function,
                    "line": text[: m.start()].count("\n") + 1,
                    "risk": risk,
                }
            )

    return rows[: max(1, min(int(limit or 50), 100))]


@app.get("/api/security/rate-limit/status")
def _qc_security_rate_limit_status_v1(limit: int = 50):
    import sqlite3

    safe_limit = max(1, min(int(limit or 50), 100))
    text = _qc_rate_limit_source_text_v1()

    source_signals = {
        "diagnosticVersion": "RATE_LIMIT_STATUS_V1",
        "rateLimitTotal": _qc_rate_limit_count_v1(r"rate[_ -]?limit|ratelimit|limiter", text),
        "burstTotal": _qc_rate_limit_count_v1(r"burst", text),
        "throttleTotal": _qc_rate_limit_count_v1(r"throttle", text),
        "cooldownTotal": _qc_rate_limit_count_v1(r"cooldown", text),
        "tooManyTotal": _qc_rate_limit_count_v1(r"too many|429", text),
        "ipClientTotal": _qc_rate_limit_count_v1(r"client\.host|request\.client|x-forwarded-for|remote_addr|ip[_ -]?address", text),
        "abuseSpamTotal": _qc_rate_limit_count_v1(r"abuse|anti[_ -]?abuse|spam|anti[_ -]?spam", text),
        "playerGuardTotal": _qc_rate_limit_count_v1(r"playerId|player_id|get_player|require_player", text),
        "telegramIdentityTotal": _qc_rate_limit_count_v1(r"TG_WEBAPP_IDENTITY|telegram/identity|initData", text),
        "idempotencyTotal": _qc_rate_limit_count_v1(r"idempot|requestId|request_id|nonce|replay", text),
        "conflictDetectorTotal": _qc_rate_limit_count_v1(r"CONFLICT_DETECTOR_STATUS_V1|conflict", text),
        "safeBoundaryTotal": _qc_rate_limit_count_v1(r"safeBoundary|safe_boundary", text),
    }

    endpoint_risk_rows = _qc_rate_limit_endpoint_audit_v1(text, safe_limit)

    conn = connect_database(_qc_rate_limit_db_path_v1())
    conn.row_factory = sqlite3.Row

    table_names = set(
        r["name"]
        for r in conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table'"
        ).fetchall()
    )

    players = []
    mining_sessions = []
    idempotency_recent = []
    open_withdraws = []
    tgwt_nonce_count = 0
    reward_count_dev_user = 0

    if "players" in table_names:
        players = _qc_rate_limit_rows_v1(
            conn,
            """
            SELECT player_id, qc_address, total_qc, total_xp, sessions, last_seen
            FROM players
            WHERE player_id IN ('dev_user','dev_user_2')
            ORDER BY player_id
            """,
        )

    if "mining_sessions" in table_names:
        mining_sessions = _qc_rate_limit_rows_v1(
            conn,
            """
            SELECT player_id, site_id, started_at
            FROM mining_sessions
            ORDER BY player_id, site_id
            LIMIT ?
            """,
            (safe_limit,),
        )

    if "action_idempotency_v1" in table_names:
        idempotency_recent = _qc_rate_limit_rows_v1(
            conn,
            """
            SELECT id, action, player_id, site_id, request_id, status,
                   created_at, completed_at, response_status
            FROM action_idempotency_v1
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

    if "withdraw_requests" in table_names:
        open_withdraws = _qc_rate_limit_rows_v1(
            conn,
            """
            SELECT id, user_id, asset, amount_u, status, created_at
            FROM withdraw_requests
            WHERE status NOT IN ('CONFIRMED','REJECTED','CANCELLED')
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

    if "tgwt_nonces" in table_names:
        try:
            tgwt_nonce_count = conn.execute("SELECT COUNT(*) FROM tgwt_nonces").fetchone()[0]
        except Exception:
            tgwt_nonce_count = -1

    if "reward_records" in table_names:
        try:
            reward_count_dev_user = conn.execute(
                "SELECT COUNT(*) FROM reward_records WHERE player_id='dev_user'"
            ).fetchone()[0]
        except Exception:
            reward_count_dev_user = -1

    conn.close()

    central_guard_detected = (
        source_signals["rateLimitTotal"] > 0
        and source_signals["idempotencyTotal"] > 0
    )

    ip_guard_visible = source_signals["ipClientTotal"] > 0
    burst_guard_visible = (
        source_signals["burstTotal"] > 0
        or source_signals["throttleTotal"] > 0
        or source_signals["cooldownTotal"] > 0
        or source_signals["tooManyTotal"] > 0
    )

    known_gaps = []
    if not ip_guard_visible:
        known_gaps.append("IP/client based guard is not clearly visible in source.")
    if not burst_guard_visible:
        known_gaps.append("Burst/throttle/cooldown markers are not clearly visible in source.")
    if endpoint_risk_rows:
        known_gaps.append("Some action endpoints still need explicit rate-limit/IP/idempotency review.")

    return {
        "ok": True,
        "diagnosticVersion": "RATE_LIMIT_STATUS_V1",
        "sourceSignals": source_signals,
        "summary": {
            "centralGuardDetected": central_guard_detected,
            "ipGuardVisible": ip_guard_visible,
            "burstGuardVisible": burst_guard_visible,
            "actionRiskCount": len(endpoint_risk_rows),
            "knownGapCount": len(known_gaps),
            "openWithdrawCount": len(open_withdraws),
            "miningSessionCount": len(mining_sessions),
            "idempotencyRecentCount": len(idempotency_recent),
            "tgwtNonceCount": tgwt_nonce_count,
            "rewardCountDevUser": reward_count_dev_user,
        },
        "actionRiskRows": endpoint_risk_rows,
        "knownGaps": known_gaps,
        "runtimeDb": {
            "tablesSeen": sorted(list(table_names)),
            "players": players,
            "miningSessions": mining_sessions,
            "idempotencyRecent": idempotency_recent,
            "openWithdraws": open_withdraws,
        },
        "safeBoundary": {
            "readOnly": True,
            "startsMining": False,
            "changesChain": False,
            "changesBalances": False,
            "mints": False,
            "changesSupply": False,
            "sendsTransactions": False,
            "adminAction": False
        }
    }



# CONFLICT_DETECTOR_STATUS_V1
# Purpose: read-only conflict/double-spend status endpoint for Telegram Game transaction paths.
# Boundary: this endpoint does not mint, mine, transfer, change balances, change chain, or change supply.

def _qc_conflict_db_path_v1() -> str:
    from pathlib import Path
    return APP_DB


def _qc_conflict_rows_v1(conn, sql: str, args=()):
    try:
        return [dict(r) for r in conn.execute(sql, args).fetchall()]
    except Exception as exc:
        return [{"error": str(exc), "sql": sql}]


def _qc_conflict_table_exists_v1(conn, table: str) -> bool:
    try:
        row = conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?",
            (table,),
        ).fetchone()
        return row is not None
    except Exception:
        return False


def _qc_conflict_table_cols_v1(conn, table: str):
    try:
        return [r["name"] for r in conn.execute("PRAGMA table_info(%s)" % table).fetchall()]
    except Exception:
        return []


def _qc_conflict_has_error_rows_v1(rows) -> bool:
    try:
        return any(isinstance(r, dict) and "error" in r for r in rows)
    except Exception:
        return True


@app.get("/api/security/conflicts/status")
def _qc_security_conflicts_status_v1(limit: int = 20):
    import sqlite3

    safe_limit = max(1, min(int(limit or 20), 100))

    conn = connect_database(_qc_conflict_db_path_v1())
    conn.row_factory = sqlite3.Row

    table_names = set(
        r["name"]
        for r in conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table'"
        ).fetchall()
    )

    duplicate_qc_addresses = []
    duplicate_reward_groups = []
    duplicate_qc_tx_ids = []
    qc_tx_status_counts = []
    qc_tx_recent = []
    withdraw_status_groups = []
    open_withdraws = []
    ready_for_chain_rewards = []
    idempotency_recent = []

    if "players" in table_names:
        duplicate_qc_addresses = _qc_conflict_rows_v1(
            conn,
            """
            SELECT qc_address, COUNT(*) AS count, GROUP_CONCAT(player_id) AS players
            FROM players
            WHERE COALESCE(qc_address, '') != ''
            GROUP BY qc_address
            HAVING COUNT(*) > 1
            ORDER BY count DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

    if "reward_records" in table_names:
        duplicate_reward_groups = _qc_conflict_rows_v1(
            conn,
            """
            SELECT player_id, source, amount, target_qc_address, chain_height, created_at,
                   COUNT(*) AS count, GROUP_CONCAT(reward_id) AS reward_ids
            FROM reward_records
            GROUP BY player_id, source, amount, target_qc_address, chain_height, created_at
            HAVING COUNT(*) > 1
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

        ready_for_chain_rewards = _qc_conflict_rows_v1(
            conn,
            """
            SELECT reward_id, player_id, amount, settlement_status, target_qc_address,
                   chain_height, created_at
            FROM reward_records
            WHERE settlement_status='READY_FOR_CHAIN'
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

    if "qc_tx_records" in table_names:
        qc_cols = _qc_conflict_table_cols_v1(conn, "qc_tx_records")

        tx_col = None
        for c in ["tx_hash", "txid", "tx_id"]:
            if c in qc_cols:
                tx_col = c
                break

        if tx_col:
            duplicate_qc_tx_ids = _qc_conflict_rows_v1(
                conn,
                """
                SELECT {tx_col} AS tx_key, COUNT(*) AS count, GROUP_CONCAT(rowid) AS rowids
                FROM qc_tx_records
                WHERE COALESCE({tx_col}, '') != ''
                GROUP BY {tx_col}
                HAVING COUNT(*) > 1
                ORDER BY count DESC
                LIMIT ?
                """.format(tx_col=tx_col),
                (safe_limit,),
            )

        if "status" in qc_cols:
            qc_tx_status_counts = _qc_conflict_rows_v1(
                conn,
                """
                SELECT status, COUNT(*) AS count
                FROM qc_tx_records
                GROUP BY status
                ORDER BY count DESC
                """,
            )

        select_cols = [
            c for c in [
                "tx_id", "txid", "tx_hash", "player_id", "from_address",
                "to_address", "amount", "status", "created_at", "confirmed_at"
            ]
            if c in qc_cols
        ]
        if select_cols:
            qc_tx_recent = _qc_conflict_rows_v1(
                conn,
                "SELECT %s FROM qc_tx_records ORDER BY rowid DESC LIMIT ?" % ", ".join(select_cols),
                (safe_limit,),
            )
        else:
            qc_tx_recent = _qc_conflict_rows_v1(
                conn,
                "SELECT * FROM qc_tx_records ORDER BY rowid DESC LIMIT ?",
                (safe_limit,),
            )

    if "withdraw_requests" in table_names:
        withdraw_status_groups = _qc_conflict_rows_v1(
            conn,
            """
            SELECT user_id, asset, amount_u, status, COUNT(*) AS count, GROUP_CONCAT(id) AS ids
            FROM withdraw_requests
            GROUP BY user_id, asset, amount_u, status
            HAVING COUNT(*) > 1
            ORDER BY count DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

        open_withdraws = _qc_conflict_rows_v1(
            conn,
            """
            SELECT id, user_id, asset, amount_u, status, created_at
            FROM withdraw_requests
            WHERE status NOT IN ('CONFIRMED','REJECTED','CANCELLED')
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

    if "action_idempotency_v1" in table_names:
        idempotency_recent = _qc_conflict_rows_v1(
            conn,
            """
            SELECT id, action, player_id, site_id, request_id, status,
                   created_at, completed_at, response_status
            FROM action_idempotency_v1
            ORDER BY created_at DESC
            LIMIT ?
            """,
            (safe_limit,),
        )

    conn.close()

    duplicate_groups = {
        "duplicateQcAddressCount": len(duplicate_qc_addresses),
        "duplicateRewardGroupCount": len(duplicate_reward_groups),
        "duplicateQcTxIdCount": len(duplicate_qc_tx_ids),
        "withdrawStatusGroupCount": len(withdraw_status_groups),
    }

    error_flags = {
        "duplicateQcAddressesQueryError": _qc_conflict_has_error_rows_v1(duplicate_qc_addresses),
        "duplicateRewardGroupsQueryError": _qc_conflict_has_error_rows_v1(duplicate_reward_groups),
        "duplicateQcTxIdsQueryError": _qc_conflict_has_error_rows_v1(duplicate_qc_tx_ids),
        "withdrawStatusGroupsQueryError": _qc_conflict_has_error_rows_v1(withdraw_status_groups),
    }

    has_duplicate_conflict = (
        duplicate_groups["duplicateQcAddressCount"] > 0
        or duplicate_groups["duplicateRewardGroupCount"] > 0
        or duplicate_groups["duplicateQcTxIdCount"] > 0
        or duplicate_groups["withdrawStatusGroupCount"] > 0
    )

    return {
        "ok": True,
        "diagnosticVersion": "CONFLICT_DETECTOR_STATUS_V1",
        "tablesSeen": sorted(list(table_names)),
        "summary": {
            **duplicate_groups,
            "openWithdrawCount": len(open_withdraws),
            "readyForChainRewardCount": len(ready_for_chain_rewards),
            "idempotencyRecentCount": len(idempotency_recent),
            "hasDuplicateConflict": has_duplicate_conflict,
            **error_flags,
        },
        "duplicateQcAddresses": duplicate_qc_addresses,
        "duplicateRewardGroups": duplicate_reward_groups,
        "duplicateQcTxIds": duplicate_qc_tx_ids,
        "withdrawStatusGroups": withdraw_status_groups,
        "qcTxStatusCounts": qc_tx_status_counts,
        "qcTxRecent": qc_tx_recent,
        "openWithdraws": open_withdraws,
        "readyForChainRewards": ready_for_chain_rewards,
        "idempotencyRecent": idempotency_recent,
        "safeBoundary": {
            "readOnly": True,
            "startsMining": False,
            "changesChain": False,
            "changesBalances": False,
            "mints": False,
            "changesSupply": False,
            "sendsTransactions": False,
            "adminAction": False
        }
    }



# IDEMPOTENCY_REPLAY_GUARD_V1
# Purpose: block duplicate/replayed POST /api/mine/finish before a second chain mine or reward insert can happen.
# Boundary: this guard does not mint, transfer, change supply, change consensus, or mine by itself.

def _qc_idem_db_path_v1() -> str:
    from pathlib import Path
    return APP_DB


def _qc_idem_now_v1() -> int:
    import time
    return int(time.time())


def _qc_idem_connect_v1():
    import sqlite3
    conn = connect_database(_qc_idem_db_path_v1())
    conn.isolation_level = None
    return conn


def _qc_idem_ensure_v1(conn) -> None:
    conn.execute("""
    CREATE TABLE IF NOT EXISTS action_idempotency_v1 (
        id TEXT PRIMARY KEY,
        action TEXT NOT NULL,
        key_hash TEXT NOT NULL,
        player_id TEXT,
        site_id TEXT,
        request_id TEXT,
        session_started_at TEXT,
        status TEXT NOT NULL,
        created_at INTEGER NOT NULL,
        completed_at INTEGER,
        response_status INTEGER,
        note TEXT,
        UNIQUE(action, key_hash)
    )
    """)
    conn.execute("CREATE INDEX IF NOT EXISTS idx_action_idem_v1_player ON action_idempotency_v1(player_id, action, created_at)")
    conn.execute("CREATE INDEX IF NOT EXISTS idx_action_idem_v1_status ON action_idempotency_v1(status, created_at)")


def _qc_idem_session_started_at_v1(conn, player_id: str, site_id: str) -> str:
    try:
        row = conn.execute(
            """
            SELECT started_at
            FROM mining_sessions
            WHERE player_id = ? AND site_id = ?
            ORDER BY started_at DESC
            LIMIT 1
            """,
            (player_id, int(site_id)),
        ).fetchone()
        if row and row[0] is not None:
            return str(row[0])
    except Exception:
        return ""
    return ""


def _qc_idem_key_hash_v1(action: str, raw_key: str) -> str:
    import hashlib
    payload = (action + "::" + raw_key).encode("utf-8", errors="replace")
    return hashlib.sha256(payload).hexdigest()


def _qc_idem_block_response_v1(player_id: str, site_id: str, request_id: str, key_hash: str):
    from starlette.responses import JSONResponse
    return JSONResponse(
        status_code=409,
        content={
            "ok": False,
            "error": "idempotency_replay_blocked",
            "message": "Duplicate mine/finish request blocked before chain action.",
            "diagnosticVersion": "IDEMPOTENCY_REPLAY_GUARD_V1",
            "action": "mine_finish",
            "playerId": player_id,
            "siteId": site_id,
            "requestId": request_id,
            "keyHash": key_hash,
            "safeBoundary": {
                "readOnly": False,
                "startsMining": False,
                "changesChain": False,
                "changesBalances": False,
                "mints": False,
                "changesSupply": False,
                "reason": "Replay was blocked before reaching the chain/reward path."
            },
        },
    )


# Retired: durable server-owned game_mining_runs_v1 now reserves completion.
@app.get("/api/security/idempotency/status")
def _qc_idempotency_status_v1(limit: int = 10):
    with closing(_db()) as conn:
        states = dict(conn.execute("SELECT status,COUNT(*) FROM game_mining_runs_v1 GROUP BY status").fetchall())
    return {"ok": True, "guard": "server_owned_mining_runs_v1", "states": states,
            "guarantee": "at_most_once_node_submission", "automaticRetry": False}


# Serve the Space Miner UI and assets from the canonical QuantumCoin repo path.
_BASE_DIR = Path(__file__).resolve().parent
_DEV_UI_DIR = _BASE_DIR / "dev_ui"
_GAME_HTML = _DEV_UI_DIR / "game.html"
_ASSETS_DIR = _DEV_UI_DIR / "assets"

if _ASSETS_DIR.exists():
    app.mount("/assets", StaticFiles(directory=str(_ASSETS_DIR)), name="space_miner_assets")

if _DEV_UI_DIR.exists():
    app.mount("/dev_ui", StaticFiles(directory=str(_DEV_UI_DIR)), name="space_miner_dev_ui")

@app.get("/", include_in_schema=False)
def space_miner_root():
    if _GAME_HTML.exists():
        return FileResponse(str(_GAME_HTML), media_type="text/html")
    return HTMLResponse("<h1>Space Miner UI not found</h1>", status_code=404)

@app.get("/game", include_in_schema=False)
def space_miner_game():
    return space_miner_root()


# DEV CORS (en sorunsuz)
app.add_middleware(
    CORSMiddleware,
    allow_origins=[origin.strip() for origin in os.getenv("QC_GAME_CORS_ORIGINS", "").split(",") if origin.strip() and origin.strip() != "*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

_lock = threading.Lock()


def _now() -> int:
    return int(time.time())


def _db():
    return connect_database(APP_DB)


def _init():
    with _lock:
        conn = _db()

        from .player_auth import ensure_schema as ensure_auth_schema
        from .mining_sessions import ensure_schema as ensure_run_schema
        ensure_auth_schema(conn)
        ensure_run_schema(conn)
        from .reward_evidence import ensure_schema as ensure_reward_evidence
        from .durable_limits import ensure_schema as ensure_limit_schema
        from .game_pow import ensure_schema as ensure_pow_schema
        ensure_reward_evidence(conn)
        ensure_limit_schema(conn)
        ensure_pow_schema(conn)
        from .x_reward_verifier import ensure_schema as ensure_x_schema
        ensure_x_schema(conn)
        from .youtube_reward_verifier import ensure_schema as ensure_youtube_schema
        ensure_youtube_schema(conn)

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
        if "qc_address" not in {row[1] for row in conn.execute("PRAGMA table_info(players)")}:
            conn.execute("ALTER TABLE players ADD COLUMN qc_address TEXT")
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

        exact_numbers.ensure_schema(conn)
        conn.commit()
        conn.close()


_init()

# ------------------- Models -------------------

class HeartbeatIn(BaseModel):
    playerId: str
    displayName: Optional[str] = None


class SessionCompleteIn(BaseModel):
    model_config = {"allow_inf_nan": False}
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
    initData: Optional[str] = None
    nftSessionId: Optional[str] = None


class MineFinishIn(BaseModel):
    model_config = {"allow_inf_nan": False}
    playerId: str
    sessionId: str = ""
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
    amount: str  # Exact decimal text; JSON floating-point amounts are rejected.
    toAddress: str
    nonce: str


class SocialClaimIn(BaseModel):
    playerId: str
    platform: str
    nonce: str
    campaignId: str = ""
    proof: Optional[dict] = None


class StoreBuyIn(BaseModel):
    playerId: str
    shipId: str
    nonce: str
    campaignId: str = ""
    proof: Optional[dict] = None



class TelegramIdentityVerifyIn(BaseModel):
    # TG_WEBAPP_IDENTITY_V1
    initData: str


# ------------------- Helpers -------------------


# TG_WEBAPP_IDENTITY_V1
# Read-only Telegram WebApp initData verification helpers.
# This does not mint, mine, transfer, or change balances.
def _telegram_bot_token() -> str:
    return (
        os.getenv("TELEGRAM_BOT_TOKEN")
        or os.getenv("QC_TELEGRAM_BOT_TOKEN")
        or os.getenv("TELEGRAM_GAME_BOT_TOKEN")
        or ""
    ).strip()


def _telegram_auth_required() -> bool:
    # Identity protection cannot be disabled by a development environment flag.
    return True


def _telegram_player_id_from_user(tg_user: dict) -> str:
    if not isinstance(tg_user, dict):
        return ""
    value = tg_user.get("id")
    if type(value) is not int or value <= 0 or value > 2**52:
        return ""
    uid = str(value)
    if not uid:
        return ""
    return "tg_" + uid


def _validate_telegram_webapp_init_data(init_data: str, max_age_sec: int = 86400) -> dict:
    raw = (init_data or "").strip()
    if not raw or len(raw.encode("utf-8")) > 16384:
        return {"ok": False, "error": "invalid_initData_size"}

    bot_token = _telegram_bot_token()
    if not bot_token:
        return {
            "ok": False,
            "error": "telegram_bot_token_not_configured",
            "message": "Set TELEGRAM_BOT_TOKEN or QC_TELEGRAM_BOT_TOKEN before enforcing Telegram WebApp identity.",
        }

    try:
        pairs = urllib.parse.parse_qsl(raw, keep_blank_values=True, strict_parsing=False)
        if len(pairs) != len({key for key, _ in pairs}):
            return {"ok": False, "error": "duplicate_initData_key"}
        data = dict(pairs)
    except Exception as e:
        return {"ok": False, "error": "initData_parse_failed", "detail": str(e)}

    received_hash = (data.get("hash") or "").strip()
    if not received_hash:
        return {"ok": False, "error": "missing_hash"}

    check_items = []
    for key in sorted(k for k in data.keys() if k != "hash"):
        check_items.append(f"{key}={data[key]}")
    data_check_string = "\n".join(check_items)

    secret_key = hmac.new(
        b"WebAppData",
        bot_token.encode("utf-8"),
        hashlib.sha256,
    ).digest()

    computed_hash = hmac.new(
        secret_key,
        data_check_string.encode("utf-8"),
        hashlib.sha256,
    ).hexdigest()

    if not hmac.compare_digest(computed_hash, received_hash):
        return {"ok": False, "error": "invalid_hash"}

    auth_date_raw = str(data.get("auth_date") or "0").strip()
    try:
        auth_date = int(auth_date_raw)
    except Exception:
        auth_date = 0

    now_ts = _now()
    if auth_date <= 0 or auth_date > now_ts + 60:
        return {"ok": False, "error": "invalid_auth_date"}

    age_sec = max(0, now_ts - auth_date)
    if max_age_sec > 0 and age_sec > max_age_sec:
        return {
            "ok": False,
            "error": "auth_date_expired",
            "ageSec": age_sec,
            "maxAgeSec": max_age_sec,
        }

    tg_user = {}
    user_raw = data.get("user") or ""
    if user_raw:
        try:
            tg_user = json.loads(user_raw)
        except Exception as e:
            return {"ok": False, "error": "user_json_parse_failed", "detail": str(e)}

    player_id = _telegram_player_id_from_user(tg_user)

    if not player_id:
        return {"ok": False, "error": "telegram_user_id_missing"}

    return {
        "ok": True,
        "playerId": player_id,
        "telegramUser": {
            "id": tg_user.get("id"),
            "username": tg_user.get("username"),
            "first_name": tg_user.get("first_name"),
            "last_name": tg_user.get("last_name"),
            "language_code": tg_user.get("language_code"),
        },
        "authDate": auth_date,
        "ageSec": age_sec,
        "hashOk": True,
        "rawKeys": sorted(data.keys()),
    }


def _nft_verified_player_id_v1(init_data: str, claimed_player_id: str) -> str:
    """Bind an NFT player ID to current, signed Telegram WebApp data."""
    verified = _validate_telegram_webapp_init_data(init_data, max_age_sec=86400)
    if not verified.get("ok"):
        return ""

    player_id = str(verified.get("playerId") or "")
    claimed = str(claimed_player_id or "").strip()
    if not player_id or not claimed:
        return ""
    if not hmac.compare_digest(player_id, claimed):
        return ""

    # The shared validator checks age; reject signatures dated in the future.
    try:
        auth_date = int(verified.get("authDate") or 0)
    except (TypeError, ValueError):
        return ""
    if auth_date > _now() + 60:
        return ""
    return player_id

def _touch_player(conn, pid: str, name: Optional[str]):
    ts = _now()
    conn.execute(
        """
        INSERT OR IGNORE INTO players(player_id, display_name, first_seen, last_seen, numeric_version)
        VALUES(?,?,?,?,1)
        """,
        (pid, (name or None), ts, ts),
    )
    if name:
        conn.execute("UPDATE players SET display_name=?, last_seen=? WHERE player_id=?", (name, ts, pid))
    else:
        conn.execute("UPDATE players SET last_seen=? WHERE player_id=?", (ts, pid))


_rl = {}  # (user_id, key) -> (win_start, count)

def _rl_hit(user_id: str, key: str, limit: int, window_sec: int) -> bool:
    from .durable_limits import admit
    _rl.clear()
    return admit(_runtime_limit_db_v1, [("domain:"+key,user_id,limit,window_sec)],_now())


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
    from .withdraw_ledger import amounts
    return amounts(conn, user_id)


def _tgwt_bump_rev(conn, user_id: str):
    conn.execute("UPDATE tgwt_state SET rev = rev + 1, updated_at=? WHERE user_id=?", (_now(), user_id))


# Legacy tgwt_nonces rows are audit history only. Active actions use their own
# proof/session/withdraw idempotency keys, never this cross-action nonce table.


def _tgwt_award_pending_from_pool(conn, user_id: str, amount_u: int, kind: str, meta: Optional[Dict[str, Any]] = None) -> bool:
    # Retired: no runtime caller may manufacture reward entitlement from amount/nonce.
    return False


def _tgwt_move_pending_to_verified(conn, user_id: str, move_u: int) -> int:
    # Historical unproven pending balances are never silently promoted.
    return 0


def _online_now(conn) -> int:
    ts = _now()
    cutoff = ts - ONLINE_TTL_SEC
    cur = conn.execute("SELECT COUNT(*) FROM players WHERE last_seen >= ?", (cutoff,))
    return int(cur.fetchone()[0])


def _total_players(conn) -> int:
    cur = conn.execute("SELECT COUNT(*) FROM players")
    return int(cur.fetchone()[0])


def _total_qc_all(conn):
    # Sum in Python so SQLite cannot overflow or promote to REAL.
    return exact_numbers.format_qc(sum(row[0] for row in conn.execute("SELECT qc_atoms FROM players")))


def _get_player_totals(conn, pid):
    row = conn.execute("SELECT qc_atoms,xp_units,total_qc,total_xp FROM players WHERE player_id=?", (pid,)).fetchone()
    atoms, xp = (exact_numbers.integer(row[0]), exact_numbers.integer(row[1])) if row else (0, 0)
    return {"qc": exact_numbers.format_qc(atoms), "qcAtoms": str(atoms), "xp": xp,
            "legacyNumericReviewRequired": bool(row and (row[2] != 0 or row[3] != 0))}


def _rank_for(conn, pid):
    atoms = int(_get_player_totals(conn, pid)["qcAtoms"])
    return 1 + conn.execute("SELECT COUNT(*) FROM players WHERE qc_atoms > ?", (atoms,)).fetchone()[0]


def _xp_to_level(total_xp: int) -> Dict[str, Any]:
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



def _ensure_unique_qc_address_index(conn):
    # QC_ADDRESS_UNIQUE_GUARD_V1
    # One non-empty QC address can belong to only one Telegram Game player.
    _ensure_qc_address_column(conn)
    conn.execute(
        """
        CREATE UNIQUE INDEX IF NOT EXISTS idx_players_qc_address_unique_nonempty
        ON players(qc_address)
        WHERE COALESCE(qc_address, '') != ''
        """
    )


def _find_qc_address_owner(conn, address: str) -> str:
    # QC_ADDRESS_UNIQUE_GUARD_V1
    _ensure_qc_address_column(conn)
    addr = (address or "").strip()
    if not addr:
        return ""
    row = conn.execute(
        "SELECT player_id FROM players WHERE qc_address=? LIMIT 1",
        (addr,),
    ).fetchone()
    return ((row[0] if row else "") or "").strip()


def _wallet_payload(conn, pid: str) -> Dict[str, Any]:
    totals = _get_player_totals(conn, pid)
    xpinfo = _xp_to_level(totals["xp"])
    return {
        "playerId": pid,
        "address": f"QC_{pid[:10]}_DEV",
        "qc_balance": totals["qc"],
        "qc_atoms": totals["qcAtoms"],
        "atomsPerQc": str(exact_numbers.ATOMS_PER_QC),
        "xpTotal": totals["xp"],
        "legacyNumericReviewRequired": totals["legacyNumericReviewRequired"],
        "tgwt_balance": 0,
        "level": xpinfo["level"],
        "xp": xpinfo["xp"],
        "xp_next": xpinfo["xpNext"],
    }


def _compute_rewards(duration_sec: int):
    # Duration comes exclusively from the persisted server run.
    return {"qc": 0, "xp": min(100, exact_numbers.integer(duration_sec))}


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
QC_GAME_MINER_BLOCK_REWARD = 0  # Compatibility only: QC reward is recorded solely by the chain.
QC_GAME_MINER_CLIENT_TYPE = "GAME_MINER"
QC_GAME_MINER_REWARD_MODE = "NETWORK_BLOCK_REWARD"
# QC_GAME_MINING_MODE_V1
# REAL_CHAIN: dev-local real release node mining.
# TESTNET_SIMULATED: testnet-safe game mining; does not call chain /api/mine.
# DISABLED: mining endpoint returns a guarded disabled response.
QC_GAME_MINING_MODE_REAL_CHAIN = "REAL_CHAIN"
QC_GAME_MINING_MODE_TESTNET_SIMULATED = "TESTNET_SIMULATED"
QC_GAME_MINING_MODE_DISABLED = "DISABLED"

QC_GAME_MINER_TESTNET_CLIENT_TYPE = "GAME_MINER_TESTNET"
QC_GAME_MINER_TESTNET_REWARD_MODE = "TESTNET_SIMULATED_NO_CHAIN_BLOCK"


def _get_game_mining_mode() -> str:
    raw = (
        os.getenv("QC_GAME_MINING_MODE")
        or os.getenv("TELEGRAM_GAME_MINING_MODE")
        or QC_GAME_MINING_MODE_REAL_CHAIN
    )
    mode = str(raw or "").strip().upper()

    if mode in ("TESTNET", "SIMULATED", "TESTNET_SIM", "TESTNET_SIMULATED"):
        return QC_GAME_MINING_MODE_TESTNET_SIMULATED

    if mode in ("OFF", "DISABLE", "DISABLED", "NONE"):
        return QC_GAME_MINING_MODE_DISABLED

    return QC_GAME_MINING_MODE_REAL_CHAIN


def _simulated_game_mine_result(target_qc_address: str, qc_health: Dict[str, Any]) -> Dict[str, Any]:
    return {
        "ok": True,
        "success": True,
        "simulated": True,
        "mode": QC_GAME_MINING_MODE_TESTNET_SIMULATED,
        "height": _current_chain_height_from_qc_health(qc_health),
        "reward": QC_GAME_MINER_BLOCK_REWARD,
        "block_hash": "",
        "targetAddress": target_qc_address,
        "note": "TESTNET_SIMULATED_NO_CHAIN_BLOCK: real chain /api/mine was not called.",
    }


def _fetch_qc_node_health_sync(timeout: float = 2.5) -> Dict[str, Any]:
    """
    QuantumCoin node health bilgisini sync okur.
    mine_finish sync endpoint olduğu için burada async helper kullanmıyoruz.
    """
    try:
        from telegram_game.utils.qc_api import get_api_base, node_urlopen
        base = get_api_base().rstrip("/")
        url = f"{base}/api/health"
        request = urllib.request.Request(url, headers={"X-QC-API-KEY": os.getenv("QC_API_TOKEN", "")})
        with node_urlopen(request, timeout=timeout) as resp:
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
    amount: int,
    xp_amount: int,
    meta: Optional[Dict[str, Any]] = None,
    chain_height: int = 0,
    block_reward: int = 0,
    mining_client_type: str = "",
    network_reward_mode: str = "",
) -> str:
    exact_numbers.integer(amount)
    exact_numbers.integer(block_reward)
    if amount != 0 or block_reward != 0:
        raise ValueError("game_cannot_issue_secondary_qc_reward")
    exact_numbers.integer(xp_amount)
    # Compatibility REAL columns receive zero only; new authority is INTEGER.
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
            0,
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
    conn.execute("UPDATE reward_records SET amount_atoms=0,xp_units=?,block_reward_atoms=0,numeric_version=1 WHERE reward_id=?", (xp_amount, reward_id))
    return reward_id


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

@app.get("/health", response_model=HealthResponse)
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
    # Legacy client-reported completion is not verified game activity.
    return {
        "ok": False,
        "error": "unverified_session_complete_disabled",
    }

@app.post("/api/social/claim")
def social_claim(inp: SocialClaimIn):
    return _verified_task_reward(inp, {"SOCIAL_FOLLOW", "SOCIAL_LIKE", "SOCIAL_JOIN", "SOCIAL_SUBSCRIBE"})


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
            SELECT player_id, COALESCE(display_name, '') as name, qc_atoms, sessions, last_seen
            FROM players
            ORDER BY qc_atoms DESC, sessions DESC, last_seen DESC
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
            "qc": exact_numbers.format_qc(total_qc),
            "qcAtoms": str(total_qc),
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

    payload["withdrawableQc"] = "0"
    payload["legacyQcSettlementExecutable"] = False
    payload["withdrawableRewardCount"] = int((withdrawable_row[0] if withdrawable_row else 0) or 0)
    payload["requestedWithdrawQc"] = "0"
    payload["requestedWithdrawCount"] = int((requested_row[0] if requested_row else 0) or 0)
    payload["minWithdrawAmount"] = QC_MIN_WITHDRAW_AMOUNT
    payload["latestWithdrawBatch"] = None

    if latest_batch:
        payload["latestWithdrawBatch"] = {
            "batchId": latest_batch[0],
            "legacyAmountUnverified": str(latest_batch[1]),
            "executable": False,
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
        payload["real_qc_ok"] = bool(real.get("ok")) and real.get("atomsPerQc") == "1000000"
        payload["real_qc_balance"] = real.get("balanceQc")
        payload["real_qc_spendable"] = real.get("spendableQc")
        payload["real_qc_atoms"] = real.get("balanceAtoms")
        payload["real_qc_spendable_atoms"] = real.get("spendableAtoms")
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
    return _verified_task_reward(inp, {"STORE_PURCHASE"})


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




# TG_WEBAPP_IDENTITY_V1
@app.get("/api/telegram/identity/status", response_model=IdentityStatusResponse)
def telegram_identity_status():
    token_configured = bool(_telegram_bot_token())
    auth_required = bool(_telegram_auth_required())
    return {
        "ok": True,
        "diagnosticVersion": "TG_WEBAPP_IDENTITY_V1",
        "tokenConfigured": token_configured,
        "authRequired": auth_required,
        "devFallbackAllowed": not auth_required,
        "playerIdSource": "server_verified_telegram_session",
        "safeBoundary": {
            "readOnly": True,
            "changesDb": False,
            "changesBalances": False,
            "startsMining": False,
            "changesChain": False,
        },
    }


@app.post("/api/telegram/identity/verify", response_model=IdentityVerificationResponse,
          responses={401: {"model": RouteError}, 503: {"model": RouteError}})
def telegram_identity_verify(inp: TelegramIdentityVerifyIn, response: Response):
    from .player_auth import issue_session
    result = _validate_telegram_webapp_init_data(inp.initData, max_age_sec=900)
    if not result.get("ok"):
        status = 503 if not _telegram_bot_token() else 401
        return JSONResponse(status_code=status, content={"ok": False, "error": result.get("error", "telegram_identity_invalid")})
    credentials = issue_session(_db, result["playerId"], _telegram_bot_token(), _now())
    response.headers["Cache-Control"] = "no-store"
    return {**result, **credentials,
        "safeBoundary": {"readOnly": False, "changesDb": True, "changesBalances": False,
                         "startsMining": False, "changesChain": False}}


@app.post("/api/telegram/identity/logout", response_model=RouteOK)
def telegram_identity_logout():
    from .player_auth import SESSION_HASH
    conn = _db()
    try:
        conn.execute("DELETE FROM player_auth_sessions_v1 WHERE token_hash=?", (SESSION_HASH.get(),))
        conn.commit()
    finally:
        conn.close()
    return {"ok": True}


# QC_GAME_CHAIN_BRIDGE_STATUS_V1
# Read-only bridge diagnostics for Telegram Game -> QuantumCoin node.
# This endpoint does not mine, transfer, mint, or change balances.
@app.get("/api/bridge/status")
async def bridge_status(playerId: str = "dev_user"):
    from .player_auth import ACTOR
    player_id = ACTOR.get() or (playerId or "").strip()

    try:
        from telegram_game.config import get_api_base
        qc_api_base = get_api_base().rstrip("/")
    except Exception as e:
        qc_api_base = ""
        cfg_error = str(e)
    else:
        cfg_error = ""

    with _lock:
        conn = _db()
        _touch_player(conn, player_id, None)
        _ensure_qc_address_column(conn)
        target_qc_address = _get_player_qc_address(conn, player_id)
        conn.commit()
        conn.close()

    qc_health = await asyncio.to_thread(_fetch_qc_node_health_sync)
    real_qc = {
        "ok": False,
        "linked": bool(target_qc_address),
        "error": "qc_address_not_linked" if not target_qc_address else "",
    }

    if target_qc_address:
        try:
            real_qc = await qc_fetch_address_balance(target_qc_address)
        except Exception as e:
            real_qc = {
                "ok": False,
                "linked": True,
                "address": target_qc_address,
                "error": "qc_balance_fetch_failed",
                "detail": str(e),
            }

    return {
        "ok": bool(qc_health.get("ok")),
        "mode": "QC_REAL_GAME_MINE_BRIDGE_V1",
        "gameMiningMode": _get_game_mining_mode(),
        "diagnosticVersion": "QC_GAME_CHAIN_BRIDGE_STATUS_V1",
        "playerId": player_id,
        "qcApiBase": qc_api_base,
        "configError": cfg_error,
        "qcApiHealth": qc_health,
        "linked": bool(target_qc_address),
        "targetQcAddress": target_qc_address,
        "realQc": real_qc,
        "safeBoundary": {
            "readOnly": True,
            "requiresLinkedAddress": True,
            "minesToLinkedAddressOnly": True,
            "changesBalances": False,
            "changesChain": False,
            "startsMining": False,
            "frontendUsesSameOriginGameApi": True
        }
    }

# QC_GAME_MINING_MODE_STATUS_V1
@app.get("/api/mining/mode")
def mining_mode_status():
    mode = _get_game_mining_mode()
    return {
        "ok": True,
        "diagnosticVersion": "QC_GAME_MINING_MODE_STATUS_V1",
        "miningMode": mode,
        "realChainMining": mode == QC_GAME_MINING_MODE_REAL_CHAIN,
        "clientPoWRequired": mode == QC_GAME_MINING_MODE_REAL_CHAIN,
        "serverMinesForPlayer": False,
        "secondaryGameQCPayment": False,
        "testnetSimulated": mode == QC_GAME_MINING_MODE_TESTNET_SIMULATED,
        "disabled": mode == QC_GAME_MINING_MODE_DISABLED,
        "safeBoundary": {
            "readOnly": True,
            "startsMining": False,
            "changesChain": False,
            "changesBalances": False,
            "testnetModeDoesNotCallChainMine": True,
        },
    }
# ------------------- Mining dev endpoints -------------------

def _nft_record_game_start_action_v1(conn, player_id, init_data, session_id):
    verified = _nft_verified_player_id_v1(init_data, player_id)
    if not verified or not session_id:
        return False

    _ensure_monthly_nft_game_session_tables_v1(conn)
    ts = _now()
    active = conn.execute(
        "SELECT 1 FROM monthly_nft_game_sessions "
        "WHERE session_id=? AND user_id=? AND status='STARTED' "
        "AND started_at<=? AND expires_at>=?",
        (session_id, verified, ts, ts),
    ).fetchone()
    if not active:
        return False

    conn.execute("""
        CREATE TABLE IF NOT EXISTS monthly_nft_game_actions (
            session_id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            action_type TEXT NOT NULL,
            recorded_at INTEGER NOT NULL
        )
    """)
    inserted = conn.execute(
        "INSERT OR IGNORE INTO monthly_nft_game_actions "
        "(session_id, user_id, action_type, recorded_at) "
        "VALUES(?,?,'MINE_START',?)",
        (session_id, verified, ts),
    ).rowcount
    return inserted == 1


@app.post("/api/mine/start")
def mine_start(inp: MineStartIn):
    with _lock:
        with closing(_db()) as conn:
            conn.execute("BEGIN IMMEDIATE")
            _touch_player(conn, inp.playerId, inp.displayName)
            _ensure_qc_address_column(conn)

            # QC_USER_LOCK_MINE_START_REQUIRES_LINKED_ADDRESS_V1
            # A player without a linked QC address cannot even open a mining session.
            target_qc_address = _get_player_qc_address(conn, inp.playerId)
            if not target_qc_address:
                conn.execute("DELETE FROM mining_sessions WHERE player_id=?", (inp.playerId,))
                conn.commit()
                conn.close()
                return {
                    "ok": False,
                    "error": "qc_address_not_linked",
                    "message": "Link or create a QC wallet address before starting game mining.",
                    "playerId": inp.playerId,
                }

            from .mining_sessions import open_run
            run = open_run(conn, inp.playerId, int(inp.siteId), _now())
            if not run.get("ok"):
                conn.rollback()
                conn.close()
                return run
            conn.commit()
            if inp.initData and inp.nftSessionId:
                try:
                    _nft_record_game_start_action_v1(
                        conn, inp.playerId, inp.initData, inp.nftSessionId
                    )
                    conn.commit()
                except sqlite3.Error:
                    # NFT evidence fails closed without undoing the accepted game start.
                    conn.rollback()
            payload = _wallet_payload(conn, inp.playerId)
            conn.close()

    return {**run, "started": True, "siteId": int(inp.siteId), "wallet": payload, "miningMode":_get_game_mining_mode()}


@app.post("/api/mine/finish")
def mine_finish(inp: MineFinishIn):
    from .mining_sessions import reserve_run
    mode = _get_game_mining_mode()
    if mode == QC_GAME_MINING_MODE_DISABLED:
        return {"ok": False, "error": "game_mining_disabled"}
    real = mode == QC_GAME_MINING_MODE_REAL_CHAIN
    run = reserve_run(_db, inp.playerId, int(inp.siteId), inp.sessionId, _now(), require_pow=real)
    if not run.get("ok"):
        return run
    proof = run.get("powReceipt") or {}
    duration = run["durationSec"]
    xp = _compute_rewards(duration)["xp"]
    with closing(_db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        exact_numbers.add_xp(conn, inp.playerId, xp)
        reward_id = _reward_record_insert(conn,player_id=inp.playerId,source="MINING_SESSION",zone_id=f"LUNA-{int(inp.siteId):02d}",
            site_id=int(inp.siteId),amount=0,xp_amount=xp,
            meta={"durationSec":duration,"startedAt":run["startedAt"],"miningMode":mode,"clientPoW":proof,"rewardPaidByChain":real},
            chain_height=int(proof.get("height") or 0),block_reward=0,
            mining_client_type="GAME_CLIENT_POW" if real else "GAME_MINER_TESTNET",network_reward_mode="COINBASE_ALREADY_ON_CHAIN" if real else "TESTNET_SIMULATED_NO_CHAIN_BLOCK")
        status = "CHAIN_REWARD_ALREADY_PAID" if real else "TESTNET_SIMULATED"
        conn.execute("UPDATE reward_records SET settlement_status=?,settlement_note=? WHERE reward_id=?",(status,"No secondary QC or TGWT payment; real block rewards belong to the linked chain wallet.",reward_id))
        conn.execute("DELETE FROM mining_sessions WHERE player_id=?",(inp.playerId,))
        conn.execute("UPDATE game_mining_runs_v1 SET status='COMPLETED',reward_id=? WHERE session_id=? AND status='PROCESSING'",(reward_id,inp.sessionId))
        payload = _wallet_payload(conn,inp.playerId)
    return {"ok":True,"durationSec":duration,"qcEarned":0,"xpEarned":xp,"rewardId":reward_id,"settlementStatus":status,
            "blockHash":proof.get("hash",""),"chainHeight":proof.get("height",0),"miningMode":mode,"wallet":payload,
            "rewardPaidByChain":real,"tgwtReward":0}


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
            "amount": exact_numbers.format_qc(int(r[2] or 0)),
            "amount_u": int(r[2] or 0),
            "meta": json.loads(r[3] or "{}"),
            "createdAt": int(r[4] or 0),
        })

    return {
        "ok": True,
        "userId": user_id,
        "pool": {"total": exact_numbers.format_qc(pool["total_u"]), "distributed": exact_numbers.format_qc(pool["distributed_u"])},
        "rev": st["rev"],
        "balances": st,
        "events": events,
        "ts": _now(),
    }


@app.post("/api/v1/tgwt/earn")
def tgwt_earn(inp: TGWTEarnIn):
    return JSONResponse(status_code=410, content={"ok": False, "error": "legacy_unproven_reward_retired"})


@app.post("/api/v1/tgwt/verify")
def tgwt_verify(inp: TGWTVerifyIn):
    return JSONResponse(status_code=410, content={"ok": False, "error": "legacy_unproven_reward_retired"})


@app.post("/api/v1/withdraw/request")
def withdraw_request(inp: WithdrawRequestIn):
    from . import withdraw_ledger
    from .player_auth import ACTOR
    actor = ACTOR.get()
    if not actor or actor != inp.userId:
        return {"ok": False, "error": "telegram_session_required"}
    if inp.asset.upper() != "TGWT":
        return {"ok": False, "error": "unsupported_asset"}
    if not _rl_hit(actor, "withdraw_request", limit=10, window_sec=60):
        return {"ok": False, "error": "rate_limited"}
    return withdraw_ledger.reserve(_db,actor,inp.toAddress,inp.amount,inp.nonce,_now())


class WithdrawTransitionIn(BaseModel):
    requestId: str
    model_config = {"extra":"forbid"}


@app.post("/api/v1/withdraw/cancel")
def withdraw_cancel(inp: WithdrawTransitionIn):
    from . import withdraw_ledger
    from .player_auth import ACTOR
    return withdraw_ledger.release(_db,inp.requestId,ACTOR.get(),_now())


@app.post("/api/admin/tgwt/withdraw/reject")
def withdraw_reject(inp: WithdrawTransitionIn):
    from . import withdraw_ledger
    return withdraw_ledger.release(_db,inp.requestId,None,_now(),operator=True,reject=True)


@app.post("/api/admin/tgwt/withdraw/confirm")
def withdraw_confirm_unavailable(inp: WithdrawTransitionIn):
    # TODO(mainnet-final): User deferred real BSC payment/receipt verification
    # to the final mainnet phase (2026-11-04). Keep this endpoint disabled until then.
    # An operator click or body txHash is not a BSC receipt. No payout adapter
    # is enabled before a deployed, request-bound user-paid claim is verified.
    return JSONResponse(status_code=503,content={"ok":False,"error":"verified_bsc_claim_adapter_not_deployed"})


@app.get("/api/v1/withdraw/status", response_model=WithdrawStatusResponse | RouteError)
def withdraw_status(id: str):
    from .player_auth import ACTOR
    actor = ACTOR.get()
    if not actor:
        return {"ok": False, "error": "telegram_session_required"}
    with _lock:
        conn = _db()
        cur = conn.execute(
            """
            SELECT id, user_id, asset, amount_u, to_address, status, tx_hash, created_at, updated_at
            FROM withdraw_requests WHERE id=? AND user_id=?
            """,
            (id, actor),
        )
        row = cur.fetchone()
        conn.close()
    if not row:
        return {"ok": False, "error": "not_found"}
    return {"ok": True, "request": {
        "id": row[0],
        "userId": row[1],
        "asset": row[2],
        "amount": exact_numbers.format_qc(int(row[3] or 0)),
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
            "amount": exact_numbers.format_qc(int(r[2] or 0)),
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
                   chain_height, block_reward, mining_client_type, network_reward_mode,
                   amount_atoms,xp_units,block_reward_atoms,numeric_version
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
                "amount": exact_numbers.format_qc(r[16]),
                "amountAtoms": str(r[16]),
                "numericVersion": r[19],
                "legacyNumericReviewRequired": r[19] == 0,
                "legacyAmountUnverified": str(r[5]) if r[19] == 0 else None,
                "xpAmount": int(r[17]),
                "status": r[7],
                "settlementStatus": r[8],
                "txHash": r[9],
                "createdAt": int(r[10] or 0),
                "confirmedAt": int(r[11] or 0),
                "chainHeight": int(r[12] or 0),
                "blockReward": exact_numbers.format_qc(r[18]),
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

    return {"ok": False, "txid": txid_clean, "confirmed": False,
            "status": "UNKNOWN", "error": "node_transaction_receipt_required"}


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
    nonce: str = ""
    publicKey: str = ""
    signature: str = ""
    model_config = {"extra":"forbid"}


# QC_WALLET_SEND_QC_V1
class WalletSendQCIn(BaseModel):
    playerId: str
    initData: str = ""
    transaction: str
    model_config = {"extra": "forbid"}



@app.post("/api/wallet/link-qc-address", response_model=WalletLinkResponse | RouteError)
async def link_qc_address(inp: LinkQCAddressIn):
    from . import wallet_link
    from .player_auth import ACTOR
    actor = ACTOR.get()
    if not actor or actor != inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    if not inp.nonce or not inp.publicKey or not inp.signature:
        return {"ok":False,"error":"wallet_ownership_proof_required"}
    with closing(_db()) as conn, conn:
        _touch_player(conn,actor,None)
        _ensure_qc_address_column(conn)
    return wallet_link.consume(_db,actor,inp.address,inp.nonce,inp.publicKey,inp.signature,_now())


@app.post("/api/wallet/link-challenge", response_model=WalletChallengeResponse | RouteError)
def wallet_link_challenge(inp: LinkQCAddressIn):
    from . import wallet_link
    from .player_auth import ACTOR
    actor = ACTOR.get()
    if not actor or actor != inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    return wallet_link.issue(_db,actor,inp.address,_now())


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
    player_id = _nft_verified_player_id_v1(inp.initData, inp.playerId)
    if not player_id:
        return {"ok": False, "error": "telegram_identity_required"}
    with _lock:
        conn = _db()
        try:
            qc_address = _get_player_qc_address(conn, player_id)
        finally:
            conn.close()
    try:
        if len(inp.transaction) > 1048576:
            return {"ok": False, "error": "signed_transaction_size_limit"}
        envelope = json.loads(inp.transaction)
        transaction = envelope["transaction"]
        if set(envelope) != {"transaction"} or not isinstance(transaction, dict):
            return {"ok": False, "error": "signed_transaction_required"}
    except (ValueError, KeyError, TypeError):
        return {"ok": False, "error": "signed_transaction_required"}
    if not qc_address or transaction.get("Sender") != qc_address:
        return {"ok": False, "error": "wallet_sender_mismatch"}
    from telegram_game.utils.qc_api import fetch_submit_signed_tx
    result = await fetch_submit_signed_tx(transaction)
    return {"ok": bool(result.get("success")), "txid": result.get("txid", ""), "status": "PENDING_CHAIN" if result.get("success") else "REJECTED", "qcSendResult": result}


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
    return JSONResponse(status_code=410, content={"ok":False,"error":"legacy_float_qc_settlement_retired","note":"Historical records retained for audit; QC mining is paid directly by the node."})




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
    return JSONResponse(status_code=410, content={"ok":False,"error":"legacy_float_qc_settlement_retired","note":"Historical records retained for audit; QC mining is paid directly by the node."})




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
    return JSONResponse(status_code=410, content={"ok":False,"error":"legacy_float_qc_settlement_retired","note":"Historical records retained for audit; QC mining is paid directly by the node."})


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
    return JSONResponse(status_code=410, content={"ok":False,"error":"legacy_float_qc_settlement_retired","note":"Historical records retained for audit; QC mining is paid directly by the node."})


@app.post("/api/withdraw/mock-confirm")
def qc_withdraw_mock_confirm(inp: QCWithdrawBatchIn):
    return JSONResponse(status_code=410, content={"ok":False,"error":"legacy_float_qc_settlement_retired","note":"Historical records retained for audit; QC mining is paid directly by the node."})






# QC_REAL_PAYOUT_TXSTATUS_READY_V1
async def qc_check_tx_status_endpoint_ready():
    from telegram_game.utils.qc_api import get_api_base, node_urlopen
    base = get_api_base().rstrip("/")
    url = base + "/api/tx/status?id=__precheck__"

    try:
        import urllib.request
        import json as _json

        def _fetch():
            req = urllib.request.Request(
                url,
                headers={"Accept": "application/json", "X-QC-API-KEY": os.getenv("QC_API_TOKEN", "")},
                method="GET",
            )
            with node_urlopen(req, timeout=5) as resp:
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

        # QC_REAL_PAYOUT_TXSTATUS_READY_V2
        # Core /api/tx/status endpoint readiness means:
        # - JSON parsed
        # - ok=true
        # - id/txid echoes the requested precheck id
        # - mempool/block status fields exist
        # confirmedHeight is accepted; a real tx is not required for endpoint readiness.
        ready = (
            data.get("ok") is True
            and (data.get("id") == "__precheck__" or data.get("txid") == "__precheck__")
            and "inBlock" in data
            and "inMempool" in data
            and ("confirmedHeight" in data or "height" in data)
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
    return JSONResponse(status_code=410, content={"ok":False,"error":"legacy_float_qc_settlement_retired","note":"Historical records retained for audit; QC mining is paid directly by the node."})


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
    return JSONResponse(status_code=410, content={"ok": False, "error": "unsafe_legacy_reset_retired"})


# WATCH_REWARDED_AD_POLICY_V1
# WATCH_CLAIM_TGWT_V1
# DEV TEST ONLY: final Watch & Earn must use verified Google Rewarded Ads completion before TGWT reward.
class WatchClaimIn(BaseModel):
    playerId: str
    videoId: str = "watch_demo"
    nonce: str
    campaignId: str = ""
    proof: Optional[dict] = None


@app.post("/api/watch/claim")
def watch_claim(inp: WatchClaimIn):
    return _verified_task_reward(inp, {"WATCH_COMPLETE"})


# TG_MONTHLY_NFT_CANDIDATE_POOL_V1
# Advisory-only monthly rare NFT candidate pool.
# This does not mint NFT, does not change QC/TGWT balances, and does not write chain metadata.
NFT_MONTHLY_POLICY_VERSION = "TG_MONTHLY_NFT_CANDIDATE_POOL_V1"
NFT_DEFAULT_TYPE = "rare"


class MonthlyNFTSessionStartIn(BaseModel):
    playerId: str
    initData: str


def _ensure_monthly_nft_game_session_tables_v1(conn):
    conn.execute("""
        CREATE TABLE IF NOT EXISTS monthly_nft_game_sessions (
            session_id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            started_at INTEGER NOT NULL,
            expires_at INTEGER NOT NULL,
            completed_at INTEGER,
            status TEXT NOT NULL CHECK(status IN ('STARTED', 'COMPLETED', 'EXPIRED'))
        )
    """)
    conn.execute(
        "CREATE UNIQUE INDEX IF NOT EXISTS idx_monthly_nft_one_active_session_v1 "
        "ON monthly_nft_game_sessions(user_id) WHERE status='STARTED'"
    )


@app.post("/api/v1/nft/monthly/session/start")
def nft_monthly_session_start(inp: MonthlyNFTSessionStartIn):
    user_id = (inp.playerId or "").strip()
    verified = _nft_verified_player_id_v1(inp.initData, user_id)
    if not verified:
        return {"ok": False, "error": "telegram_identity_required"}

    ts = _now()
    session_id = "nfts_" + secrets.token_hex(24)
    with _lock:
        conn = _db()
        _ensure_monthly_nft_game_session_tables_v1(conn)
        conn.execute(
            "UPDATE monthly_nft_game_sessions SET status='EXPIRED' "
            "WHERE user_id=? AND status='STARTED' AND expires_at<=?",
            (verified, ts),
        )
        active = conn.execute(
            "SELECT session_id FROM monthly_nft_game_sessions "
            "WHERE user_id=? AND status='STARTED' LIMIT 1",
            (verified,),
        ).fetchone()
        if active:
            conn.commit()
            conn.close()
            return {
                "ok": True,
                "sessionId": active[0],
                "status": "STARTED",
                "reused": True,
            }

        try:
            conn.execute(
                "INSERT INTO monthly_nft_game_sessions"
                "(session_id, user_id, started_at, expires_at, completed_at, status) "
                "VALUES(?,?,?,?,NULL,'STARTED')",
                (session_id, verified, ts, ts + 3600),
            )
            conn.commit()
        except sqlite3.IntegrityError:
            conn.rollback()
            conn.close()
            return {"ok": False, "error": "session_already_active"}
        conn.close()

    return {"ok": True, "sessionId": session_id, "status": "STARTED"}


class MonthlyNFTSessionCompleteIn(BaseModel):
    playerId: str
    initData: str
    sessionId: str


def _ensure_monthly_nft_verified_game_sessions_v1(conn):
    conn.execute("""
        CREATE TABLE IF NOT EXISTS monthly_nft_verified_game_sessions (
            session_id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            period TEXT NOT NULL,
            completed_at INTEGER NOT NULL,
            activity_verified INTEGER NOT NULL DEFAULT 0
                CHECK(activity_verified IN (0, 1))
        )
    """)
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_monthly_nft_verified_user_period_v1 "
        "ON monthly_nft_verified_game_sessions(user_id, period, completed_at)"
    )


@app.post("/api/v1/nft/monthly/session/complete")
def nft_monthly_session_complete(inp: MonthlyNFTSessionCompleteIn):
    user_id = (inp.playerId or "").strip()
    session_id = (inp.sessionId or "").strip()
    verified = _nft_verified_player_id_v1(inp.initData, user_id)
    if not verified or not session_id:
        return {"ok": False, "error": "telegram_identity_or_session_required"}

    ts = _now()
    with _lock:
        conn = _db()
        try:
            _ensure_monthly_nft_game_session_tables_v1(conn)
            row = conn.execute(
                "SELECT user_id, started_at, expires_at, status "
                "FROM monthly_nft_game_sessions WHERE session_id=?",
                (session_id,),
            ).fetchone()
            if not row or row[0] != verified or row[3] != "STARTED":
                conn.close()
                return {"ok": False, "error": "session_not_active"}

            started_at, expires_at = int(row[1]), int(row[2])
            if ts < started_at + 60:
                conn.close()
                return {"ok": False, "error": "session_too_short"}
            if ts > expires_at:
                conn.close()
                return {"ok": False, "error": "session_expired"}

            has_actions = conn.execute(
                "SELECT 1 FROM sqlite_master WHERE type='table' "
                "AND name='monthly_nft_game_actions'"
            ).fetchone()
            action = None
            if has_actions:
                action = conn.execute(
                    "SELECT 1 FROM monthly_nft_game_actions "
                    "WHERE session_id=? AND user_id=? AND action_type='MINE_START' "
                    "AND recorded_at>=? AND recorded_at<=?",
                    (session_id, verified, started_at, ts),
                ).fetchone()
            if not action:
                conn.close()
                return {"ok": False, "error": "verified_game_action_required"}

            changed = conn.execute(
                "UPDATE monthly_nft_game_sessions "
                "SET status='COMPLETED', completed_at=? "
                "WHERE session_id=? AND user_id=? AND status='STARTED'",
                (ts, session_id, verified),
            ).rowcount
            if changed != 1:
                conn.rollback()
                conn.close()
                return {"ok": False, "error": "session_already_completed"}

            _ensure_monthly_nft_verified_game_sessions_v1(conn)
            period = _nft_month_key(ts)
            conn.execute(
                "INSERT INTO monthly_nft_verified_game_sessions "
                "(session_id, user_id, period, completed_at, activity_verified) "
                "VALUES(?,?,?,?,1)",
                (session_id, verified, period, ts),
            )
            conn.commit()
            conn.close()
        except sqlite3.Error:
            conn.rollback()
            conn.close()
            return {"ok": False, "error": "session_store_unavailable"}

    return {
        "ok": True,
        "sessionId": session_id,
        "period": period,
        "status": "COMPLETED",
        "activityVerified": True,
    }


class MonthlyNFTCandidateIn(BaseModel):
    userId: str
    source: str = "manual"
    activity: Optional[Dict[str, Any]] = None
    period: Optional[str] = None


class MonthlyNFTAdminReviewIn(BaseModel):
    candidateId: str
    action: str
    note: Optional[str] = None
    reviewer: Optional[str] = "admin"


def _nft_month_key(ts: Optional[int] = None) -> str:
    import datetime
    if ts is None:
        ts = _now()
    dt = datetime.datetime.utcfromtimestamp(int(ts))
    return f"{dt.year:04d}-{dt.month:02d}"


def _nft_verified_mainnet_activation_at_v1():
    # Compatibility name: this is Telegram Game activation, not a mainnet date.
    from telegram_game.utils.nft_program_bridge_v1 import fetch_verified_nft_game_program_v1

    program = fetch_verified_nft_game_program_v1()
    return program["gameActivatedAt"] if program is not None else None


def _monthly_nft_award_periods_from_activation_v1(activation_at):
    from telegram_game.utils.nft_program_bridge_v1 import monthly_nft_periods_from_game_activation_v1

    return monthly_nft_periods_from_game_activation_v1(activation_at)


def _ensure_monthly_nft_tables(conn):
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS monthly_nft_candidates (
            candidate_id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            period TEXT NOT NULL,
            nft_type TEXT NOT NULL DEFAULT 'rare',
            source TEXT NOT NULL DEFAULT '',
            activity_json TEXT NOT NULL DEFAULT '{}',
            score REAL NOT NULL DEFAULT 0,
            ai_review_status TEXT NOT NULL DEFAULT 'PENDING_AI_REVIEW',
            ai_risk_level TEXT NOT NULL DEFAULT 'UNKNOWN',
            admin_status TEXT NOT NULL DEFAULT 'PENDING_ADMIN_REVIEW',
            admin_note TEXT NOT NULL DEFAULT '',
            winner_status TEXT NOT NULL DEFAULT 'NOT_SELECTED',
            minted_status TEXT NOT NULL DEFAULT 'NOT_MINTED',
            policy_version TEXT NOT NULL DEFAULT 'TG_MONTHLY_NFT_CANDIDATE_POOL_V1',
            created_at INTEGER NOT NULL DEFAULT 0,
            updated_at INTEGER NOT NULL DEFAULT 0,
            UNIQUE(user_id, period)
        )
        """
    )
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_monthly_nft_period_status ON monthly_nft_candidates(period, admin_status, winner_status)"
    )
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_monthly_nft_user_period ON monthly_nft_candidates(user_id, period)"
    )


def _monthly_nft_candidate_row_to_dict(row):
    if not row:
        return None
    activity = {}
    try:
        activity = json.loads(row[5] or "{}")
    except Exception:
        activity = {}
    return {
        "candidateId": row[0],
        "userId": row[1],
        "period": row[2],
        "nftType": row[3],
        "source": row[4],
        "activity": activity,
        "score": float(row[6] or 0),
        "aiReviewStatus": row[7],
        "aiRiskLevel": row[8],
        "adminStatus": row[9],
        "adminNote": row[10],
        "winnerStatus": row[11],
        "mintedStatus": row[12],
        "policyVersion": row[13],
        "createdAt": int(row[14] or 0),
        "updatedAt": int(row[15] or 0),
    }



# TG_MONTHLY_NFT_CANDIDATE_PLAYER_TOUCH_FIX_V1
def _nft_safe_ensure_player(conn, user_id: str):
    user_id = (user_id or "").strip()
    if not user_id:
        return None

    # Existing project helper. Keeps last_seen fresh.
    try:
        return _touch_player(conn, user_id, "")
    except TypeError:
        try:
            return _touch_player(conn, user_id)
        except Exception:
            pass
    except Exception:
        pass

    # Safe fallback: create minimal player row without changing balances.
    ts = _now()
    conn.execute(
        """
        INSERT OR IGNORE INTO players(player_id, display_name, last_seen)
        VALUES(?,?,?)
        """,
        (user_id, user_id, ts),
    )
    conn.execute("UPDATE players SET last_seen=? WHERE player_id=?", (ts, user_id))
    return None


def _monthly_nft_verified_session_count_v1(conn, user_id: str, period: str) -> int:
    """Read only server-recorded evidence; no client activity or player counter."""
    import datetime

    try:
        month_start = datetime.datetime.strptime(period, "%Y-%m").replace(
            tzinfo=datetime.timezone.utc
        )
        if month_start.strftime("%Y-%m") != period:
            return 0
        if month_start.month == 12:
            month_end = month_start.replace(year=month_start.year + 1, month=1)
        else:
            month_end = month_start.replace(month=month_start.month + 1)

        exists = conn.execute(
            "SELECT 1 FROM sqlite_master WHERE type='table' "
            "AND name='monthly_nft_verified_game_sessions'"
        ).fetchone()
        if not exists:
            return 0

        row = conn.execute(
            "SELECT COUNT(*) FROM monthly_nft_verified_game_sessions "
            "WHERE user_id=? AND period=? AND activity_verified=1 "
            "AND completed_at>=? AND completed_at<? AND completed_at<=?",
            (user_id, period, int(month_start.timestamp()), int(month_end.timestamp()), _now()),
        ).fetchone()
        return int(row[0]) if row else 0
    except (ValueError, OverflowError, sqlite3.DatabaseError):
        return 0

def _monthly_nft_candidate_select_sql():
    return """
        SELECT candidate_id, user_id, period, nft_type, source, activity_json,
               score, ai_review_status, ai_risk_level, admin_status, admin_note,
               winner_status, minted_status, policy_version, created_at, updated_at
        FROM monthly_nft_candidates
    """


@app.post("/api/v1/nft/monthly/candidate")
def nft_monthly_candidate_create(inp: MonthlyNFTCandidateIn):
    user_id = (inp.userId or "").strip()
    if not user_id:
        return {"ok": False, "error": "missing_user"}

    period = (inp.period or "").strip() or _nft_month_key()
    if period not in _monthly_nft_award_periods_from_activation_v1(
        _nft_verified_mainnet_activation_at_v1()
    ):
        return {
            "ok": False,
            "error": "monthly_nft_program_not_active_or_period_outside",
            "period": period,
        }
    source = (inp.source or "manual").strip()[:64]
    activity = inp.activity or {}
    ts = _now()
    candidate_id = "nftc_" + secrets.token_hex(16)

    with _lock:
        conn = _db()
        _ensure_monthly_nft_tables(conn)
        verified_sessions = _monthly_nft_verified_session_count_v1(
            conn, user_id, period
        )
        if verified_sessions < 1:
            conn.close()
            return {
                "ok": False,
                "error": "verified_game_session_required",
                "period": period,
            }
        source = "server_verified_game_session"
        activity = {"verifiedCompletedGameSessions": verified_sessions}
        _nft_safe_ensure_player(conn, user_id)

        existing = conn.execute(
            _monthly_nft_candidate_select_sql() + " WHERE user_id=? AND period=? LIMIT 1",
            (user_id, period),
        ).fetchone()

        if existing:
            conn.close()
            return {
                "ok": True,
                "alreadyExists": True,
                "candidate": _monthly_nft_candidate_row_to_dict(existing),
                "policy": {
                    "period": "monthly",
                    "maxAwardsPerMonth": 1,
                    "requiresAIReview": True,
                    "requiresAdmin": True,
                    "minting": "disabled_until_admin_approved_policy_path",
                },
            }

        conn.execute(
            """
            INSERT INTO monthly_nft_candidates(
                candidate_id, user_id, period, nft_type, source, activity_json,
                score, ai_review_status, ai_risk_level, admin_status, admin_note,
                winner_status, minted_status, policy_version, created_at, updated_at
            )
            VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
            """,
            (
                candidate_id,
                user_id,
                period,
                NFT_DEFAULT_TYPE,
                source,
                json.dumps(activity, ensure_ascii=False),
                0.0,
                "PENDING_AI_REVIEW",
                "UNKNOWN",
                "PENDING_ADMIN_REVIEW",
                "",
                "NOT_SELECTED",
                "NOT_MINTED",
                NFT_MONTHLY_POLICY_VERSION,
                ts,
                ts,
            ),
        )
        row = conn.execute(
            _monthly_nft_candidate_select_sql() + " WHERE candidate_id=?",
            (candidate_id,),
        ).fetchone()
        conn.commit()
        conn.close()

    return {
        "ok": True,
        "candidate": _monthly_nft_candidate_row_to_dict(row),
        "policy": {
            "period": "monthly",
            "maxAwardsPerMonth": 1,
            "requiresAIReview": True,
            "requiresAdmin": True,
            "minting": "disabled_until_admin_approved_policy_path",
        },
    }


@app.get("/api/v1/nft/monthly/status")
def nft_monthly_candidate_status(userId: str, period: Optional[str] = None):
    user_id = (userId or "").strip()
    if not user_id:
        return {"ok": False, "error": "missing_user"}

    pkey = (period or "").strip() or _nft_month_key()

    with _lock:
        conn = _db()
        _ensure_monthly_nft_tables(conn)
        row = conn.execute(
            _monthly_nft_candidate_select_sql() + " WHERE user_id=? AND period=? LIMIT 1",
            (user_id, pkey),
        ).fetchone()
        conn.close()

    return {
        "ok": True,
        "userId": user_id,
        "period": pkey,
        "candidate": _monthly_nft_candidate_row_to_dict(row),
        "policyVersion": NFT_MONTHLY_POLICY_VERSION,
    }


@app.get("/api/admin/nft/monthly/candidates")
def admin_nft_monthly_candidates(period: Optional[str] = None, limit: int = 100):
    pkey = (period or "").strip() or _nft_month_key()
    limit = max(1, min(int(limit or 100), 500))

    with _lock:
        conn = _db()
        _ensure_monthly_nft_tables(conn)
        rows = conn.execute(
            _monthly_nft_candidate_select_sql()
            + " WHERE period=? ORDER BY created_at DESC LIMIT ?",
            (pkey, limit),
        ).fetchall()
        conn.close()

    return {
        "ok": True,
        "period": pkey,
        "count": len(rows),
        "candidates": [_monthly_nft_candidate_row_to_dict(r) for r in rows],
        "policy": {
            "maxAwardsPerMonth": 1,
            "requiresAIReview": True,
            "requiresAdmin": True,
            "minting": "not_performed_by_this_endpoint",
        },
    }


# TG_MONTHLY_NFT_ADMIN_APPROVAL_V1
# Admin approval only: APPROVE / REJECT / HOLD.
# Safe boundary: no NFT mint, no winner selection, no chain metadata write, no QC/TGWT balance change.
@app.post("/api/admin/nft/monthly/review")
def admin_nft_monthly_review(inp: MonthlyNFTAdminReviewIn):
    cid = (inp.candidateId or "").strip()
    action = (inp.action or "").strip().upper()
    note = (inp.note or "").strip()
    reviewer = (inp.reviewer or "admin").strip()

    if not cid:
        return {"ok": False, "error": "missing_candidate_id"}

    allowed = {
        "APPROVE": "APPROVED_BY_ADMIN",
        "REJECT": "REJECTED_BY_ADMIN",
        "NEEDS_AI_REVIEW": "PENDING_AI_REVIEW",
        "HOLD": "ON_HOLD",
    }

    if action not in allowed:
        return {"ok": False, "error": "invalid_action", "allowed": list(allowed.keys())}

    ts = _now()
    admin_status = allowed[action]
    admin_note = f"{reviewer}: {note}"[:512]

    with _lock:
        conn = _db()
        _ensure_monthly_nft_tables(conn)
        row = conn.execute(
            _monthly_nft_candidate_select_sql() + " WHERE candidate_id=? LIMIT 1",
            (cid,),
        ).fetchone()
        if not row:
            conn.close()
            return {"ok": False, "error": "candidate_not_found"}

        winner_status = row[11]
        minted_status = row[12]
        if winner_status == "SELECTED_MONTHLY_WINNER" or minted_status != "NOT_MINTED":
            conn.close()
            return {"ok": False, "error": "candidate_selection_locked"}

        # Safe boundary: approval does not select winner and does not mint.
        if action == "APPROVE":
            winner_status = "APPROVED_CANDIDATE_NOT_WINNER"
            minted_status = "NOT_MINTED"
        elif action == "REJECT":
            winner_status = "NOT_SELECTED"
            minted_status = "NOT_MINTED"

        changed = conn.execute(
            """
            UPDATE monthly_nft_candidates
            SET admin_status=?, admin_note=?, winner_status=?, minted_status=?, updated_at=?
            WHERE candidate_id=? AND winner_status!='SELECTED_MONTHLY_WINNER'
              AND minted_status='NOT_MINTED'
            """,
            (admin_status, admin_note, winner_status, minted_status, ts, cid),
        ).rowcount
        if changed != 1:
            conn.rollback()
            conn.close()
            return {"ok": False, "error": "candidate_selection_locked"}
        updated = conn.execute(
            _monthly_nft_candidate_select_sql() + " WHERE candidate_id=? LIMIT 1",
            (cid,),
        ).fetchone()
        conn.commit()
        conn.close()

    return {
        "ok": True,
        "candidate": _monthly_nft_candidate_row_to_dict(updated),
        "safeBoundary": {
            "minted": False,
            "winnerSelected": False,
            "balanceChanged": False,
            "chainMetadataWritten": False,
            "qcSupplyChanged": False,
            "tgwtBalanceChanged": False,
            "note": "Admin approval only. Winner selection and mint path remain separate.",
        },
    }


# TG_MONTHLY_NFT_AI_REVIEW_V1
# Local advisory-only AI review for monthly rare NFT candidates.
# This does not select winners, does not mint NFT, does not change balances, and does not write chain metadata.
NFT_AI_REVIEW_VERSION = "TG_MONTHLY_NFT_AI_VERIFIED_ACTIVITY_V1"


class MonthlyNFTAIReviewIn(BaseModel):
    candidateId: Optional[str] = None
    userId: Optional[str] = None
    period: Optional[str] = None
    reviewer: Optional[str] = "local_ai"


def _monthly_nft_find_candidate(conn, candidate_id: str = "", user_id: str = "", period: str = ""):
    _ensure_monthly_nft_tables(conn)

    candidate_id = (candidate_id or "").strip()
    user_id = (user_id or "").strip()
    period = (period or "").strip() or _nft_month_key()

    if candidate_id:
        return conn.execute(
            _monthly_nft_candidate_select_sql() + " WHERE candidate_id=? LIMIT 1",
            (candidate_id,),
        ).fetchone()

    if user_id:
        return conn.execute(
            _monthly_nft_candidate_select_sql() + " WHERE user_id=? AND period=? LIMIT 1",
            (user_id, period),
        ).fetchone()

    return None


def _monthly_nft_local_ai_score(candidate: dict, verified_sessions: int = 0) -> dict:
    """Local advisory rule, not an external AI or a fraud/ownership proof.

    Only the caller's fresh server DB count is evidence. Candidate JSON,
    old mining counters and realBlockFound never contribute to this result.
    One verified completion implements the current active-user minimum;
    completing more sessions does not increase draw weight or this score.
    """
    eligible = (
        type(verified_sessions) is int and verified_sessions > 0
        and candidate.get("nftType") == "rare"
        and isinstance(candidate.get("userId"), str)
        and bool(candidate["userId"])
        and candidate["userId"] == candidate["userId"].strip()
    )
    return {
        "score": 0.65 if eligible else 0.0,
        "aiReviewStatus": "AI_REVIEW_PASSED" if eligible else "AI_REVIEW_FLAGGED",
        "aiRiskLevel": "LOW" if eligible else "HIGH",
        "reason": (
            "Server-verified monthly game activity meets the active-user minimum; manual admin review is required."
            if eligible else "No valid server-verified monthly activity for this candidate."
        ),
        "reviewVersion": NFT_AI_REVIEW_VERSION,
        "verifiedSessionCount": verified_sessions,
        "evidenceSource": "monthly_nft_verified_game_sessions",
    }


@app.post("/api/admin/nft/monthly/ai-review")
def admin_nft_monthly_ai_review(inp: MonthlyNFTAIReviewIn):
    candidate_id = (inp.candidateId or "").strip()
    user_id = (inp.userId or "").strip()
    period = (inp.period or "").strip() or _nft_month_key()
    reviewer = (inp.reviewer or "local_ai").strip()

    if not candidate_id and not user_id:
        return {"ok": False, "error": "candidateId_or_userId_required"}
    periods = _monthly_nft_award_periods_from_activation_v1(
        _nft_verified_mainnet_activation_at_v1()
    )
    if not periods:
        return {"ok": False, "error": "monthly_nft_program_not_active_or_period_outside"}

    with _lock:
        conn = None
        try:
            conn = _db()
            # A selection in another process must not race this review.
            conn.execute("BEGIN IMMEDIATE")
            row = _monthly_nft_find_candidate(conn, candidate_id, user_id, period)
            if not row:
                return {"ok": False, "error": "candidate_not_found"}
            cand = _monthly_nft_candidate_row_to_dict(row)
            if cand["winnerStatus"] == "SELECTED_MONTHLY_WINNER" or cand["mintedStatus"] != "NOT_MINTED":
                return {"ok": False, "error": "candidate_selection_locked"}
            if cand["period"] not in periods:
                return {"ok": False, "error": "monthly_nft_program_not_active_or_period_outside"}

            count = _monthly_nft_verified_session_count_v1(conn, cand["userId"], cand["period"])
            review = _monthly_nft_local_ai_score(cand, count)
            ts = _now()
            existing_note = (cand.get("adminNote") or "").strip()
            ai_note = f"{reviewer}: {review['reviewVersion']} {review['aiReviewStatus']} risk={review['aiRiskLevel']} score={review['score']} - {review['reason']}"
            admin_note = ((existing_note + " | ") if existing_note else "") + ai_note
            changed = conn.execute(
                """
                UPDATE monthly_nft_candidates
                SET score=?, ai_review_status=?, ai_risk_level=?, admin_note=?,
                    activity_json=?, source='server_verified_game_session',
                    admin_status='PENDING_ADMIN_REVIEW', winner_status='NOT_SELECTED', updated_at=?
                WHERE candidate_id=? AND winner_status!='SELECTED_MONTHLY_WINNER'
                  AND minted_status='NOT_MINTED'
                """,
                (float(review["score"]), review["aiReviewStatus"], review["aiRiskLevel"], admin_note[:512],
                 json.dumps({"verifiedCompletedGameSessions": count}), ts, cand["candidateId"]),
            ).rowcount
            if changed != 1:
                return {"ok": False, "error": "candidate_selection_locked"}
            updated = conn.execute(
                _monthly_nft_candidate_select_sql() + " WHERE candidate_id=? LIMIT 1",
                (cand["candidateId"],),
            ).fetchone()
            conn.commit()
        except sqlite3.Error:
            return {"ok": False, "error": "monthly_ai_review_store_unavailable"}
        finally:
            if conn is not None:
                if conn.in_transaction:
                    conn.rollback()
                conn.close()

    return {
        "ok": True,
        "candidate": _monthly_nft_candidate_row_to_dict(updated),
        "aiReview": review,
        "safeBoundary": {
            "minted": False, "winnerSelected": False, "balanceChanged": False,
            "chainMetadataWritten": False, "qcSupplyChanged": False, "tgwtBalanceChanged": False,
            "note": "Local advisory activity check only; fresh admin approval and separate selection are required.",
        },
    }




# TG_MONTHLY_NFT_WINNER_SELECTION_V1
# Retired manual selection: validation errors or disabled, never a winner write.
# Safe boundary: no NFT mint, no chain metadata write, no QC/TGWT balance change.
class MonthlyNFTWinnerSelectIn(BaseModel):
    candidateId: str
    reviewer: Optional[str] = "admin"
    note: Optional[str] = ""


@app.post("/api/admin/nft/monthly/select-winner")
def admin_nft_monthly_select_winner(inp: MonthlyNFTWinnerSelectIn):
    from telegram_game.utils.nft_selection_identity_v1 import select_monthly_winner_identity_v1

    cid = (inp.candidateId or "").strip()
    if not cid:
        return {"ok": False, "error": "missing_candidate_id"}
    periods = _monthly_nft_award_periods_from_activation_v1(
        _nft_verified_mainnet_activation_at_v1()
    )
    with _lock:
        return select_monthly_winner_identity_v1(
            _db, cid, periods, _now(), (inp.reviewer or "admin").strip(),
            (inp.note or "").strip(), _ensure_monthly_nft_tables,
            _monthly_nft_candidate_row_to_dict,
        )


# QMA_0070_CANDIDATE_FREEZE_V1
# Admin-only; freezing does not draw a winner, mint, or change balances.
class NFTWalletChallengeIn(BaseModel):
    playerId: str
    initData: str
    period: str


class NFTWalletProofIn(BaseModel):
    playerId: str
    initData: str
    nonce: str
    publicKey: str
    signature: str


@app.post("/api/nft/wallet/challenge")
def nft_wallet_challenge_v1(inp: NFTWalletChallengeIn):
    from telegram_game.utils.nft_program_bridge_v1 import fetch_verified_nft_game_program_v1
    from telegram_game.utils.nft_wallet_proof_v1 import issue_wallet_challenge_v1
    user = _nft_verified_player_id_v1(inp.initData, inp.playerId)
    if not user:
        return {"ok": False, "error": "telegram_identity_required"}
    return issue_wallet_challenge_v1(_db, fetch_verified_nft_game_program_v1(), user, inp.period, _now())


@app.post("/api/nft/wallet/prove")
def nft_wallet_prove_v1(inp: NFTWalletProofIn):
    from telegram_game.utils.nft_program_bridge_v1 import fetch_verified_nft_game_program_v1
    from telegram_game.utils.nft_wallet_proof_v1 import consume_wallet_challenge_v1
    user = _nft_verified_player_id_v1(inp.initData, inp.playerId)
    if not user:
        return {"ok": False, "error": "telegram_identity_required"}
    return consume_wallet_challenge_v1(_db, fetch_verified_nft_game_program_v1(), user,
                                      inp.nonce, inp.publicKey, inp.signature, _now())


class MonthlyNFTPoolFreezeIn(BaseModel):
    period: str


@app.post("/api/admin/nft/monthly/freeze-pool")
def admin_nft_monthly_freeze_pool_v1(inp: MonthlyNFTPoolFreezeIn):
    from telegram_game.utils.nft_draw_freeze_v1 import freeze_monthly_pool_v1

    periods = _monthly_nft_award_periods_from_activation_v1(
        _nft_verified_mainnet_activation_at_v1()
    )
    with _lock:
        return freeze_monthly_pool_v1(
            _db, (inp.period or "").strip(), periods, _now(),
            _ensure_monthly_nft_tables, _monthly_nft_candidate_row_to_dict,
        )


@app.get("/api/admin/nft/monthly/frozen-pool")
def admin_nft_monthly_frozen_pool_v1(period: str):
    from telegram_game.utils.nft_draw_freeze_v1 import read_monthly_pool_v1

    with _lock:
        return read_monthly_pool_v1(_db, (period or "").strip())


# TG_MONTHLY_NFT_CONTROLLED_MINT_PATH_V1
# The existing admin middleware protects this route. Neither flag alone can
# authorize an award: signed node evidence, player binding and final catalog
# commitments are checked by the bridge. Installation never enables minting.
class MonthlyNFTDrawCommitmentIn(BaseModel):
    period: str
    publication: Dict[str, Any]


class MonthlyNFTVerifiedDrawIn(BaseModel):
    period: str
    beacon: Dict[str, Any]


def _monthly_nft_evidence_context_v1():
    from telegram_game.utils.nft_program_bridge_v1 import fetch_verified_nft_game_program_v1
    from telegram_game.utils.nft_draw_evidence_v1 import load_policy_v1
    program = fetch_verified_nft_game_program_v1()
    return program, load_policy_v1(program)


@app.get("/api/admin/nft/monthly/draw-commitment")
def admin_nft_draw_commitment_v1(period: str):
    from contextlib import closing
    from telegram_game.utils.nft_draw_evidence_v1 import pool, commitment_v1, require_pool_wallets
    try:
        program, policy = _monthly_nft_evidence_context_v1()
        with closing(_db()) as conn:
            conn.execute("BEGIN")
            frozen = pool(conn, period)
            require_pool_wallets(conn, frozen, program)
            commitment = commitment_v1(frozen, program, policy)
        return {"ok": True, "commitment": commitment, "winnerSelected": False}
    except (OSError, sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "draw_evidence_unavailable"}


@app.post("/api/admin/nft/monthly/register-draw-commitment")
def admin_nft_register_draw_commitment_v1(inp: MonthlyNFTDrawCommitmentIn):
    from telegram_game.utils.nft_draw_evidence_v1 import register_draw_commitment_v1
    try:
        program, policy = _monthly_nft_evidence_context_v1()
        return register_draw_commitment_v1(_db, program, policy, inp.period, inp.publication, _now())
    except (OSError, sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "draw_evidence_unavailable"}


@app.post("/api/admin/nft/monthly/draw-winner")
def admin_nft_verified_draw_v1(inp: MonthlyNFTVerifiedDrawIn):
    from telegram_game.utils.nft_draw_evidence_v1 import draw_monthly_winner_v1
    try:
        program, policy = _monthly_nft_evidence_context_v1()
        return draw_monthly_winner_v1(_db, program, policy, inp.period, inp.beacon, _now(), _monthly_nft_candidate_row_to_dict)
    except (OSError, sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "draw_evidence_unavailable"}


class MonthlyNFTControlledMintIn(BaseModel):
    candidateId: str
    catalogItemId: Optional[str] = None
    reviewer: Optional[str] = "admin"
    note: Optional[str] = ""
    signedAward: Optional[Dict[str, Any]] = None


@app.post("/api/admin/nft/monthly/publish-draw-commitment")
def admin_nft_publish_draw_v1(inp: MonthlyNFTPoolFreezeIn):
    from telegram_game.utils.nft_external_evidence_v1 import publish_and_register_v1
    try:
        program, policy = _monthly_nft_evidence_context_v1()
        return publish_and_register_v1(_db, program, policy, inp.period, _now)
    except (OSError, sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "external_evidence_unavailable"}


@app.post("/api/admin/nft/monthly/draw-from-beacon")
def admin_nft_draw_from_beacon_v1(inp: MonthlyNFTPoolFreezeIn):
    from telegram_game.utils.nft_external_evidence_v1 import draw_from_beacon_v1
    try:
        program, policy = _monthly_nft_evidence_context_v1()
        return draw_from_beacon_v1(_db, program, policy, inp.period, _monthly_nft_candidate_row_to_dict, _now)
    except (OSError, sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "external_evidence_unavailable"}


class MonthlyNFTFinalPreparationIn(BaseModel):
    candidateId: str
    catalogItemId: str


@app.get("/api/admin/nft/monthly/catalog-draft")
def admin_nft_program_catalog_draft_v1():
    from telegram_game.utils.nft_program_bridge_v1 import fetch_verified_nft_game_program_v1
    from telegram_game.utils.nft_final_readiness_v1 import build_program_catalog_draft_v1
    from telegram_game.utils.nft_mint_bridge_v1 import _unique_object
    try:
        path = _NFT_REPOSITORY_ROOT_V1 / "nft/catalog/monthly_rare_nft_stock_2026.json"
        if path.stat().st_size > 65536:
            return {"ok": False, "error": "catalog_template_unavailable"}
        stock = json.loads(path.read_text(encoding="utf-8-sig"), object_pairs_hook=_unique_object)
        return {"ok": True, "draft": build_program_catalog_draft_v1(stock, fetch_verified_nft_game_program_v1()), "filesWritten": False}
    except (OSError, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "catalog_template_unavailable"}


def _monthly_final_preparation_v1(inp, inspect):
    from contextlib import closing
    from telegram_game.utils.nft_final_readiness_v1 import prepare_final_approval_v1, inspect_final_readiness_v1
    try:
        program, policy = _monthly_nft_evidence_context_v1()
        with closing(_db()) as conn:
            conn.execute("BEGIN")
            fn = inspect_final_readiness_v1 if inspect else prepare_final_approval_v1
            return fn(conn, _NFT_REPOSITORY_ROOT_V1, program, policy, inp.candidateId, inp.catalogItemId, _now())
    except (OSError, sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "final_preparation_unavailable", "mintAllowed": False}


@app.post("/api/admin/nft/monthly/final-approval-statement")
def admin_nft_final_approval_statement_v1(inp: MonthlyNFTFinalPreparationIn):
    return _monthly_final_preparation_v1(inp, False)


@app.post("/api/admin/nft/monthly/final-readiness")
def admin_nft_final_readiness_v1(inp: MonthlyNFTFinalPreparationIn):
    return _monthly_final_preparation_v1(inp, True)


@app.post("/api/admin/nft/monthly/controlled-mint")
def admin_nft_monthly_controlled_mint(inp: MonthlyNFTControlledMintIn):
    from telegram_game.utils.nft_mint_bridge_v1 import controlled_monthly_nft_mint_v1

    return controlled_monthly_nft_mint_v1(
        _db,
        _NFT_REPOSITORY_ROOT_V1,
        (inp.candidateId or "").strip(),
        (inp.catalogItemId or "").strip(),
        inp.signedAward,
    )


# QC_ADMIN_PANEL_AUTH_PAGINATION_V1_BRIDGE
# Central admin auth/pagination/API registry bridge.
# This does not mint, transfer, change balances, change chain, or start/stop mining.
try:
    from telegram_game.admin_panel_safety_v1 import (
        QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        qc_admin_auth_review_v1,
        qc_admin_endpoint_registry_v1,
        qc_admin_lock_summary_v1,
        qc_admin_pagination_v1,
    )
except ModuleNotFoundError:
    from admin_panel_safety_v1 import (
        QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        qc_admin_auth_review_v1,
        qc_admin_endpoint_registry_v1,
        qc_admin_lock_summary_v1,
        qc_admin_pagination_v1,
    )


def _qc26_admin_headers_v1(request):
    try:
        return dict(request.headers or {})
    except Exception:
        return {}


def _qc26_admin_panel_auth_result_v1(request):
    path = ""
    try:
        path = str(request.url.path or "")
    except Exception:
        path = "/api/admin"

    return qc_admin_auth_review_v1(_qc26_admin_headers_v1(request), path)


@app.middleware("http")
async def _qc26_admin_panel_auth_middleware_v1(request, call_next):
    path = str(getattr(getattr(request, "url", None), "path", "") or "")
    if path.startswith("/api/admin/"):
        auth = _qc26_admin_panel_auth_result_v1(request)
        if not auth.get("ok"):
            from fastapi.responses import JSONResponse as _QC26JSONResponse
            return _QC26JSONResponse(status_code=401, content=auth)

    return await call_next(request)


@app.get("/api/admin/panel/registry")
def qc26_admin_panel_registry(limit: int = 50, offset: int = 0):
    page = qc_admin_pagination_v1({"limit": limit, "offset": offset})
    reg = qc_admin_endpoint_registry_v1()
    groups = reg.get("groups", [])
    start = page["offset"]
    end = start + page["limit"]
    return {
        "ok": True,
        "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        "pagination": page,
        "totalGroups": len(groups),
        "groups": groups[start:end],
        "safeBoundary": reg.get("safeBoundary", {}),
    }


@app.get("/api/admin/panel/pagination-policy")
def qc26_admin_panel_pagination_policy(limit: int = 50, offset: int = 0):
    return {
        "ok": True,
        "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        "pagination": qc_admin_pagination_v1({"limit": limit, "offset": offset}),
        "summary": qc_admin_lock_summary_v1(),
    }

# QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_V1_BRIDGE
# Read-only admin monitoring/runbook visibility.
# This does not mint, transfer, change balances, change chain, or start/stop mining.
try:
    from telegram_game.monitoring_event_log_alert_runbook_v1 import (
        QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION,
        classify_event_v1,
        normalize_severity_v1,
        runbook_for_category_v1,
        runbook_summary_v1,
        validate_event_v1,
    )
except ModuleNotFoundError:
    from monitoring_event_log_alert_runbook_v1 import (
        QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION,
        classify_event_v1,
        normalize_severity_v1,
        runbook_for_category_v1,
        runbook_summary_v1,
        validate_event_v1,
    )


@app.get("/api/admin/monitoring/runbook")
def qc27_admin_monitoring_runbook():
    return runbook_summary_v1()


@app.get("/api/admin/monitoring/event-policy")
def qc27_admin_monitoring_event_policy(source: str = "admin-panel", category: str = "api_down", severity: str = "WARNING", message: str = "policy check"):
    event = {
        "source": source,
        "category": category,
        "severity": normalize_severity_v1(severity),
        "message": message,
    }
    return {
        "ok": True,
        "version": QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION,
        "validation": validate_event_v1(event),
        "classification": classify_event_v1(event),
        "runbook": runbook_for_category_v1(category),
    }

# QC_DB_PERSISTENCE_PLAN_V1_BRIDGE
# Read-only DB persistence plan visibility for admin panel.
# This does not migrate live DB, mint, transfer, change balances, change chain, or start/stop mining.
try:
    from telegram_game.db_persistence_plan_v1 import (
        QC_DB_PERSISTENCE_PLAN_VERSION,
        db_persistence_summary_v1,
    )
except ModuleNotFoundError:
    from db_persistence_plan_v1 import (
        QC_DB_PERSISTENCE_PLAN_VERSION,
        db_persistence_summary_v1,
    )


@app.get("/api/admin/persistence/db-plan")
def qc29_admin_persistence_db_plan():
    return db_persistence_summary_v1()

# QC_QUEUE_WORKER_INTEGRATION_PLAN_V1_BRIDGE
# Read-only queue/worker integration plan visibility for admin panel.
# This does not start workers, mint, transfer, change balances, change chain, or start/stop mining.
try:
    from telegram_game.queue_worker_plan_v1 import (
        QC_QUEUE_WORKER_PLAN_VERSION,
        queue_worker_summary_v1,
    )
except ModuleNotFoundError:
    from queue_worker_plan_v1 import (
        QC_QUEUE_WORKER_PLAN_VERSION,
        queue_worker_summary_v1,
    )


@app.get("/api/admin/queue-worker/plan")
def qc30_admin_queue_worker_plan():
    return queue_worker_summary_v1()

# QC_BACKUP_RESTORE_RUNBOOK_V1_BRIDGE
# Read-only backup/restore runbook visibility for admin panel.
# This does not run live restore, delete data, overwrite data, mint, transfer, change balances, or change chain.
try:
    from telegram_game.backup_restore_runbook_v1 import (
        QC_BACKUP_RESTORE_RUNBOOK_VERSION,
        backup_restore_summary_v1,
    )
except ModuleNotFoundError:
    from backup_restore_runbook_v1 import (
        QC_BACKUP_RESTORE_RUNBOOK_VERSION,
        backup_restore_summary_v1,
    )


@app.get("/api/admin/backup-restore/runbook")
def qc31_admin_backup_restore_runbook():
    return backup_restore_summary_v1()

# Policy displays must observe the authenticated Go node, never Python copies.
async def _node_policy_section(section):
    from .utils.qc_api import fetch_runtime_policy
    data = await fetch_runtime_policy()
    if not data.get("ok") or section not in data.get("sections", {}):
        return JSONResponse(status_code=503, content={"ok":False,"error":"node_runtime_policy_unavailable"})
    return {"ok":True,"version":data["version"],"source":data["source"],
            "context":data["context"],"policy":data["sections"][section],
            "authority":data.get("authority", "Go declaration; activation is described in context.")}


@app.get("/api/admin/economy/mainnet-before")
async def qc32_admin_mainnet_before_economy():
    return await _node_policy_section("mainnet-before")


@app.get("/api/admin/economy/burn-sink")
async def qc32c_admin_burn_sink_control():
    return await _node_policy_section("burn-sink")


@app.get("/api/admin/economy/mining-transition")
async def qc32d_admin_mining_node_transition():
    return await _node_policy_section("mining-transition")


@app.get("/api/admin/economy/network-fee")
async def qc32e_admin_network_fee_policy():
    return await _node_policy_section("network-fee")


@app.get("/api/admin/economy/founder-vesting")
async def qc32f_admin_founder_vesting_policy():
    return await _node_policy_section("founder-vesting")


@app.get("/api/admin/network/hardfork-policy")
async def qc32g_admin_hardfork_policy():
    return await _node_policy_section("hardfork-policy")


def _verified_task_reward(inp, allowed_actions=None):
    from .player_auth import ACTOR
    from .reward_evidence import claim
    actor = ACTOR.get()
    if not actor or actor != inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    from .social_reward_activation import is_social_campaign, social_rewards_enabled
    if is_social_campaign(_db, inp.campaignId) and not social_rewards_enabled():
        return {"ok":False,"error":"social_rewards_disabled_mainnet_only"}
    result = claim(_db,actor,inp.campaignId,inp.proof,_now(),allowed_actions)
    if result.get("ok"):
        with closing(_db()) as conn, conn:
            result["balances"] = _tgwt_get(conn,actor)
            result["rev"] = result["balances"]["rev"]
    return result


class VerifiedRewardClaimIn(BaseModel):
    playerId: str
    campaignId: str
    proof: dict


class RewardCampaignIn(BaseModel):
    policy: dict


class XRewardStartIn(BaseModel):
    playerId: str
    campaignId: str


class XRewardCompleteIn(BaseModel):
    playerId: str
    state: str
    code: str


class SocialProviderVerifyIn(BaseModel):
    playerId: str
    campaignId: str


class YouTubeRewardCompleteIn(BaseModel):
    playerId: str
    state: str
    code: str


@app.post("/api/social/x/start")
def start_x_reward_verification(inp: XRewardStartIn):
    from .player_auth import ACTOR
    from .x_reward_verifier import begin
    actor=ACTOR.get()
    if actor!=inp.playerId: return {"ok":False,"error":"telegram_session_required"}
    return begin(_db,actor,inp.campaignId,_now())


@app.post("/api/social/x/complete")
async def complete_x_reward_verification(inp: XRewardCompleteIn):
    from .player_auth import ACTOR
    from .x_reward_verifier import complete
    actor=ACTOR.get()
    if actor!=inp.playerId: return {"ok":False,"error":"telegram_session_required"}
    return await complete(_db,actor,inp.state,inp.code,_now())


@app.post("/api/social/telegram/verify")
async def verify_telegram_reward(inp: SocialProviderVerifyIn):
    from .player_auth import ACTOR
    from .telegram_reward_verifier import verify
    actor=ACTOR.get()
    if not actor or actor!=inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    result=await verify(_db,actor,inp.campaignId,_now())
    if result.get("ok"):
        with closing(_db()) as conn:
            result["balances"]=_tgwt_get(conn,actor)
    return result


@app.post("/api/social/youtube/start")
def start_youtube_reward_verification(inp: SocialProviderVerifyIn):
    from .player_auth import ACTOR
    from .youtube_reward_verifier import begin
    actor=ACTOR.get()
    if not actor or actor!=inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    return begin(_db,actor,inp.campaignId,_now())


@app.post("/api/social/youtube/complete")
async def complete_youtube_reward_verification(inp: YouTubeRewardCompleteIn):
    from .player_auth import ACTOR
    from .youtube_reward_verifier import complete
    actor=ACTOR.get()
    if not actor or actor!=inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    result=await complete(_db,actor,inp.state,inp.code,_now())
    if result.get("ok"):
        with closing(_db()) as conn:
            result["balances"]=_tgwt_get(conn,actor)
    return result


@app.get("/oauth/x/callback",include_in_schema=False)
def x_reward_oauth_callback(request: Request):
    payload=getattr(request.state,"x_oauth_callback",{"error":"invalid_callback"})
    encoded=json.dumps({"type":"qc:x-oauth",**payload}).replace("<","\\u003c").replace(">","\\u003e")
    nonce=secrets.token_urlsafe(24)
    page='<!doctype html><meta charset="utf-8"><title>X verification</title><p id="result">Returning verification to QuantumCoin Game…</p>'
    page+=f'<script nonce="{nonce}">history.replaceState(null,"","/oauth/x/callback");if(window.opener){{window.opener.postMessage({encoded},location.origin);window.close();}}else{{document.getElementById("result").textContent="Reopen the game and start verification in a browser that supports returning sign-in to the game.";}}</script>'
    return HTMLResponse(page,headers={"Cache-Control":"no-store","Referrer-Policy":"no-referrer",
        "Content-Security-Policy":f"default-src 'none'; script-src 'nonce-{nonce}'; frame-ancestors 'none'"})


@app.get("/oauth/youtube/callback",include_in_schema=False)
def youtube_reward_oauth_callback(request: Request):
    payload=getattr(request.state,"youtube_oauth_callback",{"error":"invalid_callback"})
    encoded=json.dumps({"type":"qc:youtube-oauth",**payload}).replace("<","\\u003c").replace(">","\\u003e")
    nonce=secrets.token_urlsafe(24)
    page='<!doctype html><meta charset="utf-8"><title>YouTube verification</title><p id="result">Returning verification to QuantumCoin Game…</p>'
    page+=f'<script nonce="{nonce}">history.replaceState(null,"","/oauth/youtube/callback");if(window.opener){{window.opener.postMessage({encoded},location.origin);window.close();}}else{{document.getElementById("result").textContent="Reopen the game and start verification in the game browser.";}}</script>'
    return HTMLResponse(page,headers={"Cache-Control":"no-store","Referrer-Policy":"no-referrer",
        "Content-Security-Policy":f"default-src 'none'; script-src 'nonce-{nonce}'; frame-ancestors 'none'"})


@app.post("/api/tgwt/rewards/claim")
def verified_reward_claim(inp: VerifiedRewardClaimIn):
    return _verified_task_reward(inp)


@app.post("/api/admin/tgwt/campaigns")
def create_verified_reward_campaign(inp: RewardCampaignIn):
    from .reward_evidence import create_campaign
    return create_campaign(_db,inp.policy)


@app.get("/api/tgwt/pool")
def verified_reward_pool():
    with closing(_db()) as conn:
        row=conn.execute("SELECT total_u,distributed_u FROM tgwt_pool WHERE id=1").fetchone()
    from .social_reward_activation import social_rewards_enabled
    return {"ok":True,"total_u":row[0],"allocated_u":row[1],"remaining_u":row[0]-row[1],
            "offChainAccounting":True,"onChainFundingVerified":False,"gasPayer":"user","gasAsset":"BNB",
            "tokenContract":"0xbc70225507b132e9926c9b09c14ff4f87f6d5451","bscClaimEnabled":False,
            "socialRewardsEnabled":social_rewards_enabled(),"socialRewardsNetwork":"bsc_mainnet"}


@app.get("/api/tgwt/campaigns")
def verified_reward_campaigns():
    with closing(_db()) as conn:
        rows=conn.execute("SELECT policy_json,policy_hash FROM tgwt_campaigns_v1 WHERE enabled=1 LIMIT 100").fetchall()
    from .social_reward_activation import social_rewards_enabled
    enabled=social_rewards_enabled()
    campaigns=[]
    for row in rows:
        policy=json.loads(row[0])
        if str(policy.get("provider","")).upper() in {"X","TELEGRAM","YOUTUBE"} and not enabled:
            continue
        campaigns.append({"policy":policy,"campaignHash":row[1]})
    return {"ok":True,"campaigns":campaigns,"verificationRequired":True,
            "providerAdaptersConfiguredAutomatically":False,"socialRewardsEnabled":enabled,
            "socialRewardsNetwork":"bsc_mainnet"}


class RewardCampaignStateIn(BaseModel):
    campaignId: str
    enabled: bool


@app.post("/api/admin/tgwt/campaign-state")
def verified_reward_campaign_state(inp: RewardCampaignStateIn):
    with closing(_db()) as conn, conn:
        changed=conn.execute("UPDATE tgwt_campaigns_v1 SET enabled=? WHERE campaign_id=?",(int(inp.enabled),inp.campaignId)).rowcount
    return {"ok":changed==1}


class GameWorkIn(BaseModel):
    playerId: str
    sessionId: str
    nextBlock: bool = False


class GameWorkSubmitIn(GameWorkIn):
    jobId: str
    nonce: str


@app.post("/api/mine/work")
async def issue_game_client_work(inp: GameWorkIn):
    from .game_pow import issue_work
    from .player_auth import ACTOR
    if ACTOR.get()!=inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    if _get_game_mining_mode()!=QC_GAME_MINING_MODE_REAL_CHAIN:
        return {"ok":False,"error":"real_client_mining_not_enabled"}
    return await issue_work(_db,inp.playerId,inp.sessionId,_now(),next_block=inp.nextBlock)


@app.post("/api/mine/work/status")
async def game_client_work_status(inp: GameWorkIn):
    from .game_pow import work_status
    from .player_auth import ACTOR
    if ACTOR.get()!=inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    if _get_game_mining_mode()!=QC_GAME_MINING_MODE_REAL_CHAIN:
        return {"ok":False,"error":"real_client_mining_not_enabled"}
    return await work_status(_db,inp.playerId,inp.sessionId,_now())


@app.post("/api/mine/work/submit")
async def submit_game_client_work(inp: GameWorkSubmitIn):
    from .game_pow import submit_work
    from .player_auth import ACTOR
    if ACTOR.get()!=inp.playerId:
        return {"ok":False,"error":"telegram_session_required"}
    if _get_game_mining_mode()!=QC_GAME_MINING_MODE_REAL_CHAIN:
        return {"ok":False,"error":"real_client_mining_not_enabled"}
    return await submit_work(_db,inp.playerId,inp.sessionId,inp.jobId,inp.nonce,_now())


@app.get("/api/security/limits/status")
def persisted_runtime_limit_status():
    from .durable_limits import MAX_ROWS
    with closing(_db()) as conn:
        count=conn.execute("SELECT COUNT(*) FROM request_limits_v1 WHERE expires_at>?",(_now(),)).fetchone()[0]
    return {"ok":True,"backend":"canonical_sqlite_fixed_windows","activeRows":count,"maxRows":MAX_ROWS,
            "actorFrom":"verified_server_session","trustClientForwardedHeaders":False,"expiredRowsPrunedOnAdmission":True}


@app.get("/mining/canonical-pow-worker.mjs",include_in_schema=False)
def canonical_game_mining_worker():
    return FileResponse(str(_NFT_REPOSITORY_ROOT_V1 / "webui/dist/canonical-pow-worker.mjs"),media_type="text/javascript")


@app.get("/mining/mining-session.mjs",include_in_schema=False)
def canonical_game_mining_lifecycle():
    return FileResponse(str(_NFT_REPOSITORY_ROOT_V1 / "webui/dist/mining-session.mjs"),media_type="text/javascript")


@app.get("/mining/canonical-pow.mjs",include_in_schema=False)
def canonical_game_mining_core():
    return FileResponse(str(_NFT_REPOSITORY_ROOT_V1 / "webui/dist/canonical-pow.mjs"),media_type="text/javascript")


# Install last so identity is checked before any legacy rate/replay middleware.
from .endpoint_response_contracts import install_openapi_contracts
install_openapi_contracts(app)
from .player_auth import PlayerIdentityBoundary
app.add_middleware(PlayerIdentityBoundary, db=lambda: _db(), bot_token=lambda: _telegram_bot_token())
