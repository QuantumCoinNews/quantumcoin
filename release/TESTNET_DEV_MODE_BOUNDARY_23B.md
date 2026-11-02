# QuantumCoin Testnet / Dev Mode Boundary - 23B

Status: QC_PRE_TESTNET_STEP_23B

## Current approved mode

Current testnet runtime is LOCAL DEV TESTNET only.

- QC_TESTNET=1
- QC_PUBLIC_TESTNET=0
- QC_NETWORK=testnet
- QC_CHAIN_MODE=testnet
- QC_NET_MAGIC=QC_TESTNET_V1
- QC_CHAIN_FILE=chain_data_testnet.dat
- QC_HTTP_PORT=:18082
- QC_P2P_PORT=:13001

## Public testnet status

Public testnet is NOT enabled yet.

RUN_PUBLIC_TESTNET_NODE_DISABLED_UNTIL_APPROVAL.ps1 exists only as a safety stop script.
It must not start a public testnet until later approval.

## Local dev testnet allowed for steps 24-26

Allowed locally:
- /api/network/label
- /api/health
- /api/telemetry
- /api/tx/status
- /api/wallet/balance
- local-only controlled test mining for dry-run
- local-only P2P on 13001

Not approved for public exposure yet:
- /api/mine
- /api/dev/fastmine
- /api/mine/job
- /api/mine/submit
- /api/tx/send
- wallet/new or wallet/address as public endpoints
- burn/stake endpoints as public endpoints
- admin/security mutation endpoints

## Safety boundary

This file does not mine, transfer, mint, change balances, change consensus, change rewards, or change total supply.

Public testnet requires later checklist approval with:
- API token policy
- CORS allowlist
- rate-limit review
- public endpoint allowlist
- faucet/premine approval decision
- backup and rollback plan
