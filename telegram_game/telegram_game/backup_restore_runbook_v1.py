"""Read-only compatibility view of the generated Go recovery declaration.

There is no independent Python artifact catalogue or live restore implementation.
Deploy configs/backup_restore.runbook.example.json with this package.
"""
import json
from pathlib import Path
from typing import Any, Dict

QC_BACKUP_RESTORE_RUNBOOK_VERSION = "QC_BACKUP_RESTORE_RUNBOOK_V1"
_DECLARATION = Path(__file__).resolve().parents[2] / "configs" / "backup_restore.runbook.example.json"


def default_backup_restore_runbook_v1() -> Dict[str, Any]:
    # Fresh ownership on each call; missing deployment data fails closed.
    with _DECLARATION.open("r", encoding="utf-8") as stream:
        value = json.load(stream)
    if not isinstance(value, dict) or value.get("version") != QC_BACKUP_RESTORE_RUNBOOK_VERSION:
        raise RuntimeError("canonical Go recovery declaration unavailable")
    return value


SAFE_BOUNDARY = default_backup_restore_runbook_v1()["safeBoundary"]
REQUIRED_ARTIFACTS = [a["name"] for a in default_backup_restore_runbook_v1()["artifacts"]]


def validate_backup_restore_runbook_v1(runbook: Dict[str, Any]) -> Dict[str, Any]:
    expected = default_backup_restore_runbook_v1()
    required = {a["name"]: a["kind"] for a in expected["artifacts"]}
    kinds = {"chain_file", "sqlite_db", "json_ledger", "jsonl_append_only", "config", "wallet_key"}
    errors = []
    if not isinstance(runbook, dict):
        return {"ok": False, "errors": ["runbook_required"]}
    if runbook.get("version") != expected["version"]:
        errors.append("invalid_version")
    if runbook.get("mode") != expected["mode"]:
        errors.append("invalid_mode")
    artifacts = runbook.get("artifacts")
    if not isinstance(artifacts, list) or not artifacts:
        errors.append("artifacts_required")
        artifacts = []
    seen = set()
    for artifact in artifacts:
        if not isinstance(artifact, dict):
            errors.append("artifact_object_required")
            continue
        name = artifact.get("name")
        if not isinstance(name, str) or not name or name != name.strip():
            errors.append("artifact_name_required")
            continue
        if name in seen:
            errors.append("duplicate_artifact:" + name)
        seen.add(name)
        kind = artifact.get("kind")
        if not isinstance(kind, str) or kind not in kinds:
            errors.append("unknown_artifact_kind:" + name)
        if name in required and kind != required[name]:
            errors.append("required_artifact_kind_mismatch:" + name)
        if artifact.get("requiresEncryption") is not True:
            errors.append("artifact_encryption_required:" + name)
        for key, label in (("testnetSeparated", "testnet_separation_required"),
                           ("mainnetSeparated", "mainnet_separation_required"),
                           ("requiresHash", "hash_required"),
                           ("requiresSizeCheck", "size_check_required"),
                           ("requiresManifestEntry", "manifest_required"),
                           ("restoreDryRunRequired", "restore_dry_run_required"),
                           ("adminApprovalRequired", "admin_approval_required"),
                           ("liveRestoreForbidden", "live_restore_forbidden_required")):
            if artifact.get(key) is not True:
                errors.append(label + ":" + name)
    for name in required:
        if name not in seen:
            errors.append("missing_required_artifact:" + name)
    safe = runbook.get("safeBoundary")
    if (not isinstance(safe, dict) or set(safe) != set(expected["safeBoundary"])
            or any(safe.get(key) is not True for key in expected["safeBoundary"])):
        errors.append("safe_boundary_mismatch")
    return {"ok": not errors, "version": QC_BACKUP_RESTORE_RUNBOOK_VERSION,
            "errors": errors, "safeBoundary": dict(expected["safeBoundary"])}


def backup_restore_summary_v1() -> Dict[str, Any]:
    runbook = default_backup_restore_runbook_v1()
    validation = validate_backup_restore_runbook_v1(runbook)
    return {"ok": validation["ok"], "runbook": runbook, "validation": validation,
            "source": "generated_go_declaration", "liveRestoreAvailable": False}
