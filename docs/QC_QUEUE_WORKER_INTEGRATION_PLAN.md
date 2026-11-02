# QuantumCoin Queue / Worker Integration Plan

Status: QC_QUEUE_WORKER_INTEGRATION_PLAN_V1

Date: 20260619_175850

## Purpose

This step defines the real queue/worker integration plan for QuantumCoin pre-testnet and later mainnet preparation.

This is a plan/validation layer only. It does not start live workers.

## Required queues

- tgwt_withdraw_review_queue
- tgwt_bsc_claim_settlement_queue
- monthly_nft_review_queue
- monitoring_alert_queue
- autopublisher_content_queue
- backup_restore_check_queue

## Rules

- Testnet and mainnet queues must be separated.
- Queue names, DB paths, dead-letter queues, and audit logs must be separated by environment.
- Idempotency is required.
- Dead-letter queue policy is required.
- Audit log is required.
- Financial, NFT, settlement, and publishing jobs require admin approval.
- Monitoring alert queue may auto-record alerts, but cannot send funds, mint, or change balances.
- AI can advise, score, and report only.
- AI cannot execute settlements, mint NFTs, send funds, change balances, or change chain state.

## Admin visibility

Telegram Game admin API exposes read-only queue worker plan visibility:

- /api/admin/queue-worker/plan

This endpoint is under /api/admin/ and is protected by the admin auth layer.

## Safe boundary

This step does not:

- start workers
- mint QC/NFT
- send QC/TGWT
- change balances
- change chain
- change consensus
- change total supply
- start or stop miners
