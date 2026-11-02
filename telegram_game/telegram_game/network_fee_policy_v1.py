# QC_NETWORK_FEE_POLICY_V1
# Micro-low network fee policy for QuantumCoin ecosystem.
#
# Boundary:
# - Control/validation only.
# - Does not execute fee transfer.
# - Does not mint.
# - Does not change balances, chain, consensus, reward, or max supply.

from typing import Any, Dict, List

QC_NETWORK_FEE_POLICY_VERSION = "QC_NETWORK_FEE_POLICY_V1"

MAX_SUPPLY_QC = 25_500_000
BASE_FEE_TARGET_QC = "0.000001"
CONGESTION_CEILING_QC = "0.00001"
FEE_RECIPIENT = "BLOCK_PRODUCER_MINER_OR_NODE_OPERATOR"
FEE_MODE = "MICRO_FEE_POLICY_FINAL_PARAMS_BEFORE_MAINNET"

SAFE_BOUNDARY = {
    "doesNotMint": True,
    "doesNotChangeMaxSupply": True,
    "doesNotChangeBalances": True,
    "doesNotSendFunds": True,
    "doesNotExecuteFeeTransfer": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeMiningRewardNow": True,
    "feeIsNotFounderIncome": True,
    "feeIsNotBurnByDefault": True,
    "feeGoesToBlockProducer": True,
    "feeMustStayMicroLow": True,
    "requiresFinalFeeParamsBeforeMainnet": True,
    "aiCannotChangeFee": True,
    "adminCannotAutoChangeFee": True,
    "gameCannotChangeFee": True,
    "workerCannotChangeFee": True,
    "controlOnly": True,
}

def default_network_fee_policy_v1() -> Dict[str, Any]:
    return {
        "version": QC_NETWORK_FEE_POLICY_VERSION,
        "mode": FEE_MODE,
        "maxSupplyQc": MAX_SUPPLY_QC,
        "baseFeeTargetQc": BASE_FEE_TARGET_QC,
        "congestionCeilingQc": CONGESTION_CEILING_QC,
        "feeRecipient": FEE_RECIPIENT,
        "miningActiveFeeDestination": "BLOCK_REWARD_PLUS_TX_FEES_TO_BLOCK_PRODUCER",
        "afterEmissionDoneFeeDestination": "TX_FEES_ONLY_TO_NODE_OR_BLOCK_PRODUCER",
        "feeIsNewMint": False,
        "feeChangesMaxSupply": False,
        "feeGoesToFounder": False,
        "feeGoesToBurnByDefault": False,
        "feeBurnOptionalFutureGovernanceOnly": True,
        "feeMustStayMicroLow": True,
        "finalFeeParamsRequiredBeforeMainnet": True,
        "aiCannotChangeFee": True,
        "adminPanelCannotAutoChangeFee": True,
        "gameCannotChangeFee": True,
        "workerCannotChangeFee": True,
        "nftCannotChangeFee": True,
        "tgwtCannotChangeFee": True,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def validate_network_fee_policy_v1(policy: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if policy.get("version") != QC_NETWORK_FEE_POLICY_VERSION:
        errors.append("invalid_version")
    if policy.get("mode") != FEE_MODE:
        errors.append("invalid_mode")
    if policy.get("maxSupplyQc") != MAX_SUPPLY_QC:
        errors.append("invalid_max_supply")
    if policy.get("baseFeeTargetQc") != BASE_FEE_TARGET_QC:
        errors.append("invalid_base_fee_target")
    if policy.get("congestionCeilingQc") != CONGESTION_CEILING_QC:
        errors.append("invalid_congestion_ceiling")
    if policy.get("feeRecipient") != FEE_RECIPIENT:
        errors.append("invalid_fee_recipient")
    if policy.get("feeIsNewMint") is not False:
        errors.append("fee_must_not_be_new_mint")
    if policy.get("feeChangesMaxSupply") is not False:
        errors.append("fee_must_not_change_max_supply")
    if policy.get("feeGoesToFounder") is not False:
        errors.append("fee_must_not_go_to_founder")
    if policy.get("feeGoesToBurnByDefault") is not False:
        errors.append("fee_must_not_burn_by_default")
    if policy.get("feeMustStayMicroLow") is not True:
        errors.append("fee_must_stay_micro_low")
    if policy.get("finalFeeParamsRequiredBeforeMainnet") is not True:
        errors.append("final_fee_params_required_before_mainnet")
    for key in ["aiCannotChangeFee", "adminPanelCannotAutoChangeFee", "gameCannotChangeFee", "workerCannotChangeFee", "nftCannotChangeFee", "tgwtCannotChangeFee"]:
        if policy.get(key) is not True:
            errors.append(key + "_required")

    safe = policy.get("safeBoundary") or {}
    for key in [
        "doesNotMint",
        "doesNotChangeMaxSupply",
        "doesNotChangeBalances",
        "doesNotSendFunds",
        "doesNotExecuteFeeTransfer",
        "doesNotChangeConsensus",
        "feeIsNotFounderIncome",
        "feeIsNotBurnByDefault",
        "feeGoesToBlockProducer",
        "feeMustStayMicroLow",
        "requiresFinalFeeParamsBeforeMainnet",
        "aiCannotChangeFee",
        "adminCannotAutoChangeFee",
        "gameCannotChangeFee",
        "workerCannotChangeFee",
        "controlOnly",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_NETWORK_FEE_POLICY_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def network_fee_summary_v1() -> Dict[str, Any]:
    policy = default_network_fee_policy_v1()
    return {
        "ok": True,
        "version": QC_NETWORK_FEE_POLICY_VERSION,
        "policy": policy,
        "validation": validate_network_fee_policy_v1(policy),
    }
