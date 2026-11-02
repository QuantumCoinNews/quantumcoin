# QC_MINING_PERIOD_BITCOIN_STYLE_CONTINUE_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.mining_node_transition_policy_v1 import (
    QC_MINING_PERIOD_NODE_TRANSITION_VERSION,
    PRIMARY_REWARD_SCHEDULE,
    POST_TEN_YEAR_EXAMPLE_SCHEDULE,
    default_mining_node_transition_policy_v1,
    validate_mining_node_transition_policy_v1,
    mining_node_transition_summary_v1,
)

def test_default_policy_valid():
    policy = default_mining_node_transition_policy_v1()
    result = validate_mining_node_transition_policy_v1(policy)
    assert result["ok"] is True
    assert result["version"] == QC_MINING_PERIOD_NODE_TRANSITION_VERSION

def test_primary_reward_schedule():
    policy = default_mining_node_transition_policy_v1()
    assert policy["primaryRewardSchedule"] == PRIMARY_REWARD_SCHEDULE
    assert [x["rewardQc"] for x in policy["primaryRewardSchedule"]] == ["50", "25", "12.5", "6.25", "3.125"]

def test_post_ten_year_bitcoin_style_schedule():
    policy = default_mining_node_transition_policy_v1()
    assert policy["postTenYearExampleSchedule"] == POST_TEN_YEAR_EXAMPLE_SCHEDULE
    assert [x["rewardQc"] for x in policy["postTenYearExampleSchedule"]] == ["1.5625", "0.78125", "0.390625"]

def test_incomplete_after_ten_years_wallet_miner_continues_bitcoin_style():
    policy = default_mining_node_transition_policy_v1()
    assert policy["primaryMiningPeriodYears"] == 10
    assert policy["halvingIntervalYears"] == 2
    assert policy["ifNotFullyMinedAfterPrimaryPeriod"] == "WALLET_MINER_CONTINUES_WITH_BITCOIN_STYLE_HALVING_UNTIL_MAX_SUPPLY"
    assert policy["walletMinerContinuesIfIncomplete"] is True
    assert policy["bitcoinStyleHalvingContinues"] is True
    assert policy["miningContinuesOnlyUntilMaxSupply"] is True
    assert policy["rewardMustBeCappedByRemainingSupply"] is True
    assert policy["nodeFeeOnlyNotForcedWhileSupplyRemains"] is True

def test_reject_auto_mint_remaining():
    policy = default_mining_node_transition_policy_v1()
    policy["autoMintRemainingForbidden"] = False
    result = validate_mining_node_transition_policy_v1(policy)
    assert result["ok"] is False
    assert "auto_mint_remaining_forbidden_required" in result["errors"]

def test_reject_node_fee_only_while_supply_remains():
    policy = default_mining_node_transition_policy_v1()
    policy["nodeFeeOnlyNotForcedWhileSupplyRemains"] = False
    result = validate_mining_node_transition_policy_v1(policy)
    assert result["ok"] is False
    assert "node_fee_only_forbidden_while_supply_remains" in result["errors"]

def test_summary_safe_boundary():
    s = mining_node_transition_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["policy"]["safeBoundary"]["requiresWalletMinerContinueIfIncomplete"] is True
    assert s["policy"]["safeBoundary"]["requiresBitcoinStyleHalvingAfterTenYears"] is True
    assert s["policy"]["safeBoundary"]["requiresRewardCapByRemainingSupply"] is True

if __name__ == "__main__":
    test_default_policy_valid()
    test_primary_reward_schedule()
    test_post_ten_year_bitcoin_style_schedule()
    test_incomplete_after_ten_years_wallet_miner_continues_bitcoin_style()
    test_reject_auto_mint_remaining()
    test_reject_node_fee_only_while_supply_remains()
    test_summary_safe_boundary()
    print("QC_MINING_PERIOD_BITCOIN_STYLE_CONTINUE_V1_TEST_PASS")
