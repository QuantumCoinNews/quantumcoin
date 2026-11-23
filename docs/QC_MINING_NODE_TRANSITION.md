# QuantumCoin Mining Period / Whole-QC Continuation

Document ID: `QC_MINING_PERIOD_BITCOIN_STYLE_CONTINUE_V1`\
Evidence stage: **Implementation**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

Original reference: 20260619_201209
Updated: 2026-11-05

## Purpose

This step defines the 10-year primary mining period, the 2-year halving schedule, and the correct continuation rule if all QC is not mined within 10 years.

This step does not mint, mine, distribute remaining coins, or start/stop mining.

## Canonical schedule basis

The user confirmed the existing whole-QC series on 5 November 2026. Consensus
halving is based on block height: every 2,102,400 blocks. Year labels below are
illustrative at the target block interval; wall-clock time does not choose rewards.
The minimum subsidy is 1 QC while issuance remains, capped by remaining supply.
Go visibility derives amounts from consensus.LegacySubsidyAtHeight. This document
supersedes the former fractional examples without changing consensus or history.

## Primary reward schedule

- Year 0-2: 50 QC per block
- Year 2-4: 25 QC per block
- Year 4-6: 12 QC per block
- Year 6-8: 6 QC per block
- Year 8-10: 3 QC per block

## Whole-QC continuation after the illustrative 10-year period

If all QC is not mined within the 10-year primary mining period:

- Wallet + Miner continues mining.
- Halving intervals continue by block height, with the 1-QC floor retained.
- Example continuation:
  - Year 10-12: 1 QC per block
  - Year 12-14: 1 QC per block
  - Year 14-16: 1 QC per block
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
