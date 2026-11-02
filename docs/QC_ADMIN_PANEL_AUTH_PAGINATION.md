# QuantumCoin Admin Panel Auth / Pagination

Status: QC_ADMIN_PANEL_AUTH_PAGINATION_V1

Date: 20260618_231600

## Decision

The existing Telegram Game admin panel approach is extended with a central API admin-panel safety layer.

## Scope

- Admin API token policy.
- Admin endpoint registry.
- Admin pagination standard.
- Admin panel registry endpoints.
- Safe bridge in Telegram Game dev_api.py.

## Admin auth

Accepted admin auth methods:

- X-Admin-Token
- X-QC-ADMIN-TOKEN
- Authorization: Bearer <token>

Query token is intentionally not part of the admin auth model.

## Pagination

- Default limit: 50
- Max limit: 100
- Offset supported for registry APIs
- Large export must later use background job

## Safety

This step does not mint, does not transfer, does not change QC/TGWT balances, does not change chain/consensus/supply, and does not start or stop miners.
