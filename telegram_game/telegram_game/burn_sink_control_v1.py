# QC_BURN_SINK_CONTROL_V1
# Burn/dead wallet control for Telegram Game + QuantumCoin ecosystem.
#
# Boundary:
# - Control/validation only.
# - no burn execution is performed by this module.
# - Does not send funds.
# - Does not mint replacement QC.
# - Does not change balances.
# - Does not change chain, consensus, mining reward, or max supply.

from typing import Any, Dict, List

QC_BURN_SINK_CONTROL_VERSION = "QC_BURN_SINK_CONTROL_V1"

MAX_SUPPLY_QC = 25_500_000
DEFAULT_BURN_ADDRESS = "QC_BURN_SINK"

SAFE_BOUNDARY = {
    "doesNotExecuteBurn": True,
    "doesNotSendFunds": True,
    "doesNotMint": True,
    "doesNotMintReplacement": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeMaxSupply": True,
    "doesNotChangeMiningReward": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "requiresBurnAddressFinalBeforeMainnet": True,
    "requiresUnspendableBurnAddress": True,
    "requiresNoBurnPrivateKey": True,
    "requiresExplicitBurnApproval": True,
    "burnReducesCirculatingSupplyOnly": True,
    "controlOnly": True,
}

def default_burn_sink_policy_v1() -> Dict[str, Any]:
    return {
        "version": QC_BURN_SINK_CONTROL_VERSION,
        "mode": "BURN_SINK_CONTROL_NO_BURN_EXECUTION",
        "maxSupplyQc": MAX_SUPPLY_QC,
        "burnAddressAlias": DEFAULT_BURN_ADDRESS,
        "burnAddressFinalBeforeMainnet": True,
        "burnAddressMustBeUnspendable": True,
        "burnPrivateKeyMustNotExist": True,
        "burnedSupplyReducesCirculatingSupply": True,
        "burnDoesNotReduceMaxSupply": True,
        "burnDoesNotMintReplacement": True,
        "autoBurnForbidden": True,
        "requiresExplicitApproval": True,
        "allowedBurnSources": [
            "explicit_user_burn",
            "explicit_admin_approved_burn",
        ],
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def validate_burn_sink_policy_v1(policy: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if policy.get("version") != QC_BURN_SINK_CONTROL_VERSION:
        errors.append("invalid_version")
    if policy.get("mode") != "BURN_SINK_CONTROL_NO_BURN_EXECUTION":
        errors.append("invalid_mode")
    if policy.get("maxSupplyQc") != MAX_SUPPLY_QC:
        errors.append("invalid_max_supply")
    if policy.get("burnAddressAlias") != DEFAULT_BURN_ADDRESS:
        errors.append("invalid_burn_address_alias")
    if policy.get("burnAddressFinalBeforeMainnet") is not True:
        errors.append("burn_address_final_before_mainnet_required")
    if policy.get("burnAddressMustBeUnspendable") is not True:
        errors.append("burn_address_must_be_unspendable")
    if policy.get("burnPrivateKeyMustNotExist") is not True:
        errors.append("burn_private_key_must_not_exist")
    if policy.get("burnedSupplyReducesCirculatingSupply") is not True:
        errors.append("burn_must_reduce_circulating_supply")
    if policy.get("burnDoesNotReduceMaxSupply") is not True:
        errors.append("burn_must_not_reduce_max_supply")
    if policy.get("burnDoesNotMintReplacement") is not True:
        errors.append("burn_must_not_mint_replacement")
    if policy.get("autoBurnForbidden") is not True:
        errors.append("auto_burn_forbidden_required")
    if policy.get("requiresExplicitApproval") is not True:
        errors.append("explicit_burn_approval_required")

    allowed = policy.get("allowedBurnSources")
    if not isinstance(allowed, list) or not allowed:
        errors.append("allowed_burn_sources_required")

    safe = policy.get("safeBoundary") or {}
    for key in [
        "doesNotExecuteBurn",
        "doesNotSendFunds",
        "doesNotMint",
        "doesNotMintReplacement",
        "doesNotChangeBalances",
        "doesNotChangeChain",
        "doesNotChangeConsensus",
        "doesNotChangeMaxSupply",
        "doesNotChangeMiningReward",
        "requiresBurnAddressFinalBeforeMainnet",
        "requiresUnspendableBurnAddress",
        "requiresNoBurnPrivateKey",
        "requiresExplicitBurnApproval",
        "burnReducesCirculatingSupplyOnly",
        "controlOnly",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_BURN_SINK_CONTROL_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def burn_sink_summary_v1() -> Dict[str, Any]:
    policy = default_burn_sink_policy_v1()
    return {
        "ok": True,
        "version": QC_BURN_SINK_CONTROL_VERSION,
        "policy": policy,
        "validation": validate_burn_sink_policy_v1(policy),
    }
