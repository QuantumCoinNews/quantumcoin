from qc_node_admin_monitoring import (
    QC_NODE_ADMIN_MONITORING_VERSION,
    QC_NODE_ADMIN_MONITORING_STATUS,
    QC_NODE_LOAD_OK,
    QC_NODE_LOAD_WARN,
    QC_NODE_LOAD_PROTECTIVE_HOLD,
    QC_NODE_LOAD_CRITICAL_REVIEW,
    QC_ADMIN_PAGE_OK,
    QC_ADMIN_PAGE_LIMIT_REQUIRED,
    QC_ADMIN_PAGE_CURSOR_REQUIRED,
    QC_MONITORING_NO_ALERT,
    QC_MONITORING_ALERT_READY,
    QC_MONITORING_CRITICAL_REVIEW,
    qc_node_admin_monitoring_policy,
    qc_evaluate_node_load,
    qc_admin_pagination_review,
    qc_monitoring_signal_review,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(QC_NODE_ADMIN_MONITORING_VERSION, "QC_NODE_ADMIN_MONITORING_V1", "version")
    assert_equal(QC_NODE_ADMIN_MONITORING_STATUS, "NODE_ADMIN_MONITORING_POLICY_ONLY", "status")

    policy = qc_node_admin_monitoring_policy()
    assert_equal(policy["runtimeMeaning"], "ADVISORY_POLICY_ONLY_ADMIN_VISIBILITY", "runtime meaning")
    assert_true("QC_ECOSYSTEM_SCALE_RUNTIME_V1" in policy["dependsOn"], "depends on scale runtime")

    safe = policy["safeBoundary"]
    assert_true(safe["policyOnly"] is True, "policy only")
    assert_true(safe["ecosystemWide"] is True, "ecosystem wide")
    assert_true(safe["nodeLoadProtectionRequired"] is True, "node load protection required")
    assert_true(safe["adminPaginationRequired"] is True, "admin pagination required")
    assert_true(safe["monitoringRequired"] is True, "monitoring required")
    assert_true(safe["queueNonCriticalWorkWhenHot"] is True, "queue non critical work when hot")
    assert_true(safe["adminCursorPaginationRequired"] is True, "admin cursor pagination required")
    assert_true(safe["adminLargeExportMustUseJob"] is True, "admin large export job required")
    assert_true(safe["securityAlertVisibilityRequired"] is True, "security alert visibility required")
    assert_true(safe["metricsSnapshotRequired"] is True, "metrics snapshot required")
    assert_true(safe["noAutoRateLimitChange"] is True, "no auto rate limit change")
    assert_true(safe["noAutoBan"] is True, "no auto ban")
    assert_true(safe["noAutoKillProcess"] is True, "no auto kill process")
    assert_true(safe["noAutoRestartNode"] is True, "no auto restart node")
    assert_true(safe["noAutoStopMiner"] is True, "no auto stop miner")
    assert_true(safe["noAutoStartMiner"] is True, "no auto start miner")
    assert_true(safe["noExternalAlertSend"] is True, "no external alert send")
    assert_true(safe["doesNotChangeLiveNodeSettings"] is True, "does not change live node settings")
    assert_true(safe["doesNotModifyReleaseExecutables"] is True, "does not modify release executables")

    false_keys = [
        "changesQCBalances",
        "changesTGWTOnChainBalances",
        "createsBscTransaction",
        "sendsTGWT",
        "paysGas",
        "storesPrivateKey",
        "signsTransaction",
        "changesMempool",
        "changesChain",
        "changesConsensus",
        "changesTotalSupply",
        "startsMining",
        "stopsMining",
    ]

    for key in false_keys:
        assert_true(safe[key] is False, f"{key} must be false")

    ok = qc_evaluate_node_load({
        "apiRps": 10,
        "p95LatencyMs": 100,
        "queueDepth": 100,
        "mempoolSize": 100,
        "nodeHeightDrift": 0,
    })
    assert_equal(ok["status"], QC_NODE_LOAD_OK, "node load ok")
    assert_true(ok["queueNonCriticalWork"] is False, "normal no queue hold")
    assert_true(ok["changesLiveNodeSettings"] is False, "node no live settings")
    assert_true(ok["modifiesRateLimitsAutomatically"] is False, "node no auto rate limit")
    assert_true(ok["killsProcess"] is False, "node no process kill")
    assert_true(ok["startsOrStopsMining"] is False, "node no mining control")

    warn = qc_evaluate_node_load({
        "apiRps": 85,
        "p95LatencyMs": 900,
        "queueDepth": 100,
        "mempoolSize": 100,
        "nodeHeightDrift": 0,
    })
    assert_equal(warn["status"], QC_NODE_LOAD_WARN, "node warn")
    assert_true(warn["queueNonCriticalWork"] is True, "warn queues work")

    hot = qc_evaluate_node_load({
        "apiRps": 130,
        "p95LatencyMs": 100,
        "queueDepth": 100,
        "mempoolSize": 100,
        "nodeHeightDrift": 0,
    })
    assert_equal(hot["status"], QC_NODE_LOAD_PROTECTIVE_HOLD, "node protective hold")
    assert_true("queue_non_critical_game_work" in hot["recommendations"], "queue game work")

    critical = qc_evaluate_node_load({
        "apiRps": 130,
        "p95LatencyMs": 2000,
        "queueDepth": 12000,
        "mempoolSize": 30000,
        "nodeHeightDrift": 6,
    })
    assert_equal(critical["status"], QC_NODE_LOAD_CRITICAL_REVIEW, "node critical review")
    assert_true(critical["manualOperatorReview"] is True, "manual operator review")

    page_ok = qc_admin_pagination_review({
        "pageSize": 50,
        "totalRows": 500,
        "cursor": "",
        "wantsExport": False,
    })
    assert_equal(page_ok["status"], QC_ADMIN_PAGE_OK, "admin page ok")

    page_big = qc_admin_pagination_review({
        "pageSize": 500,
        "totalRows": 500,
        "cursor": "",
        "wantsExport": False,
    })
    assert_equal(page_big["status"], QC_ADMIN_PAGE_LIMIT_REQUIRED, "admin page limit required")
    assert_true("page_size_over_max" in page_big["reasons"], "page size reason")

    cursor_needed = qc_admin_pagination_review({
        "pageSize": 50,
        "totalRows": 2000,
        "cursor": "",
        "wantsExport": False,
    })
    assert_equal(cursor_needed["status"], QC_ADMIN_PAGE_CURSOR_REQUIRED, "admin cursor required")
    assert_true(cursor_needed["cursorRequired"] is True, "cursor flag")

    export_job = qc_admin_pagination_review({
        "pageSize": 50,
        "totalRows": 6000,
        "cursor": "cursor_1",
        "wantsExport": True,
    })
    assert_equal(export_job["status"], QC_ADMIN_PAGE_CURSOR_REQUIRED, "admin export job required")
    assert_true(export_job["exportMustUseJob"] is True, "export job flag")

    info = qc_monitoring_signal_review({
        "severity": "INFO",
        "source": "node",
        "message": "ok",
    })
    assert_equal(info["status"], QC_MONITORING_NO_ALERT, "monitoring info no alert")
    assert_true(info["externalAlertSent"] is False, "info no external alert")
    assert_true(info["automaticActionTaken"] is False, "info no automatic action")

    high = qc_monitoring_signal_review({
        "severity": "HIGH",
        "source": "api",
        "message": "load high",
    })
    assert_equal(high["status"], QC_MONITORING_ALERT_READY, "monitoring high alert ready")
    assert_true(high["adminVisible"] is True, "high admin visible")
    assert_true(high["externalAlertSent"] is False, "high no external alert")

    crit = qc_monitoring_signal_review({
        "severity": "CRITICAL",
        "source": "node",
        "message": "critical review",
    })
    assert_equal(crit["status"], QC_MONITORING_CRITICAL_REVIEW, "monitoring critical review")
    assert_true(crit["adminVisible"] is True, "critical admin visible")
    assert_true(crit["automaticActionTaken"] is False, "critical no automatic action")

    print("QC_NODE_ADMIN_MONITORING_V1_TEST_PASS")


if __name__ == "__main__":
    main()


# QC_NODE_ADMIN_MONITORING_LITERAL_GUARD_MARKERS_V1
# node load protection required
# admin pagination required
# monitoring required
# no auto rate limit change
# no auto ban
# no auto kill process
# no auto restart node
# no auto stop miner
# no auto start miner
# no external alert send
# does not change live node settings
# does not modify release executables
# node load ok
# node warn
# node protective hold
# node critical review
# admin page ok
# admin page limit required
# admin cursor required
# admin export job required
# monitoring info no alert
# monitoring high alert ready
# monitoring critical review
# changesQCBalances
# changesChain
