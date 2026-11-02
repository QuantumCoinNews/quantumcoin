from tgwt_pool_policy import TGWT_POOL_TOTAL_U
from tgwt_state_flow import (
    TGWT_STATE_FLOW_VERSION,
    TGWT_STATE_FLOW_STATUS,
    TGWT_WITHDRAWN_LEGACY_ALIAS,
    tgwt_state_flow_policy,
    tgwt_state_summary,
    tgwt_validate_state_invariants,
    tgwt_create_earn_pending_event,
    tgwt_create_verify_event,
    tgwt_create_claim_confirmed_event,
    tgwt_create_admin_hold_event,
    tgwt_create_reject_event,
    tgwt_apply_state_event_log,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(TGWT_STATE_FLOW_VERSION, "TGWT_STATE_FLOW_V1", "version")
    assert_equal(TGWT_STATE_FLOW_STATUS, "OFF_CHAIN_STATE_FLOW_ACTIVE_USER_PAID_BSC_CLAIM", "status")
    assert_equal(TGWT_WITHDRAWN_LEGACY_ALIAS, "claimed", "withdrawn alias")

    policy = tgwt_state_flow_policy()
    assert_equal(policy["version"], "TGWT_STATE_FLOW_V1", "policy version")
    assert_equal(policy["claimModel"], "USER_PAID_BSC_CLAIM", "claim model")
    assert_equal(policy["feeModel"], "BNB_GAS_PAID_BY_USER", "fee model")
    assert_true(policy["networkSeparation"]["qcNativeNetworkOnly"] is True, "QC native only")
    assert_true(policy["networkSeparation"]["bscClaimLayerSeparate"] is True, "BSC separate")
    assert_true(policy["networkSeparation"]["mixesQCAndBSC"] is False, "no network mixing")

    safe = policy["safeBoundary"]
    assert_true(safe["offChainOnly"] is True, "offChainOnly")
    assert_true(safe["recordsRightsOnly"] is True, "recordsRightsOnly")
    assert_true(safe["userPaysBscFee"] is True, "userPaysBscFee")
    assert_true(safe["usesClaimModel"] is True, "usesClaimModel")
    assert_true(safe["withdrawnIsLegacyAliasForClaimed"] is True, "withdrawn alias boundary")

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
    events = []

    ev1 = tgwt_create_earn_pending_event("ev1", user, "10", "WATCH_AND_EARN", "watch reward pending")
    events, summary, ok, reason = tgwt_apply_state_event_log(user, events, ev1)
    assert_true(ok, "earn pending applied")
    assert_equal(summary["earned"], "10", "earned after earn")
    assert_equal(summary["pending"], "10", "pending after earn")
    assert_equal(summary["verified"], "0", "verified after earn")
    assert_equal(summary["claimable"], "0", "claimable after earn")
    assert_true(summary["canClaim"] is False, "cannot claim pending only")

    ev2 = tgwt_create_verify_event("ev2", user, "7")
    events, summary, ok, reason = tgwt_apply_state_event_log(user, events, ev2)
    assert_true(ok, "verify applied")
    assert_equal(summary["earned"], "10", "earned after verify")
    assert_equal(summary["pending"], "3", "pending after verify")
    assert_equal(summary["verified"], "7", "verified after verify")
    assert_equal(summary["claimable"], "7", "claimable after verify")
    assert_true(summary["canClaim"] is True, "can claim verified")
    assert_true(summary["requiresConnectedBscWallet"] is True, "requires BSC wallet")
    assert_equal(summary["gasPaidBy"], "USER", "gas paid by user")
    assert_true(summary["backendPaysGas"] is False, "backend does not pay gas")
    assert_true(summary["backendSendsToken"] is False, "backend does not send token")

    ev3 = tgwt_create_claim_confirmed_event(
        "ev3",
        user,
        "2",
        "0xabc123",
        "0x1111111111111111111111111111111111111111",
    )
    events, summary, ok, reason = tgwt_apply_state_event_log(user, events, ev3)
    assert_true(ok, "claim confirmation applied")
    assert_equal(summary["claimed"], "2", "claimed after claim")
    assert_equal(summary["withdrawn"], "2", "withdrawn legacy alias")
    assert_equal(summary["claimable"], "5", "claimable after claim")

    valid, valid_reason = tgwt_validate_state_invariants(summary)
    assert_true(valid, "summary invariants valid")
    assert_equal(valid_reason, "ok", "valid reason")

    events_dup, summary_dup, ok, reason = tgwt_apply_state_event_log(user, events, ev3)
    assert_true(not ok, "duplicate event rejected")
    assert_equal(reason, "duplicate_event", "duplicate reason")
    assert_equal(summary_dup["claimed"], "2", "claimed unchanged after duplicate")

    ev_hold = tgwt_create_admin_hold_event("ev4", user, "1", "suspicious pending claim")
    events, summary, ok, reason = tgwt_apply_state_event_log(user, events, ev_hold)
    assert_true(ok, "hold applied")
    assert_equal(summary["pending"], "2", "pending after hold")
    assert_equal(summary["held"], "1", "held after hold")

    ev_reject = tgwt_create_reject_event("ev5", user, "1", "invalid pending claim")
    events, summary, ok, reason = tgwt_apply_state_event_log(user, events, ev_reject)
    assert_true(ok, "reject applied")
    assert_equal(summary["pending"], "1", "pending after reject")
    assert_equal(summary["rejected"], "1", "rejected after reject")

    bad_summary = dict(summary)
    bad_summary["claimable_u"] = 999
    valid, valid_reason = tgwt_validate_state_invariants(bad_summary)
    assert_true(not valid, "bad claimable should fail")
    assert_equal(valid_reason, "claimable_mismatch", "claimable mismatch reason")

    huge = dict(summary)
    huge["earned_u"] = TGWT_POOL_TOTAL_U + 1
    valid, valid_reason = tgwt_validate_state_invariants(huge)
    assert_true(not valid, "over pool should fail")
    assert_equal(valid_reason, "pool_cap_exceeded", "pool cap reason")

    direct_summary = tgwt_state_summary(user, events)
    assert_equal(direct_summary["earned"], summary["earned"], "direct earned")
    assert_equal(direct_summary["claimable"], summary["claimable"], "direct claimable")

    print("TGWT_STATE_FLOW_V1_TEST_PASS")


if __name__ == "__main__":
    main()
