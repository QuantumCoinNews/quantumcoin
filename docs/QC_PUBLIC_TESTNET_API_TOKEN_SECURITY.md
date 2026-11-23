# API token and listener policy

Protected routes require a configured `QC_API_TOKEN` in all network profiles.
A non-loopback listener refuses to start without it, including with local-release
or mainnet selected. The public-testnet switches also require the token on
loopback. Configure the same header for the canonical `/api/health` probe.

Pass the credential in `X-QC-API-KEY` or `Authorization: Bearer`. Query and form
tokens are rejected in all modes. Store the credential outside source control.

For browser callers on exposed listeners, configure the exact origin allowlist
as described in [API browser origin policy](QC_CORS_PUBLIC_MODE_SECURITY.md).
