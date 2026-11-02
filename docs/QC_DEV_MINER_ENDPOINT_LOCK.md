# QuantumCoin Dev / Miner Endpoint Lock

Status: QC_DEV_MINER_ENDPOINT_LOCK_V1

Date: 20260617_211754

Dev and miner control endpoints must not be usable in public testnet mode.

## Root node endpoints

- /api/dev/fastmine
- /api/miner/start
- /api/miner/stop
- /api/miner/status

## Alternate api package endpoints

- /api/miner/start
- /api/miner/stop

## Allowed mode

Only all of the following:

- QC_DEV_MODE=1 / true / yes
- request comes from loopback
- not public testnet mode

## Public testnet behavior

Public testnet mode rejects these endpoints even with a valid API token.

## Safety boundary

This does not change chain state, balances, mining rewards, total supply, consensus, release executables, or stable miner scripts.

It only prevents dev/miner control endpoints from being exposed outside explicit local development.
