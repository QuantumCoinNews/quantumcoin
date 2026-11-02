# TGWT_BSC_CLAIM_MAINNET_TESTNET_LOCK_V1
# Core safety boundary for TGWT BSC claim configuration.
#
# Rules:
# - TGWT claim is user-paid on BSC. User pays BNB gas/fee.
# - Backend must not create BSC transactions.
# - Backend must not store/use treasury private key, seed, mnemonic, or hot-wallet secret.
# - Distribution wallet / token contract / claim contract are public addresses only.
# - Testnet and mainnet BSC claim configs must not be mixed.

import os
import re
from typing import Any, Dict, List, Tuple

TGWT_BSC_CLAIM_LOCK_VERSION = "TGWT_BSC_CLAIM_MAINNET_TESTNET_LOCK_V1"

TGWT_BSC_NETWORK_TESTNET = "bsc_testnet"
TGWT_BSC_NETWORK_MAINNET = "bsc_mainnet"

TGWT_BSC_TESTNET_CHAIN_ID = 97
TGWT_BSC_MAINNET_CHAIN_ID = 56

TGWT_BSC_ZERO_ADDRESS = "0x0000000000000000000000000000000000000000"

TGWT_PRIVATE_SECRET_ENV_NAMES = [
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
]

TGWT_PUBLIC_CONFIG_ENV_NAMES = {
    "claim_enabled": "TGWT_BSC_CLAIM_ENABLED",
    "network": "TGWT_BSC_NETWORK",
    "chain_id": "TGWT_BSC_CHAIN_ID",
    "distribution_wallet_address": "TGWT_BSC_DISTRIBUTION_WALLET_ADDRESS",
    "token_contract_address": "TGWT_BSC_TOKEN_CONTRACT_ADDRESS",
    "claim_contract_address": "TGWT_BSC_CLAIM_CONTRACT_ADDRESS",
    "mainnet_approval": "TGWT_BSC_MAINNET_CLAIM_APPROVAL",
}

TGWT_SAFE_BOUNDARY_V1 = {
    "claimModel": "USER_PAID_BSC_CLAIM",
    "userPaysBscFee": True,
    "feeAsset": "BNB",
    "backendPaysGas": False,
    "createsBscTransaction": False,
    "storesPrivateKey": False,
    "usesPrivateKey": False,
    "requiresConnectedBscWallet": True,
    "requiresUserWalletSignatureLater": True,
    "bscClaimLayerSeparate": True,
    "mixesQCAndBSC": False,
}


class TGWTBSCClaimSafetyError(ValueError):
    pass


def _env(name: str, default: str = "") -> str:
    return str(os.getenv(name) or default).strip()


def _env_bool(name: str) -> bool:
    return _env(name).lower() in {"1", "true", "yes", "on", "enabled"}


def _as_int(value: Any, default: int) -> int:
    try:
        if value is None or str(value).strip() == "":
            return int(default)
        return int(str(value).strip())
    except Exception:
        return int(default)


def tgwt_is_bsc_address_v1(value: Any) -> bool:
    text = str(value or "").strip()
    if not re.fullmatch(r"0x[a-fA-F0-9]{40}", text):
        return False
    return text.lower() != TGWT_BSC_ZERO_ADDRESS.lower()


def tgwt_private_secret_env_hits_v1() -> List[str]:
    hits = []
    for name in TGWT_PRIVATE_SECRET_ENV_NAMES:
        if _env(name):
            hits.append(name)
    return hits


def tgwt_resolve_bsc_network_v1(network: str = "") -> str:
    raw = (str(network or "").strip().lower() or _env("TGWT_BSC_NETWORK").lower())
    if raw in {"mainnet", "main", "bsc", "bsc_mainnet", "56"}:
        return TGWT_BSC_NETWORK_MAINNET
    if raw in {"testnet", "test", "bsc_testnet", "97"}:
        return TGWT_BSC_NETWORK_TESTNET

    # Project-level testnet mode defaults TGWT BSC to BSC testnet.
    project_mode = " ".join([
        _env("QC_TESTNET"),
        _env("QC_CHAIN_MODE"),
        _env("QC_NETWORK"),
        _env("TELEGRAM_GAME_CHAIN_MODE"),
        _env("TELEGRAM_GAME_NETWORK"),
    ]).lower()
    if "testnet" in project_mode or _env_bool("QC_TESTNET"):
        return TGWT_BSC_NETWORK_TESTNET

    return TGWT_BSC_NETWORK_TESTNET


def tgwt_expected_chain_id_v1(network: str) -> int:
    network = tgwt_resolve_bsc_network_v1(network)
    if network == TGWT_BSC_NETWORK_MAINNET:
        return TGWT_BSC_MAINNET_CHAIN_ID
    return TGWT_BSC_TESTNET_CHAIN_ID


def tgwt_bsc_public_config_from_env_v1() -> Dict[str, Any]:
    network = tgwt_resolve_bsc_network_v1()
    default_chain = tgwt_expected_chain_id_v1(network)

    return {
        "version": TGWT_BSC_CLAIM_LOCK_VERSION,
        "claimEnabled": _env_bool("TGWT_BSC_CLAIM_ENABLED"),
        "network": network,
        "chainId": _as_int(_env("TGWT_BSC_CHAIN_ID"), default_chain),
        "distributionWalletAddress": _env("TGWT_BSC_DISTRIBUTION_WALLET_ADDRESS"),
        "tokenContractAddress": _env("TGWT_BSC_TOKEN_CONTRACT_ADDRESS"),
        "claimContractAddress": _env("TGWT_BSC_CLAIM_CONTRACT_ADDRESS"),
        "mainnetApproval": _env("TGWT_BSC_MAINNET_CLAIM_APPROVAL"),
        "claimModel": "USER_PAID_BSC_CLAIM",
        "userPaysBscFee": True,
        "feeAsset": "BNB",
        "backendPaysGas": False,
        "createsBscTransaction": False,
        "safeBoundary": dict(TGWT_SAFE_BOUNDARY_V1),
        "publicOnly": True,
    }


def tgwt_validate_bsc_public_config_v1(config: Dict[str, Any]) -> Tuple[bool, str]:
    cfg = dict(config or {})

    secret_hits = tgwt_private_secret_env_hits_v1()
    if secret_hits:
        return False, "private_secret_env_forbidden:" + ",".join(secret_hits)

    network = tgwt_resolve_bsc_network_v1(str(cfg.get("network", "")))
    chain_id = _as_int(cfg.get("chainId"), tgwt_expected_chain_id_v1(network))
    expected_chain = tgwt_expected_chain_id_v1(network)
    claim_enabled = bool(cfg.get("claimEnabled", False))

    if chain_id != expected_chain:
        return False, "bsc_chain_id_network_mismatch"

    if network == TGWT_BSC_NETWORK_MAINNET:
        approval = str(cfg.get("mainnetApproval", "") or "").strip()
        if approval != "I_APPROVE_TGWT_BSC_MAINNET_CLAIM":
            return False, "mainnet_claim_requires_explicit_manual_approval"

    if bool(cfg.get("backendPaysGas", False)):
        return False, "backend_gas_payment_forbidden"

    if bool(cfg.get("createsBscTransaction", False)):
        return False, "backend_bsc_transaction_forbidden"

    if cfg.get("userPaysBscFee") is not True:
        return False, "user_pays_bsc_fee_required"

    if str(cfg.get("claimModel", "")) != "USER_PAID_BSC_CLAIM":
        return False, "claim_model_must_be_user_paid_bsc_claim"

    if claim_enabled:
        if not tgwt_is_bsc_address_v1(cfg.get("distributionWalletAddress")):
            return False, "valid_distribution_wallet_required_when_claim_enabled"
        if not tgwt_is_bsc_address_v1(cfg.get("tokenContractAddress")):
            return False, "valid_token_contract_required_when_claim_enabled"
        if not tgwt_is_bsc_address_v1(cfg.get("claimContractAddress")):
            return False, "valid_claim_contract_required_when_claim_enabled"

    return True, "ok"


def tgwt_assert_bsc_public_config_safe_v1(config: Dict[str, Any] = None) -> Dict[str, Any]:
    cfg = tgwt_bsc_public_config_from_env_v1() if config is None else dict(config)
    valid, reason = tgwt_validate_bsc_public_config_v1(cfg)
    if not valid:
        raise TGWTBSCClaimSafetyError(reason)
    return cfg


def tgwt_bsc_claim_frontend_public_payload_v1(config: Dict[str, Any] = None) -> Dict[str, Any]:
    cfg = tgwt_assert_bsc_public_config_safe_v1(config)
    return {
        "version": TGWT_BSC_CLAIM_LOCK_VERSION,
        "claimEnabled": bool(cfg.get("claimEnabled", False)),
        "network": cfg.get("network", TGWT_BSC_NETWORK_TESTNET),
        "chainId": int(cfg.get("chainId", TGWT_BSC_TESTNET_CHAIN_ID)),
        "distributionWalletAddress": cfg.get("distributionWalletAddress", ""),
        "tokenContractAddress": cfg.get("tokenContractAddress", ""),
        "claimContractAddress": cfg.get("claimContractAddress", ""),
        "claimModel": "USER_PAID_BSC_CLAIM",
        "userPaysBscFee": True,
        "feeAsset": "BNB",
        "backendPaysGas": False,
        "createsBscTransaction": False,
        "requiresConnectedBscWallet": True,
        "requiresUserWalletSignatureLater": True,
        "safeBoundary": dict(TGWT_SAFE_BOUNDARY_V1),
    }


def tgwt_claim_core_todo_v1() -> List[str]:
    return [
        "wire_frontend_wallet_connect_later",
        "wire_user_wallet_signature_later",
        "set_distribution_wallet_public_address_in_private_env",
        "set_tgwt_token_contract_public_address_in_private_env",
        "set_claim_contract_public_address_in_private_env",
        "keep_private_keys_out_of_code_git_ui",
        "keep_backend_bsc_transaction_creation_disabled",
        "keep_user_paid_bnb_gas_model",
    ]
