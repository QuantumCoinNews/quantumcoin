# Snapshot consistency, encrypted custody and game authority

## Capture boundaries

Recovery snapshots remain offline, cross-artifact operations: operators quiesce
all writers externally. SQLite's live transactional backup capability does not
make chain/DB/logs one distributed transaction. The snapshot manifest records a
capture boundary for every artifact:

- Chain: `committed_file_under_writer_lock`. The offline executor takes the same
  canonical destination and OS lock used by mining, NFT commits, autosave and
  shutdown persistence. It copies the latest complete disk commit, not newer RAM
  state. The chain is a full-file replacement, not an append-only stream.
- SQLite: `sqlite_backup_transaction`. Python's native SQLite backup API includes
  committed WAL data and integrity-checks the database. Checkpointing/copying a
  lone live `.db` file is not used. Logical ledger roles sharing a source reuse
  the same captured DB generation.
- JSONL: `closed_segment`. Source plans must explicitly set
  `rotationBoundary: "closed_segment"`. Rotate/close the producer segment before
  snapshot; do not point at an actively appended segment. The copy verifies the
  source remains unchanged and each nonempty line is valid JSON. Maximum JSONL
  segment size is 16 MiB; larger histories must be rotated/inventoried separately.
- Other JSON/config/wallet sources: `quiesced_stable_file`, with change/hash checks.

## Protected V2 backups

All required artifacts, including DBs, configs, logs, ledgers and public custody
records, require encryption. The default runbook no longer permits a plaintext
artifact opt-out. `manifest.json` is an encrypted envelope, not a plaintext hash
list. The executor accepts authenticated V2 snapshots only.

PBKDF2-HMAC-SHA256 with 600,000 iterations and a fresh 32-byte snapshot salt
derives a backup master key from the securely supplied custody passphrase. This
is domain-separated from the inner wallet envelope. Purpose-specific HMAC keys
separate manifest encryption, manifest authentication and every artifact. The
encrypted manifest is authenticated with HMAC-SHA256 before decryption. Artifact
frames use AES-256-GCM with unique counter nonces under per-artifact keys; AAD
binds artifact name, frame index and length. An authenticated EOF prevents prefix
truncation, and frame checks reject append/reorder/substitution. New snapshot
salts prevent cross-snapshot ciphertext swaps. Plaintext sizes and SHA256 hashes
are authenticated inside the manifest and checked after decryption.

The trust anchor is a strong passphrase in independent custody outside backup
storage. A user who can rewrite artifacts and the manifest but lacks this secret
cannot forge authentication. Do not put the secret in arguments, source control,
the snapshot or a custody-reference document. AES/HMAC authentication does not
provide public-key signer identity, protect a compromised custody secret, or
prove which valid historical snapshot is newest. Pin the expected manifest hash
and network/genesis through an independent trusted channel to prevent rollback.
The restore journal remains an offline selection record; preserve its latest
digest independently when operational anti-rollback assurance is required.

Every verify/restore needs `--scratch C:\QC_PRIVATE_ENCRYPTED\scratch`, an existing
operator-controlled directory on protected/encrypted storage. The tool rejects
an implicit scratch path. SQLite verification and trusted chain replay need
plaintext working files. Temporary scratch files are private and cleaned on
normal completion/failure; crash remnants need operator inspection. The tool
checks paths/permissions, not whether the underlying volume is encrypted. Restrict
Windows NTFS ACLs and use encrypted storage; mode 0600 alone is not an ACL policy.

`dry-run` and restore generations preserve encrypted backup bytes and verify
decryption. To obtain usable offline files, use `materialize --offline` into a
NEW private directory, then run the existing trusted-context chain validator and
application DB checks. The inner wallet envelope remains encrypted until the
separate `recover-wallet --offline` command creates a NEW wallet file. Neither
operation copies into live paths or starts services. Never commit materialized
files or include them in a new backup as an unprotected snapshot.

Plaintext V1 backups are not silently relabelled V2. Preserve historical copies
offline; inspect and re-inventory their original quiesced sources, create a V2
snapshot, verify it and retain provenance. There is no automatic deletion or
in-place migration of historical backups.

## One gameplay authority

The authenticated Telegram backend, SQLite session/evidence flow and TGWT ledger
are canonical. Its leaderboard is `/api/leaderboard/top` on that service. The
root score and leaderboard registrations have been removed. Writable mode
uses the generic API JSON 404 boundary and observer mode uses its capability
boundary. See [endpoint ownership](ENDPOINT_OWNERSHIP.md).

The Go `game` signed-score library remains an offline compatibility projection
with signature, scope, persistence, sequence and replay tests. It is not a second
runtime game economy. Existing caches are preserved, not imported into Telegram
balances. `AddScore`, `HandleTelegramScore`, `GiveReward` and
`EvaluateAndReward` return `ErrLegacyGameDisabled`. Positive/negative deltas do
nothing, and no fake payment is printed. Historical milestone constants are not
reward policy. There is no milestone claim state to consume for nonexistent
rewards. A real future milestone feature belongs in canonical verified task and
ledger accounting; this work creates no QC/TGWT/NFT payout or new reward rate.

## Regression evidence

The test suite combines existing actual-root auth/monitoring routes, loopback P2P
handshake and validated-height observations, multi-process log writers,
writer-termination recovery, corrupted-log preservation and network separation
with new capture/encryption/game retirement tests. New tests include concurrent
SQLite transactions during backup, an OS-locked commit in another process,
encrypted artifact+manifest tampering, truncation/substitution, trusted chain
replay after materialization, wallet recovery, unsigned concurrent deltas, stable
leaderboard ties, and repeat milestone rejection. All use synthetic data/keys.
This is technical regression coverage, not live mainnet recovery acceptance.
