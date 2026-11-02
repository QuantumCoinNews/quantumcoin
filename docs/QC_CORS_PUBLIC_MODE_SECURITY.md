# QuantumCoin CORS Public Mode Security

Status: QC_CORS_PUBLIC_MODE_SECURITY_V1

Date: 20260617_210658

Public testnet API mode must not use wildcard CORS.

## Local/dev behavior

Local/dev mode keeps wildcard CORS for compatibility with existing local tools.

## Public testnet behavior

When public testnet mode is active, CORS allows only exact origins configured by:

- QC_CORS_ALLOWED_ORIGINS
- QC_CORS_ALLOWED_ORIGIN

Comma-separated origins are supported.

Wildcard * is ignored in public testnet mode.

## Affected code

- root main.go withCORS
- api/http_server.go withCORS

## Safety boundary

This does not change chain state, balances, mining rewards, total supply, consensus, or wallet data.

It only controls browser CORS headers for public testnet API exposure.
