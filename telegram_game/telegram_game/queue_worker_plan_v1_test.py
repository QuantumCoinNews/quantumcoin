# QC_QUEUE_WORKER_INTEGRATION_PLAN_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.queue_worker_plan_v1 import (
    QC_QUEUE_WORKER_PLAN_VERSION,
    REQUIRED_QUEUES,
    default_queue_worker_plan_v1,
    validate_queue_worker_plan_v1,
    queue_worker_summary_v1,
)

def test_default_plan_valid():
    plan = default_queue_worker_plan_v1()
    result = validate_queue_worker_plan_v1(plan)
    assert result["ok"] is True
    assert result["version"] == QC_QUEUE_WORKER_PLAN_VERSION

def test_required_queues_present():
    plan = default_queue_worker_plan_v1()
    names = {q["name"] for q in plan["queues"]}
    for name in REQUIRED_QUEUES:
        assert name in names

def test_unsafe_capability_rejected():
    plan = default_queue_worker_plan_v1()
    plan["queues"][0]["canSendFunds"] = True
    result = validate_queue_worker_plan_v1(plan)
    assert result["ok"] is False
    assert any(x.startswith("unsafe_capability_forbidden") for x in result["errors"])

def test_missing_dead_letter_rejected():
    plan = default_queue_worker_plan_v1()
    plan["queues"][0]["requiresDeadLetter"] = False
    result = validate_queue_worker_plan_v1(plan)
    assert result["ok"] is False
    assert any(x.startswith("dead_letter_required") for x in result["errors"])

def test_summary_safe_boundary():
    s = queue_worker_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["plan"]["safeBoundary"]["doesNotStartWorkers"] is True
    assert s["plan"]["safeBoundary"]["doesNotSendFunds"] is True

if __name__ == "__main__":
    test_default_plan_valid()
    test_required_queues_present()
    test_unsafe_capability_rejected()
    test_missing_dead_letter_rejected()
    test_summary_safe_boundary()
    print("QC_QUEUE_WORKER_INTEGRATION_PLAN_V1_TEST_PASS")
