# QuantumCoin Testnet Faucet / Premine Security

Status: QC_TESTNET_FAUCET_PREMINE_SECURITY_V1

Date: 20260617_230052

Testnet faucet and premine must remain disabled until explicit later approval.

## Rules

- QC_TESTNET_FAUCET_ENABLED must not be enabled before faucet approval.
- Faucet private key/address env values must not be set in testnet mode.
- QC_TESTNET_PREMINE_ENABLED must not be enabled before premine approval.
- QC_MAIN_ADDRESS must not be used as a testnet premine fallback.
- QC_PREMINE_* address values must not be set in testnet mode.
- QC_PREMINE_* percent values must be empty or zero in testnet mode.
- testnet/faucet.json is disabled placeholder only.
- testnet/genesis.json has empty premine list.

## Safety boundary

This does not mine, transfer, mint, change balances, change consensus, change rewards, or change total supply.
