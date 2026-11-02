# QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.monitoring_event_log_alert_runbook_v1 import (
    QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION,
    classify_event_v1,
    normalize_severity_v1,
    runbook_for_category_v1,
    runbook_summary_v1,
    validate_event_v1,
)

def test_normalize_severity():
    assert normalize_severity_v1("fatal") == "CRITICAL"
    assert normalize_severity_v1("warn") == "WARNING"
    assert normalize_severity_v1("anything") == "INFO"

def test_validate_event():
    event = {
        "source": "node-api",
        "category": "api_down",
        "severity": "CRITICAL",
        "message": "node API unreachable",
    }
    result = validate_event_v1(event)
    assert result["ok"] is True
    assert result["version"] == QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_VERSION

def test_invalid_event_rejected():
    result = validate_event_v1({"severity": "bad"})
    assert result["ok"] is False
    assert "source_required" in result["errors"]
    assert "category_required" in result["errors"]
    assert "message_required" in result["errors"]
    assert "severity_must_be_normalized" in result["errors"]

def test_classify_critical_event_escalates():
    result = classify_event_v1({
        "source": "node-api",
        "category": "chain_mismatch",
        "severity": "CRITICAL",
        "message": "wrong chain file",
    })
    assert result["alertOpen"] is True
    assert result["escalate"] is True
    assert result["safeBoundary"]["doesNotChangeSupply"] is True

def test_runbook_entries_exist():
    summary = runbook_summary_v1()
    cats = [x["category"] for x in summary["entries"]]
    assert "api_down" in cats
    assert "auth_failure" in cats
    assert "chain_mismatch" in cats
    assert "payout_guard" in cats
    assert "nft_mint_guard" in cats
    assert "watch_verification_failure" in cats

def test_unknown_category_safe():
    entry = runbook_for_category_v1("new_unknown")
    assert entry["category"] == "new_unknown"
    assert "do_not_mint" in entry["forbidden"]

if __name__ == "__main__":
    test_normalize_severity()
    test_validate_event()
    test_invalid_event_rejected()
    test_classify_critical_event_escalates()
    test_runbook_entries_exist()
    test_unknown_category_safe()
    print("QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_V1_TEST_PASS")
