# TGWT_BSC_CLAIM_CONFIG_V1
#
# Safe configuration boundary for future user-paid BSC TGWT claim.
#
# This module does NOT:
# - deploy a contract
# - create a BSC transaction
# - send TGWT
# - store private keys
# - sign transactions
# - use hot wallets or treasury wallets
# - pay BNB gas
# - touch QC balances, QC chain, QC mining, or QC consensus
#
# It only defines and validates testnet/mainnet separation and contract-address policy.

from __future__ import annotations

from typing import Any, Dict, Tuple


TGWT_BSC_CLAIM_CONFIG_VERSION = "TGWT_BSC_CLAIM_CONFIG_V1"
TGWT_BSC_CLAIM_CONFIG_STATUS = "CONFIG_ONLY_NO_BSC_TRANSACTION"

TGWT_BSC_NETWORK_TESTNET = "bsc_testnet"
TGWT_BSC_NETWORK_MAINNET = "bsc_mainnet"

TGWT_BSC_TESTNET_CHAIN_ID = 97
TGWT_BSC_MAINNET_CHAIN_ID = 56

TGWT_CONTRACT_PLACEHOLDER = "0x0000000000000000000000000000000000000000"

TGWT_BSC_CLAIM_CONFIG_SAFE_BOUNDARY = {
    "configOnly": True,
    "offChainOnly": True,
    "userPaysBscFee": True,
    "claimModel": "USER_PAID_BSC_CLAIM",
    "contractAddressRequiredNow": False,
    "contractAddressRequiredWhenClaimEnabled": True,
    "tokenContractAddressRequiredWhenClaimEnabled": True,
    "claimContractAddressRequiredWhenClaimEnabled": True,
    "testnetMainnetSeparated": True,
    "createsBscTransaction": False,
    "deploysContract": False,
    "sendsTGWT": False,
    "autoPayout": False,
    "storesPrivateKey": False,
    "signsTransaction": False,
    "paysGas": False,
    "usesHotWallet": False,
    "usesTreasuryWallet": False,
    "mintsToken": False,
    "burnsToken": False,
    "changesQCBalances": False,
    "changesTGWTOnChainBalances": False,
    "changesMempool": False,
    "changesChain": False,
    "replacesChain": False,
    "changesConsensus": False,
    "changesTotalSupply": False,
    "startsMining": False,
    "stopsMining": False,
}


def tgwt_is_zero_address(value: Any) -> bool:
    return str(value or "").strip().lower() == TGWT_CONTRACT_PLACEHOLDER.lower()


def tgwt_is_bsc_contract_address(value: Any) -> bool:
    s = str(value or "").strip()
    if not (s.startswith("0x") and len(s) == 42):
        return False
    if tgwt_is_zero_address(s):
        return False
    body = s[2:]
    for ch in body:
        if ch not in "0123456789abcdefABCDEF":
            return False
    return True


def tgwt_bsc_network_config(network: str = TGWT_BSC_NETWORK_TESTNET) -> Dict[str, Any]:
    network = str(network or "").strip().lower()

    if network == TGWT_BSC_NETWORK_MAINNET:
        chain_id = TGWT_BSC_MAINNET_CHAIN_ID
        label = "BSC_MAINNET"
        mainnet = True
    else:
        chain_id = TGWT_BSC_TESTNET_CHAIN_ID
        label = "BSC_TESTNET"
        mainnet = False
        network = TGWT_BSC_NETWORK_TESTNET

    return {
        "version": TGWT_BSC_CLAIM_CONFIG_VERSION,
        "status": TGWT_BSC_CLAIM_CONFIG_STATUS,
        "network": network,
        "networkLabel": label,
        "chainId": chain_id,
        "isMainnet": mainnet,
        "isTestnet": not mainnet,
        "claimEnabled": False,
        "tokenContract": TGWT_CONTRACT_PLACEHOLDER,
        "claimContract": TGWT_CONTRACT_PLACEHOLDER,
        "claimModel": "USER_PAID_BSC_CLAIM",
        "feeModel": "BNB_GAS_PAID_BY_USER",
        "networkSeparation": {
            "qcNativeNetworkOnly": True,
            "bscClaimLayerSeparate": True,
            "telegramGameIsClaimGateway": True,
            "mixesQCAndBSC": False,
        },
        "safeBoundary": dict(TGWT_BSC_CLAIM_CONFIG_SAFE_BOUNDARY),
    }


def tgwt_bsc_claim_config(
    network: str = TGWT_BSC_NETWORK_TESTNET,
    claim_enabled: bool = False,
    token_contract: str = TGWT_CONTRACT_PLACEHOLDER,
    claim_contract: str = TGWT_CONTRACT_PLACEHOLDER,
) -> Dict[str, Any]:
    cfg = tgwt_bsc_network_config(network)
    cfg["claimEnabled"] = bool(claim_enabled)
    cfg["tokenContract"] = str(token_contract or "").strip()
    cfg["claimContract"] = str(claim_contract or "").strip()
    cfg["contractAddressPolicy"] = {
        "requiredNow": False,
        "requiredWhenClaimEnabled": True,
        "tokenContractAddressRequiredWhenClaimEnabled": True,
        "claimContractAddressRequiredWhenClaimEnabled": True,
    }
    valid, reason = tgwt_validate_bsc_claim_config(cfg)
    cfg["valid"] = valid
    cfg["validationReason"] = reason
    return cfg


def tgwt_validate_bsc_claim_config(config: Dict[str, Any]) -> Tuple[bool, str]:
    cfg = dict(config or {})
    network = str(cfg.get("network", "")).strip().lower()
    chain_id = int(cfg.get("chainId", 0) or 0)
    claim_enabled = bool(cfg.get("claimEnabled", False))
    token_contract = str(cfg.get("tokenContract", "")).strip()
    claim_contract = str(cfg.get("claimContract", "")).strip()

    if network == TGWT_BSC_NETWORK_TESTNET:
        if chain_id != TGWT_BSC_TESTNET_CHAIN_ID:
            return False, "testnet_chain_id_mismatch"
    elif network == TGWT_BSC_NETWORK_MAINNET:
        if chain_id != TGWT_BSC_MAINNET_CHAIN_ID:
            return False, "mainnet_chain_id_mismatch"
    else:
        return False, "unsupported_bsc_network"

    if not claim_enabled:
        return True, "claim_disabled_contract_addresses_not_required"

    if not tgwt_is_bsc_contract_address(token_contract):
        return False, "missing_or_invalid_token_contract"

    if not tgwt_is_bsc_contract_address(claim_contract):
        return False, "missing_or_invalid_claim_contract"

    if token_contract.lower() == claim_contract.lower():
        return False, "token_and_claim_contract_must_differ"

    return True, "claim_enabled_config_valid"


def tgwt_bsc_claim_frontend_payload(config: Dict[str, Any]) -> Dict[str, Any]:
    cfg = dict(config or {})
    return {
        "version": TGWT_BSC_CLAIM_CONFIG_VERSION,
        "network": cfg.get("network", TGWT_BSC_NETWORK_TESTNET),
        "chainId": int(cfg.get("chainId", TGWT_BSC_TESTNET_CHAIN_ID) or TGWT_BSC_TESTNET_CHAIN_ID),
        "claimEnabled": bool(cfg.get("claimEnabled", False)),
        "tokenContract": str(cfg.get("tokenContract", TGWT_CONTRACT_PLACEHOLDER)),
        "claimContract": str(cfg.get("claimContract", TGWT_CONTRACT_PLACEHOLDER)),
        "claimModel": "USER_PAID_BSC_CLAIM",
        "feeModel": "BNB_GAS_PAID_BY_USER",
        "backendPaysGas": False,
        "backendSendsToken": False,
        "createsBscTransaction": False,
        "requiresMetaMask": True,
        "requiresUserWalletSignature": True,
        "safeBoundary": dict(TGWT_BSC_CLAIM_CONFIG_SAFE_BOUNDARY),
    }

# TGWT_BSC_CLAIM_MAINNET_TESTNET_LOCK_V1_BRIDGE
# Runtime-safe bridge to the core public config guard.
def tgwt_bsc_claim_public_runtime_config_v1():
    from telegram_game.tgwt_bsc_claim_safety_v1 import tgwt_bsc_public_config_from_env_v1

    return tgwt_bsc_public_config_from_env_v1()


def tgwt_validate_bsc_claim_public_runtime_config_v1():
    from telegram_game.tgwt_bsc_claim_safety_v1 import tgwt_assert_bsc_public_config_safe_v1

    return tgwt_assert_bsc_public_config_safe_v1()


def tgwt_bsc_claim_frontend_public_payload_runtime_v1():
    from telegram_game.tgwt_bsc_claim_safety_v1 import tgwt_bsc_claim_frontend_public_payload_v1

    return tgwt_bsc_claim_frontend_public_payload_v1()
