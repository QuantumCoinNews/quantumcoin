# TGWT_BSC_CLAIM_MAINNET_TESTNET_LOCK_V1_TEST

import os
import sys
from pathlib import Path
from contextlib import contextmanager

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.tgwt_bsc_claim_safety_v1 import (
    TGWT_BSC_CLAIM_LOCK_VERSION,
    TGWT_BSC_MAINNET_CHAIN_ID,
    TGWT_BSC_NETWORK_MAINNET,
    TGWT_BSC_NETWORK_TESTNET,
    TGWT_BSC_TESTNET_CHAIN_ID,
    TGWTBSCClaimSafetyError,
    tgwt_assert_bsc_public_config_safe_v1,
    tgwt_bsc_claim_frontend_public_payload_v1,
    tgwt_bsc_public_config_from_env_v1,
    tgwt_claim_core_todo_v1,
    tgwt_is_bsc_address_v1,
    tgwt_validate_bsc_public_config_v1,
)


@contextmanager
def clean_env():
    keys = [
        "TGWT_BSC_CLAIM_ENABLED",
        "TGWT_BSC_NETWORK",
        "TGWT_BSC_CHAIN_ID",
        "TGWT_BSC_DISTRIBUTION_WALLET_ADDRESS",
        "TGWT_BSC_TOKEN_CONTRACT_ADDRESS",
        "TGWT_BSC_CLAIM_CONTRACT_ADDRESS",
        "TGWT_BSC_MAINNET_CLAIM_APPROVAL",
        "TGWT_BSC_PRIVATE_KEY",
        "TGWT_BSC_TREASURY_PRIVATE_KEY",
        "TGWT_BSC_DISTRIBUTION_PRIVATE_KEY",
        "TGWT_BSC_SEED",
        "TGWT_BSC_MNEMONIC",
        "BSC_PRIVATE_KEY",
        "BSC_TREASURY_PRIVATE_KEY",
        "BSC_DISTRIBUTION_PRIVATE_KEY",
        "BSC_SEED",
        "BSC_MNEMONIC",
        "TREASURY_PRIVATE_KEY",
        "DISTRIBUTION_PRIVATE_KEY",
        "HOT_WALLET_PRIVATE_KEY",
        "QC_TESTNET",
        "QC_CHAIN_MODE",
        "QC_NETWORK",
        "TELEGRAM_GAME_CHAIN_MODE",
        "TELEGRAM_GAME_NETWORK",
    ]
    old = {k: os.environ.get(k) for k in keys}
    try:
        for k in keys:
            os.environ.pop(k, None)
        yield
    finally:
        for k, v in old.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v


GOOD_DIST = "0x1111111111111111111111111111111111111111"
GOOD_TOKEN = "0x2222222222222222222222222222222222222222"
GOOD_CLAIM = "0x3333333333333333333333333333333333333333"


def expect_error(fn, label):
    try:
        fn()
    except TGWTBSCClaimSafetyError:
        return
    raise AssertionError("expected TGWTBSCClaimSafetyError: " + label)


def test_address_validation():
    assert tgwt_is_bsc_address_v1(GOOD_DIST)
    assert not tgwt_is_bsc_address_v1("0x123")
    assert not tgwt_is_bsc_address_v1("0x0000000000000000000000000000000000000000")


def test_disabled_testnet_default_safe():
    with clean_env():
        cfg = tgwt_bsc_public_config_from_env_v1()
        assert cfg["version"] == TGWT_BSC_CLAIM_LOCK_VERSION
        assert cfg["claimEnabled"] is False
        assert cfg["network"] == TGWT_BSC_NETWORK_TESTNET
        assert cfg["chainId"] == TGWT_BSC_TESTNET_CHAIN_ID
        safe = tgwt_assert_bsc_public_config_safe_v1(cfg)
        assert safe["userPaysBscFee"] is True
        assert safe["createsBscTransaction"] is False


def test_enabled_testnet_requires_public_addresses():
    with clean_env():
        os.environ["TGWT_BSC_CLAIM_ENABLED"] = "1"
        expect_error(tgwt_assert_bsc_public_config_safe_v1, "enabled claim requires addresses")

        os.environ["TGWT_BSC_DISTRIBUTION_WALLET_ADDRESS"] = GOOD_DIST
        os.environ["TGWT_BSC_TOKEN_CONTRACT_ADDRESS"] = GOOD_TOKEN
        os.environ["TGWT_BSC_CLAIM_CONTRACT_ADDRESS"] = GOOD_CLAIM
        cfg = tgwt_assert_bsc_public_config_safe_v1()
        assert cfg["claimEnabled"] is True
        assert cfg["network"] == TGWT_BSC_NETWORK_TESTNET


def test_testnet_rejects_mainnet_chain_id():
    with clean_env():
        cfg = {
            "claimEnabled": False,
            "network": TGWT_BSC_NETWORK_TESTNET,
            "chainId": TGWT_BSC_MAINNET_CHAIN_ID,
            "claimModel": "USER_PAID_BSC_CLAIM",
            "userPaysBscFee": True,
            "backendPaysGas": False,
            "createsBscTransaction": False,
        }
        valid, reason = tgwt_validate_bsc_public_config_v1(cfg)
        assert valid is False
        assert reason == "bsc_chain_id_network_mismatch"


def test_mainnet_requires_manual_approval():
    with clean_env():
        cfg = {
            "claimEnabled": False,
            "network": TGWT_BSC_NETWORK_MAINNET,
            "chainId": TGWT_BSC_MAINNET_CHAIN_ID,
            "claimModel": "USER_PAID_BSC_CLAIM",
            "userPaysBscFee": True,
            "backendPaysGas": False,
            "createsBscTransaction": False,
        }
        valid, reason = tgwt_validate_bsc_public_config_v1(cfg)
        assert valid is False
        assert reason == "mainnet_claim_requires_explicit_manual_approval"

        cfg["mainnetApproval"] = "I_APPROVE_TGWT_BSC_MAINNET_CLAIM"
        valid, reason = tgwt_validate_bsc_public_config_v1(cfg)
        assert valid is True
        assert reason == "ok"


def test_private_key_env_forbidden():
    with clean_env():
        os.environ["TGWT_BSC_PRIVATE_KEY"] = "never_put_private_key_here"
        expect_error(tgwt_assert_bsc_public_config_safe_v1, "private key env forbidden")


def test_backend_bsc_tx_forbidden():
    with clean_env():
        cfg = {
            "claimEnabled": False,
            "network": TGWT_BSC_NETWORK_TESTNET,
            "chainId": TGWT_BSC_TESTNET_CHAIN_ID,
            "claimModel": "USER_PAID_BSC_CLAIM",
            "userPaysBscFee": True,
            "backendPaysGas": False,
            "createsBscTransaction": True,
        }
        valid, reason = tgwt_validate_bsc_public_config_v1(cfg)
        assert valid is False
        assert reason == "backend_bsc_transaction_forbidden"


def test_frontend_payload_public_only():
    with clean_env():
        cfg = {
            "claimEnabled": True,
            "network": TGWT_BSC_NETWORK_TESTNET,
            "chainId": TGWT_BSC_TESTNET_CHAIN_ID,
            "distributionWalletAddress": GOOD_DIST,
            "tokenContractAddress": GOOD_TOKEN,
            "claimContractAddress": GOOD_CLAIM,
            "claimModel": "USER_PAID_BSC_CLAIM",
            "userPaysBscFee": True,
            "backendPaysGas": False,
            "createsBscTransaction": False,
        }
        payload = tgwt_bsc_claim_frontend_public_payload_v1(cfg)
        assert payload["claimModel"] == "USER_PAID_BSC_CLAIM"
        assert payload["userPaysBscFee"] is True
        assert payload["backendPaysGas"] is False
        assert payload["createsBscTransaction"] is False
        assert "privateKey" not in str(payload)
        assert payload["distributionWalletAddress"] == GOOD_DIST


def test_todo_contains_wallet_and_contract_steps():
    todo = tgwt_claim_core_todo_v1()
    assert "wire_frontend_wallet_connect_later" in todo
    assert "set_distribution_wallet_public_address_in_private_env" in todo
    assert "set_tgwt_token_contract_public_address_in_private_env" in todo
    assert "keep_user_paid_bnb_gas_model" in todo


if __name__ == "__main__":
    test_address_validation()
    test_disabled_testnet_default_safe()
    test_enabled_testnet_requires_public_addresses()
    test_testnet_rejects_mainnet_chain_id()
    test_mainnet_requires_manual_approval()
    test_private_key_env_forbidden()
    test_backend_bsc_tx_forbidden()
    test_frontend_payload_public_only()
    test_todo_contains_wallet_and_contract_steps()
    print("TGWT_BSC_CLAIM_MAINNET_TESTNET_LOCK_V1_TEST_PASS")
