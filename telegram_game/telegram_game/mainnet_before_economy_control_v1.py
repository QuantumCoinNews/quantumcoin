# QC_MAINNET_BEFORE_ECONOMY_CONTROL_V1
# Mainnet-before economy invariant control for Telegram Game + QuantumCoin ecosystem.
#
# Boundary:
# - Pre-mainnet control only.
# - no mainnet launch is executed by this module.
# - Does not mint.
# - Does not send funds.
# - Does not change balances.
# - Does not change chain, consensus, mining reward, or total supply.

from typing import Any, Dict, List

QC_MAINNET_BEFORE_ECONOMY_CONTROL_VERSION = "QC_MAINNET_BEFORE_ECONOMY_CONTROL_V1"

TOTAL_SUPPLY_QC = 25_500_000
FOUNDER_TOTAL_BPS = 500
FOUNDER_LIQUID_BPS = 100
FOUNDER_LOCKED_BPS = 400
NON_FOUNDER_BPS = 9500
FULL_ALLOCATION_BPS = 10000
INITIAL_MINING_REWARD_QC = 50
COINBASE_MATURITY_BLOCKS = 10

SAFE_BOUNDARY = {
    "doesNotLaunchMainnet": True,
    "doesNotMint": True,
    "doesNotSendFunds": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeSupply": True,
    "doesNotChangeMiningReward": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "doesNotSelectNFTWinners": True,
    "doesNotExecuteSettlement": True,
    "preMainnetControlOnly": True,
    "requiresFinalHumanApproval": True,
    "requiresImmutableSupplyCheck": True,
    "requiresFounderSplitCheck": True,
    "requiresFounderWalletBinding": True,
    "requiresFinalFounderWalletAddress": True,
}

def default_mainnet_before_economy_policy_v1() -> Dict[str, Any]:
    return {
        "version": QC_MAINNET_BEFORE_ECONOMY_CONTROL_VERSION,
        "mode": "PRE_MAINNET_ECONOMY_CONTROL_NO_LAUNCH",
        "totalSupplyQc": TOTAL_SUPPLY_QC,
        "founderTotalBasisPoints": FOUNDER_TOTAL_BPS,
        "founderLiquidBasisPoints": FOUNDER_LIQUID_BPS,
        "founderLockedBasisPoints": FOUNDER_LOCKED_BPS,
        "founderWalletBindingRequired": True,
        "founderWalletAddressFinalBeforeMainnet": True,
        "initialMiningRewardQc": INITIAL_MINING_REWARD_QC,
        "coinbaseMaturityBlocks": COINBASE_MATURITY_BLOCKS,
        "allocations": [
            {
                "name": "founder_liquid_operations",
                "basisPoints": FOUNDER_LIQUID_BPS,
                "locked": False,
                "liquid": True,
                "mintsNewQc": False,
            },
            {
                "name": "founder_locked_vesting",
                "basisPoints": FOUNDER_LOCKED_BPS,
                "locked": True,
                "liquid": False,
                "mintsNewQc": False,
            },
            {
                "name": "non_founder_network_supply",
                "basisPoints": NON_FOUNDER_BPS,
                "locked": False,
                "liquid": False,
                "mintsNewQc": False,
            },
        ],
        "safeBoundary": dict(SAFE_BOUNDARY),
        "notes": {
            "status": "Pre-mainnet economy control only. This does not launch mainnet.",
            "supply": "Total supply is fixed at 25,500,000 QC.",
            "founder": "Founder/main allocation is 5% total: 1% liquid and 4% locked/vesting. Final founder wallet address binding is required before mainnet.",
            "ai": "AI can monitor, score, and report only. AI cannot mint, change balances, change consensus, or alter supply.",
            "tgwt": "TGWT cannot mint QC or change QC supply.",
            "nft": "NFT flow cannot change QC supply, QC balances, mining rewards, or consensus.",
        },
    }

def validate_mainnet_before_economy_policy_v1(policy: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if policy.get("version") != QC_MAINNET_BEFORE_ECONOMY_CONTROL_VERSION:
        errors.append("invalid_version")
    if policy.get("mode") != "PRE_MAINNET_ECONOMY_CONTROL_NO_LAUNCH":
        errors.append("invalid_mode")
    if policy.get("totalSupplyQc") != TOTAL_SUPPLY_QC:
        errors.append("invalid_total_supply")
    if policy.get("founderTotalBasisPoints") != FOUNDER_TOTAL_BPS:
        errors.append("invalid_founder_total")
    if policy.get("founderLiquidBasisPoints") != FOUNDER_LIQUID_BPS:
        errors.append("invalid_founder_liquid")
    if policy.get("founderLockedBasisPoints") != FOUNDER_LOCKED_BPS:
        errors.append("invalid_founder_locked")
    if policy.get("founderLiquidBasisPoints", 0) + policy.get("founderLockedBasisPoints", 0) != policy.get("founderTotalBasisPoints"):
        errors.append("invalid_founder_split_sum")
    if policy.get("founderWalletBindingRequired") is not True:
        errors.append("founder_wallet_binding_required")
    if policy.get("founderWalletAddressFinalBeforeMainnet") is not True:
        errors.append("founder_wallet_address_final_before_mainnet_required")
    if policy.get("initialMiningRewardQc") != INITIAL_MINING_REWARD_QC:
        errors.append("invalid_initial_mining_reward")
    if policy.get("coinbaseMaturityBlocks") != COINBASE_MATURITY_BLOCKS:
        errors.append("invalid_coinbase_maturity")

    allocations = policy.get("allocations")
    if not isinstance(allocations, list) or not allocations:
        errors.append("allocations_required")
        allocations = []

    seen = set()
    total_bps = 0
    for allocation in allocations:
        name = str(allocation.get("name") or "").strip()
        if not name:
            errors.append("allocation_name_required")
            continue
        if name in seen:
            errors.append("duplicate_allocation:" + name)
        seen.add(name)

        bps = allocation.get("basisPoints")
        if not isinstance(bps, int) or bps <= 0:
            errors.append("invalid_allocation_bps:" + name)
        else:
            total_bps += bps

        if allocation.get("mintsNewQc") is True:
            errors.append("allocation_must_not_mint_qc:" + name)

    for required in [
        "founder_liquid_operations",
        "founder_locked_vesting",
        "non_founder_network_supply",
    ]:
        if required not in seen:
            errors.append("missing_required_allocation:" + required)

    if total_bps != FULL_ALLOCATION_BPS:
        errors.append("allocation_total_must_equal_10000")

    safe = policy.get("safeBoundary") or {}
    for key in [
        "doesNotLaunchMainnet",
        "doesNotMint",
        "doesNotSendFunds",
        "doesNotChangeBalances",
        "doesNotChangeChain",
        "doesNotChangeConsensus",
        "doesNotChangeSupply",
        "doesNotChangeMiningReward",
        "doesNotSelectNFTWinners",
        "doesNotExecuteSettlement",
        "requiresFinalHumanApproval",
        "requiresImmutableSupplyCheck",
        "requiresFounderSplitCheck",
        "requiresFounderWalletBinding",
        "requiresFinalFounderWalletAddress",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_MAINNET_BEFORE_ECONOMY_CONTROL_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def mainnet_before_economy_summary_v1() -> Dict[str, Any]:
    policy = default_mainnet_before_economy_policy_v1()
    return {
        "ok": True,
        "version": QC_MAINNET_BEFORE_ECONOMY_CONTROL_VERSION,
        "policy": policy,
        "validation": validate_mainnet_before_economy_policy_v1(policy),
    }
