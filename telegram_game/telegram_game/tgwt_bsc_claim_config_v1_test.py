from tgwt_bsc_claim_config import (
    TGWT_BSC_CLAIM_CONFIG_VERSION,
    TGWT_BSC_CLAIM_CONFIG_STATUS,
    TGWT_BSC_NETWORK_TESTNET,
    TGWT_BSC_NETWORK_MAINNET,
    TGWT_BSC_TESTNET_CHAIN_ID,
    TGWT_BSC_MAINNET_CHAIN_ID,
    TGWT_CONTRACT_PLACEHOLDER,
    tgwt_bsc_network_config,
    tgwt_bsc_claim_config,
    tgwt_validate_bsc_claim_config,
    tgwt_bsc_claim_frontend_payload,
    tgwt_is_bsc_contract_address,
)


def assert_true(value, message):
    if not value:
        raise AssertionError(message)


def assert_equal(got, want, message):
    if got != want:
        raise AssertionError(f"{message}: got={got!r} want={want!r}")


def main():
    assert_equal(TGWT_BSC_CLAIM_CONFIG_VERSION, "TGWT_BSC_CLAIM_CONFIG_V1", "version")
    assert_equal(TGWT_BSC_CLAIM_CONFIG_STATUS, "CONFIG_ONLY_NO_BSC_TRANSACTION", "status")

    testnet = tgwt_bsc_network_config(TGWT_BSC_NETWORK_TESTNET)
    assert_equal(testnet["chainId"], 97, "testnet chain id")
    assert_true(testnet["isTestnet"] is True, "is testnet")
    assert_true(testnet["claimEnabled"] is False, "claim disabled by default")
    assert_equal(testnet["tokenContract"], TGWT_CONTRACT_PLACEHOLDER, "token placeholder")
    assert_equal(testnet["claimContract"], TGWT_CONTRACT_PLACEHOLDER, "claim placeholder")

    mainnet = tgwt_bsc_network_config(TGWT_BSC_NETWORK_MAINNET)
    assert_equal(mainnet["chainId"], 56, "mainnet chain id")
    assert_true(mainnet["isMainnet"] is True, "is mainnet")

    cfg_disabled = tgwt_bsc_claim_config()
    assert_true(cfg_disabled["valid"] is True, "disabled config valid")
    assert_equal(cfg_disabled["validationReason"], "claim_disabled_contract_addresses_not_required", "disabled reason")
    assert_true(cfg_disabled["contractAddressPolicy"]["requiredNow"] is False, "contract not required now")
    assert_true(cfg_disabled["contractAddressPolicy"]["requiredWhenClaimEnabled"] is True, "contract required when enabled")

    cfg_enabled_missing = tgwt_bsc_claim_config(claim_enabled=True)
    assert_true(cfg_enabled_missing["valid"] is False, "enabled missing invalid")
    assert_equal(cfg_enabled_missing["validationReason"], "missing_or_invalid_token_contract", "missing token reason")

    token = "0x1111111111111111111111111111111111111111"
    claim = "0x2222222222222222222222222222222222222222"
    assert_true(tgwt_is_bsc_contract_address(token), "token address valid")
    assert_true(tgwt_is_bsc_contract_address(claim), "claim address valid")
    assert_true(not tgwt_is_bsc_contract_address(TGWT_CONTRACT_PLACEHOLDER), "zero placeholder invalid when enabled")

    cfg_enabled = tgwt_bsc_claim_config(
        network=TGWT_BSC_NETWORK_TESTNET,
        claim_enabled=True,
        token_contract=token,
        claim_contract=claim,
    )
    assert_true(cfg_enabled["valid"] is True, "enabled config valid")
    assert_equal(cfg_enabled["validationReason"], "claim_enabled_config_valid", "enabled reason")

    cfg_same = tgwt_bsc_claim_config(
        network=TGWT_BSC_NETWORK_TESTNET,
        claim_enabled=True,
        token_contract=token,
        claim_contract=token,
    )
    assert_true(cfg_same["valid"] is False, "same contracts invalid")
    assert_equal(cfg_same["validationReason"], "token_and_claim_contract_must_differ", "same contract reason")

    bad_testnet = dict(cfg_enabled)
    bad_testnet["chainId"] = TGWT_BSC_MAINNET_CHAIN_ID
    valid, reason = tgwt_validate_bsc_claim_config(bad_testnet)
    assert_true(valid is False, "bad testnet chain invalid")
    assert_equal(reason, "testnet_chain_id_mismatch", "bad testnet reason")

    bad_mainnet = tgwt_bsc_claim_config(
        network=TGWT_BSC_NETWORK_MAINNET,
        claim_enabled=True,
        token_contract=token,
        claim_contract=claim,
    )
    bad_mainnet["chainId"] = TGWT_BSC_TESTNET_CHAIN_ID
    valid, reason = tgwt_validate_bsc_claim_config(bad_mainnet)
    assert_true(valid is False, "bad mainnet chain invalid")
    assert_equal(reason, "mainnet_chain_id_mismatch", "bad mainnet reason")

    payload = tgwt_bsc_claim_frontend_payload(cfg_enabled)
    assert_equal(payload["version"], "TGWT_BSC_CLAIM_CONFIG_V1", "payload version")
    assert_equal(payload["claimModel"], "USER_PAID_BSC_CLAIM", "payload claim model")
    assert_equal(payload["feeModel"], "BNB_GAS_PAID_BY_USER", "payload fee model")
    assert_true(payload["backendPaysGas"] is False, "backend does not pay gas")
    assert_true(payload["backendSendsToken"] is False, "backend does not send token")
    assert_true(payload["createsBscTransaction"] is False, "backend does not create bsc tx")
    assert_true(payload["requiresMetaMask"] is True, "metamask required")
    assert_true(payload["requiresUserWalletSignature"] is True, "user signature required")

    safe = payload["safeBoundary"]
    assert_true(safe["configOnly"] is True, "config only")
    assert_true(safe["offChainOnly"] is True, "offChainOnly")
    assert_true(safe["userPaysBscFee"] is True, "user pays bsc fee")
    assert_true(safe["testnetMainnetSeparated"] is True, "separated")

    false_keys = [
        "createsBscTransaction",
        "deploysContract",
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

    print("TGWT_BSC_CLAIM_CONFIG_V1_TEST_PASS")


if __name__ == "__main__":
    main()
