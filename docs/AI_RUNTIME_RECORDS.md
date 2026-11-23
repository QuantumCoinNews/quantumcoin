# Advisory runtime records

AI cannot create monetary rewards. `SetBonus` and `AddBonus` are retired no-ops;
`GetBonusSnapshot` always returns zero. Bonus builders return no suggestions, and
`WriteAIBonusesTo` / `CollectAndWrite` always return `ErrAIBonusDisabled`, including
empty requests. Existing bonus files are neither read as authority nor deleted.
Legacy `bonus` fields in AI state are ignored and omitted from new snapshots.

Canonical node bootstrap installs one advisory observer. Lifecycle events feed
the same `ai.AIAlert` buffer exposed by the root API and compatibility API helper.
There is no second root alert store. Runtime bootstrap loads state; events,
autosave ticks and successful shutdown save it. Read-only API mode never creates
or writes advisory files. Bare `NewRuntime` does not install persistence.

The canonical path is `<absolute configured chain filename>.advisory/`:

- `ai_state.json`: telemetry and the last 100 alerts.
- `ai_security_reviews.json`: the last 1,000 transaction reviews, with a
  cumulative `dropped_reviews` count for retention removals.

The path is independent of working directory and isolates different chain
filenames. Older files in working directories are not automatically copied or
deleted. `LoadState`/`SaveState` remain compatibility wrappers using `QC_NODE_DIR`;
the runtime uses explicit `LoadStateFrom`/`SaveStateTo` paths.

Telemetry refreshes from the live peer count at bootstrap, runtime events,
autosave and the root telemetry request. The root response includes `ai` and
`ai_persistence`. No reliable runtime-wide hash counter exists yet:
`hashrate_available=false` denotes unavailable measurement, not a measured zero.
This is telemetry, independent of AI enable/disable decisions.

Local mining, accepted external PoW and controlled NFT mining invoke advisory
recording after successful commit and after releasing runtime/chain mining
locks. Review amounts use their block's protocol denomination. Coinbase is
excluded. Standalone blockchain methods do not write AI files. Peer transaction
admission remains independent of advisory review; the bounded analysis endpoint
can observe transactions arriving from all ingress paths.

Writes use unique temporary files, flush/close, and OS file replacement. A process
mutex plus an OS directory lock serialize concurrent read/modify/write review
updates, including multiple processes in the same OS session. Windows uses a named mutex; Unix uses a
sidecar flock. State is the latest owning runtime snapshot, not a cross-process
merge of telemetry or alert buffers. Operate one canonical runtime per process
and one owning runtime per chain.

Reads reject files larger than 4 MiB. Corrupt review history is preserved and
reported rather than silently replaced. Review payloads are bounded; atomic
write failures preserve the last good file. These are bounded operational
records, not an immutable audit archive or a promise of unlimited retention.

Alert IDs combine a random process prefix with an atomic sequence. Legacy
duplicate/missing IDs are repaired on state load. Alert strings are capped at
1,024 bytes; returned slices are detached snapshots.

Storage failures increment process-local `ai_persistence.failures`, set
`last_failure_at` / `last_error`, log an operational error and enqueue an
`AI_PERSISTENCE_FAILED` alert when AI is enabled. If disk is broken, in-memory
status and logs remain available. Counters start fresh with the process; saved
alerts retain earlier errors once a subsequent snapshot succeeds. Advisory
failure never changes an accepted block into a failed mining response, changes
transaction admission, or changes subsidy, balances or total supply.
