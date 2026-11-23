# Runtime persistence and source delivery

All supported ASGI entry points export the same `telegram_game.runtime:app`.
`api_server` is a compatibility alias with loopback binding, no reload by default
and proxy headers disabled. It has no independent earn policy, database,
simulated payout worker or administrator fallback. The compatibility handler
routers intentionally contain no routes; the canonical app owns the live routes.

## Existing and new databases

For the existing local game database, explicitly select its absolute path before
starting a launcher or uvicorn. For example, in this checkout:

```powershell
$env:QC_GAME_DB = 'C:\Projects\quantumcoin\telegram_game\dev_api_state.db'
Set-Location C:\Projects\quantumcoin\telegram_game
python -m uvicorn telegram_game.runtime:app --host 127.0.0.1 --port 8080 --no-proxy-headers
```

Back up and independently reconcile legacy data before production rollout.
This selection does not certify old rewards or simulated confirmations. No
database is automatically moved, merged, reset or replaced.

Fresh installations default to a user data directory outside the source tree,
with separate `local`, `testnet` and `mainnet` paths. See
[the runtime guide](PLAYER_RUNTIME_AND_IDENTITY.md) for platform paths and
`QC_GAME_DATA_DIR`. A known old game database or a retired `QC_TGWT_DB` setting
without an explicit canonical selection stops startup rather than silently
creating a different empty database. Relative `QC_GAME_DB` values are rejected.
The four server launchers use this same policy.

Every canonical SQLite connection uses WAL and a five-second busy timeout.
SQLite still serializes writes; short legacy Python lock sections remain. The
game does not hold a global lock or open transaction while waiting for a node
job. Mining completion consumes stored accepted PoW and never asks the server
to search for a nonce. Slow bridge health I/O is moved off the event loop.
Concurrency tests pause node health/job responses and verify that authenticated
heartbeat, wallet and TGWT requests complete before the paused call is released.

## Source delivery checks

Repository and standalone game ignore rules cover SQLite databases, WAL/SHM,
rollback journals, SQLite backup variants, local environment files and generated
settlement reports. Matching runtime files are also excluded by source archive
attributes. Runtime data stays on the local machine; no real DB is a test fixture.

From the repository root, check the index and current committed tree:

```powershell
python scripts/source_hygiene.py
```

The check fails if runtime DBs or settlement reports are tracked, including a
file added with `git add -f`. To create a source archive of an already reviewed
and committed revision, choose a **new** output filename:

```powershell
python scripts/source_hygiene.py --archive C:\path\to\quantumcoin-source.zip
```

This archives **HEAD**, applies archive exclusion attributes and refuses to
overwrite an existing file. It does not include uncommitted working files. The
archive path above is an example; no source archive or GitHub update is created
automatically by runtime startup. Regression scripts set an explicit isolated
test database per run. The old mini diagnostic opens the selected database in
read-only mode and cannot create a cwd-relative empty DB.

Existing delivery ZIPs and previously published Git history are not rewritten by
these safeguards. A previously distributed copy must be replaced separately.
No claim is made that past distributed data has been recalled. No history rewrite
or deletion of the local database is performed.

Real BSC payment confirmation remains deferred to the final mainnet phase.
