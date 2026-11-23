# HTTP limiter resource contract

The canonical node owns one global HTTP guard and one shared endpoint limiter.
These are node-local, in-memory ingress protections, not consensus or spending
authorization. Authentication, signed transaction validation and the route
order remain in place.

| Resource | Bound |
| --- | --- |
| Guard identities | 32 shards, at most 256 keys each: 8,192 total upper bound |
| Endpoint identity/route pairs | 32 shards, at most 256 pairs each: 8,192 total upper bound |
| Retained key bytes | 256; endpoint bound includes identity, separator and route key |
| Guard history | 100 requests per 10 seconds, plus one ban-triggering timestamp; no timestamp growth while banned |
| Endpoint history | Only requests admitted by that limiter; at most its configured count, with a hard policy maximum of 256 |
| Supported endpoint window | Positive and no more than 24 hours |
| Limiter denial log writes | At most one writer in flight per limiter instance; a one-second cooldown between reservations |
| HTTP monitoring notifications | Three fixed category slots (auth failure, rate limit, API failure), each with one writer in flight and a one-second cooldown |

An endpoint may still consume an admitted limiter slot when downstream auth or
business validation rejects the request, according to its existing wrapper
order. Rejections by the limiter itself do not accumulate timestamps or extend
the endpoint record's expiration. Invalid endpoint policies fail closed before
allocating a record. The canonical router always creates its endpoint limiter;
the private nil-limiter compatibility path remains a no-op.

The guard's unbanned idle records expire with its request window; active bans
remain protected for the existing ten-minute duration. Endpoint expiry belongs
to each record's own admitted-request window. A short-window route cannot
prematurely remove a long-window quota in the same shard. Cleanup is opportunistic
every 64 operations and when capacity is reached. An idle instance may retain
bounded state until another request triggers cleanup; there is no cleanup
goroutine per identity.

If a shard still contains live records after cleanup, a new key receives a
denial (HTTP 429 at the middleware boundary). Live quotas and bans are not evicted
to admit fresh identities. Known clients with remaining quota can continue.
Hash skew may reach a shard limit before the total 8,192-key bound. Once records
expire, new identities can be admitted again. A process restart clears these
in-memory protections; there is no distributed or persistent ban database here.

The real socket peer supplies identity; forwarded headers do not create fresh
limiter identities. Endpoint keys are registered route budget names, not arbitrary
request paths. New guard keys own their bounded string storage. Request labels
written to denial logs are truncated to the label budget.

Logging happens after the shard mutex is released. If storage stalls, the first
selected writer can remain blocked; later denials skip logging and do not queue
more writers behind it. A reservation is released on completion/error. HTTP
monitoring has separate fixed category budgets so warning floods cannot suppress
the first critical API-failure category. The existing bounded live-evidence
collector continues to receive eligible observations even when persistent
notifications are coalesced. This does not impose a global budget on unrelated
application/monitoring operations.

Windows acceptance includes concentrated same-shard churn, a 25,000-request
rejected flood, mixed route windows, exact expiry/recovery, capacity HTTP status
and forwarded-header isolation, concurrent churn, and a real locked security
log in isolated test child processes. Node acceptance also covers distinct-IP
monitoring floods and a stalled backend. Previous CORS, early-rejection,
miner-control and monitoring integration checks remain covered.
