# QC_HARDFORK_POLICY_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.hardfork_policy_v1 import (
    QC_HARDFORK_POLICY_VERSION,
    default_hardfork_policy_v1,
    validate_hardfork_policy_v1,
    hardfork_policy_summary_v1,
)

def test_default_policy_valid():
    policy = default_hardfork_policy_v1()
    result = validate_hardfork_policy_v1(policy)
    assert result["ok"] is True
    assert result["version"] == QC_HARDFORK_POLICY_VERSION

def test_hardfork_does_not_execute_now():
    policy = default_hardfork_policy_v1()
    assert policy["hardforkExecutionNow"] is False

def test_hardfork_emergency_allowed():
    policy = default_hardfork_policy_v1()
    assert policy["hardforkAllowed"] is True
    assert "CRITICAL_CONSENSUS_BUG" in policy["allowedReasons"]

def test_hardfork_forbidden_economy_changes():
    policy = default_hardfork_policy_v1()
    assert "CHANGE_MAX_SUPPLY" in policy["forbiddenReasons"]
    assert "MINT_EXTRA_QC" in policy["forbiddenReasons"]
    assert policy["hardforkCannotChangeMaxSupply"] is True
    assert policy["hardforkCannotMintExtraQc"] is True

def test_hardfork_requires_process():
    policy = default_hardfork_policy_v1()
    assert policy["requiresTestnetRehearsal"] is True
    assert policy["requiresMigrationHeight"] is True
    assert policy["requiresRollbackPlan"] is True

def test_reject_execution_now():
    policy = default_hardfork_policy_v1()
    policy["hardforkExecutionNow"] = True
    result = validate_hardfork_policy_v1(policy)
    assert result["ok"] is False
    assert "hardfork_execution_now_forbidden" in result["errors"]

def test_reject_ai_hardfork_permission():
    policy = default_hardfork_policy_v1()
    policy["hardforkCannotAutoRunByAi"] = False
    result = validate_hardfork_policy_v1(policy)
    assert result["ok"] is False
    assert "hardforkCannotAutoRunByAi_required" in result["errors"]

def test_summary_safe_boundary():
    s = hardfork_policy_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["policy"]["safeBoundary"]["doesNotExecuteHardfork"] is True
    assert s["policy"]["safeBoundary"]["requiresTestnetRehearsal"] is True

if __name__ == "__main__":
    test_default_policy_valid()
    test_hardfork_does_not_execute_now()
    test_hardfork_emergency_allowed()
    test_hardfork_forbidden_economy_changes()
    test_hardfork_requires_process()
    test_reject_execution_now()
    test_reject_ai_hardfork_permission()
    test_summary_safe_boundary()
    print("QC_HARDFORK_POLICY_V1_TEST_PASS")
