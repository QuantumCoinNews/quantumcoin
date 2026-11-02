from tgwt_pool_policy import TGWT_POOL_TOTAL_U, TGWT_UNIT
from tgwt_ledger_events import (
    TGWT_LEDGER_EVENTS_VERSION,
    TGWT_LEDGER_STATUS_ACTIVE,
    TGWT_EVENT_EARN_PENDING,
    TGWT_EVENT_VERIFY_PENDING,
    TGWT_EVENT_CLAIM_CONFIRMED,
    TGWT_EVENT_ADMIN_HOLD,
    TGWT_EVENT_REJECT,
    tgwt_amount_to_u,
    tgwt_amount_from_u,
    tgwt_ledger_event,
    tgwt_ledger_initial,
    tgwt_apply_event,
    tgwt_build_ledger_from_events,
    tgwt_append_event_log,
    tgwt_ledger_policy,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(TGWT_LEDGER_EVENTS_VERSION, "TGWT_LEDGER_EVENTS_V1", "version")
    assert_equal(TGWT_LEDGER_STATUS_ACTIVE, "OFF_CHAIN_LEDGER_ACTIVE_NO_BSC_TRANSFER", "status")

    assert_equal(tgwt_amount_to_u("1"), TGWT_UNIT, "1 TGWT to micro")
    assert_equal(tgwt_amount_to_u("1.25"), 1_250_000, "1.25 TGWT to micro")
    assert_equal(tgwt_amount_from_u(1_250_000), "1.25", "micro to amount")

    policy = tgwt_ledger_policy()
    assert_equal(policy["version"], "TGWT_LEDGER_EVENTS_V1", "policy version")
    assert_equal(policy["claimModel"], "USER_PAID_BSC_CLAIM_LATER", "claim model")

    safe = policy["safeBoundary"]
    assert_true(safe["offChainOnly"] is True, "offChainOnly")
    assert_true(safe["recordsRightsOnly"] is True, "recordsRightsOnly")
    assert_true(safe["userPaysBscFee"] is True, "userPaysBscFee")

    false_keys = [
        "createsBscTransaction",
        "sendsTGWT",
        "autoPayout",
        "storesPrivateKey",
        "signsTransaction",
        "paysGas",
        "usesHotWallet",
        "usesTreasuryWallet",
        "mintsToken",
        "burnsToken",
        "changesQCBalances",
        "changesTGWTOnChainBalances",
        "changesMempool",
        "changesChain",
        "replacesChain",
        "changesConsensus",
        "changesTotalSupply",
        "startsMining",
        "stopsMining",
    ]

    for key in false_keys:
        assert_true(safe[key] is False, f"{key} must be false")

    user = "dev_user"
    ledger = tgwt_ledger_initial(user)

    e1 = tgwt_ledger_event(
        "ev1",
        user,
        TGWT_EVENT_EARN_PENDING,
        tgwt_amount_to_u("10"),
        "WATCH_AND_EARN",
        "watch claim recorded off-chain",
    )

    ledger, ok, reason = tgwt_apply_event(ledger, e1, set())
    assert_true(ok, "earn pending should apply")
    assert_equal(reason, "applied", "earn reason")
    assert_equal(ledger["earned_u"], 10 * TGWT_UNIT, "earned after e1")
    assert_equal(ledger["pending_u"], 10 * TGWT_UNIT, "pending after e1")
    assert_equal(ledger["verified_u"], 0, "verified after e1")
    assert_equal(ledger["claimable_u"], 0, "claimable after e1")

    e2 = tgwt_ledger_event(
        "ev2",
        user,
        TGWT_EVENT_VERIFY_PENDING,
        tgwt_amount_to_u("7"),
        "ADMIN_REVIEW",
        "verified for future user-paid BSC claim",
    )

    ledger, ok, reason = tgwt_apply_event(ledger, e2, {"ev1"})
    assert_true(ok, "verify should apply")
    assert_equal(ledger["pending_u"], 3 * TGWT_UNIT, "pending after verify")
    assert_equal(ledger["verified_u"], 7 * TGWT_UNIT, "verified after verify")
    assert_equal(ledger["claimable_u"], 7 * TGWT_UNIT, "claimable after verify")

    e3 = tgwt_ledger_event(
        "ev3",
        user,
        TGWT_EVENT_CLAIM_CONFIRMED,
        tgwt_amount_to_u("2"),
        "BSC_USER_PAID_CLAIM_RECORD",
        "record confirmed user-paid claim; no backend gas payment",
    )

    ledger, ok, reason = tgwt_apply_event(ledger, e3, {"ev1", "ev2"})
    assert_true(ok, "claim record should apply")
    assert_equal(ledger["claimed_u"], 2 * TGWT_UNIT, "claimed after claim")
    assert_equal(ledger["claimable_u"], 5 * TGWT_UNIT, "claimable after claim")

    ledger2, ok, reason = tgwt_apply_event(ledger, e3, {"ev1", "ev2", "ev3"})
    assert_true(not ok, "duplicate should not apply")
    assert_equal(reason, "duplicate_event", "duplicate reason")
    assert_equal(ledger2["claimed_u"], 2 * TGWT_UNIT, "claimed unchanged after duplicate")

    too_big = tgwt_ledger_event(
        "ev4",
        user,
        TGWT_EVENT_EARN_PENDING,
        TGWT_POOL_TOTAL_U,
        "ABUSE_TEST",
        "should not exceed pool after existing earned",
    )

    ledger3, ok, reason = tgwt_apply_event(ledger, too_big, {"ev1", "ev2", "ev3"})
    assert_true(not ok, "pool over cap should reject")
    assert_equal(reason, "pool_cap_exceeded", "pool cap reason")
    assert_equal(ledger3["earned_u"], ledger["earned_u"], "earned unchanged after over cap")

    events = []
    events, ok, reason = tgwt_append_event_log(events, e1)
    assert_true(ok, "append e1")
    events, ok, reason = tgwt_append_event_log(events, e1)
    assert_true(not ok, "duplicate append rejected")
    assert_equal(reason, "duplicate_event", "duplicate append reason")

    rebuilt = tgwt_build_ledger_from_events(user, [e1, e2, e3])
    assert_equal(rebuilt["earned_u"], 10 * TGWT_UNIT, "rebuilt earned")
    assert_equal(rebuilt["pending_u"], 3 * TGWT_UNIT, "rebuilt pending")
    assert_equal(rebuilt["verified_u"], 7 * TGWT_UNIT, "rebuilt verified")
    assert_equal(rebuilt["claimed_u"], 2 * TGWT_UNIT, "rebuilt claimed")
    assert_equal(rebuilt["claimable_u"], 5 * TGWT_UNIT, "rebuilt claimable")

    hold_event = tgwt_ledger_event(
        "ev_hold",
        user,
        TGWT_EVENT_ADMIN_HOLD,
        tgwt_amount_to_u("1"),
        "ADMIN_REVIEW",
        "suspicious claim hold",
    )
    held_ledger, ok, reason = tgwt_apply_event(tgwt_ledger_initial(user), hold_event, set())
    assert_true(not ok, "hold without pending should fail")
    assert_equal(reason, "insufficient_pending", "hold reason without pending")

    reject_event = tgwt_ledger_event(
        "ev_reject",
        user,
        TGWT_EVENT_REJECT,
        tgwt_amount_to_u("1"),
        "ADMIN_REVIEW",
        "reject pending amount",
    )
    reject_ledger, ok, reason = tgwt_apply_event(tgwt_ledger_initial(user), reject_event, set())
    assert_true(not ok, "reject without pending should fail")
    assert_equal(reason, "insufficient_pending", "reject reason without pending")

    print("TGWT_LEDGER_EVENTS_V1_TEST_PASS")


if __name__ == "__main__":
    main()
