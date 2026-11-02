# QC_ADMIN_PANEL_AUTH_PAGINATION_V1
# Central admin auth + pagination policy for Telegram Game admin APIs.
#
# Boundary:
# - Does not mint.
# - Does not send QC/TGWT.
# - Does not change balances.
# - Does not change chain/consensus/supply.
# - Does not start/stop miners.
# - Provides admin auth/pagination/registry safety only.

import os
from typing import Any, Dict, List, Tuple

QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION = "QC_ADMIN_PANEL_AUTH_PAGINATION_V1"

QC_ADMIN_HEADER_PRIMARY = "X-Admin-Token"
QC_ADMIN_HEADER_ALT = "X-QC-ADMIN-TOKEN"

QC_ADMIN_DEFAULT_LIMIT = 50
QC_ADMIN_MAX_LIMIT = 100
QC_ADMIN_MAX_OFFSET = 100000

QC_ADMIN_SAFE_BOUNDARY = {
    "authRequiredForAdminEndpoints": True,
    "tokenHeaderPrimary": QC_ADMIN_HEADER_PRIMARY,
    "tokenHeaderAlt": QC_ADMIN_HEADER_ALT,
    "bearerTokenAllowed": True,
    "queryTokenRejected": True,
    "paginationRequired": True,
    "maxPageSize": QC_ADMIN_MAX_LIMIT,
    "offsetSupported": True,
    "cursorRecommendedForLargeLists": True,
    "doesNotMint": True,
    "doesNotSendQC": True,
    "doesNotSendTGWT": True,
    "doesNotPayGas": True,
    "doesNotStorePrivateKey": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeSupply": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
}

QC_ADMIN_ENDPOINT_GROUPS = [
    {
        "group": "NFT_MONTHLY",
        "endpoints": [
            "/api/admin/nft/monthly/candidates",
            "/api/admin/nft/monthly/review",
            "/api/admin/nft/monthly/ai-review",
            "/api/admin/nft/monthly/select-winner",
            "/api/admin/nft/monthly/freeze-pool",
            "/api/admin/nft/monthly/frozen-pool",
            "/api/admin/nft/monthly/draw-commitment",
            "/api/admin/nft/monthly/register-draw-commitment",
            "/api/admin/nft/monthly/draw-winner",
            "/api/admin/nft/monthly/publish-draw-commitment",
            "/api/admin/nft/monthly/draw-from-beacon",
            "/api/admin/nft/monthly/final-approval-statement",
            "/api/admin/nft/monthly/final-readiness",
            "/api/admin/nft/monthly/catalog-draft",
            "/api/admin/nft/monthly/controlled-mint",
        ],
        "requiresAdminAuth": True,
        "requiresPaginationForList": True,
        "safeBoundary": "NO_AUTO_MINT_NO_BALANCE_CHANGE",
    },
    {
        "group": "TGWT_WITHDRAW_CLAIM",
        "endpoints": [
            "/api/v1/withdraw/list",
            "/api/v1/withdraw/status",
            "/api/settlement/summary",
            "/api/withdraw/real-payout-precheck",
        ],
        "requiresAdminAuth": True,
        "requiresPaginationForList": True,
        "safeBoundary": "ADMIN_REVIEW_NO_BACKEND_GAS_NO_PRIVATE_KEY",
    },
    {
        "group": "WATCH_EARN",
        "endpoints": [
            "/api/watch",
            "/api/watch/claim",
        ],
        "requiresAdminAuth": False,
        "requiresPaginationForList": False,
        "safeBoundary": "FINAL_REWARD_REQUIRES_VERIFIED_COMPLETION",
    },
    {
        "group": "NODE_SECURITY",
        "endpoints": [
            "/api/security/action-burst/status",
            "/api/security/rate-limit/status",
            "/api/security/conflicts/status",
        ],
        "requiresAdminAuth": True,
        "requiresPaginationForList": True,
        "safeBoundary": "READ_ONLY_SECURITY_VISIBILITY",
    },
]

def _env(name: str, default: str = "") -> str:
    return str(os.getenv(name) or default).strip()

def _env_bool(name: str) -> bool:
    return _env(name).lower() in {"1", "true", "yes", "on", "enabled"}

def qc_admin_expected_token_v1() -> str:
    return (
        _env("TELEGRAM_GAME_ADMIN_TOKEN")
        or _env("QC_TG_ADMIN_TOKEN")
        or _env("QC_ADMIN_TOKEN")
        or _env("ADMIN_TOKEN")
    )

def qc_admin_mode_v1() -> str:
    raw = " ".join([
        _env("TELEGRAM_GAME_CHAIN_MODE"),
        _env("TELEGRAM_GAME_NETWORK"),
        _env("QC_CHAIN_MODE"),
        _env("QC_NETWORK"),
        _env("WATCH_EARN_MODE"),
        _env("TGWT_BSC_NETWORK"),
    ]).lower()

    if "mainnet" in raw or "final" in raw or "prod" in raw:
        return "mainnet"
    if "testnet" in raw or _env_bool("QC_TESTNET"):
        return "testnet"
    return "local"

def qc_admin_auth_required_v1() -> bool:
    if _env_bool("TELEGRAM_GAME_ADMIN_AUTH_REQUIRED") or _env_bool("QC_ADMIN_AUTH_REQUIRED"):
        return True
    return qc_admin_mode_v1() in {"testnet", "mainnet"}

def qc_admin_extract_token_v1(headers: Dict[str, str]) -> str:
    h = {str(k).lower(): str(v).strip() for k, v in dict(headers or {}).items()}

    token = h.get(QC_ADMIN_HEADER_PRIMARY.lower()) or h.get(QC_ADMIN_HEADER_ALT.lower()) or ""
    if token:
        return token

    auth = h.get("authorization") or ""
    if auth.lower().startswith("bearer "):
        return auth[7:].strip()

    return ""

def qc_admin_auth_review_v1(headers: Dict[str, str], path: str = "/api/admin") -> Dict[str, Any]:
    expected = qc_admin_expected_token_v1()
    required = qc_admin_auth_required_v1() or str(path or "").startswith("/api/admin/")

    if not required and not expected:
        return {
            "ok": True,
            "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
            "mode": qc_admin_mode_v1(),
            "adminAuthRequired": False,
            "devFallbackAllowed": True,
            "safeBoundary": dict(QC_ADMIN_SAFE_BOUNDARY),
        }

    if required and not expected:
        return {
            "ok": False,
            "error": "admin_token_not_configured",
            "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
            "mode": qc_admin_mode_v1(),
            "adminAuthRequired": True,
            "safeBoundary": dict(QC_ADMIN_SAFE_BOUNDARY),
        }

    provided = qc_admin_extract_token_v1(headers)
    if not provided or provided != expected:
        return {
            "ok": False,
            "error": "admin_auth_failed",
            "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
            "mode": qc_admin_mode_v1(),
            "adminAuthRequired": True,
            "safeBoundary": dict(QC_ADMIN_SAFE_BOUNDARY),
        }

    return {
        "ok": True,
        "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        "mode": qc_admin_mode_v1(),
        "adminAuthRequired": True,
        "safeBoundary": dict(QC_ADMIN_SAFE_BOUNDARY),
    }

def qc_admin_pagination_v1(params: Dict[str, Any]) -> Dict[str, Any]:
    raw_limit = params.get("limit", QC_ADMIN_DEFAULT_LIMIT)
    raw_offset = params.get("offset", 0)

    try:
        limit = int(raw_limit)
    except Exception:
        limit = QC_ADMIN_DEFAULT_LIMIT

    try:
        offset = int(raw_offset)
    except Exception:
        offset = 0

    limit = max(1, min(limit, QC_ADMIN_MAX_LIMIT))
    offset = max(0, min(offset, QC_ADMIN_MAX_OFFSET))

    return {
        "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        "limit": limit,
        "offset": offset,
        "maxLimit": QC_ADMIN_MAX_LIMIT,
        "hasPagination": True,
        "cursorRecommendedAfterRows": 1000,
        "largeExportMustUseBackgroundJob": True,
    }

def qc_admin_endpoint_registry_v1() -> Dict[str, Any]:
    return {
        "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        "panelName": "QuantumCoin Unified API Admin Panel",
        "scope": "telegram_game_admin_panel_extended_to_quantumcoin_api_registry",
        "groups": list(QC_ADMIN_ENDPOINT_GROUPS),
        "safeBoundary": dict(QC_ADMIN_SAFE_BOUNDARY),
    }

def qc_admin_lock_summary_v1() -> Dict[str, Any]:
    return {
        "version": QC_ADMIN_PANEL_AUTH_PAGINATION_VERSION,
        "auth": {
            "headers": [QC_ADMIN_HEADER_PRIMARY, QC_ADMIN_HEADER_ALT, "Authorization: Bearer <token>"],
            "env": ["TELEGRAM_GAME_ADMIN_TOKEN", "QC_TG_ADMIN_TOKEN", "QC_ADMIN_TOKEN", "ADMIN_TOKEN"],
            "queryTokenAccepted": False,
        },
        "pagination": {
            "defaultLimit": QC_ADMIN_DEFAULT_LIMIT,
            "maxLimit": QC_ADMIN_MAX_LIMIT,
            "maxOffset": QC_ADMIN_MAX_OFFSET,
        },
        "registry": qc_admin_endpoint_registry_v1(),
    }
