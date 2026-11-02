# QuantumCoin Blockchain Package Standard

Status: QC_BLOCKCHAIN_PACKAGE_STANDARD_V1

This document defines the expected blockchain package structure before testnet preparation.

## Expected active package

The canonical active blockchain package is:

- quantumcoin/blockchain
- folder: blockchain/

All production node, wallet, miner, API, Telegram Game bridge and security logic that needs chain data must use this canonical package or controlled API boundaries.

## Not allowed before testnet

Before testnet, the repo must not contain ambiguous active blockchain implementations such as:

- another active folder with `package blockchain`
- copied blockchain implementation under cmd/
- copied blockchain implementation under old test folders that is still part of `go list ./...`
- duplicate package names that can be confused with the canonical chain package

## Archive folders

Backup/review/archive folders may contain old blockchain copies only if they are clearly ignored by active build tools.

Examples:
- _legacy_noise_archive/
- _local_audit_backups/
- _local_audit_reports/
- _checkpoints/
- _wm_review/
- release_backup_*/

## Rule

Step 8 is an audit step.

If a duplicate active blockchain package is found, do not delete blindly. First classify it, then move/archive or exclude it in a separate controlled repair step.

No release overwrite.
No GitHub push.
