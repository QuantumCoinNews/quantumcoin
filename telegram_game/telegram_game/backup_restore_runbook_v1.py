# QC_BACKUP_RESTORE_RUNBOOK_V1
# Backup/restore runbook for Telegram Game + QuantumCoin ecosystem.
#
# Boundary:
# - Runbook/validation only.
# - no live restore is executed by this module.
# - Does not delete data.
# - Does not overwrite data.
# - Does not send funds.
# - Does not mint.
# - Does not change balances.
# - Does not change chain or supply.

from typing import Any, Dict, List

QC_BACKUP_RESTORE_RUNBOOK_VERSION = "QC_BACKUP_RESTORE_RUNBOOK_V1"

SAFE_BOUNDARY = {
    "doesNotRunLiveRestore": True,
    "doesNotDeleteData": True,
    "doesNotOverwriteData": True,
    "doesNotMint": True,
    "doesNotSendFunds": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeSupply": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "runbookOnly": True,
    "requiresManifest": True,
    "requiresHashCheck": True,
    "requiresDryRun": True,
    "requiresAdminApproval": True,
}

REQUIRED_ARTIFACTS = [
    "quantumcoin_chain_file",
    "telegram_game_sqlite_db",
    "tgwt_ledger",
    "monthly_nft_review_log",
    "admin_event_log",
    "monitoring_event_log",
    "runtime_config_examples",
]

def default_backup_restore_runbook_v1() -> Dict[str, Any]:
    artifact_defaults = {
        "testnetSeparated": True,
        "mainnetSeparated": True,
        "requiresHash": True,
        "requiresSizeCheck": True,
        "requiresManifestEntry": True,
        "requiresEncryption": False,
        "restoreDryRunRequired": True,
        "adminApprovalRequired": True,
        "liveRestoreForbidden": True,
    }

    artifacts = [
        {"name": "quantumcoin_chain_file", "kind": "chain_file", "appendOnlyPreferred": True},
        {"name": "telegram_game_sqlite_db", "kind": "sqlite_db", "appendOnlyPreferred": False},
        {"name": "tgwt_ledger", "kind": "json_ledger", "appendOnlyPreferred": True},
        {"name": "monthly_nft_review_log", "kind": "json_ledger", "appendOnlyPreferred": True},
        {"name": "admin_event_log", "kind": "jsonl_append_only", "appendOnlyPreferred": True},
        {"name": "monitoring_event_log", "kind": "jsonl_append_only", "appendOnlyPreferred": True},
        {"name": "runtime_config_examples", "kind": "config", "appendOnlyPreferred": False},
    ]

    for artifact in artifacts:
        artifact.update(artifact_defaults)

    return {
        "version": QC_BACKUP_RESTORE_RUNBOOK_VERSION,
        "mode": "RUNBOOK_ONLY_NO_LIVE_RESTORE",
        "artifacts": artifacts,
        "safeBoundary": dict(SAFE_BOUNDARY),
        "notes": {
            "dev": "Runbook only. No live restore is executed in this step.",
            "testnet": "Testnet backup names, manifests, and restore dry-runs must be separated from mainnet.",
            "mainnet": "Mainnet restore requires admin approval, manifest verification, hash verification, dry-run, and downtime plan.",
            "keys": "Private keys are never shared in reports; encrypted external custody/manual procedure only.",
        },
    }

def validate_backup_restore_runbook_v1(runbook: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if runbook.get("version") != QC_BACKUP_RESTORE_RUNBOOK_VERSION:
        errors.append("invalid_version")
    if runbook.get("mode") != "RUNBOOK_ONLY_NO_LIVE_RESTORE":
        errors.append("invalid_mode")

    artifacts = runbook.get("artifacts")
    if not isinstance(artifacts, list) or not artifacts:
        errors.append("artifacts_required")
        artifacts = []

    seen = set()
    for artifact in artifacts:
        name = str(artifact.get("name") or "").strip()
        if not name:
            errors.append("artifact_name_required")
            continue
        if name in seen:
            errors.append("duplicate_artifact:" + name)
        seen.add(name)

        if not str(artifact.get("kind") or "").strip():
            errors.append("artifact_kind_required:" + name)
        if artifact.get("testnetSeparated") is not True:
            errors.append("testnet_separation_required:" + name)
        if artifact.get("mainnetSeparated") is not True:
            errors.append("mainnet_separation_required:" + name)
        if artifact.get("requiresHash") is not True:
            errors.append("hash_required:" + name)
        if artifact.get("requiresSizeCheck") is not True:
            errors.append("size_check_required:" + name)
        if artifact.get("requiresManifestEntry") is not True:
            errors.append("manifest_required:" + name)
        if artifact.get("restoreDryRunRequired") is not True:
            errors.append("restore_dry_run_required:" + name)
        if artifact.get("adminApprovalRequired") is not True:
            errors.append("admin_approval_required:" + name)
        if artifact.get("liveRestoreForbidden") is not True:
            errors.append("live_restore_forbidden_required:" + name)

    for required in REQUIRED_ARTIFACTS:
        if required not in seen:
            errors.append("missing_required_artifact:" + required)

    safe = runbook.get("safeBoundary") or {}
    for key in [
        "doesNotRunLiveRestore",
        "doesNotDeleteData",
        "doesNotOverwriteData",
        "doesNotMint",
        "doesNotSendFunds",
        "doesNotChangeBalances",
        "doesNotChangeChain",
        "doesNotChangeSupply",
        "requiresManifest",
        "requiresHashCheck",
        "requiresDryRun",
        "requiresAdminApproval",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_BACKUP_RESTORE_RUNBOOK_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def backup_restore_summary_v1() -> Dict[str, Any]:
    runbook = default_backup_restore_runbook_v1()
    return {
        "ok": True,
        "version": QC_BACKUP_RESTORE_RUNBOOK_VERSION,
        "runbook": runbook,
        "validation": validate_backup_restore_runbook_v1(runbook),
    }
