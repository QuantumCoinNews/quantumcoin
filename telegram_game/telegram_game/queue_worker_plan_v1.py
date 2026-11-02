# QC_QUEUE_WORKER_INTEGRATION_PLAN_V1
# Queue/worker integration plan for Telegram Game + QuantumCoin ecosystem.
#
# Boundary:
# - Plan/validation only.
# - no worker start is executed by this module.
# - Does not migrate live DB automatically.
# - Does not send funds.
# - Does not mint.
# - Does not change balances.
# - Does not change chain or supply.

from typing import Any, Dict, List

QC_QUEUE_WORKER_PLAN_VERSION = "QC_QUEUE_WORKER_INTEGRATION_PLAN_V1"

SAFE_BOUNDARY = {
    "doesNotStartWorkers": True,
    "doesNotMint": True,
    "doesNotSendFunds": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeSupply": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "planOnly": True,
    "requiresAdminApproval": True,
    "requiresIdempotency": True,
    "requiresDeadLetter": True,
    "requiresAuditLog": True,
}

REQUIRED_QUEUES = [
    "tgwt_withdraw_review_queue",
    "tgwt_bsc_claim_settlement_queue",
    "monthly_nft_review_queue",
    "monitoring_alert_queue",
    "autopublisher_content_queue",
    "backup_restore_check_queue",
]

def default_queue_worker_plan_v1() -> Dict[str, Any]:
    return {
        "version": QC_QUEUE_WORKER_PLAN_VERSION,
        "mode": "PLAN_ONLY_NO_WORKER_START",
        "queues": [
            {
                "name": "tgwt_withdraw_review_queue",
                "kind": "review",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "requiresAdminApproval": True,
                "requiresIdempotency": True,
                "requiresDeadLetter": True,
                "requiresAuditLog": True,
                "autoExecAllowed": False,
                "canSendFunds": False,
                "canMint": False,
                "canChangeBalances": False,
                "canChangeChain": False,
            },
            {
                "name": "tgwt_bsc_claim_settlement_queue",
                "kind": "settlement",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "requiresAdminApproval": True,
                "requiresIdempotency": True,
                "requiresDeadLetter": True,
                "requiresAuditLog": True,
                "autoExecAllowed": False,
                "canSendFunds": False,
                "canMint": False,
                "canChangeBalances": False,
                "canChangeChain": False,
            },
            {
                "name": "monthly_nft_review_queue",
                "kind": "review",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "requiresAdminApproval": True,
                "requiresIdempotency": True,
                "requiresDeadLetter": True,
                "requiresAuditLog": True,
                "autoExecAllowed": False,
                "canSendFunds": False,
                "canMint": False,
                "canChangeBalances": False,
                "canChangeChain": False,
            },
            {
                "name": "monitoring_alert_queue",
                "kind": "monitoring",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "requiresAdminApproval": False,
                "requiresIdempotency": True,
                "requiresDeadLetter": True,
                "requiresAuditLog": True,
                "autoExecAllowed": True,
                "canSendFunds": False,
                "canMint": False,
                "canChangeBalances": False,
                "canChangeChain": False,
            },
            {
                "name": "autopublisher_content_queue",
                "kind": "publishing",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "requiresAdminApproval": True,
                "requiresIdempotency": True,
                "requiresDeadLetter": True,
                "requiresAuditLog": True,
                "autoExecAllowed": False,
                "canSendFunds": False,
                "canMint": False,
                "canChangeBalances": False,
                "canChangeChain": False,
            },
            {
                "name": "backup_restore_check_queue",
                "kind": "maintenance",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "requiresAdminApproval": True,
                "requiresIdempotency": True,
                "requiresDeadLetter": True,
                "requiresAuditLog": True,
                "autoExecAllowed": False,
                "canSendFunds": False,
                "canMint": False,
                "canChangeBalances": False,
                "canChangeChain": False,
            },
        ],
        "safeBoundary": dict(SAFE_BOUNDARY),
        "notes": {
            "dev": "Queue worker integration is plan-only in this step.",
            "testnet": "Testnet queues must be separated from mainnet.",
            "mainnet": "Mainnet workers require explicit approval and dry-run validation.",
            "ai": "AI can advise and score only; AI cannot execute settlements, mint, send funds, or alter balances.",
        },
    }

def validate_queue_worker_plan_v1(plan: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if plan.get("version") != QC_QUEUE_WORKER_PLAN_VERSION:
        errors.append("invalid_version")
    if plan.get("mode") != "PLAN_ONLY_NO_WORKER_START":
        errors.append("invalid_mode")

    queues = plan.get("queues")
    if not isinstance(queues, list) or not queues:
        errors.append("queues_required")
        queues = []

    seen = set()
    for q in queues:
        name = str(q.get("name") or "").strip()
        if not name:
            errors.append("queue_name_required")
            continue
        if name in seen:
            errors.append("duplicate_queue:" + name)
        seen.add(name)

        if not str(q.get("kind") or "").strip():
            errors.append("queue_kind_required:" + name)
        if q.get("testnetSeparated") is not True:
            errors.append("testnet_separation_required:" + name)
        if q.get("mainnetSeparated") is not True:
            errors.append("mainnet_separation_required:" + name)
        if q.get("requiresIdempotency") is not True:
            errors.append("idempotency_required:" + name)
        if q.get("requiresDeadLetter") is not True:
            errors.append("dead_letter_required:" + name)
        if q.get("requiresAuditLog") is not True:
            errors.append("audit_log_required:" + name)

        for forbidden in ["canSendFunds", "canMint", "canChangeBalances", "canChangeChain"]:
            if q.get(forbidden) is True:
                errors.append("unsafe_capability_forbidden:" + name + ":" + forbidden)

    for required in REQUIRED_QUEUES:
        if required not in seen:
            errors.append("missing_required_queue:" + required)

    safe = plan.get("safeBoundary") or {}
    for key in [
        "doesNotStartWorkers",
        "doesNotMint",
        "doesNotSendFunds",
        "doesNotChangeBalances",
        "doesNotChangeChain",
        "doesNotChangeSupply",
        "requiresIdempotency",
        "requiresDeadLetter",
        "requiresAuditLog",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_QUEUE_WORKER_PLAN_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def queue_worker_summary_v1() -> Dict[str, Any]:
    plan = default_queue_worker_plan_v1()
    return {
        "ok": True,
        "version": QC_QUEUE_WORKER_PLAN_VERSION,
        "plan": plan,
        "validation": validate_queue_worker_plan_v1(plan),
    }
