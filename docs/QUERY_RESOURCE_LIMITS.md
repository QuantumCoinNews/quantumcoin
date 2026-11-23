# Query resource limits

The canonical node API owns these reads. The standalone `api` HTTP server is
retired and fails closed; its former mempool listing is not reintroduced.

| Route | Work and response contract |
| --- | --- |
| `GET /api/blocks` | Current-tip-relative pages, default 20 and maximum 200 block summaries; no transaction payload cloning or signature verification. |
| `GET /api/tx/status?id=...` | Canonical transaction-location indexes; `txid` remains an alias. Unknown identifiers return `unknown`; missing identifiers return 400. |
| `GET /api/block?hash=...` | Canonical block-hash index, then a defensive copy of the matching block. Unknown hashes do not scan the chain. |
| `/api/mempool` and descendants | Unregistered routes return the JSON API 404 boundary. They do not invoke the retired list/signature-validation path. |

Block pages expose `X-Total-Count`, `X-Page-Limit`, `X-Page-Offset`,
`X-Has-More`, and `X-Next-Offset` when another page exists. Positive limits
above 200 are capped. Invalid/nonpositive/overflowing limits fall back to 20;
invalid/negative/overflowing offsets fall back to zero. An offset at or past
the end returns an empty JSON array without a next-page cursor. Page contents
and counts share one state read lock. Offsets refer to the tip at each request;
they do not pin a chain snapshot across multiple requests or reorganizations.

Transaction and block-hash indexes are private, derived, in-memory data. The
existing canonical constructor, append/admission, load and replacement hooks
maintain them under the chain state lock. Load/reorg rebuild them from the
authoritative ledger, rather than trusting a serialized index. A manually
constructed compatibility instance initializes missing indexes on its first
lookup; subsequent lookups use the derived maps. Index initialization and reorg
can therefore take linear work once; ordinary warm lookups do not.

The block-hash index references the chain-owned block, including append paths
that clone their candidate. It does not retain a returned caller-owned candidate.
Every returned block is defensively copied. Indexed lookup does not make block
copying/encoding independent of that block's own payload size; it removes work
over unrelated blocks. The existing per-block validation rules are unchanged.

Mempool telemetry uses the canonical count. Security/advisory sampling uses the
existing bounded snapshot (up to 200 blocks, 2,000 transactions and its byte
budget), analysis concurrency limit and endpoint rate limit. Sampling is not a
replacement mempool admission or signature-verification authority.

Acceptance records cover the actual mux, paging edge cases and large transaction
cardinality, unknown/confirmed transaction status, retired routes, canonical
admission/replacement, disk reload, block snapshot ownership and concurrent
canonical appends. Scaling benchmarks report timings/allocations for different
chain sizes without fragile timing-based CI assertions.
