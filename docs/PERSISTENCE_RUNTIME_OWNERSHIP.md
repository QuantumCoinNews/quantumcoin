# Persistence ownership and acceptance

The node owns its chain through `node.Runtime`, the trusted `blockchain`
loader, and the shared atomic blockchain writer. Mining, autosave, shutdown
and read-only reload must use those boundaries. Network identity, checksums
and consensus replay are mandatory there. No second chain codec is supported.
The unused `persistence.SaveChainV1` / `LoadChainV1` API is retired and fails
without disk I/O. Historical envelopes are not trusted migration input.

Telegram Game owns its SQLite database through its canonical Python runtime.
The Go persistence plan does not create or migrate that database, and is not a
startup gate or proof of mainnet readiness. Schema names are inventory labels,
not chain-file header versions. Chain storage is an atomic snapshot, not an
append-only physical file.

The token bridge uses the persistence file lock. Its sidecar remains on disk;
OS handle ownership, not sidecar existence, determines whether storage is busy.
Release is idempotent and process termination releases ownership automatically.
An old sentinel sidecar no longer blocks recovery after a crash.

Inventory paths reject unsafe segments and existing symlink/junction ancestors.
Operators must protect parent directories from concurrent replacement. Runtime
readiness requires real independent restored bytes matching backup size/hash,
plus trusted storage-specific schema/encryption verifiers. A dry-run is only
a plan and never establishes successful restore. Test marker verifiers do not
prove encryption of production data. Offline authenticated/encrypted snapshots
and live SQLite backup remain owned by `backuprestore/offline`.

Independent off-host backups, real custody recovery and mainnet deployment
acceptance remain separate tracked operational work.
