# QC_NODE_ADMIN_MONITORING_V1
#
# QuantumCoin node/API load protection, admin pagination and monitoring policy.
#
# Scope:
# - QC Node/API
# - Telegram Game API boundary
# - Admin lists and review pages
# - Security logs and operational alerts
# - Stable Wallet/Miner integration boundary
#
# Purpose:
# - Define safe load-protection decisions as advisory/policy results.
# - Define admin pagination constraints for large data sets.
# - Define monitoring alert readiness without sending alerts automatically.
# - Build on QC_ECOSYSTEM_SCALE_RUNTIME_V1.
#
# This module does NOT:
# - change live node settings
# - modify rate limits automatically
# - ban IPs automatically
# - kill processes
# - start or stop miners
# - create real monitoring integrations
# - send external alerts
# - modify QC consensus
# - modify balances
# - modify stable release executables

from __future__ import annotations

from typing import Any, Dict, List

from qc_ecosystem_scale_runtime import QC_ECOSYSTEM_SCALE_RUNTIME_VERSION


QC_NODE_ADMIN_MONITORING_VERSION = "QC_NODE_ADMIN_MONITORING_V1"
QC_NODE_ADMIN_MONITORING_STATUS = "NODE_ADMIN_MONITORING_POLICY_ONLY"

QC_NODE_LOAD_OK = "NODE_LOAD_OK"
QC_NODE_LOAD_WARN = "NODE_LOAD_WARN_REDUCE_NON_CRITICAL_WORK"
QC_NODE_LOAD_PROTECTIVE_HOLD = "NODE_LOAD_PROTECTIVE_HOLD_QUEUE_NON_CRITICAL_WORK"
QC_NODE_LOAD_CRITICAL_REVIEW = "NODE_LOAD_CRITICAL_ADMIN_REVIEW_REQUIRED"

QC_ADMIN_PAGE_OK = "ADMIN_PAGE_OK"
QC_ADMIN_PAGE_LIMIT_REQUIRED = "ADMIN_PAGE_LIMIT_REQUIRED"
QC_ADMIN_PAGE_CURSOR_REQUIRED = "ADMIN_PAGE_CURSOR_REQUIRED"

QC_MONITORING_NO_ALERT = "MONITORING_NO_ALERT"
QC_MONITORING_ALERT_READY = "MONITORING_ALERT_READY_ADMIN_VISIBILITY"
QC_MONITORING_CRITICAL_REVIEW = "MONITORING_CRITICAL_REVIEW_REQUIRED"

QC_DEFAULT_NODE_LIMITS = {
    "maxApiRps": 120,
    "warnApiRps": 80,
    "maxP95LatencyMs": 1500,
    "warnP95LatencyMs": 800,
    "maxQueueDepth": 10000,
    "warnQueueDepth": 5000,
    "maxMempoolSize": 25000,
    "warnMempoolSize": 10000,
    "maxNodeHeightDrift": 5,
    "warnNodeHeightDrift": 2,
}

QC_DEFAULT_ADMIN_PAGE_LIMITS = {
    "defaultPageSize": 50,
    "maxPageSize": 100,
    "cursorRequiredAfterRows": 1000,
    "maxExportRowsWithoutJob": 5000,
}

QC_MONITORING_SEVERITIES = ["INFO", "WARN", "HIGH", "CRITICAL"]

QC_NODE_ADMIN_MONITORING_SAFE_BOUNDARY = {
    "policyOnly": True,
    "ecosystemWide": True,
    "dependsOnScaleRuntime": QC_ECOSYSTEM_SCALE_RUNTIME_VERSION,
    "nodeLoadProtectionRequired": True,
    "adminPaginationRequired": True,
    "monitoringRequired": True,
    "queueNonCriticalWorkWhenHot": True,
    "adminCursorPaginationRequired": True,
    "adminLargeExportMustUseJob": True,
    "securityAlertVisibilityRequired": True,
    "metricsSnapshotRequired": True,
    "noAutoRateLimitChange": True,
    "noAutoBan": True,
    "noAutoKillProcess": True,
    "noAutoRestartNode": True,
    "noAutoStopMiner": True,
    "noAutoStartMiner": True,
    "noExternalAlertSend": True,
    "doesNotChangeLiveNodeSettings": True,
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


def qc_node_admin_monitoring_policy() -> Dict[str, Any]:
    return {
        "version": QC_NODE_ADMIN_MONITORING_VERSION,
        "status": QC_NODE_ADMIN_MONITORING_STATUS,
        "dependsOn": [QC_ECOSYSTEM_SCALE_RUNTIME_VERSION],
        "nodeLimits": dict(QC_DEFAULT_NODE_LIMITS),
        "adminPageLimits": dict(QC_DEFAULT_ADMIN_PAGE_LIMITS),
        "monitoringSeverities": list(QC_MONITORING_SEVERITIES),
        "runtimeMeaning": "ADVISORY_POLICY_ONLY_ADMIN_VISIBILITY",
        "safeBoundary": dict(QC_NODE_ADMIN_MONITORING_SAFE_BOUNDARY),
    }


def qc_evaluate_node_load(metrics: Dict[str, Any], limits: Dict[str, Any] | None = None) -> Dict[str, Any]:
    metrics = dict(metrics or {})
    limits = dict(limits or QC_DEFAULT_NODE_LIMITS)

    reasons: List[str] = []
    recommendations: List[str] = []

    api_rps = _num(metrics.get("apiRps", 0))
    p95_latency_ms = _num(metrics.get("p95LatencyMs", 0))
    queue_depth = _num(metrics.get("queueDepth", 0))
    mempool_size = _num(metrics.get("mempoolSize", 0))
    height_drift = _num(metrics.get("nodeHeightDrift", 0))

    if api_rps >= _num(limits.get("maxApiRps", 120)):
        reasons.append("api_rps_over_max")
    elif api_rps >= _num(limits.get("warnApiRps", 80)):
        reasons.append("api_rps_warn")

    if p95_latency_ms >= _num(limits.get("maxP95LatencyMs", 1500)):
        reasons.append("p95_latency_over_max")
    elif p95_latency_ms >= _num(limits.get("warnP95LatencyMs", 800)):
        reasons.append("p95_latency_warn")

    if queue_depth >= _num(limits.get("maxQueueDepth", 10000)):
        reasons.append("queue_depth_over_max")
    elif queue_depth >= _num(limits.get("warnQueueDepth", 5000)):
        reasons.append("queue_depth_warn")

    if mempool_size >= _num(limits.get("maxMempoolSize", 25000)):
        reasons.append("mempool_size_over_max")
    elif mempool_size >= _num(limits.get("warnMempoolSize", 10000)):
        reasons.append("mempool_size_warn")

    if height_drift >= _num(limits.get("maxNodeHeightDrift", 5)):
        reasons.append("node_height_drift_over_max")
    elif height_drift >= _num(limits.get("warnNodeHeightDrift", 2)):
        reasons.append("node_height_drift_warn")

    over_max = [r for r in reasons if r.endswith("_over_max")]
    warns = [r for r in reasons if r.endswith("_warn")]

    if over_max:
        status = QC_NODE_LOAD_PROTECTIVE_HOLD
        recommendations.extend([
            "queue_non_critical_game_work",
            "slow_admin_exports_to_background_jobs",
            "show_admin_load_warning",
            "keep_consensus_and_balances_unchanged",
        ])
    elif warns:
        status = QC_NODE_LOAD_WARN
        recommendations.extend([
            "prefer_queue_worker_for_non_critical_work",
            "reduce_admin_page_size_if_needed",
            "watch_node_metrics",
        ])
    else:
        status = QC_NODE_LOAD_OK
        reasons.append("node_load_within_policy_limits")
        recommendations.append("normal_operation")

    if len(over_max) >= 3:
        status = QC_NODE_LOAD_CRITICAL_REVIEW
        recommendations.append("manual_operator_review_required")

    return {
        "version": QC_NODE_ADMIN_MONITORING_VERSION,
        "status": status,
        "reasons": reasons,
        "recommendations": recommendations,
        "metrics": {
            "apiRps": api_rps,
            "p95LatencyMs": p95_latency_ms,
            "queueDepth": queue_depth,
            "mempoolSize": mempool_size,
            "nodeHeightDrift": height_drift,
        },
        "queueNonCriticalWork": status in {QC_NODE_LOAD_WARN, QC_NODE_LOAD_PROTECTIVE_HOLD, QC_NODE_LOAD_CRITICAL_REVIEW},
        "manualOperatorReview": status == QC_NODE_LOAD_CRITICAL_REVIEW,
        "changesLiveNodeSettings": False,
        "modifiesRateLimitsAutomatically": False,
        "killsProcess": False,
        "startsOrStopsMining": False,
        "safeBoundary": dict(QC_NODE_ADMIN_MONITORING_SAFE_BOUNDARY),
    }


def qc_admin_pagination_review(request: Dict[str, Any], limits: Dict[str, Any] | None = None) -> Dict[str, Any]:
    request = dict(request or {})
    limits = dict(limits or QC_DEFAULT_ADMIN_PAGE_LIMITS)

    reasons: List[str] = []
    page_size = int(_num(request.get("pageSize", limits.get("defaultPageSize", 50))))
    total_rows = int(_num(request.get("totalRows", 0)))
    cursor = str(request.get("cursor", "")).strip()
    wants_export = bool(request.get("wantsExport", False))

    max_page_size = int(_num(limits.get("maxPageSize", 100)))
    cursor_required_after_rows = int(_num(limits.get("cursorRequiredAfterRows", 1000)))
    max_export_rows_without_job = int(_num(limits.get("maxExportRowsWithoutJob", 5000)))

    if page_size <= 0:
        reasons.append("page_size_must_be_positive")

    if page_size > max_page_size:
        reasons.append("page_size_over_max")

    if total_rows > cursor_required_after_rows and not cursor:
        reasons.append("cursor_required_for_large_admin_list")

    if wants_export and total_rows > max_export_rows_without_job:
        reasons.append("large_export_must_use_background_job")

    if "page_size_over_max" in reasons or "page_size_must_be_positive" in reasons:
        status = QC_ADMIN_PAGE_LIMIT_REQUIRED
    elif reasons:
        status = QC_ADMIN_PAGE_CURSOR_REQUIRED
    else:
        status = QC_ADMIN_PAGE_OK
        reasons.append("admin_page_request_within_policy_limits")

    return {
        "version": QC_NODE_ADMIN_MONITORING_VERSION,
        "status": status,
        "reasons": reasons,
        "pageSize": page_size,
        "maxPageSize": max_page_size,
        "totalRows": total_rows,
        "cursorRequired": total_rows > cursor_required_after_rows,
        "exportMustUseJob": wants_export and total_rows > max_export_rows_without_job,
        "requiresPagination": True,
        "requiresCursorForLargeList": True,
        "requiresBackgroundJobForLargeExport": True,
        "safeBoundary": dict(QC_NODE_ADMIN_MONITORING_SAFE_BOUNDARY),
    }


def qc_monitoring_signal_review(signal: Dict[str, Any]) -> Dict[str, Any]:
    signal = dict(signal or {})
    severity = str(signal.get("severity", "INFO")).strip().upper()
    source = str(signal.get("source", "")).strip() or "UNKNOWN_SOURCE"
    message = str(signal.get("message", "")).strip() or "no_message"

    if severity not in QC_MONITORING_SEVERITIES:
        severity = "INFO"

    if severity == "CRITICAL":
        status = QC_MONITORING_CRITICAL_REVIEW
        reasons = ["critical_signal_requires_manual_review"]
    elif severity in {"WARN", "HIGH"}:
        status = QC_MONITORING_ALERT_READY
        reasons = ["monitoring_alert_ready_for_admin_visibility"]
    else:
        status = QC_MONITORING_NO_ALERT
        reasons = ["monitoring_signal_record_only"]

    return {
        "version": QC_NODE_ADMIN_MONITORING_VERSION,
        "status": status,
        "reasons": reasons,
        "severity": severity,
        "source": source,
        "message": message,
        "adminVisible": status in {QC_MONITORING_ALERT_READY, QC_MONITORING_CRITICAL_REVIEW},
        "externalAlertSent": False,
        "automaticActionTaken": False,
        "safeBoundary": dict(QC_NODE_ADMIN_MONITORING_SAFE_BOUNDARY),
    }


def _num(value: Any) -> float:
    try:
        return float(value)
    except Exception:
        return 0.0
