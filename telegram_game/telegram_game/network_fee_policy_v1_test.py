# QC_NETWORK_FEE_POLICY_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.network_fee_policy_v1 import (
    QC_NETWORK_FEE_POLICY_VERSION,
    BASE_FEE_TARGET_QC,
    CONGESTION_CEILING_QC,
    FEE_RECIPIENT,
    default_network_fee_policy_v1,
    validate_network_fee_policy_v1,
    network_fee_summary_v1,
)

def test_default_policy_valid():
    policy = default_network_fee_policy_v1()
    result = validate_network_fee_policy_v1(policy)
    assert result["ok"] is True
    assert result["version"] == QC_NETWORK_FEE_POLICY_VERSION

def test_fee_is_micro_low():
    policy = default_network_fee_policy_v1()
    assert policy["baseFeeTargetQc"] == BASE_FEE_TARGET_QC == "0.000001"
    assert policy["congestionCeilingQc"] == CONGESTION_CEILING_QC == "0.00001"
    assert policy["feeMustStayMicroLow"] is True

def test_fee_destination():
    policy = default_network_fee_policy_v1()
    assert policy["feeRecipient"] == FEE_RECIPIENT
    assert policy["feeGoesToFounder"] is False
    assert policy["feeGoesToBurnByDefault"] is False

def test_fee_not_mint():
    policy = default_network_fee_policy_v1()
    assert policy["feeIsNewMint"] is False
    assert policy["feeChangesMaxSupply"] is False

def test_reject_old_higher_fee():
    policy = default_network_fee_policy_v1()
    policy["baseFeeTargetQc"] = "0.0001"
    result = validate_network_fee_policy_v1(policy)
    assert result["ok"] is False
    assert "invalid_base_fee_target" in result["errors"]

def test_reject_founder_fee():
    policy = default_network_fee_policy_v1()
    policy["feeGoesToFounder"] = True
    result = validate_network_fee_policy_v1(policy)
    assert result["ok"] is False
    assert "fee_must_not_go_to_founder" in result["errors"]

def test_summary_safe_boundary():
    s = network_fee_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["policy"]["safeBoundary"]["feeMustStayMicroLow"] is True
    assert s["policy"]["safeBoundary"]["feeGoesToBlockProducer"] is True

if __name__ == "__main__":
    test_default_policy_valid()
    test_fee_is_micro_low()
    test_fee_destination()
    test_fee_not_mint()
    test_reject_old_higher_fee()
    test_reject_founder_fee()
    test_summary_safe_boundary()
    print("QC_NETWORK_FEE_POLICY_MICRO_V1_TEST_PASS")
