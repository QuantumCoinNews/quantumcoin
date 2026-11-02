# QuantumCoin Mining Period / Bitcoin-Style Continuation

Status: QC_MINING_PERIOD_BITCOIN_STYLE_CONTINUE_V1

Date: 20260619_201209

## Purpose

This step defines the 10-year primary mining period, the 2-year halving schedule, and the correct continuation rule if all QC is not mined within 10 years.

This step does not mint, mine, distribute remaining coins, or start/stop mining.

## Primary reward schedule

- Year 0-2: 50 QC per block
- Year 2-4: 25 QC per block
- Year 4-6: 12.5 QC per block
- Year 6-8: 6.25 QC per block
- Year 8-10: 3.125 QC per block

## Bitcoin-style continuation after 10 years

If all QC is not mined within the 10-year primary mining period:

- Wallet + Miner continues mining.
- Halving continues every 2 years.
- Example continuation:
  - Year 10-12: 1.5625 QC per block
  - Year 12-14: 0.78125 QC per block
  - Year 14-16: 0.390625 QC per block
- Mining continues only until max supply is reached.
- Rewards must be capped by remaining supply.
- Remaining QC is not auto-minted.
- Remaining QC is not admin-minted.
- AI, game, worker, admin panel, TGWT, NFT, Watch & Earn cannot distribute remaining QC.
- Node fee-only mode is not forced while supply remains.
- Node/fee mode becomes the continuation model only after emission is complete.

## If all QC is mined within 10 years

- Normal emission is complete.
- Network continues through node/fee mode.

## Supply rule

- Max supply remains 25,500,000 QC.
- No rule in this step changes max supply.
- No rule in this step creates replacement minting.

## Admin visibility

Telegram Game admin API exposes read-only mining transition visibility:

- /api/admin/economy/mining-transition

This endpoint is under /api/admin/ and is protected by the admin auth layer.

## Safe boundary

This step does not:

- mint QC directly
- auto-mint remaining QC
- admin-mint remaining QC
- distribute remaining QC outside mining
- change balances
- change chain
- change consensus
- change max supply
- start or stop miners
