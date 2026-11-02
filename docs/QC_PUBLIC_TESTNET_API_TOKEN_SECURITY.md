# QuantumCoin Public Testnet API Token Security

Status: QC_PUBLIC_TESTNET_API_TOKEN_REQUIRED_V1

Date: 20260617_195318

Public testnet API exposure must not run with an empty QC_API_TOKEN.

## Local/dev behavior

When no public testnet signal is enabled and QC_API_TOKEN is empty, local development behavior remains unchanged.

## Public testnet behavior

QC_API_TOKEN is mandatory when any of these are true:

- QC_PUBLIC_TESTNET=1
- QC_API_PUBLIC_TESTNET=1
- QC_TESTNET_PUBLIC_API=1
- QC_TESTNET=1 with public QC_API_LISTEN
- QC_NET_MAGIC / QC_NETWORK / QC_NETWORK_MODE / QC_CHAIN_MODE / QC_MODE contains testnet and QC_API_LISTEN is public

Public listen examples:

- 0.0.0.0:8082
- [::]:8082
- non-loopback IP addresses

## Token locations

For this step, existing supported token locations are preserved:

- X-QC-API-KEY
- Authorization: Bearer TOKEN
- query token fallback: disabled in public testnet mode
- form token fallback: disabled in public testnet mode

Query/form fallback is disabled in public testnet mode by QC_QUERY_FORM_TOKEN_FALLBACK_CLOSED_FOR_PUBLIC_TESTNET_V1.

## Safety boundary

This does not change chain state, balances, mining rewards, supply, consensus, or wallet data.

It only prevents protected API routes from becoming tokenless in public testnet mode.

