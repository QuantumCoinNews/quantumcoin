# Miner attach and exclusive startup

This contract concerns the compatibility miner managed by `miner.Start` and its
optional local token bridge. It does not activate the bridge or change PoW.

- `AttachToRunningMiner` always returns
  `ErrAttachToRunningMinerUnsupportedV1`, including a nil bridge. It does not
  replace callbacks, stop a miner or claim that an attachment succeeded.
- `StartMiningWithToken` uses `miner.StartExclusiveV1`. The lifecycle mutex
  covers the idle check, request/difficulty validation, bridge preparation and
  loop installation. Preparation is not run if a miner is already active.
- A busy bridge startup returns a nil bridge and an error matching both
  `ErrAttachToRunningMinerUnsupportedV1` and `miner.ErrMinerAlreadyActiveV1`.
  It does not create a new registry, update an existing registry, or change the
  active miner's callbacks. The caller must explicitly stop, await completion,
  then perform a cold startup with the bridge options.
- A preparation error does not activate a loop. Preparation callbacks are
  trusted startup code and must not re-enter Start/Stop or other lifecycle
  mutations while holding the lifecycle mutex.
- Plain `miner.Start` retains its compatibility behavior: an already-active
  call is idempotent and does not reconfigure the running miner. Its nil result
  is not an API for attaching a new bridge. Exclusive startup distinguishes an
  actual new loop from that no-op.

Shared option precedence is defined in
[mining startup options](MINING_OPTIONS_CONTRACT.md). No automatic stop/restart,
hot callback replacement, token issuance policy or new consensus rule is added.
