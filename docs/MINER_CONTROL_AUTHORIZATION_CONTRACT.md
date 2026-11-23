# Miner control authorization

The canonical node owns the following development-only control routes:

| Route | Method | Purpose |
| --- | --- | --- |
| /api/miner/start | POST | Start the node's continuous miner |
| /api/miner/stop | POST | Request that node miner to stop |
| /api/miner/status | GET | Read that node miner's state |

These routes require explicit QC_DEV_MODE, a direct loopback client, a loopback
effective API bind, a local Host and an explicitly configured QC_API_TOKEN.
Public-testnet/exposed listeners are excluded. API read-only mode rejects
start/stop. Credentials are accepted through X-QC-API-KEY or Authorization:
Bearer, never through query/form data. Authentication precedes the control
endpoint's rate limiter; unauthorized attempts do not exhaust the authenticated
three-requests-per-ten-seconds start/stop budgets. Those budgets remain enabled.

Forwarded, X-Real-IP, Via and all X-Forwarded-* header names are rejected, even
when a value is empty. Browser Origin must be the exact HTTP/HTTPS request
origin without credentials, path, query or fragment; incompatible fetch-site
metadata is rejected. Native clients can omit browser Origin metadata.
Operators must not expose these development controls through a proxy that
injects their node credential and hides the original request metadata.

Start accepts only a single JSON address field with a valid QC reward address:
{"address":"QC_REWARD_ADDRESS"}. Missing/malformed/ambiguous/unknown fields,
trailing JSON and query-based addresses are rejected. Body size is bounded.
There is no configuration/default-wallet fallback for this control request.
Same-address start while already running is idempotent; a different valid
address receives 409, and an invalid address receives 400. Neither retargets
the current miner. Missing runtime/chain returns 503 rather than panicking.

The native GUI follows a separate client responsibility: it computes real PoW
in its local worker and uses the node's canonical job/tip/submit/result routes.
It does not call these development process-control routes. The configured API
token is sent through headers, the expected genesis is pinned from trusted
local config, and a failed job preflight starts no hashing worker. GUI Stop
cancels that GUI's own worker; it does not stop another client's or the node's
miner. Both routes still participate in the same QC blockchain/PoW rules.

Mining lifecycle acceptance is documented in MINER_LIFECYCLE_CONTRACT.md.
Compatibility/isolated searches now observe cancellation and shutdown as well
as network-V4. Stop requests and worker completion are distinct operations;
per-run ownership prevents late stop/UI callbacks from affecting a new run.
No block reward, halving, supply, pool distribution or consensus policy changes
are introduced by this control contract.
