# QC_DB_PERSISTENCE_PLAN_V1
# Real DB persistence plan for Telegram Game + QuantumCoin ecosystem.
#
# Boundary:
# - Plan/validation only.
# - no live migration is executed by this module.
# - Does not migrate live DB automatically.
# - Does not send funds.
# - Does not mint.
# - Does not change balances.
# - Does not change chain or supply.

from typing import Any, Dict, List

QC_DB_PERSISTENCE_PLAN_VERSION = "QC_DB_PERSISTENCE_PLAN_V1"

SAFE_BOUNDARY = {
    "doesNotMint": True,
    "doesNotSendFunds": True,
    "doesNotChangeBalances": True,
    "doesNotChangeChain": True,
    "doesNotChangeConsensus": True,
    "doesNotChangeSupply": True,
    "doesNotStartMining": True,
    "doesNotStopMining": True,
    "planOnly": True,
    "requiresBackupPlan": True,
    "requiresMigrationPlan": True,
    "requiresRestoreTest": True,
}

REQUIRED_DOMAINS = [
    "quantumcoin_chain",
    "telegram_game_state",
    "tgwt_ledger",
    "withdraw_claim_review",
    "monthly_nft_review",
    "admin_event_log",
    "monitoring_events",
]

def default_db_persistence_plan_v1() -> Dict[str, Any]:
    return {
        "version": QC_DB_PERSISTENCE_PLAN_VERSION,
        "mode": "PLAN_ONLY_NO_LIVE_MIGRATION",
        "domains": [
            {
                "name": "quantumcoin_chain",
                "storageMode": "file",
                "schemaVersion": "chain_file_v1",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "backupRequired": True,
                "migrationRequired": False,
                "appendOnlyLedger": True,
            },
            {
                "name": "telegram_game_state",
                "storageMode": "sqlite",
                "schemaVersion": "tg_game_db_v1",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "backupRequired": True,
                "migrationRequired": True,
                "appendOnlyLedger": False,
            },
            {
                "name": "tgwt_ledger",
                "storageMode": "sqlite",
                "schemaVersion": "tgwt_ledger_v1",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "backupRequired": True,
                "migrationRequired": True,
                "appendOnlyLedger": True,
            },
            {
                "name": "withdraw_claim_review",
                "storageMode": "sqlite",
                "schemaVersion": "withdraw_claim_review_v1",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "backupRequired": True,
                "migrationRequired": True,
                "appendOnlyLedger": True,
            },
            {
                "name": "monthly_nft_review",
                "storageMode": "sqlite",
                "schemaVersion": "monthly_nft_review_v1",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "backupRequired": True,
                "migrationRequired": True,
                "appendOnlyLedger": True,
            },
            {
                "name": "admin_event_log",
                "storageMode": "append_only",
                "schemaVersion": "admin_event_log_v1",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "backupRequired": True,
                "migrationRequired": True,
                "appendOnlyLedger": True,
            },
            {
                "name": "monitoring_events",
                "storageMode": "append_only",
                "schemaVersion": "monitoring_events_v1",
                "testnetSeparated": True,
                "mainnetSeparated": True,
                "backupRequired": True,
                "migrationRequired": True,
                "appendOnlyLedger": True,
            },
        ],
        "safeBoundary": dict(SAFE_BOUNDARY),
        "notes": {
            "dev": "Local SQLite/file storage is acceptable for development.",
            "testnet": "Testnet DB paths must be separate from mainnet.",
            "mainnet": "Mainnet requires backup, restore, schema, and audit-log validation before launch.",
        },
    }

def validate_db_persistence_plan_v1(plan: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []

    if plan.get("version") != QC_DB_PERSISTENCE_PLAN_VERSION:
        errors.append("invalid_version")

    domains = plan.get("domains")
    if not isinstance(domains, list) or not domains:
        errors.append("domains_required")
        domains = []

    seen = set()
    for d in domains:
        name = str(d.get("name") or "").strip()
        if not name:
            errors.append("domain_name_required")
            continue
        if name in seen:
            errors.append("duplicate_domain:" + name)
        seen.add(name)

        if not str(d.get("storageMode") or "").strip():
            errors.append("storage_mode_required:" + name)
        if not str(d.get("schemaVersion") or "").strip():
            errors.append("schema_version_required:" + name)
        if d.get("testnetSeparated") is not True:
            errors.append("testnet_separation_required:" + name)
        if d.get("mainnetSeparated") is not True:
            errors.append("mainnet_separation_required:" + name)
        if d.get("backupRequired") is not True:
            errors.append("backup_required:" + name)

    for required in REQUIRED_DOMAINS:
        if required not in seen:
            errors.append("missing_required_domain:" + required)

    safe = plan.get("safeBoundary") or {}
    for key in [
        "doesNotMint",
        "doesNotSendFunds",
        "doesNotChangeBalances",
        "doesNotChangeChain",
        "doesNotChangeSupply",
        "requiresBackupPlan",
        "requiresMigrationPlan",
        "requiresRestoreTest",
    ]:
        if safe.get(key) is not True:
            errors.append("safe_boundary_missing:" + key)

    return {
        "ok": len(errors) == 0,
        "version": QC_DB_PERSISTENCE_PLAN_VERSION,
        "errors": errors,
        "safeBoundary": dict(SAFE_BOUNDARY),
    }

def db_persistence_summary_v1() -> Dict[str, Any]:
    plan = default_db_persistence_plan_v1()
    return {
        "ok": True,
        "version": QC_DB_PERSISTENCE_PLAN_VERSION,
        "plan": plan,
        "validation": validate_db_persistence_plan_v1(plan),
    }

