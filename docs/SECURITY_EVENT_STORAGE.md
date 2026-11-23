# Security event storage and alarm lifecycle

The canonical bootstrap binds compatibility security event producers to
`<absolute chain path>.security/<validated network profile>/security_events.json`.
Profiles are `local-release`, `testnet-local`, `testnet-public`, and `mainnet`.
The profile comes from the same validated network configuration as the node.
The process hosts one canonical runtime. Separate node processes use separate chain
paths when they need independent observation stores. Explicit directory arguments
remain available to isolated tools and tests.

Existing root-level `security_events.json` files are preserved and are not imported
automatically: their network of origin cannot be established. Archive them as
historical evidence; migrate only after independently establishing their network.

The local store retains at most 1000 observations and at most 8 MiB. Truncation is
reported. Updates use an OS file lock across processes, a unique temporary file,
file sync, and atomic replacement (Windows write-through; directory sync on Unix).
Readers share the store lock. Invalid/empty/oversized/unsupported records are
preserved and reported as errors, never silently replaced with an empty history.
Write failures remain visible in the producing process through `writeError`.
This is a bounded local operational record, not an immutable external audit archive.
Atomic replacement protects against partial JSON; hardware that ignores flushes
cannot be guaranteed durable. Export evidence to a separate archive when required.

The score and all event summaries use the last 24 hours of unresolved evidence.
Repeated `(source,type,subject)` observations contribute once at their highest
severity. Informational events do not lower the score. Acknowledgement retains
risk; resolution clears the selected group up to that event, and later observations
remain open. The unfiltered events endpoint retains history and actor/note fields.

`POST /api/security/events/action` accepts
`{"id":"<event id>","action":"acknowledge|resolve","note":"<operator note>"}`.
It uses the existing API authentication plus the separate
`X-QC-MONITORING-ADMIN` credential matching server-side
`QC_MONITORING_ADMIN_TOKEN`. There is no default operator credential. Read-only
API mode rejects mutations. Actors and paths are selected server-side.
Lifecycle records persist across restarts. These operator actions do not modify
chain state, balances, fee rules, peer bans, mining, or consensus.

P2P `ReputationSnapshot()` publishes `overallRisk`, `peerCount`, `peers`, and
`scope`. The aggregate is the highest observed local reputation risk; an empty
local observation set is LOW, while unavailable peer security is UNKNOWN at the
runtime boundary. It is not a proof of global network safety.
