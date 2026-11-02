# QC_LOAD_TEST_SCRIPTS_SAFE_V1
# Safe load test plan and endpoint rules for QuantumCoin + Telegram Game.
#
# Boundary:
# - Read-only GET load testing only.
# - Blocks mutating endpoints.
# - Localhost-only by default.
# - Does not mint, send funds, change balances, change chain/supply, or start/stop miners.

from typing import Any, Dict, List
from urllib.parse import urlparse

QC_LOAD_TEST_SCRIPTS_VERSION = "QC_LOAD_TEST_SCRIPTS_SAFE_V1"

MAX_REQUESTS = 500
MAX_CONCURRENCY = 25
DEFAULT_REQUESTS = 20
DEFAULT_CONCURRENCY = 2

LOCAL_HOSTS = {"127.0.0.1", "localhost", "::1"}

BLOCKED_PREFIXES = [
    "/api/mine",
    "/api/miner/start",
    "/api/miner/stop",
    "/api/wallet/send",
    "/api/wallet/send-qc",
    "/api/tx/send",
    "/api/watch/claim",
    "/api/social/claim",
    "/api/store/buy",
    "/api/v1/tgwt/earn",
    "/api/v1/tgwt/verify",
    "/api/v1/withdraw/request",
    "/api/withdraw/request",
    "/api/withdraw/mark-pending",
    "/api/withdraw/mock-confirm",
    "/api/withdraw/real-payout-precheck",
    "/api/settlement/mark-pending",
    "/api/settlement/mock-confirm",
    "/api/admin/nft/monthly/review",
    "/api/admin/nft/monthly/ai-review",
    "/api/admin/nft/monthly/select-winner",
    "/api/admin/nft/monthly/controlled-mint",
]

SAFE_BOUNDARY = {
    "onlyGET": True,
    "blocksMutatingEndpoints": True,
    "doesNotMint": True,
    "doesNotSendFunds": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeSupply": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "localOnlyByDefault": True,
}

def _clamp_int(value: Any, default: int, minimum: int, maximum: int) -> int:
    try:
        n = int(value)
    except Exception:
        n = default
    return max(minimum, min(n, maximum))

def normalize_endpoint_v1(endpoint: str) -> str:
    clean = str(endpoint or "").strip()
    if not clean:
        return "/health"
    if not clean.startswith("/"):
        clean = "/" + clean
    return clean

def is_blocked_endpoint_v1(endpoint: str) -> bool:
    clean = normalize_endpoint_v1(endpoint).lower()
    return any(clean.startswith(prefix.lower()) for prefix in BLOCKED_PREFIXES)

def validate_load_test_case_v1(case: Dict[str, Any]) -> Dict[str, Any]:
    base_url = str(case.get("baseUrl") or "http://127.0.0.1:8082").strip()
    endpoint = normalize_endpoint_v1(str(case.get("endpoint") or "/health"))
    method = str(case.get("method") or "GET").upper().strip()
    allow_remote = bool(case.get("allowRemote", False))

    parsed = urlparse(base_url)
    errors: List[str] = []

    if not parsed.scheme or not parsed.netloc:
        errors.append("base_url_invalid")

    if not allow_remote and parsed.hostname not in LOCAL_HOSTS:
        errors.append("remote_base_url_blocked")

    if method != "GET":
        errors.append("only_get_allowed")

    if is_blocked_endpoint_v1(endpoint):
        errors.append("mutating_endpoint_blocked")

    requests = _clamp_int(case.get("requests"), DEFAULT_REQUESTS, 1, MAX_REQUESTS)
    concurrency = _clamp_int(case.get("concurrency"), DEFAULT_CONCURRENCY, 1, MAX_CONCURRENCY)

    return {
        "ok": len(errors) == 0,
        "version": QC_LOAD_TEST_SCRIPTS_VERSION,
        "errors": errors,
        "normalized": {
            "baseUrl": base_url,
            "endpoint": endpoint,
            "method": method,
            "requests": requests,
            "concurrency": concurrency,
            "allowRemote": allow_remote,
        },
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def build_load_test_matrix_v1() -> Dict[str, Any]:
    cases = [
        {
            "name": "qc_node_health_local",
            "component": "QC_NODE",
            "baseUrl": "http://127.0.0.1:8082",
            "endpoint": "/health",
            "method": "GET",
            "requests": 20,
            "concurrency": 2,
        },
        {
            "name": "qc_node_network_label_local",
            "component": "QC_NODE",
            "baseUrl": "http://127.0.0.1:8082",
            "endpoint": "/api/network/label",
            "method": "GET",
            "requests": 20,
            "concurrency": 2,
        },
        {
            "name": "telegram_game_health_local",
            "component": "TELEGRAM_GAME",
            "baseUrl": "http://127.0.0.1:8080",
            "endpoint": "/health",
            "method": "GET",
            "requests": 20,
            "concurrency": 2,
        },
        {
            "name": "telegram_game_qc_health_bridge",
            "component": "TELEGRAM_GAME",
            "baseUrl": "http://127.0.0.1:8080",
            "endpoint": "/api/qc/health",
            "method": "GET",
            "requests": 20,
            "concurrency": 2,
        },
        {
            "name": "admin_panel_registry_local",
            "component": "ADMIN_PANEL",
            "baseUrl": "http://127.0.0.1:8080",
            "endpoint": "/api/admin/panel/registry?limit=10&offset=0",
            "method": "GET",
            "requests": 10,
            "concurrency": 1,
            "requiresAdminToken": True,
        },
        {
            "name": "admin_monitoring_runbook_local",
            "component": "ADMIN_PANEL",
            "baseUrl": "http://127.0.0.1:8080",
            "endpoint": "/api/admin/monitoring/runbook",
            "method": "GET",
            "requests": 10,
            "concurrency": 1,
            "requiresAdminToken": True,
        },
    ]

    validations = [validate_load_test_case_v1(c) for c in cases]

    return {
        "ok": all(v["ok"] for v in validations),
        "version": QC_LOAD_TEST_SCRIPTS_VERSION,
        "cases": cases,
        "validations": validations,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }
