# QuantumCoin Monitoring / Event Log / Alert Runbook

Status: QC_MONITORING_EVENT_LOG_ALERT_RUNBOOK_V1

Date: 20260619_000753

## Purpose

This step defines the monitoring, event log, alert severity, and admin runbook standard for the QuantumCoin ecosystem.

## Scope

- QuantumCoin node/API monitoring
- Wallet/miner API monitoring
- Telegram Game API monitoring
- TGWT claim/withdraw monitoring
- Watch & Earn verification monitoring
- Monthly NFT review/mint-guard monitoring
- Admin auth/security monitoring
- Testnet/mainnet chain mismatch monitoring

## Severity

- INFO: visible event, no alert
- WARNING: admin review required
- CRITICAL: escalation required before promotion or public operation continues

## Categories

- api_down
- auth_failure
- chain_mismatch
- payout_guard
- nft_mint_guard
- watch_verification_failure

## Admin API visibility

Telegram Game admin API now exposes read-only monitoring/runbook endpoints:

- /api/admin/monitoring/runbook
- /api/admin/monitoring/event-policy

These endpoints are protected by the admin auth layer from item 26 because they are under /api/admin/.

## Safety boundary

This system is read-only alerting/runbook infrastructure.

It must not:

- mint QC or NFT
- send QC/TGWT
- change balances
- change consensus
- change total supply
- start or stop miners
- auto-ban users without admin review

## Next future step

Actual persistent event storage can later use DB/JSONL/queue workers after DB persistence and queue/worker plans are completed.
