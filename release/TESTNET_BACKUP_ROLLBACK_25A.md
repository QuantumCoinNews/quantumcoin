# QuantumCoin Testnet Backup / Rollback Plan - 25A

Status: QC_PRE_TESTNET_STEP_25A

Scope: LOCAL DEV TESTNET only.
Public testnet remains disabled until approval.

Mainnet file must not be restored or overwritten by these scripts.
Mainnet chain file: chain_data.dat
Testnet chain file: chain_data_testnet.dat

Backup script:
BACKUP_TESTNET_STATE_25A.ps1

Restore script:
RESTORE_TESTNET_STATE_25A_REQUIRE_EXPLICIT_PATH.ps1

Restore requires explicit backup folder path.
Restore must only target testnet files.

Backed up files:
- config.testnet.json
- chain_data_testnet.dat
- bonus_store_testnet.json if exists
- wallet_data_testnet.json if exists
- RUN_TESTNET_API_18082.ps1
- STOP_TESTNET_API_18082.ps1
- RUN_TESTNET_NODE_18082_13001.ps1
- STOP_TESTNET_NODE_18082_13001.ps1
- TESTNET_DEV_MODE_BOUNDARY_23B.md
- TESTNET_RUNBOOK_24A.md

Safety:
- No GitHub
- No transfer
- No mining
- No testnet start
- No mainnet chain touch
- No restore unless explicit path is passed later

Next:
26 first local testnet dry-run
