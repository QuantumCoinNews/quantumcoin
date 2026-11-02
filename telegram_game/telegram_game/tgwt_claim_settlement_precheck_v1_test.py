from tgwt_claim_settlement_precheck import (
    TGWT_CLAIM_SETTLEMENT_PRECHECK_VERSION,
    TGWT_CLAIM_SETTLEMENT_PRECHECK_STATUS,
    TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW,
    TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED,
    TGWT_CLAIM_SETTLEMENT_REJECTED,
    TGWT_CLAIM_SETTLEMENT_HOLD,
    TGWT_CLAIM_READY_STATUS,
    TGWT_CLAIM_PENDING_ADMIN_STATUS,
    TGWT_BSC_TESTNET_CHAIN_ID,
    TGWT_BSC_MAINNET_CHAIN_ID,
    tgwt_claim_settlement_precheck_policy,
    tgwt_claim_settlement_precheck,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def claim_request(**overrides):
    data = {
        "userId": "dev_user",
        "claimId": "claim_1",
        "bscAddress": "0x1111111111111111111111111111111111111111",
        "claimStatus": TGWT_CLAIM_READY_STATUS,
        "adminStatus": TGWT_CLAIM_PENDING_ADMIN_STATUS,
        "amountTGWT": 100,
        "claimableTGWT": 100,
    }
    data.update(overrides)
    return data


def bsc_config(**overrides):
    data = {
        "claimEnabled": True,
        "chainId": TGWT_BSC_TESTNET_CHAIN_ID,
        "tokenContractAddress": "0x2222222222222222222222222222222222222222",
        "claimContractAddress": "0x3333333333333333333333333333333333333333",
        "userPaysGas": True,
        "requiresUserSignature": True,
        "backendPaysGas": False,
        "backendSendsToken": False,
        "createsBscTransaction": False,
    }
    data.update(overrides)
    return data


def main():
    assert_equal(TGWT_CLAIM_SETTLEMENT_PRECHECK_VERSION, "TGWT_CLAIM_SETTLEMENT_PRECHECK_V1", "version")
    assert_equal(TGWT_CLAIM_SETTLEMENT_PRECHECK_STATUS, "PRECHECK_ONLY_NO_BSC_TRANSACTION", "status")

    policy = tgwt_claim_settlement_precheck_policy()
    assert_equal(policy["settlementModel"], "USER_PAID_BSC_CLAIM_LATER", "settlement model")
    assert_true(policy["adminApprovalHandledByNextLayer"] is True, "admin approval next layer")

    safe = policy["safeBoundary"]
    assert_true(safe["precheckOnly"] is True, "precheck only")
    assert_true(safe["offChainOnly"] is True, "off-chain only")
    assert_true(safe["adminReviewSignalOnly"] is True, "admin review signal only")
    assert_true(safe["requiresUserPaidClaim"] is True, "requires user paid claim")
    assert_true(safe["requiresBscAddress"] is True, "requires bsc address")
    assert_true(safe["requiresClaimStatusReady"] is True, "requires claim status ready")
    assert_true(safe["requiresAmountWithinClaimable"] is True, "amount within claimable")
    assert_true(safe["requiresChainConfigWhenEnabled"] is True, "chain config required")
    assert_true(safe["requiresTokenContractWhenEnabled"] is True, "token contract required")
    assert_true(safe["requiresClaimContractWhenEnabled"] is True, "claim contract required")
    assert_true(safe["doesNotCreateTransaction"] is True, "does not create transaction")
    assert_true(safe["doesNotSendToken"] is True, "does not send token")
    assert_true(safe["doesNotPayGas"] is True, "does not pay gas")
    assert_true(safe["doesNotSign"] is True, "does not sign")
    assert_true(safe["doesNotStorePrivateKey"] is True, "does not store private key")
    assert_true(safe["doesNotCallRpc"] is True, "does not call rpc")
    assert_true(safe["userPaysBnbGasLater"] is True, "user pays bnb gas later")
    assert_true(safe["automaticTransferAllowed"] is False, "automatic transfer false")

    false_keys = [
        "backendPaysGas",
        "backendSendsToken",
        "createsBscTransaction",
        "deploysContract",
        "broadcastsTransaction",
        "storesPrivateKey",
        "signsTransaction",
        "changesQCBalances",
        "changesTGWTOnChainBalances",
        "changesMempool",
        "changesChain",
        "changesConsensus",
        "changesTotalSupply",
        "startsMining",
        "stopsMining",
    ]

    for key in false_keys:
        assert_true(safe[key] is False, f"{key} must be false")

    ok = tgwt_claim_settlement_precheck(claim_request(), bsc_config(), "TESTNET")
    assert_equal(ok["status"], TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW, "ready for admin review")
    assert_true(ok["readyForAdminReview"] is True, "ready flag")
    assert_true(ok["readyForAutomaticTransfer"] is False, "no automatic transfer")
    assert_true(ok["createsBscTransaction"] is False, "no bsc transaction")
    assert_true(ok["backendPaysGas"] is False, "backend no gas")
    assert_true(ok["backendSendsToken"] is False, "backend no send")
    assert_true("precheck_passed_ready_for_admin_review" in ok["reasons"], "precheck reason")

    disabled = tgwt_claim_settlement_precheck(claim_request(), bsc_config(claimEnabled=False), "TESTNET")
    assert_equal(disabled["status"], TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED, "config disabled")
    assert_true("claim_config_disabled" in disabled["reasons"], "config disabled reason")

    invalid_address = tgwt_claim_settlement_precheck(
        claim_request(bscAddress="not_an_address"),
        bsc_config(),
        "TESTNET",
    )
    assert_equal(invalid_address["status"], TGWT_CLAIM_SETTLEMENT_REJECTED, "invalid address rejected")
    assert_true("invalid_bsc_address" in invalid_address["reasons"], "invalid address reason")

    too_much = tgwt_claim_settlement_precheck(
        claim_request(amountTGWT=101, claimableTGWT=100),
        bsc_config(),
        "TESTNET",
    )
    assert_equal(too_much["status"], TGWT_CLAIM_SETTLEMENT_REJECTED, "amount exceeds rejected")
    assert_true("claim_amount_exceeds_claimable" in too_much["reasons"], "amount exceeds reason")

    wrong_chain = tgwt_claim_settlement_precheck(
        claim_request(),
        bsc_config(chainId=TGWT_BSC_MAINNET_CHAIN_ID),
        "TESTNET",
    )
    assert_equal(wrong_chain["status"], TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED, "wrong chain config required")
    assert_true("invalid_chain_id" in wrong_chain["reasons"], "wrong chain reason")

    missing_contract = tgwt_claim_settlement_precheck(
        claim_request(),
        bsc_config(tokenContractAddress=""),
        "TESTNET",
    )
    assert_equal(missing_contract["status"], TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED, "missing token contract")
    assert_true("missing_token_contract_address" in missing_contract["reasons"], "missing token reason")

    same_contract = tgwt_claim_settlement_precheck(
        claim_request(),
        bsc_config(
            tokenContractAddress="0x2222222222222222222222222222222222222222",
            claimContractAddress="0x2222222222222222222222222222222222222222",
        ),
        "TESTNET",
    )
    assert_equal(same_contract["status"], TGWT_CLAIM_SETTLEMENT_CONFIG_REQUIRED, "same contract config")
    assert_true("token_and_claim_contract_must_differ" in same_contract["reasons"], "same contract reason")

    backend_gas = tgwt_claim_settlement_precheck(
        claim_request(),
        bsc_config(backendPaysGas=True),
        "TESTNET",
    )
    assert_equal(backend_gas["status"], TGWT_CLAIM_SETTLEMENT_REJECTED, "backend gas rejected")
    assert_true("backend_pays_gas_not_allowed" in backend_gas["reasons"], "backend gas reason")

    hold = tgwt_claim_settlement_precheck(
        claim_request(),
        bsc_config(requiresUserSignature=False),
        "TESTNET",
    )
    assert_equal(hold["status"], TGWT_CLAIM_SETTLEMENT_HOLD, "signature warning hold")
    assert_true("user_signature_requirement_should_be_enabled" in hold["warnings"], "signature warning")

    mainnet_ok = tgwt_claim_settlement_precheck(
        claim_request(claimId="claim_mainnet"),
        bsc_config(chainId=TGWT_BSC_MAINNET_CHAIN_ID),
        "MAINNET",
    )
    assert_equal(mainnet_ok["status"], TGWT_CLAIM_SETTLEMENT_READY_FOR_ADMIN_REVIEW, "mainnet chain ok")

    print("TGWT_CLAIM_SETTLEMENT_PRECHECK_V1_TEST_PASS")


if __name__ == "__main__":
    main()
