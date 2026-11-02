# QuantumCoin Active Source Standard

Status: QC_ACTIVE_SOURCE_STANDARD_V1

This document defines what is treated as active source, local-only data, runtime data, archive/backup material, or review material before testnet preparation.

## Active source

Active source means files/folders that can be used for build, test, review, or maintained project logic.

Core active source includes:

- main.go
- http_security.go
- go.mod
- go.sum
- README.md
- config examples
- ai/
- api/
- blockchain/
- cmd/
- config/
- consolefx/
- docs/
- game/
- gui/
- i18n/
- internal/
- miner/
- nft/
- p2p/
- qc-desktop/
- security/
- src/
- telegram_game/telegram_game/
- testnet/
- tests/
- token/
- tools/
- ui/
- utils/
- wallet/
- webui/
- webwallet/
- autopublisher/src/ and autopublisher source/config templates only

## Local-only sensitive material

These must never be committed or included in public audit/share packages:

- .env
- .wallet
- wallet_priv.hex
- miner_priv.hex
- private keys
- credentials.json
- token.json
- api keys
- secrets

## Runtime/generated material

These are not active source:

- chain_data.dat
- chain_data_testnet.dat
- qc_blockchain.db
- *.db
- *.dat
- *.log
- *.exe
- *.dll
- *.pdb
- *.zip
- node_modules/
- cache/
- tmp/
- build outputs
- release runtime outputs

## Archive/backup material

These are not active source:

- *.bak
- *.before_*
- main.go.before_*
- dev_api_before_*
- old_working_release/
- release_backup_*/
- release_clean_test*/
- _legacy_runtime_*/
- _checkpoints/
- _root_exe_quarantine/
- _wm_review/

## Rule

Before testnet, active source must be distinguishable from local/runtime/archive material. Local/runtime/archive material may remain on the developer machine, but it must not be treated as active source or pushed to GitHub.

## Environment template rule

.env and .env.* files are local-only sensitive, except .env.example template files.

Allowed example templates:
- .env.example
- autopublisher/.env.example
- telegram_game/.env.example

These files must not contain real keys, tokens, passwords, private keys, wallet secrets, or production credentials.
