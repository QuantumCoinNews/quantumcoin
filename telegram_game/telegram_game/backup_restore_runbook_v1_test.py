# QC_BACKUP_RESTORE_RUNBOOK_V1_TEST

import sys
from pathlib import Path

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.backup_restore_runbook_v1 import (
    QC_BACKUP_RESTORE_RUNBOOK_VERSION,
    REQUIRED_ARTIFACTS,
    default_backup_restore_runbook_v1,
    validate_backup_restore_runbook_v1,
    backup_restore_summary_v1,
)

def test_default_runbook_valid():
    runbook = default_backup_restore_runbook_v1()
    result = validate_backup_restore_runbook_v1(runbook)
    assert result["ok"] is True
    assert result["version"] == QC_BACKUP_RESTORE_RUNBOOK_VERSION

def test_required_artifacts_present():
    runbook = default_backup_restore_runbook_v1()
    names = {a["name"] for a in runbook["artifacts"]}
    for name in REQUIRED_ARTIFACTS:
        assert name in names

def test_live_restore_allowed_rejected():
    runbook = default_backup_restore_runbook_v1()
    runbook["artifacts"][0]["liveRestoreForbidden"] = False
    result = validate_backup_restore_runbook_v1(runbook)
    assert result["ok"] is False
    assert any(x.startswith("live_restore_forbidden_required") for x in result["errors"])

def test_missing_hash_rejected():
    runbook = default_backup_restore_runbook_v1()
    runbook["artifacts"][0]["requiresHash"] = False
    result = validate_backup_restore_runbook_v1(runbook)
    assert result["ok"] is False
    assert any(x.startswith("hash_required") for x in result["errors"])

def test_summary_safe_boundary():
    s = backup_restore_summary_v1()
    assert s["ok"] is True
    assert s["validation"]["ok"] is True
    assert s["runbook"]["safeBoundary"]["doesNotRunLiveRestore"] is True
    assert s["runbook"]["safeBoundary"]["doesNotOverwriteData"] is True

if __name__ == "__main__":
    test_default_runbook_valid()
    test_required_artifacts_present()
    test_live_restore_allowed_rejected()
    test_missing_hash_rejected()
    test_summary_safe_boundary()
    print("QC_BACKUP_RESTORE_RUNBOOK_V1_TEST_PASS")
