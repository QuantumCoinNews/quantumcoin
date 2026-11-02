# QuantumCoin Burn / Dead Wallet Control

Status: QC_BURN_SINK_CONTROL_V1

Date: 20260619_191812

## Purpose

This step defines burn / dead wallet infrastructure rules.

This step does not burn coins and does not execute transfers.

## Rules

- Burn address alias: QC_BURN_SINK
- Burn address must be finalized before mainnet.
- Burn address must be unspendable.
- Burn private key must not exist.
- Burned QC cannot be recovered.
- Burned QC must not be replaced by new minting.
- Burned QC reduces circulating/spendable supply only.
- Max supply remains 25,500,000 QC.
- Auto burn is forbidden.
- Explicit user/admin approval is required for any future burn flow.
- AI, worker, game, NFT, TGWT, Watch & Earn, monitoring, or admin visibility endpoints cannot auto-burn.

## Supply interpretation

- Max supply: 25,500,000 QC
- Burned supply: amount sent to the burn sink
- Circulating/spendable supply: max/mined supply minus burned and locked amounts where applicable

## Admin visibility

Telegram Game admin API exposes read-only burn sink visibility:

- /api/admin/economy/burn-sink

This endpoint is under /api/admin/ and is protected by the admin auth layer.

## Safe boundary

This step does not:

- execute burn
- send funds
- mint replacement QC
- change balances
- change chain
- change consensus
- change max supply
- change mining reward
- start or stop miners
