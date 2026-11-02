# QuantumCoin Legacy / Build / Error / Backup Separation

Status: QC_LEGACY_NOISE_SEPARATION_V1

This document defines the separation rule for old build, error, backup, and temporary files before testnet preparation.

## Keep active source untouched

Never move or delete these stable release files during cleanup/audit steps:

- release/wallet-gui.exe
- release/quantumcoin.exe
- release/run_miner.cmd
- release/_qc_api_mempool_miner.ps1

Never move or delete local chain/wallet state unless there is a separate explicit backup/restore plan:

- .wallet
- .env
- chain_data.dat
- qc_blockchain.db
- wallet_priv.hex
- miner_address.txt
- release/chain_data.dat
- release/wallet_priv.hex

## Legacy / backup candidates

These are not active source:

- main.go.before_*.bak
- *.before_*
- *_before_*
- *.bak
- *_errors.txt
- build_errors.txt
- build.log
- api_out.log
- e2e_api_out.log
- e2e_api_err.log
- old tree/list snapshots
- old generated diagnostic text files

## Old backup folders

These are archive/review/runtime material, not active source:

- _checkpoints/
- _legacy_runtime_*/
- _root_exe_quarantine/
- _wm_review/
- old_working_release/
- release_backup_*/
- release_clean_test*/
- __qcdiag/

## Rule

Step 6A is read-only audit.

Step 6B may move safe legacy/noise files into a local ignored archive folder, but only after reviewing the 6A report.

No deletion.
No release overwrite.
No GitHub push.
