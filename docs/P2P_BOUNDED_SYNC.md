# Bounded peer synchronization

Updated peers advertise `SyncVersion=1` in the existing network handshake.
They request ranges of at most eight blocks, reducing each response to a
896 KiB payload budget. The existing 1 MiB message limit and 512 KiB individual
block payload limit remain in force. Transport reads also have a per-message
budget including Gob framing overhead; messages are dispatched in stream order.

A continuation pins the advertised tip and preceding block hash. The receiver
replays blocks in a detached candidate using only local genesis, economics,
activation, vesting and NFT authority context. The completed candidate goes
through canonical replay and cumulative-work selection under the chain writer
lock. Claimed height or difficulty alone cannot select a branch.

An interrupted connection retains verified progress in memory for two minutes.
A new connection uses a fresh session identifier and can continue from that
progress, including through a different peer that still has the pinned branch.
If that branch is unavailable, one reset starts from genesis; repeated resets
are rejected. Process restart or expiration starts a fresh download. This cache
does not replace canonical disk persistence or backup recovery.

Only one download owns a node chain at a time; the process cache has four
slots. Untried or interrupted peers retry on heartbeat. Completed histories are
not periodically re-downloaded. Invalid chunk positions, identifiers, genesis,
PoW, changing tips, oversized data and completed-session replays are rejected.

Older peers retain bounded legacy compatibility. A history larger than the old
single-message limit requires range-capable peers; the legacy response sends an
explicit error rather than an oversized chain. No consensus rule is relaxed.

Acceptance uses two independent test-node processes over TCP, the canonical
handshake/connection/transaction validation paths, and a transparent proxy that
cuts a connection mid-transfer. A roughly 1.97 MB valid fork is transferred
while the source mines, and completion after reconnect must actually resume.
The proxy records the largest payload. Tests also cover malformed/replayed
chunks and run both node processes under the race detector.

```powershell
python scripts/verify_testnet_sync.py --output C:/Temp/qc-sync-acceptance.json
```

The command snapshots the working tree and preserves source fingerprints,
commands, exit codes and logs outside the checkout. The helper-only test is
intentionally skipped in its parent process and executed in each child process.
This proves the P2P component acceptance; the final signed application binary,
selected testnet profile, independent hosts and backup/restore have separate
release acceptance gates.
