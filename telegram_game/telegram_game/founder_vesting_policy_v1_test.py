# QC_FOUNDER_VESTING_POLICY_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.founder_vesting_policy_v1 import (
    QC_FOUNDER_VESTING_POLICY_VERSION,
    default_founder_vesting_policy_v1,
    validate_founder_vesting_policy_v1,
    founder_vesting_summary_v1,
)

def test_default_policy_valid():
    policy = default_founder_vesting_policy_v1()
    result = validate_founder_vesting_policy_v1(policy)
    assert result["ok"] is True
    assert result["version"] == QC_FOUNDER_VESTING_POLICY_VERSION

def test_founder_split():
    policy = default_founder_vesting_policy_v1()
    assert policy["founderTotalQc"] == 1_275_000
    assert policy["founderLiquidQc"] == 255_000
    assert policy["founderLockedQc"] == 1_020_000
    assert policy["founderLiquidQc"] + policy["founderLockedQc"] == policy["founderTotalQc"]

def test_vesting_schedule():
    policy = default_founder_vesting_policy_v1()
    assert policy["cliffMonths"] == 12
    assert policy["vestingMonths"] == 48
    assert policy["monthlyUnlockQc"] == 21_250
    assert policy["monthlyUnlockQc"] * policy["vestingMonths"] == policy["founderLockedQc"]
    assert policy["unlockStartsAtMonth"] == 13
    assert policy["unlockEndsAtMonth"] == 60

def test_founder_wallet_final_before_mainnet():
    policy = default_founder_vesting_policy_v1()
    assert policy["founderWalletFinalBeforeMainnet"] is True
    assert policy["founderWalletChangeAfterMainnet"] is False

def test_reject_early_unlock():
    policy = default_founder_vesting_policy_v1()
    policy["earlyUnlockForbidden"] = False
    result = validate_founder_vesting_policy_v1(policy)
    assert result["ok"] is False
    assert "early_unlock_forbidden_required" in result["errors"]

def test_reject_wrong_monthly_unlock():
    policy = default_founder_vesting_policy_v1()
    policy["monthlyUnlockQc"] = 30_000
    result = validate_founder_vesting_policy_v1(policy)
    assert result["ok"] is False
    assert "invalid_monthly_unlock" in result["errors"]

def test_summary_safe_boundary():
    s = founder_vesting_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["policy"]["safeBoundary"]["requiresMonthlyUnlock21250QC"] is True
    assert s["policy"]["safeBoundary"]["forbidsEarlyUnlock"] is True
    assert s["policy"]["safeBoundary"]["fixedSupplyOnly"] is True

if __name__ == "__main__":
    test_default_policy_valid()
    test_founder_split()
    test_vesting_schedule()
    test_founder_wallet_final_before_mainnet()
    test_reject_early_unlock()
    test_reject_wrong_monthly_unlock()
    test_summary_safe_boundary()
    print("QC_FOUNDER_VESTING_POLICY_V1_TEST_PASS")
