# QuantumCoin Network Fee Policy

Status: QC_NETWORK_FEE_POLICY_V1

Date: 20260619_203752

## Purpose

This step defines the micro-low network fee policy.

This step does not execute a fee transfer, mint QC, change balances, change consensus, or change max supply.

## Fee target

- Base fee target: 0.000001 QC
- Congestion / anti-spam ceiling target: 0.00001 QC
- Final atomic-unit compatibility must be confirmed before mainnet.
- Final fee parameters require explicit mainnet approval.

## Fee destination

During active mining:

- Block reward + transaction fees go to the block producer / miner.

After emission is complete:

- No new QC mint.
- Transaction fees go to the node / block producer.

## Forbidden destinations

Network fee does not go to:

- founder wallet
- burn wallet by default
- AI system
- admin panel
- Telegram Game
- TGWT
- NFT system
- Watch & Earn system

## Burn relation

Fee burn is not enabled by default.

Any future fee-burn design would require separate governance/final approval and must not bypass the burn sink rules.

## Safe boundary

This step does not:

- mint QC
- change max supply
- change balances
- send funds
- execute fee transfer
- change consensus
- change mining reward
- start or stop miners
