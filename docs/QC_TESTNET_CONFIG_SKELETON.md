# QuantumCoin Testnet Config Skeleton

Status: QC_TESTNET_CONFIG_SKELETON_V1

Date: 20260617_223732

This step creates the testnet configuration skeleton only. It does not start testnet.

## Required boundaries

- network = testnet
- network magic = QC_TESTNET_V1
- chain file = chain_data_testnet.dat
- main chain file chain_data.dat must not be used
- public API exposure requires API token
- public CORS requires explicit allowed origins
- dev/miner endpoints remain disabled in public mode
- faucet disabled in skeleton
- premine disabled in skeleton
- total supply cap remains 25,500,000 QC

## Files

- config/testnet_config_skeleton_v1.go
- config/testnet_config_skeleton_v1_test.go
- configs/testnet.config.skeleton.json
- configs/testnet.env.example

## Safety boundary

This does not mine, transfer, mint, change balances, change consensus, change rewards, or change total supply.
