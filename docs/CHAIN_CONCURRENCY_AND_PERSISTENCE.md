# Chain concurrency and persistence

The root application no longer has a global `bc` pointer. Every HTTP operation
captures its own chain reference through the locked `node.Runtime.Snapshot`
boundary, after any requested disk reload. A response uses that same reference
for block lookup, amount conversion and presentation. The CLI has a local chain
reference; configuration remains immutable after startup.

## Runtime ownership

- Observer reload serializes before capturing the current chain and file stat.
  A validated replacement is published under the runtime write lock. Reload
  events are delivered after service locks are released.
- Ordinary mining holds its runtime reference until the operation completes.
  Observer mode rejects mining and P2P startup, and skips chain persistence.
- Reload refuses a runtime with an active miner or a chain previously handed
  to P2P. A peer handler can outlive a socket close, so such a runtime must be
  restarted as an observer before replacing its chain pointer.
- Shutdown closes admission before inspecting services, stops periodic autosave,
  waits for the registered miner and uses the same final-save boundary. New
  mining/P2P work and NFT mint work cannot enter after shutdown starts.
- The retired miner compatibility API uses an atomic pointer and pins one
  reference when starting a worker.

## One chain-file installation path

`SaveToFile`, `CommitValidatedBlockAndPersist` and durable NFT candidate commit
all call `persistChainFileV1`. HTTP/CLI saves delegate to `Runtime.SaveChain`.

Lock order is consistent: runtime ownership (when applicable), chain state,
destination mutex, then OS writer lock. Ordinary saves hold the chain read lock
through installation; commits hold its write lock. Encoding begins only after
acquiring the destination lock. A delayed save on that chain cannot install
bytes captured before a newer block/NFT commit.

Destination keys resolve absolute paths and existing symlinks; Windows keys
also normalize case. The in-process lock is shared across chain instances.
An OS lock also serializes cooperating processes on the same host: a named mutex on Windows,
or `flock` on a persistent `.lock` sidecar on Unix. Unix lock files are not
unlinked on release because another process may already be waiting on the inode.

The common writer creates a unique temporary file beside the destination,
encodes while locked, flushes the file, closes it and installs it with the
platform replacement operation. Windows uses `MoveFileEx` with replace/write-
through flags; Unix uses same-directory rename. Chain-file readers on Windows
allow delete sharing so an observer does not prevent an atomic replacement.
Readers see complete old or complete new files on filesystems supporting the
native replacement operation; they never read a partially written temporary.

Encoding/write/install failure preserves the previously installed destination.
Temporary files are removed on normal failure, worker commits roll back their
live append, and NFT commits publish live state only after successful install.
The old shared `<chain>.tmp` filename is no longer used by these writers.

## Scope of the guarantee

The supported deployment remains one authoritative writable runtime per chain
file with observer readers. OS serialization prevents overlapping cooperating
process writes; it does not merge independent writable runtimes' different
histories or replace peer consensus/fork selection. Likewise, this file flush
and native replacement boundary is not a claim of full power-loss transactional
durability on every filesystem. A process killed before installation can leave
an unused temporary file; the previous configured chain file remains the load
target.

## Verification

Tests cover real concurrent API reload/read traffic; simultaneous saves, mining
and trusted readers; save/NFT-commit overlap; partial encode/install failure;
an actual child process blocked by the OS writer lock; observer write denial;
P2P pointer pinning; and autosave/mining shutdown. Architecture guards prevent
reintroducing a root global chain pointer, direct adapter file saves or an
independent chain installation path.

Run `go test ./... -race -count=1` for the full regression suite.
