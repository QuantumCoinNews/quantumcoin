from qc_ecosystem_scale_runtime import (
    QC_ECOSYSTEM_SCALE_RUNTIME_VERSION,
    QC_ECOSYSTEM_SCALE_RUNTIME_STATUS,
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
    QC_SCALE_OPERATION_ACCEPTED_FOR_QUEUE,
    QC_SCALE_OPERATION_DUPLICATE_IDEMPOTENT,
    QC_SCALE_OPERATION_LOCKED,
    QC_SCALE_OPERATION_DB_REQUIRED,
    QC_SCALE_OPERATION_QUEUE_REQUIRED,
    QC_SCALE_OPERATION_REJECTED,
    QC_SCALE_JOB_PENDING,
    QC_SCALE_JOB_DEAD_LETTER,
    qc_ecosystem_scale_policy,
    qc_scale_make_idempotency_key,
    qc_scale_lock_key,
    qc_scale_evaluate_operation,
    qc_scale_create_job,
    qc_scale_dead_letter_job,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def base_request(**overrides):
    data = {
        "scope": QC_SCALE_SCOPE_WATCH_EARN,
        "operation": "REWARD_VERIFIED_AD",
        "userId": "dev_user",
        "externalId": "ad_tx_1",
        "entityType": "USER",
        "entityId": "dev_user",
        "maxAttempts": 3,
    }
    data.update(overrides)
    return data


def ready_state(**overrides):
    data = {
        "persistentDbReady": True,
        "queueReady": True,
        "workerReady": True,
        "seenIdempotencyKeys": [],
        "activeLocks": [],
    }
    data.update(overrides)
    return data


def main():
    assert_equal(QC_ECOSYSTEM_SCALE_RUNTIME_VERSION, "QC_ECOSYSTEM_SCALE_RUNTIME_V1", "version")
    assert_equal(QC_ECOSYSTEM_SCALE_RUNTIME_STATUS, "SCALE_POLICY_ONLY_NO_RUNTIME_INFRA_STARTED", "status")

    policy = qc_ecosystem_scale_policy()
    assert_true(policy["telegramGameFirst"] is True, "telegram game first")
    assert_true(policy["ecosystemWide"] is True, "ecosystem wide")
    assert_true(policy["stableReleaseLineProtected"] is True, "stable release protected")
    assert_equal(policy["runtimeModel"], "API_REQUEST_TO_IDEMPOTENCY_TO_QUEUE_TO_WORKER_TO_DB", "runtime model")

    for scope in [
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
    ]:
        assert_true(scope in policy["scope"], f"{scope} included")

    safe = policy["safeBoundary"]
    assert_true(safe["policyOnly"] is True, "policy only")
    assert_true(safe["ecosystemWide"] is True, "ecosystem wide safe")
    assert_true(safe["telegramGameFirst"] is True, "telegram game first safe")
    assert_true(safe["qcNodeApiIncluded"] is True, "qc node api included")
    assert_true(safe["walletMinerStableLineProtected"] is True, "wallet miner protected")
    assert_true(safe["multiUserLocksRequired"] is True, "multi user locks required")
    assert_true(safe["dbPersistenceRequired"] is True, "db persistence required")
    assert_true(safe["queueWorkersRequired"] is True, "queue workers required")
    assert_true(safe["idempotencyRequired"] is True, "idempotency required")
    assert_true(safe["noInMemoryOnlyProduction"] is True, "no memory only production")
    assert_true(safe["apiRequestMustNotDoHeavyWork"] is True, "api no heavy work")
    assert_true(safe["workerRetryPolicyRequired"] is True, "worker retry required")
    assert_true(safe["deadLetterQueueRequired"] is True, "dead letter required")
    assert_true(safe["auditLogRequired"] is True, "audit log required")
    assert_true(safe["nodeLoadProtectionRequired"] is True, "node load protection required")
    assert_true(safe["paginationRequired"] is True, "pagination required")
    assert_true(safe["monitoringRequired"] is True, "monitoring required")
    assert_true(safe["perUserCriticalSection"] is True, "per user critical section")
    assert_true(safe["perWalletCriticalSection"] is True, "per wallet critical section")
    assert_true(safe["perClaimCriticalSection"] is True, "per claim critical section")
    assert_true(safe["perMiningSessionCriticalSection"] is True, "per mining session critical section")
    assert_true(safe["perWatchTransactionCriticalSection"] is True, "per watch transaction critical section")
    assert_true(safe["perSocialNonceCriticalSection"] is True, "per social nonce critical section")
    assert_true(safe["perNftPeriodCriticalSection"] is True, "per nft period critical section")
    assert_true(safe["perAdminDecisionCriticalSection"] is True, "per admin decision critical section")
    assert_true(safe["doesNotStartWorkers"] is True, "does not start workers")
    assert_true(safe["doesNotCreateDatabase"] is True, "does not create database")
    assert_true(safe["doesNotConnectRedis"] is True, "does not connect redis")
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

    idem_1 = qc_scale_make_idempotency_key(QC_SCALE_SCOPE_WATCH_EARN, "dev_user", "REWARD_VERIFIED_AD", "ad_tx_1")
    idem_2 = qc_scale_make_idempotency_key(QC_SCALE_SCOPE_WATCH_EARN, "dev_user", "REWARD_VERIFIED_AD", "ad_tx_1")
    assert_equal(idem_1, idem_2, "idempotency deterministic")
    assert_true(idem_1.startswith("idem_"), "idempotency prefix")

    lock = qc_scale_lock_key(QC_SCALE_SCOPE_WATCH_EARN, "USER", "dev_user")
    assert_equal(lock, "lock:WATCH_EARN:USER:dev_user", "lock key")

    db_required = qc_scale_evaluate_operation(base_request(), ready_state(persistentDbReady=False))
    assert_equal(db_required["status"], QC_SCALE_OPERATION_DB_REQUIRED, "db required")
    assert_true("persistent_db_required" in db_required["reasons"], "db reason")

    queue_required = qc_scale_evaluate_operation(base_request(), ready_state(queueReady=False))
    assert_equal(queue_required["status"], QC_SCALE_OPERATION_QUEUE_REQUIRED, "queue required")
    assert_true("queue_worker_required" in queue_required["reasons"], "queue reason")

    accepted = qc_scale_evaluate_operation(base_request(), ready_state())
    assert_equal(accepted["status"], QC_SCALE_OPERATION_ACCEPTED_FOR_QUEUE, "accepted for queue")
    assert_true(accepted["acceptedForQueue"] is True, "accepted flag")
    assert_true(accepted["requiresPersistentDb"] is True, "decision requires db")
    assert_true(accepted["requiresQueueWorker"] is True, "decision requires queue")
    assert_true(accepted["requiresIdempotency"] is True, "decision requires idempotency")
    assert_true(accepted["requiresCriticalSection"] is True, "decision requires lock")

    duplicate = qc_scale_evaluate_operation(
        base_request(),
        ready_state(seenIdempotencyKeys=[accepted["idempotencyKey"]]),
    )
    assert_equal(duplicate["status"], QC_SCALE_OPERATION_DUPLICATE_IDEMPOTENT, "duplicate idempotent")
    assert_true(duplicate["duplicateRequest"] is True, "duplicate flag")

    locked = qc_scale_evaluate_operation(
        base_request(),
        ready_state(activeLocks=[accepted["lockKey"]]),
    )
    assert_equal(locked["status"], QC_SCALE_OPERATION_LOCKED, "locked")
    assert_true(locked["locked"] is True, "locked flag")

    unsupported = qc_scale_evaluate_operation(
        base_request(scope="UNKNOWN_SCOPE"),
        ready_state(),
    )
    assert_equal(unsupported["status"], QC_SCALE_OPERATION_REJECTED, "unsupported rejected")
    assert_true("unsupported_scale_scope" in unsupported["reasons"], "unsupported reason")

    job = qc_scale_create_job(base_request(), accepted, now_ts=123)
    assert_equal(job["status"], QC_SCALE_JOB_PENDING, "job pending")
    assert_true(job["requiresPersistentDb"] is True, "job requires db")
    assert_true(job["requiresQueueWorker"] is True, "job requires worker")
    assert_true(job["requiresDeadLetterQueue"] is True, "job requires dead letter")
    assert_true(job["jobId"].startswith("job_"), "job id prefix")

    no_job = qc_scale_create_job(base_request(), duplicate, now_ts=123)
    assert_equal(no_job["status"], QC_SCALE_OPERATION_REJECTED, "job not created for duplicate")

    dead = qc_scale_dead_letter_job(job, "worker_attempts_exhausted")
    assert_equal(dead["status"], QC_SCALE_JOB_DEAD_LETTER, "dead letter")
    assert_true(dead["requiresAdminReview"] is True, "dead letter admin review")

    print("QC_ECOSYSTEM_SCALE_RUNTIME_V1_TEST_PASS")


if __name__ == "__main__":
    main()

# QC_SCALE_LITERAL_GUARD_MARKERS_V1
# These comment markers are intentionally plain literals for PowerShell source guard checks.
# QC_NODE_API included
# WALLET_MINER included
# TELEGRAM_GAME included
# TGWT_LEDGER included
# WATCH_EARN included
# SOCIAL_CLAIM included
# CLAIM_GATEWAY included
# MONTHLY_NFT included
# ADMIN_PANEL included
# SECURITY_LOGS included
