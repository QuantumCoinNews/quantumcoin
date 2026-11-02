# QC_FOUNDER_VESTING_POLICY_V1
# Founder allocation and vesting policy.
#
# Boundary:
# - Control/validation only.
# - Does not execute unlock.
# - Does not mint.
# - Does not change balances, chain, consensus, reward, or max supply.

from typing import Any, Dict, List

QC_FOUNDER_VESTING_POLICY_VERSION = "QC_FOUNDER_VESTING_POLICY_V1"

MAX_SUPPLY_QC = 25_500_000
FOUNDER_TOTAL_QC = 1_275_000
FOUNDER_LIQUID_QC = 255_000
FOUNDER_LOCKED_QC = 1_020_000
FOUNDER_TOTAL_PERCENT = 5
FOUNDER_LIQUID_PERCENT = 1
FOUNDER_LOCKED_PERCENT = 4
CLIFF_MONTHS = 12
VESTING_MONTHS = 48
TOTAL_SCHEDULE_MONTHS = 60
MONTHLY_UNLOCK_QC = 21_250

SAFE_BOUNDARY = {
    "doesNotMint": True,
    "doesNotChangeMaxSupply": True,
    "doesNotChangeBalances": True,
    "doesNotSendFunds": True,
    "doesNotExecuteUnlock": True,
    "doesNotChangeConsensus": True,
    "requiresFounderWalletFinalBeforeMainnet": True,
    "forbidsFounderWalletChangeAfterMainnet": True,
    "requiresOneYearCliff": True,
    "requiresFourYearMonthlyVesting": True,
    "requiresFortyEightMonthlyUnlocks": True,
    "requiresMonthlyUnlock21250QC": True,
    "forbidsEarlyUnlock": True,
    "adminCannotEarlyUnlock": True,
    "aiCannotUnlock": True,
    "gameCannotUnlock": True,
    "workerCannotUnlock": True,
    "tgwtCannotUnlock": True,
    "nftCannotUnlock": True,
    "fixedSupplyOnly": True,
    "controlOnly": True,
}

def default_founder_vesting_policy_v1() -> Dict[str, Any]:
    return {
        "version": QC_FOUNDER_VESTING_POLICY_VERSION,
        "mode": "FOUNDER_VESTING_LOCK_POLICY_NO_UNLOCK_EXECUTION",
        "maxSupplyQc": MAX_SUPPLY_QC,
        "founderTotalQc": FOUNDER_TOTAL_QC,
        "founderLiquidQc": FOUNDER_LIQUID_QC,
        "founderLockedQc": FOUNDER_LOCKED_QC,
        "founderTotalPercent": FOUNDER_TOTAL_PERCENT,
        "founderLiquidPercent": FOUNDER_LIQUID_PERCENT,
        "founderLockedPercent": FOUNDER_LOCKED_PERCENT,
        "cliffMonths": CLIFF_MONTHS,
        "vestingMonths": VESTING_MONTHS,
        "totalScheduleMonths": TOTAL_SCHEDULE_MONTHS,
        "monthlyUnlockQc": MONTHLY_UNLOCK_QC,
        "unlockStartsAtMonth": 13,
        "unlockEndsAtMonth": 60,
        "founderWalletFinalBeforeMainnet": True,
        "founderWalletChangeAfterMainnet": False,
        "earlyUnlockForbidden": True,
        "adminEarlyUnlockForbidden": True,
        "aiCannotUnlock": True,
        "gameCannotUnlock": True,
        "workerCannotUnlock": True,
        "tgwtCannotUnlock": True,
        "nftCannotUnlock": True,
        "doesNotMintFounderAllocation": True,
        "founderAllocationFromFixedSupplyOnly": True,
        "maxSupplyAlwaysUnchanged": True,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def validate_founder_vesting_policy_v1(policy: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if policy.get("version") != QC_FOUNDER_VESTING_POLICY_VERSION:
        errors.append("invalid_version")
    if policy.get("mode") != "FOUNDER_VESTING_LOCK_POLICY_NO_UNLOCK_EXECUTION":
        errors.append("invalid_mode")
    if policy.get("maxSupplyQc") != MAX_SUPPLY_QC:
        errors.append("invalid_max_supply")
    if policy.get("founderTotalQc") != FOUNDER_TOTAL_QC:
        errors.append("invalid_founder_total")
    if policy.get("founderLiquidQc") != FOUNDER_LIQUID_QC:
        errors.append("invalid_founder_liquid")
    if policy.get("founderLockedQc") != FOUNDER_LOCKED_QC:
        errors.append("invalid_founder_locked")
    if policy.get("founderLiquidQc", 0) + policy.get("founderLockedQc", 0) != policy.get("founderTotalQc"):
        errors.append("founder_split_mismatch")
    if policy.get("founderTotalPercent") != FOUNDER_TOTAL_PERCENT:
        errors.append("invalid_founder_total_percent")
    if policy.get("founderLiquidPercent") != FOUNDER_LIQUID_PERCENT:
        errors.append("invalid_founder_liquid_percent")
    if policy.get("founderLockedPercent") != FOUNDER_LOCKED_PERCENT:
        errors.append("invalid_founder_locked_percent")
    if policy.get("cliffMonths") != CLIFF_MONTHS:
        errors.append("invalid_cliff_months")
    if policy.get("vestingMonths") != VESTING_MONTHS:
        errors.append("invalid_vesting_months")
    if policy.get("totalScheduleMonths") != TOTAL_SCHEDULE_MONTHS:
        errors.append("invalid_total_schedule_months")
    if policy.get("monthlyUnlockQc") != MONTHLY_UNLOCK_QC:
        errors.append("invalid_monthly_unlock")
    if policy.get("monthlyUnlockQc", 0) * policy.get("vestingMonths", 0) != policy.get("founderLockedQc"):
        errors.append("monthly_unlock_total_mismatch")
    if policy.get("unlockStartsAtMonth") != 13 or policy.get("unlockEndsAtMonth") != 60:
        errors.append("invalid_unlock_month_range")
    if policy.get("founderWalletFinalBeforeMainnet") is not True:
        errors.append("founder_wallet_final_required_before_mainnet")
    if policy.get("founderWalletChangeAfterMainnet") is not False:
        errors.append("founder_wallet_change_after_mainnet_forbidden")
    if policy.get("earlyUnlockForbidden") is not True:
        errors.append("early_unlock_forbidden_required")
    if policy.get("adminEarlyUnlockForbidden") is not True:
        errors.append("admin_early_unlock_forbidden_required")
    for key in ["aiCannotUnlock", "gameCannotUnlock", "workerCannotUnlock", "tgwtCannotUnlock", "nftCannotUnlock"]:
        if policy.get(key) is not True:
            errors.append(key + "_required")
    if policy.get("doesNotMintFounderAllocation") is not True:
        errors.append("founder_allocation_mint_forbidden_required")
    if policy.get("founderAllocationFromFixedSupplyOnly") is not True:
        errors.append("fixed_supply_only_required")
    if policy.get("maxSupplyAlwaysUnchanged") is not True:
        errors.append("max_supply_unchanged_required")

    safe = policy.get("safeBoundary") or {}
    for key in [
        "doesNotMint",
        "doesNotChangeMaxSupply",
        "doesNotChangeBalances",
        "doesNotSendFunds",
        "doesNotExecuteUnlock",
        "doesNotChangeConsensus",
        "requiresFounderWalletFinalBeforeMainnet",
        "forbidsFounderWalletChangeAfterMainnet",
        "requiresOneYearCliff",
        "requiresFourYearMonthlyVesting",
        "requiresFortyEightMonthlyUnlocks",
        "requiresMonthlyUnlock21250QC",
        "forbidsEarlyUnlock",
        "adminCannotEarlyUnlock",
        "aiCannotUnlock",
        "gameCannotUnlock",
        "workerCannotUnlock",
        "tgwtCannotUnlock",
        "nftCannotUnlock",
        "fixedSupplyOnly",
        "controlOnly",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_FOUNDER_VESTING_POLICY_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def founder_vesting_summary_v1() -> Dict[str, Any]:
    policy = default_founder_vesting_policy_v1()
    return {
        "ok": True,
        "version": QC_FOUNDER_VESTING_POLICY_VERSION,
        "policy": policy,
        "validation": validate_founder_vesting_policy_v1(policy),
    }
