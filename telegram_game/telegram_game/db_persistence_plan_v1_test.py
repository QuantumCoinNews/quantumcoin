# QC_DB_PERSISTENCE_PLAN_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.db_persistence_plan_v1 import (
    QC_DB_PERSISTENCE_PLAN_VERSION,
    REQUIRED_DOMAINS,
    db_persistence_summary_v1,
    default_db_persistence_plan_v1,
    validate_db_persistence_plan_v1,
)

def test_default_plan_valid():
    plan = default_db_persistence_plan_v1()
    result = validate_db_persistence_plan_v1(plan)
    assert result["ok"] is True
    assert result["version"] == QC_DB_PERSISTENCE_PLAN_VERSION

def test_required_domains_present():
    plan = default_db_persistence_plan_v1()
    names = {d["name"] for d in plan["domains"]}
    for name in REQUIRED_DOMAINS:
        assert name in names

def test_missing_separation_rejected():
    plan = default_db_persistence_plan_v1()
    plan["domains"][0]["testnetSeparated"] = False
    result = validate_db_persistence_plan_v1(plan)
    assert result["ok"] is False
    assert any(x.startswith("testnet_separation_required") for x in result["errors"])

def test_duplicate_domain_rejected():
    plan = default_db_persistence_plan_v1()
    plan["domains"].append(dict(plan["domains"][0]))
    result = validate_db_persistence_plan_v1(plan)
    assert result["ok"] is False
    assert any(x.startswith("duplicate_domain") for x in result["errors"])

def test_summary_safe_boundary():
    s = db_persistence_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["plan"]["safeBoundary"]["doesNotChangeSupply"] is True
    assert s["plan"]["safeBoundary"]["requiresRestoreTest"] is True

if __name__ == "__main__":
    test_default_plan_valid()
    test_required_domains_present()
    test_missing_separation_rejected()
    test_duplicate_domain_rejected()
    test_summary_safe_boundary()
    print("QC_DB_PERSISTENCE_PLAN_V1_TEST_PASS")
