# QuantumCoin Query/Form Token Fallback Security

Status: QC_QUERY_FORM_TOKEN_FALLBACK_CLOSED_FOR_PUBLIC_TESTNET_V1

Date: 20260617_200656

Public testnet API mode must not accept API tokens from URL query strings or form body fallback.

## Public testnet accepted token locations

- X-QC-API-KEY
- Authorization: Bearer TOKEN

## Public testnet rejected token locations

- ?token=TOKEN
- form/body token=TOKEN fallback

## Local/dev behavior

Local/dev mode keeps the old fallback behavior so existing local tools are not broken before testnet.

## Reason

Query/form tokens can leak through:

- browser history
- terminal logs
- proxy logs
- analytics/referrer headers
- screenshots

## Safety boundary

This does not change chain state, balances, mining rewards, total supply, consensus, or wallet data.

It only hardens protected API authentication in public testnet mode.
