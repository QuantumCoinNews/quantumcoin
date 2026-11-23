# QuantumCoin offline backup and recovery

The Go `backuprestore` package defines and validates the read-only recovery
contract. `backuprestore/offline` and `cmd/recovery` implement explicit offline
operations. No HTTP endpoint or node startup path invokes the executor. The
admin runbook view does not grant permission to restore a live service.

## Preparation and required coverage

Stop/quiesce the node, miners, game backend, bot and workers externally. The tool
requires an explicit offline declaration; it does not detect or stop all writers.
SQLite backup includes committed WAL pages, but a cross-file snapshot requires
all writers to remain quiesced. Use a private, access-controlled directory outside
Git, with distinct local-release/testnet/mainnet roots. On Windows restrict NTFS
ACLs to the operator; Unix mode 0600 is not a Windows ACL guarantee.

A valid runbook must include all nine roles with the correct kinds:

- quantumcoin_chain_file
- telegram_game_sqlite_db
- tgwt_ledger
- monthly_nft_review_log
- admin_event_log
- monitoring_event_log
- runtime_config_examples
- wallet_encrypted_backup (wallet_key, encryption mandatory)
- wallet_custody_record (config, public references only)

Copy `configs/recovery.sources.example.json` outside the repository, set real
absolute paths, independently pin the expected genesis SHA256, select the
network, then set offline=true only after quiescing. The example deliberately
fails as supplied. Paths are operator inputs, not guessed from the working
folder. Copy `configs/wallet-custody.example.json` outside Git and record every
public address, custodian/procedure references, and the actual recovery-drill UTC.
Never place a password, key or seed in that record.

The TGWT and NFT logical ledger roles can reference the same SQLite source as
the game DB. The executor captures that source once, then makes identical copies;
it does not invent JSON ledger exports. Legacy artifact kind names describe roles;
the manifest `format` records the actual storage encoding. Log roles accept JSON
state stores or JSONL. Runtime examples use the separated security event JSON and
monitoring state JSON; confirm actual configured paths. If a required source is
absent, inventory and resolve it explicitly. Do not create fake history to pass.

## Build and password input

```powershell
go build -o C:\QC_PROJECT_EXTERNAL_BACKUPS\recovery.exe ./cmd/recovery
```

Pass the wallet password through a secret manager or a securely prompted pipe
with `--password-stdin`. The tool refuses echoed terminal input, command-line
passwords, and passwords shorter than 12 bytes. Use a strong unique passphrase;
length alone does not guarantee entropy. Do not save it in shell history or a
report. In the commands below `<secure-password-producer>` denotes your chosen
secure input process; it is not a literal command or a plaintext password.
The tool never writes private material to stdout. All snapshot, verify and restore commands require `--scratch`; standalone seal/unseal accept it but do not need scratch. Commands read the password once
from stdin, optionally stripping one terminal CRLF/LF.

## Encrypt and rehearse wallet recovery

Older Windows wallet runtimes may contain `wallet_priv.hex`,
`wallet_recovery_words.txt` and `wallet_recovery_manifest.json`. Treat these
as custody files. Restrict their Windows ACL to the wallet owner and SYSTEM and
verify the permissions after replacing a file. Keep the runtime separate from
the signed distribution package. For independent recovery, preserve the wallet
and seed using a protected offline backup, then test recovery with the public
address before retiring the legacy files. The current JSON envelope command
expects the formats below. The legacy three-file layout needs its own custody
migration.

The GUI can import its existing encrypted `.qcbak` into a separate encrypted
legacy profile without recreating the three plaintext files or replacing an
existing key. It verifies private-key/address consistency and preserves the
encrypted legacy notes. See [QC wallet recovery](QC_WALLET_RECOVERY.md). This GUI
flow does not extend the offline JSON `seal-wallet` input formats listed below.

Supported inputs are current QC `wallet_data.json` (`wallets` plus `default`) and
operator `wallet.json` (`address` plus `privateKeyHex`) with raw 32-byte secp256k1
keys. Keys must derive the claimed public addresses. A wallet containing only an
address is not recoverable and is rejected. Other seed/hardware/browser formats
need their own custody procedures; the tool does not claim to convert them.

```text
<secure-password-producer> | recovery seal-wallet --source C:\QC_PRIVATE\wallet_data.json --target C:\QC_OFFLINE_INPUT\testnet\operator.qcwallet --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
<secure-password-producer> | recovery unseal-wallet --source C:\QC_OFFLINE_INPUT\testnet\operator.qcwallet --target C:\QC_PRIVATE\NEW-recovered-wallet.json --offline --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
```

The envelope uses AES-256-GCM, PBKDF2-HMAC-SHA256 with 600,000 iterations, a random
16-byte salt and random 12-byte nonce. The version header is authenticated. Every
snapshot/verify/restore decrypts in memory and checks address coverage against
custody. A renamed plaintext wallet is rejected. `unseal-wallet` creates a new
0600 file and refuses replacement. Keep that private recovery file offline;
confirm wallet signing/balance against a trusted node before operational use.
Keep the passphrase in separate protected custody. Losing it loses recovery.
All persistent snapshot artifacts, including DB/config/logs/custody, are now
AES-256-GCM encrypted. The encrypted manifest additionally requires HMAC-SHA256
verification. The independent trust anchor is the custody passphrase, not a hash
stored beside the data. See [snapshot security and game authority](SNAPSHOT_SECURITY_AND_GAME_AUTHORITY.md)
for framing, key separation, capture boundaries and migration limits.

Verification requires an explicitly configured `--scratch` directory on private,
encrypted storage. Temporary plaintext is necessary for SQLite/replay checks;
ordinary cleanup does not promise secure erasure after a crash. The tool checks
the directory, not underlying volume encryption. Do not use public/default temp.


## Snapshot, verification and restore

```text
<secure-password-producer> | recovery snapshot --plan C:\QC_PRIVATE\testnet.recovery-plan.local.json --target C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\snapshot-NEW --python C:\PATH\TO\python.exe --offline --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
<secure-password-producer> | recovery verify --source C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\snapshot-NEW --network testnet --genesis EXPECTED_HASH --python C:\PATH\TO\python.exe --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
<secure-password-producer> | recovery dry-run --source C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\snapshot-NEW --target C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\rehearsal-NEW --network testnet --genesis EXPECTED_HASH --python C:\PATH\TO\python.exe --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
<secure-password-producer> | recovery restore --source C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\snapshot-NEW --target C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\offline-recovery --network testnet --genesis EXPECTED_HASH --python C:\PATH\TO\python.exe --offline --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
<secure-password-producer> | recovery current --target C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\offline-recovery --network testnet --genesis EXPECTED_HASH --python C:\PATH\TO\python.exe --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
<secure-password-producer> | recovery rollback --target C:\QC_PROJECT_EXTERNAL_BACKUPS\testnet\offline-recovery --network testnet --genesis EXPECTED_HASH --python C:\PATH\TO\python.exe --offline --scratch C:\QC_PRIVATE_ENCRYPTED\scratch --password-stdin
```

Use the same custody passphrase for generations in one workspace. Password
rotation needs a separately managed workspace/custody migration. Python 3's
standard `sqlite3.Connection.backup()` and `PRAGMA integrity_check` are invoked
without a shell; no third-party SQLite package is required. Failures are reported
without provider output or DB rows. Each operation has bounded files and SQLite
timeouts. Source-change checks cover copied files; they do not replace quiescing.

The authenticated encrypted manifest payload contains version/network/genesis, artifact names, sizes, formats and
SHA256 hashes, never private key values or source paths. Preserve its own SHA256
and custody verification evidence outside the backup through a trusted channel.
HMAC and authenticated encryption reject rewriting without the independently held custody password. Hashes alone remain insufficient; pin manifest identity/freshness out of band.
Network/genesis values are pinned inventory labels; before promotion use the
node's existing trusted-context full chain replay. Byte/hash verification is not
consensus validation, schema migration, or proof of mainnet origin.

A dry-run is an actual copy to a new isolated destination followed by verification.
Restore creates a new immutable `generation-*` directory, verifies it, then records
selection in a sequential hash-linked journal. It never changes live paths.
Rollback writes a new journal event selecting the previous verified generation;
newer generations remain intact. Rolling back the first restore selects none.
Consumers must use `current`, which validates the journal and selected generation.
An unfinished generation is not selected. Invalid/truncated journal records stop
recovery; they are never silently discarded. A stale `.offline-recovery-lock`
after a crash requires inspection and manual removal only when no operator is
running. The private workspace assumption excludes hostile same-account path
replacement. No service is automatically reconfigured, started or stopped.

Only after rehearsal, offline wallet recovery and the trusted startup validator
pass may an operator plan live promotion with the required downtime/custody
approval. This implementation does not activate mainnet, send funds, mint or pay
BSC withdrawals. Backup verification cannot replace separate-device disaster
recovery acceptance or offsite custody checks.

## Verification

```powershell
go run ./cmd/policy-export --check
go test ./backuprestore/... ./cmd/recovery ./policyexamples
```

SQLite tests require Python 3 (`python` on PATH, or set
`QC_RECOVERY_TEST_PYTHON` explicitly). They create synthetic databases and wallets,
including committed WAL data, and never open the real game DB or operator wallet.

## Protected V2 working-file recovery

V2 backups and restore generations contain encrypted artifact bytes, including
the manifest. Plaintext V1 snapshots are not accepted or automatically migrated.
Preserve and re-inventory older sources offline; create and verify a new V2 copy.
A closed JSONL source must declare `rotationBoundary: "closed_segment"` in the
source plan; chain `AppendOnlyPreferred` is false because persistence replaces
the whole file. Snapshot captures share the mining/autosave OS destination lock.

Use the explicit offline working-copy step before invoking trusted chain replay:

```text
<secure-password-producer> | recovery materialize --source SNAPSHOT --target NEW_PRIVATE_WORKING_DIRECTORY --network testnet --genesis EXPECTED_HASH --python PYTHON_EXE --scratch PRIVATE_ENCRYPTED_SCRATCH --offline --password-stdin
<secure-password-producer> | recovery recover-wallet --source SNAPSHOT --target NEW_PRIVATE_WALLET_FILE --network testnet --genesis EXPECTED_HASH --python PYTHON_EXE --scratch PRIVATE_ENCRYPTED_SCRATCH --offline --password-stdin
```

`materialize` creates verified plaintext working files in the new private target;
its inner wallet artifact retains wallet-envelope encryption. `recover-wallet`
creates a separate new usable wallet file after verifying the protected snapshot.
Never point either target at live data, publish the files, or mistake the working
directory for an authenticated encrypted backup. The separate `unseal-wallet`
command is for standalone inner wallet envelopes, not V2 snapshot artifacts.
