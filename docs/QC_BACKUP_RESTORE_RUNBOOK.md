# QuantumCoin Backup / Restore Runbook

Status: QC_BACKUP_RESTORE_RUNBOOK_V1

Date: 20260619_181553

## Purpose

This step defines the backup and restore runbook for QuantumCoin pre-testnet and later mainnet preparation.

This is a runbook and validation layer only. It does not run live restore.

## Required backup artifacts

- quantumcoin_chain_file
- telegram_game_sqlite_db
- tgwt_ledger
- monthly_nft_review_log
- admin_event_log
- monitoring_event_log
- runtime_config_examples

## Rules

- Testnet and mainnet backup roots must be separated.
- Testnet and mainnet manifests must be separated.
- Hash check is required.
- Size check is required.
- Manifest entry is required.
- Restore dry-run is required before any real restore.
- Admin approval is required before restore.
- No live restore is allowed in this runbook step.
- No overwrite is allowed without explicit approval.
- Private keys must never be printed into reports.
- Chain, ledger, NFT review, admin log, and monitoring log backups must be preserved.

## Admin visibility

Telegram Game admin API exposes read-only backup/restore runbook visibility:

- /api/admin/backup-restore/runbook

This endpoint is under /api/admin/ and is protected by the admin auth layer.

## Safe boundary

This step does not:

- run live restore
- delete data
- overwrite data
- mint QC/NFT
- send QC/TGWT
- change balances
- change chain
- change consensus
- change total supply
- start or stop miners
