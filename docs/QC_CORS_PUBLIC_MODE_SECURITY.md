# API browser origin policy

The node selects CORS policy from the effective HTTP listener address. A
numeric loopback listener keeps wildcard compatibility for local tools. Any
other address, including wildcard and hostname binds, requires an exact browser
origin from `QC_CORS_ALLOWED_ORIGINS` or the single-origin compatibility variable
`QC_CORS_ALLOWED_ORIGIN`. The plural setting accepts comma-separated origins.
The public-testnet switches also select the restricted policy on loopback.

Restricted listeners ignore wildcard origins and return 403 for an origin
outside the allowlist, including on preflight. Set the origin including its
scheme and port if the browser sends that origin. Clients that omit the Origin
header use the normal route authentication rules.

An omitted Origin is different from an explicitly empty one. Empty/whitespace,
repeated Origin values (even identical values), or duplicate case variants are
rejected with 403 before application handlers, including for ordinary GET/POST.
These malformed requests are rejected on local listeners too; the documented
local wildcard compatibility for a single nonempty Origin remains available.
The middleware preserves existing Vary fields and adds Origin, and does not
leave a stale Access-Control-Allow-Origin header on rejected responses.

CORS is a browser-origin boundary, not authentication. An allowed browser or a
native client without Origin still needs the route credential and a valid
wallet-signed transaction to submit a transfer. Preflight does not submit or
authorize a transaction.
