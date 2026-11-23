# HTTP response schema

The root node, runtime NFT adapters and retained package-api response helpers
share `internal/httpresponse`. This package serializes responses; authentication,
CORS, rate limits and operation ownership remain with existing callers.

Application errors use the appropriate non-2xx HTTP status and a JSON object:

```json
{"success":false,"message":"method not allowed"}
```

Endpoints may retain additional machine-readable fields such as `code` and
`version`. NFT errors keep both and include a nonempty message. Method rejection
retains Allow, authentication retains WWW-Authenticate, and CORS headers survive
serialization. Unsupported features are error statuses rather than successful
"coming soon" responses. Unknown API paths return JSON and do not become SPA
navigation.

Successful responses retain their existing endpoint DTO: object, list or null.
There is no new global data wrapper, renamed transaction ID or changed balance
representation. Wallet and bridge consumers must continue to validate the
specific response contract; the common error envelope is not a substitute for
a valid signed transaction/mining receipt.

The writer marshals the complete JSON document before committing headers. An
encoding failure becomes HTTP 500 with a fixed, safe error message. It removes
stale Content-Length, writes one JSON document plus newline, and marks JSON as
nosniff. Encoding and transport failures are logged by error type without
logging payloads or arbitrary error strings. Write errors and short writes are
observable; after headers are committed the writer cannot replace the response
or roll back a transaction. Callers reconcile uncertain delivery using existing
transaction/receipt lookup and idempotency rules.

Protocol exceptions are explicit: HEAD has no response body, OPTIONS preflight
may return empty 204, and net/http can reject malformed framing or oversized
headers before application dispatch using its own transport response. Static
assets/navigation remain WebUI responses. CLI stdout/stderr is a separate local
interface; raw CLI output is not an HTTP response envelope.

The old package-api server remains retired. Compatibility alert/telemetry
helpers keep their read DTOs but share JSON method/encoding errors; they do not
create a second listener. Its old wallet route remains fail-closed 410/405.
