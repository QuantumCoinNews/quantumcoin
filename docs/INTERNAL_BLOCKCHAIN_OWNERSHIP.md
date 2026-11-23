# Internal and blockchain ownership

`blockchain.Blockchain` owns accepted transactions, pending outpoints, UTXO,
coinbase maturity, reward, fees, supply and chain persistence. Node runtime owns
the live chain reference, lifecycle and trusted reload boundary.

| Internal API | Role | Runtime boundary |
| --- | --- | --- |
| Mempool | Compatibility adapter bound to one canonical Blockchain | No separate pending pool; Add delegates to canonical admission; Clear/Remove/Pop cannot mutate it. Root/CLI/P2P admission does not use this adapter as a second acceptance gate |
| UTXOSet | Bound read/reindex adapter | Detached snapshots and canonical maturity/spendable selection; no independent balance map or authority switching |
| Reward helpers | Canonical calculation facade | Delegates height/state/cap rules to blockchain; does not pay or mint |
| FeeManager | Retired fail-closed compatibility API | No local accumulated fee balance; canonical transaction fee validation owns economic effects |
| Bonus history | Bounded advisory memory history and optional JSONL audit helper | `pending_settlement` never means paid. Root exposes a read view; bootstrap configures its audit path. The yearly miner hook and AI bonus execution remain disabled |
| internal/httpresponse | HTTP response formatting utility | No monetary authority; separate subpackage |

Bonus JSONL is not a durable balance/settlement/idempotency ledger and is not
loaded into spendable entitlements on restart. Duplicate audit calls are not
duplicate payments. No active runtime producer is enabled by this review.
Multi-process monetary settlement and crash-safe payment recovery must stay in
their canonical chain or service ledger, never be inferred from this helper.

Validation combines real canonical adapter tests, concurrent read/reindex and
reward/fee tests, concurrent JSONL audit writes, write-error memory invariants,
hidden child-process restart tests, disabled miner hooks across restart, actual
root source/import admission guards, and trusted chain replay/persistence tests.
Synthetic UTXO projection fixtures establish adapter behavior only, not valid
consensus blocks. Targeted race tests do not replace the separately tracked
full blockchain race/release acceptance.
