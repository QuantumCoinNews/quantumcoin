# QC_MAINNET_BEFORE_ECONOMY_CONTROL_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.mainnet_before_economy_control_v1 import (
    QC_MAINNET_BEFORE_ECONOMY_CONTROL_VERSION,
    TOTAL_SUPPLY_QC,
    FOUNDER_TOTAL_BPS,
    FOUNDER_LIQUID_BPS,
    FOUNDER_LOCKED_BPS,
    default_mainnet_before_economy_policy_v1,
    validate_mainnet_before_economy_policy_v1,
    mainnet_before_economy_summary_v1,
)

def test_default_policy_valid():
    policy = default_mainnet_before_economy_policy_v1()
    result = validate_mainnet_before_economy_policy_v1(policy)
    assert result["ok"] is True
    assert result["version"] == QC_MAINNET_BEFORE_ECONOMY_CONTROL_VERSION

def test_supply_and_founder_split():
    policy = default_mainnet_before_economy_policy_v1()
    assert policy["totalSupplyQc"] == TOTAL_SUPPLY_QC == 25_500_000
    assert policy["founderTotalBasisPoints"] == FOUNDER_TOTAL_BPS == 500
    assert policy["founderLiquidBasisPoints"] == FOUNDER_LIQUID_BPS == 100
    assert policy["founderLockedBasisPoints"] == FOUNDER_LOCKED_BPS == 400
    assert policy["founderLiquidBasisPoints"] + policy["founderLockedBasisPoints"] == policy["founderTotalBasisPoints"]

def test_founder_wallet_binding_required():
    policy = default_mainnet_before_economy_policy_v1()
    assert policy["founderWalletBindingRequired"] is True
    assert policy["founderWalletAddressFinalBeforeMainnet"] is True
    assert policy["safeBoundary"]["requiresFounderWalletBinding"] is True
    assert policy["safeBoundary"]["requiresFinalFounderWalletAddress"] is True

def test_founder_wallet_binding_missing_rejected():
    policy = default_mainnet_before_economy_policy_v1()
    policy["founderWalletBindingRequired"] = False
    result = validate_mainnet_before_economy_policy_v1(policy)
    assert result["ok"] is False
    assert "founder_wallet_binding_required" in result["errors"]

def test_supply_change_rejected():
    policy = default_mainnet_before_economy_policy_v1()
    policy["totalSupplyQc"] = 25_500_001
    result = validate_mainnet_before_economy_policy_v1(policy)
    assert result["ok"] is False
    assert "invalid_total_supply" in result["errors"]

def test_minting_allocation_rejected():
    policy = default_mainnet_before_economy_policy_v1()
    policy["allocations"][0]["mintsNewQc"] = True
    result = validate_mainnet_before_economy_policy_v1(policy)
    assert result["ok"] is False
    assert any(x.startswith("allocation_must_not_mint_qc") for x in result["errors"])

def test_summary_safe_boundary():
    s = mainnet_before_economy_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["policy"]["safeBoundary"]["doesNotLaunchMainnet"] is True
    assert s["policy"]["safeBoundary"]["doesNotChangeSupply"] is True
    assert s["policy"]["safeBoundary"]["requiresFounderSplitCheck"] is True
    assert s["policy"]["safeBoundary"]["requiresFounderWalletBinding"] is True
    assert s["policy"]["safeBoundary"]["requiresFinalFounderWalletAddress"] is True

if __name__ == "__main__":
    test_default_policy_valid()
    test_supply_and_founder_split()
    test_founder_wallet_binding_required()
    test_founder_wallet_binding_missing_rejected()
    test_supply_change_rejected()
    test_minting_allocation_rejected()
    test_summary_safe_boundary()
    print("QC_MAINNET_BEFORE_ECONOMY_CONTROL_V1_TEST_PASS")
