# QuantumCoin Mainnet-Before Economy Control

Document ID: `QC_MAINNET_BEFORE_ECONOMY_CONTROL_V1`\
Evidence stage: **Partial implementation**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

Date: 20260619_190310

## Purpose

This step defines the final pre-mainnet economy invariant control.

This does not launch mainnet.

## Fixed invariants

- Total supply: 25,500,000 QC
- Founder/main allocation: 5% total
- Founder liquid allocation: 1%
- Founder locked/vesting allocation: 4%
- Founder wallet binding: required before mainnet
- Final founder wallet address: must be explicitly approved before mainnet launch
- Initial mining reward: 50 QC
- Coinbase maturity: 10 blocks

## Founder allocation

- Founder total: 1,275,000 QC
- Founder liquid/open: 255,000 QC
- Founder locked/vesting: 1,020,000 QC
- This allocation is not new minting. It must come from the fixed 25,500,000 QC supply.

## Ecosystem boundaries

AI, bonus systems, TGWT, Watch & Earn, NFT review, queue workers, monitoring, admin panels, and autopublisher systems cannot:

- mint QC
- change QC total supply
- change balances
- change consensus
- change mining rewards
- start or stop miners
- execute settlement without admin approval
- select NFT winners alone
- change founder allocation
- bypass founder wallet binding

## Admin visibility

Telegram Game admin API exposes read-only economy control visibility:

- /api/admin/economy/mainnet-before

This endpoint is under /api/admin/ and is protected by the admin auth layer.

## Safe boundary

This step does not:

- launch mainnet
- mint QC/NFT
- send QC/TGWT
- change balances
- change chain
- change consensus
- change total supply
- change mining reward
- start or stop miners
