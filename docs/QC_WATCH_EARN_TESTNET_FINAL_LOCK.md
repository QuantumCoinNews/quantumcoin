# QuantumCoin Watch & Earn Testnet / Final Lock

Document ID: `WATCH_EARN_TESTNET_FINAL_LOCK_V1`\
Evidence stage: **Implementation**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

Date: 20260618_213251

## Decision

Watch & Earn is a future revenue model. In testnet/dev it can use safe simulation for UI and ledger tests. In final/mainnet it must require verified rewarded ad completion.

## Final rules

- Final/mainnet activation requires explicit approval.
- Final provider must be Google rewarded ads or AdMob rewarded.
- Server-side verification is required in final mode.
- No reward for ad load only.
- No reward for ad open only.
- No reward for click only.
- No fake impressions.
- No bot traffic.
- No invalid traffic.
- DEV timer simulation is not final revenue.
- TGWT reward enters pending ledger only.
- Revenue log stays separate from TGWT reward log.

## Scope

This step does not call Google APIs, does not run ad SDK, does not create real ad impressions, does not enable revenue, and does not send on-chain TGWT.
