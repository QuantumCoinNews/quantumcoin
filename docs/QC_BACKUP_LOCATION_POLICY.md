# QuantumCoin Backup Location Policy

Status: QC_33B_BACKUP_EXTERNALIZATION_V1

Date: 20260619_213620

## Rule

Project backup folders and generated local audit backups must not stay inside the active project source tree.

Default external backup root:

C:\QC_PROJECT_EXTERNAL_BACKUPS\quantumcoin

Current batch:

C:\QC_PROJECT_EXTERNAL_BACKUPS\quantumcoin\QC_33B_BACKUP_EXTERNALIZATION_20260619_213620

## Why

Keeping backup folders inside the project causes:

- false positive lint/analyzer warnings
- duplicate source scanning
- old review file noise
- accidental edits to backup copies
- confusion between active source and archived source

## Protected active paths

The following must not be moved by backup cleanup:

- release
- .git
- active Go/Python/PowerShell/JSON source files
- Telegram Game active package files
- wallet/miner stable release files
- chain/wallet runtime files

## Telegram Game rule

Telegram Game backup folders must be outside the active project tree.

Allowed active Telegram Game location:

C:\Projects\quantumcoin\telegram_game

Backup/archive location:

C:\QC_PROJECT_EXTERNAL_BACKUPS\quantumcoin

## This step

This step moves known backup directories outside the project.
It does not delete backups.
It does not modify stable release files.
It does not touch Git/GitHub.
