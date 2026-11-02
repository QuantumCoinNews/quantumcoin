# QuantumCoin TESTNET Label UI/API

Status: QC_TESTNET_LABEL_UI_API_V1

Date: 20260617_231316

UI/API must visibly expose the active network label.

## API

- GET /api/network/label
- Response includes:
  - version
  - network
  - label
  - is_testnet
  - badge
  - warning
  - chain_file
  - net_magic
- Headers include:
  - X-QuantumCoin-Network
  - X-QuantumCoin-Network-Label
  - X-QuantumCoin-Testnet

## UI

- Web UI includes QC_TESTNET_LABEL_UI_V1 badge script.
- In testnet mode, the UI displays a visible TESTNET badge.
- Page title is prefixed with TESTNET when testnet is detected.

## Safety boundary

This does not mine, transfer, mint, change balances, change consensus, change rewards, or change total supply.
