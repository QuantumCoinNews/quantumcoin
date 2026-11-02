# QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_V1
# Read-only event log / alert / runbook policy for Telegram Game + QuantumCoin admin APIs.
#
# Boundary:
# - Does not mint.
# - Does not send QC/TGWT.
# - Does not change balances.
# - Does not change chain/consensus/supply.
# - Does not start/stop miners.
# - Does not auto-ban.
# - Produces read-only alert/runbook guidance only.

from typing import Any, Dict, List

QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION = "QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_V1"

SEVERITIES = ["INFO", "WARNING", "CRITICAL"]

SAFE_BOUNDARY = {
    "doesNotMint": True,
    "doesNotSendFunds": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeSupply": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "doesNotAutoBan": True,
    "readOnlyAlertingStandard": True,
    "requiresAdminReviewBeforeAction": True,
}

FORBIDDEN_ACTIONS = [
    "do_not_mint",
    "do_not_change_qc_supply",
    "do_not_change_balances",
    "do_not_send_funds",
    "do_not_start_or_stop_miners",
    "do_not_auto_ban_without_review",
]

RUNBOOK_ENTRIES = [
    {
        "category": "api_down",
        "severity": "CRITICAL",
        "summary": "Expected API is unreachable or repeatedly failing.",
        "adminAction": "Check process, port, env mode, logs, and last deployment before restart decision.",
        "forbidden": FORBIDDEN_ACTIONS,
    },
    {
        "category": "auth_failure",
        "severity": "WARNING",
        "summary": "Admin/API authentication failed or token is missing.",
        "adminAction": "Verify token config, reject query token usage, and review source client/IP.",
        "forbidden": FORBIDDEN_ACTIONS,
    },
    {
        "category": "chain_mismatch",
        "severity": "CRITICAL",
        "summary": "Mainnet/testnet chain file, network label, or height looks inconsistent.",
        "adminAction": "Stop promotion, verify chain file separation, compare node labels, keep release files untouched.",
        "forbidden": FORBIDDEN_ACTIONS,
    },
    {
        "category": "payout_guard",
        "severity": "WARNING",
        "summary": "TGWT/claim/withdraw payout requires admin review.",
        "adminAction": "Review claim status, settlement precheck, and confirm backend has no private key/gas-payment path.",
        "forbidden": FORBIDDEN_ACTIONS,
    },
    {
        "category": "nft_mint_guard",
        "severity": "WARNING",
        "summary": "NFT monthly flow reached a review/mint-sensitive state.",
        "adminAction": "Verify AI review, admin approval, uniqueness, final metadata hash, and manual approval token.",
        "forbidden": FORBIDDEN_ACTIONS,
    },
    {
        "category": "watch_verification_failure",
        "severity": "WARNING",
        "summary": "Watch & Earn completion is not server-side verified.",
        "adminAction": "Reject reward, keep final revenue mode disabled, review provider verification logs.",
        "forbidden": FORBIDDEN_ACTIONS,
    },
]

def normalize_severity_v1(value: str) -> str:
    raw = str(value or "").strip().upper()
    if raw in {"CRITICAL", "FATAL", "PANIC", "HIGH"}:
        return "CRITICAL"
    if raw in {"WARNING", "WARN", "MEDIUM"}:
        return "WARNING"
    return "INFO"

def validate_event_v1(event: Dict[str, Any]) -> Dict[str, Any]:
    source = str(event.get("source") or "").strip()
    category = str(event.get("category") or "").strip()
    message = str(event.get("message") or "").strip()
    severity = str(event.get("severity") or "").strip().upper()

    errors: List[str] = []
    if not source:
        errors.append("source_required")
    if not category:
        errors.append("category_required")
    if not message:
        errors.append("message_required")
    if severity not in SEVERITIES:
        errors.append("severity_must_be_normalized")

    return {
        "ok": len(errors) == 0,
        "version": QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def runbook_for_category_v1(category: str) -> Dict[str, Any]:
    clean = str(category or "").strip().lower()
    for entry in RUNBOOK_ENTRIES:
        if entry["category"] == clean:
            return dict(entry)
    return {
        "category": clean or "general",
        "severity": "INFO",
        "summary": "General monitoring event.",
        "adminAction": "Review event details and record admin note.",
        "forbidden": list(FORBIDDEN_ACTIONS),
    }

def classify_event_v1(event: Dict[str, Any]) -> Dict[str, Any]:
    severity = normalize_severity_v1(str(event.get("severity") or "INFO"))
    category = str(event.get("category") or "general").strip().lower()
    entry = runbook_for_category_v1(category)

    return {
        "ok": True,
        "version": QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION,
        "alertOpen": severity != "INFO",
        "severity": severity,
        "category": category,
        "escalate": severity == "CRITICAL",
        "runbook": entry,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def runbook_summary_v1() -> Dict[str, Any]:
    return {
        "ok": True,
        "version": QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION,
        "scope": "quantumcoin_and_telegram_game_monitoring_event_log_alert_runbook",
        "severities": list(SEVERITIES),
        "entries": [dict(e) for e in RUNBOOK_ENTRIES],
        "safeBoundary": dict(SAFE_BOUNDARY),
    }
