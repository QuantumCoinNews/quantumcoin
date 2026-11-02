# QC_ECOSYSTEM_SCALE_RUNTIME_V1
#
# QuantumCoin ecosystem scale runtime policy.
#
# Scope:
# - Telegram Game
# - TGWT ledger / Watch & Earn / Social Claim / Claim Gateway
# - Monthly NFT review flow
# - Admin panels and security logs
# - QC Node/API boundary
# - Stable Wallet/Miner integration boundary
#
# Purpose:
# - Define production-scale runtime requirements before testnet/mainnet promotion.
# - Require multi-user locks, DB persistence, idempotency and queues/workers.
# - Prevent duplicate rewards and race-condition state changes.
# - Keep heavy/slow work out of direct API request path.
#
# This module does NOT:
# - start real workers
# - create a real DB
# - connect Redis/RabbitMQ/Kafka
# - modify QC consensus
# - modify balances
# - create BSC transactions
# - send TGWT
# - pay gas
# - touch stable release executables

from __future__ import annotations

import hashlib
from typing import Any, Dict, List


QC_ECOSYSTEM_SCALE_RUNTIME_VERSION = "QC_ECOSYSTEM_SCALE_RUNTIME_V1"
QC_ECOSYSTEM_SCALE_RUNTIME_STATUS = "SCALE_POLICY_ONLY_NO_RUNTIME_INFRA_STARTED"

QC_SCALE_SCOPE_QC_NODE_API = "QC_NODE_API"
QC_SCALE_SCOPE_WALLET_MINER = "WALLET_MINER"
QC_SCALE_SCOPE_TELEGRAM_GAME = "TELEGRAM_GAME"
QC_SCALE_SCOPE_TGWT_LEDGER = "TGWT_LEDGER"
QC_SCALE_SCOPE_WATCH_EARN = "WATCH_EARN"
QC_SCALE_SCOPE_SOCIAL_CLAIM = "SOCIAL_CLAIM"
QC_SCALE_SCOPE_CLAIM_GATEWAY = "CLAIM_GATEWAY"
QC_SCALE_SCOPE_MONTHLY_NFT = "MONTHLY_NFT"
QC_SCALE_SCOPE_ADMIN_PANEL = "ADMIN_PANEL"
QC_SCALE_SCOPE_SECURITY_LOGS = "SECURITY_LOGS"

QC_SCALE_OPERATION_ACCEPTED_FOR_QUEUE = "SCALE_OPERATION_ACCEPTED_FOR_QUEUE"
QC_SCALE_OPERATION_DUPLICATE_IDEMPOTENT = "SCALE_OPERATION_DUPLICATE_IDEMPOTENT_RETURN_SAME_RESULT"
QC_SCALE_OPERATION_LOCKED = "SCALE_OPERATION_LOCKED_RETRY_LATER"
QC_SCALE_OPERATION_DB_REQUIRED = "SCALE_OPERATION_DB_PERSISTENCE_REQUIRED"
QC_SCALE_OPERATION_QUEUE_REQUIRED = "SCALE_OPERATION_QUEUE_WORKER_REQUIRED"
QC_SCALE_OPERATION_REJECTED = "SCALE_OPERATION_REJECTED"

QC_SCALE_JOB_PENDING = "SCALE_JOB_PENDING"
QC_SCALE_JOB_PROCESSING = "SCALE_JOB_PROCESSING"
QC_SCALE_JOB_DONE = "SCALE_JOB_DONE"
QC_SCALE_JOB_DEAD_LETTER = "SCALE_JOB_DEAD_LETTER"

QC_SCALE_SUPPORTED_SCOPES = [
    QC_SCALE_SCOPE_QC_NODE_API,
    QC_SCALE_SCOPE_WALLET_MINER,
    QC_SCALE_SCOPE_TELEGRAM_GAME,
    QC_SCALE_SCOPE_TGWT_LEDGER,
    QC_SCALE_SCOPE_WATCH_EARN,
    QC_SCALE_SCOPE_SOCIAL_CLAIM,
    QC_SCALE_SCOPE_CLAIM_GATEWAY,
    QC_SCALE_SCOPE_MONTHLY_NFT,
    QC_SCALE_SCOPE_ADMIN_PANEL,
    QC_SCALE_SCOPE_SECURITY_LOGS,
]

QC_SCALE_CRITICAL_LOCKS = [
    "perUserCriticalSection",
    "perWalletCriticalSection",
    "perClaimCriticalSection",
    "perMiningSessionCriticalSection",
    "perWatchTransactionCriticalSection",
    "perSocialNonceCriticalSection",
    "perNftPeriodCriticalSection",
    "perAdminDecisionCriticalSection",
]

QC_SCALE_REQUIRED_PERSISTENCE_TABLES = [
    "users",
    "wallet_links",
    "mining_sessions",
    "tgwt_ledger_events",
    "watch_earn_events",
    "social_claim_events",
    "claim_requests",
    "claim_settlement_reviews",
    "monthly_nft_candidates",
    "admin_decisions",
    "security_event_logs",
    "rate_limit_counters",
    "idempotency_keys",
    "job_queue",
    "dead_letter_queue",
]

QC_SCALE_SAFE_BOUNDARY = {
    "policyOnly": True,
    "ecosystemWide": True,
    "telegramGameFirst": True,
    "qcNodeApiIncluded": True,
    "walletMinerStableLineProtected": True,
    "multiUserLocksRequired": True,
    "dbPersistenceRequired": True,
    "queueWorkersRequired": True,
    "idempotencyRequired": True,
    "noInMemoryOnlyProduction": True,
    "apiRequestMustNotDoHeavyWork": True,
    "workerRetryPolicyRequired": True,
    "deadLetterQueueRequired": True,
    "auditLogRequired": True,
    "adminDecisionPersistenceRequired": True,
    "securityLogPersistenceRequired": True,
    "rateLimitPersistenceRequired": True,
    "nodeLoadProtectionRequired": True,
    "paginationRequired": True,
    "monitoringRequired": True,
    "perUserCriticalSection": True,
    "perWalletCriticalSection": True,
    "perClaimCriticalSection": True,
    "perMiningSessionCriticalSection": True,
    "perWatchTransactionCriticalSection": True,
    "perSocialNonceCriticalSection": True,
    "perNftPeriodCriticalSection": True,
    "perAdminDecisionCriticalSection": True,
    "doesNotStartWorkers": True,
    "doesNotCreateDatabase": True,
    "doesNotConnectRedis": True,
    "doesNotConnectKafka": True,
    "doesNotConnectRabbitMq": True,
    "doesNotModifyReleaseExecutables": True,
    "changesQCBalances": False,
    "changesTGWTOnChainBalances": False,
    "createsBscTransaction": False,
    "sendsTGWT": False,
    "paysGas": False,
    "storesPrivateKey": False,
    "signsTransaction": False,
    "changesMempool": False,
    "changesChain": False,
    "changesConsensus": False,
    "changesTotalSupply": False,
    "startsMining": False,
    "stopsMining": False,
}


def qc_ecosystem_scale_policy() -> Dict[str, Any]:
    return {
        "version": QC_ECOSYSTEM_SCALE_RUNTIME_VERSION,
        "status": QC_ECOSYSTEM_SCALE_RUNTIME_STATUS,
        "scope": list(QC_SCALE_SUPPORTED_SCOPES),
        "criticalLocks": list(QC_SCALE_CRITICAL_LOCKS),
        "requiredPersistenceTables": list(QC_SCALE_REQUIRED_PERSISTENCE_TABLES),
        "runtimeModel": "API_REQUEST_TO_IDEMPOTENCY_TO_QUEUE_TO_WORKER_TO_DB",
        "telegramGameFirst": True,
        "ecosystemWide": True,
        "stableReleaseLineProtected": True,
        "safeBoundary": dict(QC_SCALE_SAFE_BOUNDARY),
    }


def qc_scale_make_idempotency_key(scope: str, user_id: str, operation: str, external_id: str) -> str:
    scope = _norm(scope)
    user_id = str(user_id or "").strip()
    operation = _norm(operation)
    external_id = str(external_id or "").strip()
    raw = f"{scope}|{user_id}|{operation}|{external_id}"
    digest = hashlib.sha256(raw.encode("utf-8")).hexdigest()
    return f"idem_{digest}"


def qc_scale_lock_key(scope: str, entity_type: str, entity_id: str) -> str:
    scope = _norm(scope)
    entity_type = _norm(entity_type)
    entity_id = str(entity_id or "").strip()
    if not entity_id:
        return ""
    return f"lock:{scope}:{entity_type}:{entity_id}"


def qc_scale_evaluate_operation(
    request: Dict[str, Any],
    runtime_state: Dict[str, Any],
) -> Dict[str, Any]:
    request = dict(request or {})
    runtime_state = dict(runtime_state or {})

    reasons: List[str] = []
    warnings: List[str] = []

    scope = _norm(request.get("scope", ""))
    operation = _norm(request.get("operation", ""))
    user_id = str(request.get("userId", "")).strip()
    external_id = str(request.get("externalId", "")).strip()
    entity_type = _norm(request.get("entityType", "USER"))
    entity_id = str(request.get("entityId", user_id)).strip()

    if scope not in QC_SCALE_SUPPORTED_SCOPES:
        reasons.append("unsupported_scale_scope")

    if not operation:
        reasons.append("missing_operation")

    if not user_id and scope in {
        QC_SCALE_SCOPE_TELEGRAM_GAME,
        QC_SCALE_SCOPE_TGWT_LEDGER,
        QC_SCALE_SCOPE_WATCH_EARN,
        QC_SCALE_SCOPE_SOCIAL_CLAIM,
        QC_SCALE_SCOPE_CLAIM_GATEWAY,
    }:
        reasons.append("missing_user_id")

    if not external_id:
        reasons.append("missing_external_id")

    persistent_db_ready = bool(runtime_state.get("persistentDbReady", False))
    queue_ready = bool(runtime_state.get("queueReady", False))
    worker_ready = bool(runtime_state.get("workerReady", False))

    if persistent_db_ready is not True:
        return _qc_scale_result(
            QC_SCALE_OPERATION_DB_REQUIRED,
            reasons + ["persistent_db_required"],
            warnings,
            request,
            "",
            "",
        )

    if queue_ready is not True or worker_ready is not True:
        return _qc_scale_result(
            QC_SCALE_OPERATION_QUEUE_REQUIRED,
            reasons + ["queue_worker_required"],
            warnings,
            request,
            "",
            "",
        )

    idem_key = str(request.get("idempotencyKey", "")).strip()
    if not idem_key:
        idem_key = qc_scale_make_idempotency_key(scope, user_id, operation, external_id)

    seen_keys = set(str(x).strip() for x in runtime_state.get("seenIdempotencyKeys", []) or [])
    if idem_key in seen_keys:
        return _qc_scale_result(
            QC_SCALE_OPERATION_DUPLICATE_IDEMPOTENT,
            ["duplicate_idempotency_key_return_same_result"],
            warnings,
            request,
            idem_key,
            "",
        )

    lock_key = str(request.get("lockKey", "")).strip()
    if not lock_key:
        lock_key = qc_scale_lock_key(scope, entity_type, entity_id)

    active_locks = set(str(x).strip() for x in runtime_state.get("activeLocks", []) or [])
    if lock_key and lock_key in active_locks:
        return _qc_scale_result(
            QC_SCALE_OPERATION_LOCKED,
            ["critical_section_locked_retry_later"],
            warnings,
            request,
            idem_key,
            lock_key,
        )

    if reasons:
        return _qc_scale_result(
            QC_SCALE_OPERATION_REJECTED,
            reasons,
            warnings,
            request,
            idem_key,
            lock_key,
        )

    return _qc_scale_result(
        QC_SCALE_OPERATION_ACCEPTED_FOR_QUEUE,
        ["accepted_for_queue_worker_processing"],
        warnings,
        request,
        idem_key,
        lock_key,
    )


def qc_scale_create_job(request: Dict[str, Any], decision: Dict[str, Any], now_ts: int = 0) -> Dict[str, Any]:
    request = dict(request or {})
    decision = dict(decision or {})

    if str(decision.get("status", "")) != QC_SCALE_OPERATION_ACCEPTED_FOR_QUEUE:
        return {
            "version": QC_ECOSYSTEM_SCALE_RUNTIME_VERSION,
            "status": QC_SCALE_OPERATION_REJECTED,
            "reasons": ["job_not_created_without_accepted_scale_decision"],
            "safeBoundary": dict(QC_SCALE_SAFE_BOUNDARY),
        }

    job_id = qc_scale_make_idempotency_key(
        str(request.get("scope", "")),
        str(request.get("userId", "")),
        str(request.get("operation", "")),
        str(request.get("externalId", "")),
    ).replace("idem_", "job_")

    return {
        "version": QC_ECOSYSTEM_SCALE_RUNTIME_VERSION,
        "status": QC_SCALE_JOB_PENDING,
        "jobId": job_id,
        "scope": _norm(request.get("scope", "")),
        "operation": _norm(request.get("operation", "")),
        "userId": str(request.get("userId", "")).strip(),
        "externalId": str(request.get("externalId", "")).strip(),
        "idempotencyKey": str(decision.get("idempotencyKey", "")).strip(),
        "lockKey": str(decision.get("lockKey", "")).strip(),
        "createdAt": int(now_ts or 0),
        "maxAttempts": int(request.get("maxAttempts", 3) or 3),
        "requiresPersistentDb": True,
        "requiresQueueWorker": True,
        "requiresDeadLetterQueue": True,
        "safeBoundary": dict(QC_SCALE_SAFE_BOUNDARY),
    }


def qc_scale_dead_letter_job(job: Dict[str, Any], reason: str) -> Dict[str, Any]:
    job = dict(job or {})
    return {
        "version": QC_ECOSYSTEM_SCALE_RUNTIME_VERSION,
        "status": QC_SCALE_JOB_DEAD_LETTER,
        "jobId": str(job.get("jobId", "")).strip(),
        "reason": str(reason or "").strip() or "dead_letter_reason_required",
        "requiresAdminReview": True,
        "safeBoundary": dict(QC_SCALE_SAFE_BOUNDARY),
    }


def _qc_scale_result(
    status: str,
    reasons: List[str],
    warnings: List[str],
    request: Dict[str, Any],
    idempotency_key: str,
    lock_key: str,
) -> Dict[str, Any]:
    return {
        "version": QC_ECOSYSTEM_SCALE_RUNTIME_VERSION,
        "status": status,
        "reasons": list(reasons or []),
        "warnings": list(warnings or []),
        "scope": _norm(request.get("scope", "")),
        "operation": _norm(request.get("operation", "")),
        "userId": str(request.get("userId", "")).strip(),
        "externalId": str(request.get("externalId", "")).strip(),
        "idempotencyKey": idempotency_key,
        "lockKey": lock_key,
        "acceptedForQueue": status == QC_SCALE_OPERATION_ACCEPTED_FOR_QUEUE,
        "duplicateRequest": status == QC_SCALE_OPERATION_DUPLICATE_IDEMPOTENT,
        "locked": status == QC_SCALE_OPERATION_LOCKED,
        "requiresPersistentDb": True,
        "requiresQueueWorker": True,
        "requiresIdempotency": True,
        "requiresCriticalSection": True,
        "safeBoundary": dict(QC_SCALE_SAFE_BOUNDARY),
    }


def _norm(value: Any) -> str:
    return str(value or "").strip().upper()
