# QuantumCoin Chain File Separation

Status: QC_CHAIN_FILE_SEPARATION_V1

Date: 20260617_224803

Mainnet/local chain and testnet chain must not share the same data file.

## Rules

- Main/local default chain file remains chain_data.dat.
- Testnet mode must not use chain_data.dat.
- Testnet mode must use a clearly labeled testnet chain file, e.g. chain_data_testnet.dat.
- A testnet chain file must not be used unless explicit testnet mode is enabled.
- QC_CHAIN_FILE can override the chain path, but the mode guard still applies.
- Active miner CLI is also guarded.

## Testnet mode signals

- QC_TESTNET=1
- QC_PUBLIC_TESTNET=1
- QC_API_PUBLIC_TESTNET=1
- QC_TESTNET_PUBLIC_API=1
- QC_NET_MAGIC / QC_NETWORK / QC_NETWORK_MODE / QC_CHAIN_MODE / QC_MODE contains testnet

## Safety boundary

This does not mine, transfer, mint, change balances, change consensus, change rewards, or change total supply.
