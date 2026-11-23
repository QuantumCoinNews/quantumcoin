# API routing and navigation boundary

The root HTTP router owns the active API. The former standalone `api.StartHTTP`
does not open a listener. Embedded WebUI navigation is a separate fallback.

| Route | Current contract |
| --- | --- |
| GET /api/wallet/address | Header auth; configured local wallet address. It never generates a wallet or returns a key. Configuration failure is an error. |
| GET /api/network/label | Public JSON network label with no-store headers; no credential or wallet secret. |
| GET /api/ai/analysis | Header auth; advisory analysis object with anomaly_report, recommendations and reward_suggestions. Disabled AI returns empty observations. |
| GET /api/ai/bonus | Header auth; read-only legacy bonus history (array or null), not a mint/award operation. |
| GET /api/miner/status | Header auth plus direct loopback, explicit dev mode, local Host and no proxy forwarding. JSON running/height/bits/httpPort. |

These reads remain available in read-only mode subject to their existing auth
and local-development constraints. Mutation methods are rejected; these routes
do not create balances, mining work or wallet keys.

Unregistered canonical `/api` or `/api/...` paths return HTTP 404 with the JSON
error envelope `success: false` and a nonempty `message`. They never receive the
SPA index, regardless of filename suffix or method. In read-only mode the
capability gate can return 401/403 before an unknown route is dispatched.
Header/body limits, CORS and rate gates can likewise take precedence. An OPTIONS
CORS preflight is transport negotiation and may return 204; it is not a claim
that the requested application route exists.

Dot segments or duplicate slashes that would make ServeMux redirect an API path
are rejected with JSON 404 before mux normalization. Both the original decoded
path and its cleaned namespace are checked, preventing API-to-navigation and
navigation-to-API redirects. A legitimate single trailing slash is preserved.
The path is never rewritten to an actionable API route by this middleware.
Existing CORS and read-only gates remain outside this check.

Normal navigation such as `/wallet/receive` still receives the public landing
page. Missing assets/files return 404. Go's case-sensitive routing is retained;
this policy does not invent aliases for endpoints or change the existing JSON
schema of registered endpoints.
