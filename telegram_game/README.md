# QuantumCoin Telegram Game / Space Miner

QuantumCoin Space Miner is a game-styled wallet, mining and reward interface for the QuantumCoin ecosystem.

## Current scope

- Real QC wallet/mining bridge through the local QuantumCoin backend/node
- Mining zone UX with real block/reward flow
- Wallet send/receive UI
- Local transaction status fallback
- TGWT ledger-backed rewards
- Watch & Earn development flow with final Google Rewarded Ads policy
- Social claim reward flow
- Admin settlement/reward classification
- Ship Store public USDT model with hidden public prices

## Important rules

- The game does not create a separate blockchain.
- QC balance must not be faked by UI clicks.
- QC rewards must come from valid mining/chain/backend flow.
- XP is only gameplay progression.
- TGWT is a community/game reward layer.
- Premium ships must not change QC total supply or block reward rules.
- Public GitHub code must not contain real USDT ship prices.

## Security

Do not commit:

- .env
- private keys
- seed phrases
- local databases
- real USDT price files
- runtime backups
- generated checkpoints
