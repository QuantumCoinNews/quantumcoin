# QuantumCoin Main Entrypoints

Status: QC_MAIN_ENTRYPOINTS_V1

This document classifies every main.go entrypoint before testnet preparation.

## Entrypoint classes

### ACTIVE_ROOT_NODE
Main QuantumCoin node/API/miner/P2P entrypoint.

Expected:
- main.go

### ACTIVE_MINER_CLI
Miner command entrypoint.

Expected:
- cmd/miner/main.go

### ACTIVE_GUI
Desktop GUI entrypoint.

Expected:
- cmd/quantumcoin-gui/main.go

### ACTIVE_WALLET_TOOL
Wallet/recovery/seed tools.

Expected examples:
- cmd/qc_deterministic_wallet/main.go
- cmd/qc_recovery_init/main.go
- cmd/qc_seed_canonical_check/main.go
- cmd/qc_seed_manager/main.go
- cmd/qc_wallet_create/main.go
- cmd/qc_wallet_restore/main.go

### TEST_OR_HELPER
API test or helper entrypoints.

Expected examples:
- cmd/apitest/main.go
- cmd/apitest/quantumcoin-gui/main.go

### ARCHIVE_REVIEW
Old review/archive copies. These are not active source.

Expected examples:
- _wm_review/cmd/miner/main.go
- _wm_review/cmd/quantumcoin-gui/main.go

## Rule

Root main.go is the main node entrypoint.

Other main.go files are allowed only if they are clearly classified as:
- active CLI/tool
- active GUI
- test/helper
- archive/review

Before testnet, there must be no unknown main.go entrypoint.
