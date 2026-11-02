# QuantumCoin Go Build Targets

Status: QC_GO_BUILD_TARGETS_V1

This document defines the Go build targets before testnet preparation.

## Required active build targets

### ROOT_NODE

Main node/API/P2P/mining entrypoint.

Package:
- .

Temporary audit build output:
- quantumcoin-node-audit.exe

Release output later:
- release/quantumcoin.exe

Important:
- The audit build must not overwrite release/quantumcoin.exe.

### MINER_CLI

Miner command entrypoint.

Package:
- ./cmd/miner

Temporary audit build output:
- quantumcoin-miner-cli-audit.exe

### DESKTOP_GUI

Desktop GUI entrypoint.

Package:
- ./cmd/quantumcoin-gui

Temporary audit build output:
- wallet-gui-audit.exe

Release output later:
- release/wallet-gui.exe

Important:
- The audit build must not overwrite release/wallet-gui.exe.

### WALLET_TOOLS

Wallet/recovery/seed helper tools.

Packages:
- ./cmd/qc_deterministic_wallet
- ./cmd/qc_recovery_init
- ./cmd/qc_seed_canonical_check
- ./cmd/qc_seed_manager
- ./cmd/qc_wallet_create
- ./cmd/qc_wallet_restore

These are active tools, but they are not the main release wallet GUI.

## Test/helper build targets

These may be built for development checks but are not release outputs:

- ./cmd/apitest
- ./cmd/apitest/quantumcoin-gui

## Archive/review targets

These must not be built as active release targets:

- ./_wm_review/...

## Rule

Before testnet, every active Go executable target must be explicit.

Audit builds must go to _local_audit_reports only.

No audit build may overwrite files in release/.
