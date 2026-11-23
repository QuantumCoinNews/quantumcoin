# Canonical mining and block lookup routes

The old standalone `api.StartHTTP` is retired and always returns
`ErrStandaloneHTTPRetiredV1`; it owns no listener or routes. Its old stub mining
and block endpoints are not a second runtime.

The root node API owns the following active routes, under header token auth and
the shared CORS/read-only boundaries:

| Route | Contract |
| --- | --- |
| GET /api/block?index=N or ?hash=HEX | Actual canonical-chain block, HTTP 200. JSON uses the existing explorer presentation, including byte arrays encoded as base64. |
| GET /api/block with invalid/missing selector | HTTP 400. |
| GET /api/block for a well-formed absent block | HTTP 404. |
| GET /api/block when the chain is unavailable | HTTP 503, not a missing-block 404. |
| POST /api/mine | Authenticated node-side single-block CPU mining. HTTP 200 success identifies the block committed by the runtime; this is not a queued-job or simulated-accepted response. |
| /api/mine in read-only API mode | HTTP 403 for an authenticated request; no mining or chain write. |
| /api/mine with malformed JSON / missing miner | HTTP 400. |
| /api/mine with no chain | HTTP 503. |
| /api/mine when durable write fails | Error response; no successful receipt or installed block. |

Wrong methods get HTTP 405 after auth in writable mode; a missing/incorrect
credential gets HTTP 401. Earlier CORS, body-size, limiter or read-only rejection
can take precedence. Application errors use the existing JSON error envelope;
neither HTML 200 nor `accepted: true` substitutes for a committed mining result.

`/api/mine` runs computation on the node. GUI/browser/Telegram client PoW work
uses the separate canonical `/api/mine/job` and `/api/mine/submit` protocol; it
must not describe a call to `/api/mine` as the client's own PoW. This contract
adds no chain, reward, pool split, difficulty or consensus rule. A local commit
receipt remains subject to coinbase maturity, network acceptance and reorgs;
it is not a finality guarantee.
