# Queue and worker ownership

The Go `queueworker` package is a standalone preparation library. It has no
production root/node/recovery entry point. Its constructors, manual-start plan,
handlers and tests do not authorize live work. No central Go worker is added.
Its queue names are inventory labels, not service bindings or proof of launch.

| Work | Canonical owner | Activation boundary |
| --- | --- | --- |
| TGWT withdraw review / settlement | `telegram_game/telegram_game/runtime.py`, `dev_api.py`, canonical SQLite ledger | Authenticated server-side review/ledger transitions; real BSC confirmation remains disabled pending mainnet evidence |
| Monthly NFT review | Telegram Game review endpoints | Verified server activity and admin review; final assets/approval remain separately tracked |
| Monitoring alerts | `node.Runtime` and monitoring engine | Node-owned lifecycle and storage; no independent Go queue process |
| Content publication | `autopublisher/src/runtime/engine.ts`, `scheduler.ts` | Publisher-owned job ledger, scheduler authority and delivery receipts |
| Backup / restore | `cmd/recovery`, `backuprestore/offline` | Explicit offline operator command; node startup does not launch recovery |

These owners retain their own durable state and authentication. A Go queue job
cannot grant mint, fund transfer, balance or chain mutation authority. Future
integration requires an explicit migration decision, service authentication,
network-bound durable state, bounded retry/dead-letter policy, shutdown contract
and service-side idempotency. Handler callbacks must honor cancellation; this
preparation library does not sandbox arbitrary callbacks or promise exactly-once
external effects. File serialization alone does not make independent in-memory
queues a distributed broker. None is started by current runtime code.

The standalone file-store lock uses the shared persistence OS lock. Persistent
`.lock` files are rendezvous files; file existence does not indicate an active
owner. Abrupt process termination releases the lock without sentinel deletion.
This does not replace canonical SQLite transactions or publisher locking.

Verification: dependency graphs for root/node/recovery must exclude
`quantumcoin/queueworker`; standalone race tests exercise explicit start/stop,
approval, lease/ack, retry/dead-letter, audit, durable restart/idempotency and
process-exit lock recovery. These results do not claim live BSC payouts, NFT
activation or social publication acceptance.
