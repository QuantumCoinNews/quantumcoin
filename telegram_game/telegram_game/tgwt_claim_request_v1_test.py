from tgwt_state_flow import (
    tgwt_create_earn_pending_event,
    tgwt_create_verify_event,
    tgwt_apply_state_event_log,
)
from tgwt_claim_request import (
    TGWT_CLAIM_REQUEST_VERSION,
    TGWT_CLAIM_REQUEST_STATUS_ACTIVE,
    TGWT_CLAIM_STATUS_READY_FOR_USER_PAID_CLAIM,
    TGWT_CLAIM_STATUS_PENDING_ADMIN_REVIEW,
    TGWT_CLAIM_STATUS_HOLD,
    TGWT_CLAIM_STATUS_REJECTED,
    tgwt_claim_request_policy,
    tgwt_make_claim_id,
    tgwt_claim_request_record,
    tgwt_claim_request_from_events,
    tgwt_claim_request_is_ready,
    tgwt_claim_request_requires_admin,
    tgwt_claim_request_is_rejected,
    tgwt_claim_request_append,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def build_events():
    user = "dev_user"
    events = []
    ev1 = tgwt_create_earn_pending_event("ev1", user, "100", "WATCH_AND_EARN", "pending reward")
    events, summary, ok, reason = tgwt_apply_state_event_log(user, events, ev1)
    assert_true(ok, "earn event")
    ev2 = tgwt_create_verify_event("ev2", user, "80")
    events, summary, ok, reason = tgwt_apply_state_event_log(user, events, ev2)
    assert_true(ok, "verify event")
    return user, events


def main():
    assert_equal(TGWT_CLAIM_REQUEST_VERSION, "TGWT_CLAIM_REQUEST_V1", "version")
    assert_equal(TGWT_CLAIM_REQUEST_STATUS_ACTIVE, "OFF_CHAIN_CLAIM_REQUEST_ACTIVE_NO_BSC_TRANSFER", "status")

    policy = tgwt_claim_request_policy()
    assert_equal(policy["version"], "TGWT_CLAIM_REQUEST_V1", "policy version")
    assert_equal(policy["claimModel"], "USER_PAID_BSC_CLAIM", "claim model")
    assert_equal(policy["feeModel"], "BNB_GAS_PAID_BY_USER", "fee model")
    assert_true(policy["contractAddressPolicy"]["requiredNow"] is False, "contract address not required now")
    assert_true(policy["contractAddressPolicy"]["requiredLater"] is True, "contract address required later")
    assert_true(policy["networkSeparation"]["qcNativeNetworkOnly"] is True, "QC native only")
    assert_true(policy["networkSeparation"]["bscClaimLayerSeparate"] is True, "BSC separate")
    assert_true(policy["networkSeparation"]["mixesQCAndBSC"] is False, "no network mixing")

    safe = policy["safeBoundary"]
    assert_true(safe["offChainOnly"] is True, "offChainOnly")
    assert_true(safe["recordsClaimRequestOnly"] is True, "recordsClaimRequestOnly")
    assert_true(safe["userPaysBscFee"] is True, "userPaysBscFee")
    assert_true(safe["requiresConnectedBscWallet"] is True, "requires BSC wallet")
    assert_true(safe["requiresUserWalletSignatureLater"] is True, "requires user wallet signature later")
    assert_true(safe["contractAddressRequiredNow"] is False, "contract address required now false")
    assert_true(safe["contractAddressRequiredLater"] is True, "contract address later")

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

    good_addr = "0x1111111111111111111111111111111111111111"
    user, events = build_events()

    claim_id = tgwt_make_claim_id(user, good_addr, "10", "nonce1")
    assert_true(claim_id.startswith("tgwt_claim_"), "claim id prefix")

    ready = tgwt_claim_request_from_events(
        user_id=user,
        events=events,
        amount="10",
        bsc_address=good_addr,
        nonce="nonce1",
    )
    assert_equal(ready["status"], TGWT_CLAIM_STATUS_READY_FOR_USER_PAID_CLAIM, "ready status")
    assert_true(tgwt_claim_request_is_ready(ready), "ready helper")
    assert_true(ready["backendPaysGas"] is False, "backend does not pay gas")
    assert_true(ready["backendSendsToken"] is False, "backend does not send token")
    assert_true(ready["createsBscTransaction"] is False, "backend does not create bsc tx")
    assert_true(ready["contractAddressRequiredNow"] is False, "contract address not required now")
    assert_true(ready["tokenContractAddressRequiredLater"] is True, "token contract later")
    assert_true(ready["claimContractAddressRequiredLater"] is True, "claim contract later")

    records = []
    records, ok, reason = tgwt_claim_request_append(records, ready)
    assert_true(ok, "append ready")
    records, ok, reason = tgwt_claim_request_append(records, ready)
    assert_true(not ok, "duplicate append rejected")
    assert_equal(reason, "duplicate_claim_id", "duplicate append reason")

    medium = tgwt_claim_request_from_events(
        user_id=user,
        events=events,
        amount="60",
        bsc_address=good_addr,
        nonce="nonce2",
    )
    assert_equal(medium["status"], TGWT_CLAIM_STATUS_PENDING_ADMIN_REVIEW, "medium status")
    assert_true(tgwt_claim_request_requires_admin(medium), "medium requires admin")

    flagged = tgwt_claim_request_from_events(
        user_id=user,
        events=events,
        amount="10",
        bsc_address=good_addr,
        nonce="nonce3",
        risk_flags=["duplicate_watch_pattern"],
    )
    assert_equal(flagged["status"], TGWT_CLAIM_STATUS_HOLD, "flagged status")
    assert_true(tgwt_claim_request_requires_admin(flagged), "flagged requires admin")

    rejected_addr = tgwt_claim_request_from_events(
        user_id=user,
        events=events,
        amount="10",
        bsc_address="0x123",
        nonce="nonce4",
    )
    assert_equal(rejected_addr["status"], TGWT_CLAIM_STATUS_REJECTED, "invalid address rejected")
    assert_true(tgwt_claim_request_is_rejected(rejected_addr), "invalid address rejected helper")

    rejected_amount = tgwt_claim_request_from_events(
        user_id=user,
        events=events,
        amount="1000",
        bsc_address=good_addr,
        nonce="nonce5",
    )
    assert_equal(rejected_amount["status"], TGWT_CLAIM_STATUS_REJECTED, "insufficient amount rejected")
    assert_true("insufficient_claimable" in rejected_amount["review"]["reasons"], "insufficient reason")

    direct = tgwt_claim_request_record(
        user_id=user,
        amount="10",
        claimable_amount="80",
        bsc_address=good_addr,
        nonce="nonce6",
        previous_claim_ids=[],
    )
    assert_equal(direct["status"], TGWT_CLAIM_STATUS_READY_FOR_USER_PAID_CLAIM, "direct ready")

    print("TGWT_CLAIM_REQUEST_V1_TEST_PASS")


if __name__ == "__main__":
    main()
