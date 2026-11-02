from tgwt_admin_review import (
    TGWT_ADMIN_REVIEW_VERSION,
    TGWT_ADMIN_REVIEW_STATUS_ACTIVE,
    TGWT_REVIEW_APPROVED_FOR_USER_PAID_CLAIM,
    TGWT_REVIEW_PENDING_ADMIN_REVIEW,
    TGWT_REVIEW_HOLD,
    TGWT_REVIEW_REJECTED,
    tgwt_admin_review_policy,
    tgwt_admin_review_decision,
    tgwt_admin_review_is_allowed,
    tgwt_admin_review_requires_admin,
    tgwt_admin_review_is_rejected,
    tgwt_is_bsc_address,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(TGWT_ADMIN_REVIEW_VERSION, "TGWT_ADMIN_REVIEW_V1", "version")
    assert_equal(TGWT_ADMIN_REVIEW_STATUS_ACTIVE, "OFF_CHAIN_ADMIN_REVIEW_ACTIVE_NO_BSC_TRANSFER", "status")

    policy = tgwt_admin_review_policy()
    assert_equal(policy["version"], "TGWT_ADMIN_REVIEW_V1", "policy version")
    assert_equal(policy["claimModel"], "USER_PAID_BSC_CLAIM", "claim model")
    assert_equal(policy["feeModel"], "BNB_GAS_PAID_BY_USER", "fee model")
    assert_true(policy["networkSeparation"]["qcNativeNetworkOnly"] is True, "QC native only")
    assert_true(policy["networkSeparation"]["bscClaimLayerSeparate"] is True, "BSC separate")
    assert_true(policy["networkSeparation"]["mixesQCAndBSC"] is False, "no network mixing")

    safe = policy["safeBoundary"]
    assert_true(safe["offChainOnly"] is True, "offChainOnly")
    assert_true(safe["recordsDecisionOnly"] is True, "recordsDecisionOnly")
    assert_true(safe["userPaysBscFee"] is True, "userPaysBscFee")
    assert_true(safe["requiresUserWalletSignatureLater"] is True, "user signature later")

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
    assert_true(tgwt_is_bsc_address(good_addr), "valid bsc address")
    assert_true(not tgwt_is_bsc_address("0x123"), "invalid short bsc address")

    small = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="10",
        claimable_amount="100",
        bsc_address=good_addr,
        claim_id="claim1",
    )
    assert_equal(small["status"], TGWT_REVIEW_APPROVED_FOR_USER_PAID_CLAIM, "small status")
    assert_true(tgwt_admin_review_is_allowed(small), "small allowed")
    assert_true(not tgwt_admin_review_requires_admin(small), "small no admin")
    assert_true(small["backendPaysGas"] is False, "backend does not pay gas")
    assert_true(small["backendSendsToken"] is False, "backend does not send token")
    assert_true(small["createsBscTransaction"] is False, "backend does not create bsc tx")

    medium = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="100",
        claimable_amount="200",
        bsc_address=good_addr,
        claim_id="claim2",
    )
    assert_equal(medium["status"], TGWT_REVIEW_PENDING_ADMIN_REVIEW, "medium status")
    assert_true(tgwt_admin_review_requires_admin(medium), "medium admin required")

    flagged = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="10",
        claimable_amount="100",
        bsc_address=good_addr,
        risk_flags=["duplicate_watch_pattern"],
        claim_id="claim3",
    )
    assert_equal(flagged["status"], TGWT_REVIEW_HOLD, "flagged status")
    assert_true(tgwt_admin_review_requires_admin(flagged), "flagged admin required")

    invalid_addr = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="10",
        claimable_amount="100",
        bsc_address="0x123",
        claim_id="claim4",
    )
    assert_equal(invalid_addr["status"], TGWT_REVIEW_REJECTED, "invalid addr rejected")
    assert_true(tgwt_admin_review_is_rejected(invalid_addr), "invalid addr rejected helper")

    insufficient = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="101",
        claimable_amount="100",
        bsc_address=good_addr,
        claim_id="claim5",
    )
    assert_equal(insufficient["status"], TGWT_REVIEW_REJECTED, "insufficient rejected")
    assert_true("insufficient_claimable" in insufficient["reasons"], "insufficient reason")

    duplicate = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="10",
        claimable_amount="100",
        bsc_address=good_addr,
        previous_claim_ids=["claim6"],
        claim_id="claim6",
    )
    assert_equal(duplicate["status"], TGWT_REVIEW_REJECTED, "duplicate rejected")
    assert_true("duplicate_claim_id" in duplicate["reasons"], "duplicate reason")

    daily_hold = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="10",
        claimable_amount="100",
        bsc_address=good_addr,
        user_daily_claimed_amount="1000",
        claim_id="claim7",
    )
    assert_equal(daily_hold["status"], TGWT_REVIEW_HOLD, "daily hold")
    assert_true("user_daily_limit_review" in daily_hold["reasons"], "daily reason")

    global_hold = tgwt_admin_review_decision(
        user_id="dev_user",
        amount="10",
        claimable_amount="100",
        bsc_address=good_addr,
        global_daily_claimed_amount="50000",
        claim_id="claim8",
    )
    assert_equal(global_hold["status"], TGWT_REVIEW_HOLD, "global hold")
    assert_true("global_daily_limit_review" in global_hold["reasons"], "global reason")

    print("TGWT_ADMIN_REVIEW_V1_TEST_PASS")


if __name__ == "__main__":
    main()
