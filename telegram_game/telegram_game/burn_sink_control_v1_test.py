# QC_BURN_SINK_CONTROL_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.burn_sink_control_v1 import (
    QC_BURN_SINK_CONTROL_VERSION,
    MAX_SUPPLY_QC,
    DEFAULT_BURN_ADDRESS,
    default_burn_sink_policy_v1,
    validate_burn_sink_policy_v1,
    burn_sink_summary_v1,
)

def test_default_burn_policy_valid():
    policy = default_burn_sink_policy_v1()
    result = validate_burn_sink_policy_v1(policy)
    assert result["ok"] is True
    assert result["version"] == QC_BURN_SINK_CONTROL_VERSION

def test_burn_address_and_supply():
    policy = default_burn_sink_policy_v1()
    assert policy["maxSupplyQc"] == MAX_SUPPLY_QC == 25_500_000
    assert policy["burnAddressAlias"] == DEFAULT_BURN_ADDRESS == "QC_BURN_SINK"
    assert policy["burnDoesNotReduceMaxSupply"] is True

def test_spendable_burn_address_rejected():
    policy = default_burn_sink_policy_v1()
    policy["burnAddressMustBeUnspendable"] = False
    result = validate_burn_sink_policy_v1(policy)
    assert result["ok"] is False
    assert "burn_address_must_be_unspendable" in result["errors"]

def test_burn_replacement_mint_rejected():
    policy = default_burn_sink_policy_v1()
    policy["burnDoesNotMintReplacement"] = False
    result = validate_burn_sink_policy_v1(policy)
    assert result["ok"] is False
    assert "burn_must_not_mint_replacement" in result["errors"]

def test_auto_burn_rejected():
    policy = default_burn_sink_policy_v1()
    policy["autoBurnForbidden"] = False
    result = validate_burn_sink_policy_v1(policy)
    assert result["ok"] is False
    assert "auto_burn_forbidden_required" in result["errors"]

def test_summary_safe_boundary():
    s = burn_sink_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["policy"]["safeBoundary"]["doesNotExecuteBurn"] is True
    assert s["policy"]["safeBoundary"]["doesNotMintReplacement"] is True
    assert s["policy"]["safeBoundary"]["requiresUnspendableBurnAddress"] is True

if __name__ == "__main__":
    test_default_burn_policy_valid()
    test_burn_address_and_supply()
    test_spendable_burn_address_rejected()
    test_burn_replacement_mint_rejected()
    test_auto_burn_rejected()
    test_summary_safe_boundary()
    print("QC_BURN_SINK_CONTROL_V1_TEST_PASS")
