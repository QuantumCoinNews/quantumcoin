# QuantumCoin Load Test Scripts

Status: QC_LOAD_TEST_SCRIPTS_SAFE_V1

Date: 20260619_002150

## Purpose

This step prepares safe load test scripts for the pre-testnet phase.

It does not run destructive or high-volume tests now.

## Created files

- tools/qc_load_test_safe_v1.ps1
- telegram_game/telegram_game/load_test_safety_v1.py
- telegram_game/telegram_game/load_test_safety_v1_test.py
- configs/load_test.safe.example.json

## Rules

Load tests are read-only by default.

Allowed:

- GET /health
- GET /api/network/label
- GET /api/qc/health
- GET /api/admin/panel/registry
- GET /api/admin/monitoring/runbook

Blocked:

- mining endpoints
- send/tx endpoints
- withdraw/settlement mutation endpoints
- watch/social claim endpoints
- NFT review/select/mint endpoints
- store buy endpoints

## Limits

- Default requests: 20
- Default concurrency: 2
- Max requests: 500
- Max concurrency: 25
- Localhost only by default

## Safety

The load test scripts must not:

- mint QC/NFT
- send QC/TGWT
- change balances
- change chain
- change supply
- start or stop miners
- approve admin actions
- execute payouts

## Later real testnet use

When testnet is running, approved read-only targets can be tested using:

powershell -ExecutionPolicy Bypass -File tools/qc_load_test_safe_v1.ps1 -BaseUrl http://127.0.0.1:18082 -Endpoint /health -Requests 100 -Concurrency 5

Remote/public targets require explicit approval and -AllowRemote.
