# QuantumCoin DB Persistence Plan

Status: QC_DB_PERSISTENCE_PLAN_V1

Date: 20260619_173735

## Purpose

This step defines the real database persistence plan for QuantumCoin pre-testnet and later mainnet preparation.

This is a plan/validation layer only. It does not migrate live data automatically.

## Persistence domains

- quantumcoin_chain
- telegram_game_state
- tgwt_ledger
- withdraw_claim_review
- monthly_nft_review
- admin_event_log
- monitoring_events

## Rules

- Testnet and mainnet DB/files must be separated.
- Financial/reward/review/event history must not rely on in-memory-only state.
- TGWT ledger, withdraw/claim review, NFT review, admin event log, and monitoring events require persistent storage.
- Append-only ledger style is required for financial/admin/security event history.
- Backup is required before migration.
- Restore test is required before mainnet.
- Schema version must be tracked.
- No automatic mainnet migration should run on startup without explicit admin approval.

## Admin visibility

Telegram Game admin API exposes read-only persistence plan visibility:

- /api/admin/persistence/db-plan

This endpoint is under /api/admin/ and is protected by the admin auth layer.

## Safe boundary

This step does not:

- mint QC/NFT
- send QC/TGWT
- change balances
- change chain
- change consensus
- change total supply
- start or stop miners
- migrate live DB automatically
