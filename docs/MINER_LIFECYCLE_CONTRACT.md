# Miner lifecycle and cooperative cancellation

The native wallet owns its local PoW worker. Node development controls own the
node's continuous miner. Stopping one does not stop other clients or create a
second reward system. Both use the existing QC consensus rules.

## Node mining

Each continuous run owns an immutable stop channel and an atomic, single-use
stop signal. Stop requests do not acquire the runtime or blockchain locks held
by a legacy PoW search. Repeated or late requests cannot close a replacement
run's channel. A run remains registered until its worker has completed cleanup.
Callers that require completion must use WaitMiningContext, not just StopMining.

Runtime mining contexts observe caller cancellation and the runtime's shutdown
signal. Shutdown signals cancellation before acquiring runtime locks. Waiting
for the runtime mining mutex or blockchain mining lock is cancellable as well.
Continuous and one-shot searches observe shutdown, including compatibility and
isolated-local mining. Normal cancellation is not reported as a mining failure.

Context-aware PoW retains existing nonce order, hash preimages, difficulty,
transaction validation, subsidy, fees and persistence rules. Network-V4 keeps
its existing detached job search. Legacy, continuity, atom and NFT-V4 block
builders check cancellation during nonce search. Existing synchronous library
entry points retain their signatures and use a background context; callers
requiring cancellation use their context-aware counterparts.

Cancellation before the persistent commit barrier installs no candidate. Once
disk installation begins, it finishes with the corresponding in-memory commit:
cancellation must not leave disk and memory describing different chains. Stop
completion can therefore wait for an in-progress storage operation. Arbitrary
blocking callbacks or operating-system I/O cannot be forcibly interrupted.

## Native wallet worker

A running wallet worker has one owner and one completion channel. Starting
another returns an explicit busy error. Completion clears only the matching
run. Stop cancels that worker and waits up to five seconds for completion; if
completion is still pending it returns an error and retains active ownership.
It does not show a false stopped state or start a replacement over that worker.

Queued UI callbacks verify their run generation before updating state. Old
completion or stop callbacks cannot clear a replacement worker's state. A
matching stopped notification is delivered at most once. The tab's final queued
render reads current worker ownership instead of a stale requested boolean;
neither delayed stop nor delayed start rendering overrides the real state.
The wallet uses its
native worker, not an external miner executable or node process-control route.

## Acceptance evidence

The 2026-11-15 lifecycle acceptance covers bounded legacy cancellation,
concurrent stop, shutdown, autosave, restart, per-run signal ownership, queued
UI callbacks and pending-stop behavior under the race detector. The complete
blockchain suite and prior control-authorization regression are recorded
separately in the local acceptance report. This source acceptance does not
replace release provenance or manual GUI acceptance.
