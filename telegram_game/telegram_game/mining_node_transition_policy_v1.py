# QC_MINING_PERIOD_BITCOIN_STYLE_CONTINUE_V1
# Primary 10-year mining period, 2-year halving, and bitcoin-style continuation.
#
# Boundary:
# - Control/validation only.
# - Does not mint directly.
# - Does not distribute remaining coins.
# - Does not start/stop mining.
# - Does not change balances, chain, consensus, reward, or max supply.

from typing import Any, Dict, List

QC_MINING_PERIOD_NODE_TRANSITION_VERSION = "QC_MINING_PERIOD_BITCOIN_STYLE_CONTINUE_V1"

MAX_SUPPLY_QC = 25_500_000
INITIAL_REWARD_QC = 50
PRIMARY_MINING_PERIOD_YEARS = 10
HALVING_INTERVAL_YEARS = 2

PRIMARY_REWARD_SCHEDULE = [
    {"index": 1, "fromYear": 0, "toYear": 2, "rewardQc": "50"},
    {"index": 2, "fromYear": 2, "toYear": 4, "rewardQc": "25"},
    {"index": 3, "fromYear": 4, "toYear": 6, "rewardQc": "12.5"},
    {"index": 4, "fromYear": 6, "toYear": 8, "rewardQc": "6.25"},
    {"index": 5, "fromYear": 8, "toYear": 10, "rewardQc": "3.125"},
]

POST_TEN_YEAR_EXAMPLE_SCHEDULE = [
    {"index": 6, "fromYear": 10, "toYear": 12, "rewardQc": "1.5625"},
    {"index": 7, "fromYear": 12, "toYear": 14, "rewardQc": "0.78125"},
    {"index": 8, "fromYear": 14, "toYear": 16, "rewardQc": "0.390625"},
]

SAFE_BOUNDARY = {
    "doesNotMint": True,
    "doesNotAutoMintRemaining": True,
    "doesNotAdminMintRemaining": True,
    "doesNotAIDistributeRemaining": True,
    "doesNotGameDistributeRemaining": True,
    "doesNotWorkerDistributeRemaining": True,
    "doesNotChangeMaxSupply": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeMiningRewardNow": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "requiresTenYearPrimaryMiningPeriod": True,
    "requiresTwoYearHalving": True,
    "requiresWalletMinerContinueIfIncomplete": True,
    "requiresBitcoinStyleHalvingAfterTenYears": True,
    "requiresRewardCapByRemainingSupply": True,
    "requiresNodeFeeModeAfterEmissionDone": True,
    "forbidsNodeFeeOnlyWhileSupplyRemains": True,
    "controlOnly": True,
}

def default_mining_node_transition_policy_v1() -> Dict[str, Any]:
    return {
        "version": QC_MINING_PERIOD_NODE_TRANSITION_VERSION,
        "mode": "MINING_PERIOD_BITCOIN_STYLE_CONTINUE_NO_MINT_EXECUTION",
        "maxSupplyQc": MAX_SUPPLY_QC,
        "initialRewardQc": INITIAL_REWARD_QC,
        "primaryMiningPeriodYears": PRIMARY_MINING_PERIOD_YEARS,
        "halvingIntervalYears": HALVING_INTERVAL_YEARS,
        "primaryRewardSchedule": list(PRIMARY_REWARD_SCHEDULE),
        "postTenYearExampleSchedule": list(POST_TEN_YEAR_EXAMPLE_SCHEDULE),
        "ifFullyMinedWithinPrimaryPeriod": "NODE_FEE_MODE_AFTER_EMISSION_COMPLETE",
        "ifNotFullyMinedAfterPrimaryPeriod": "WALLET_MINER_CONTINUES_WITH_BITCOIN_STYLE_HALVING_UNTIL_MAX_SUPPLY",
        "walletMinerContinuesIfIncomplete": True,
        "bitcoinStyleHalvingContinues": True,
        "miningContinuesOnlyUntilMaxSupply": True,
        "rewardMustBeCappedByRemainingSupply": True,
        "autoMintRemainingForbidden": True,
        "adminMintRemainingForbidden": True,
        "aiCannotDistributeRemaining": True,
        "gameCannotDistributeRemaining": True,
        "workerCannotDistributeRemaining": True,
        "nodeFeeModeEnabledAfterEmissionDone": True,
        "nodeFeeOnlyNotForcedWhileSupplyRemains": True,
        "maxSupplyAlwaysUnchanged": True,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def validate_mining_node_transition_policy_v1(policy: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if policy.get("version") != QC_MINING_PERIOD_NODE_TRANSITION_VERSION:
        errors.append("invalid_version")
    if policy.get("mode") != "MINING_PERIOD_BITCOIN_STYLE_CONTINUE_NO_MINT_EXECUTION":
        errors.append("invalid_mode")
    if policy.get("maxSupplyQc") != MAX_SUPPLY_QC:
        errors.append("invalid_max_supply")
    if policy.get("initialRewardQc") != INITIAL_REWARD_QC:
        errors.append("invalid_initial_reward")
    if policy.get("primaryMiningPeriodYears") != PRIMARY_MINING_PERIOD_YEARS:
        errors.append("invalid_primary_mining_period")
    if policy.get("halvingIntervalYears") != HALVING_INTERVAL_YEARS:
        errors.append("invalid_halving_interval")
    if policy.get("primaryRewardSchedule") != PRIMARY_REWARD_SCHEDULE:
        errors.append("invalid_primary_reward_schedule")
    if policy.get("postTenYearExampleSchedule") != POST_TEN_YEAR_EXAMPLE_SCHEDULE:
        errors.append("invalid_post_ten_year_example_schedule")
    if policy.get("ifFullyMinedWithinPrimaryPeriod") != "NODE_FEE_MODE_AFTER_EMISSION_COMPLETE":
        errors.append("invalid_full_emission_transition")
    if policy.get("ifNotFullyMinedAfterPrimaryPeriod") != "WALLET_MINER_CONTINUES_WITH_BITCOIN_STYLE_HALVING_UNTIL_MAX_SUPPLY":
        errors.append("invalid_incomplete_emission_transition")
    if policy.get("walletMinerContinuesIfIncomplete") is not True:
        errors.append("wallet_miner_continue_required_if_incomplete")
    if policy.get("bitcoinStyleHalvingContinues") is not True:
        errors.append("bitcoin_style_halving_continue_required")
    if policy.get("miningContinuesOnlyUntilMaxSupply") is not True:
        errors.append("mining_until_max_supply_required")
    if policy.get("rewardMustBeCappedByRemainingSupply") is not True:
        errors.append("reward_cap_by_remaining_supply_required")
    if policy.get("autoMintRemainingForbidden") is not True:
        errors.append("auto_mint_remaining_forbidden_required")
    if policy.get("adminMintRemainingForbidden") is not True:
        errors.append("admin_mint_remaining_forbidden_required")
    if policy.get("aiCannotDistributeRemaining") is not True:
        errors.append("ai_remaining_distribution_forbidden_required")
    if policy.get("gameCannotDistributeRemaining") is not True:
        errors.append("game_remaining_distribution_forbidden_required")
    if policy.get("workerCannotDistributeRemaining") is not True:
        errors.append("worker_remaining_distribution_forbidden_required")
    if policy.get("nodeFeeModeEnabledAfterEmissionDone") is not True:
        errors.append("node_fee_mode_after_emission_done_required")
    if policy.get("nodeFeeOnlyNotForcedWhileSupplyRemains") is not True:
        errors.append("node_fee_only_forbidden_while_supply_remains")
    if policy.get("maxSupplyAlwaysUnchanged") is not True:
        errors.append("max_supply_unchanged_required")

    safe = policy.get("safeBoundary") or {}
    for key in [
        "doesNotMint",
        "doesNotAutoMintRemaining",
        "doesNotAdminMintRemaining",
        "doesNotAIDistributeRemaining",
        "doesNotGameDistributeRemaining",
        "doesNotWorkerDistributeRemaining",
        "doesNotChangeMaxSupply",
        "doesNotChangeBalances",
        "doesNotChangeChain",
        "doesNotChangeConsensus",
        "requiresTenYearPrimaryMiningPeriod",
        "requiresTwoYearHalving",
        "requiresWalletMinerContinueIfIncomplete",
        "requiresBitcoinStyleHalvingAfterTenYears",
        "requiresRewardCapByRemainingSupply",
        "requiresNodeFeeModeAfterEmissionDone",
        "forbidsNodeFeeOnlyWhileSupplyRemains",
        "controlOnly",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_MINING_PERIOD_NODE_TRANSITION_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def mining_node_transition_summary_v1() -> Dict[str, Any]:
    policy = default_mining_node_transition_policy_v1()
    return {
        "ok": True,
        "version": QC_MINING_PERIOD_NODE_TRANSITION_VERSION,
        "policy": policy,
        "validation": validate_mining_node_transition_policy_v1(policy),
    }
