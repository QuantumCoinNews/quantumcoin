# QuantumCoin Source Clean Standard

This project separates source code, runtime state, release artifacts, secrets, and audit outputs.

See [source/runtime ownership and inventory](SOURCE_RUNTIME_LAYOUT.md) for the
canonical per-service paths, maintained embedded assets and legacy path rules.
Use `python scripts/source_hygiene.py --layout` for a metadata-only placement
inventory; its source-gate result remains separate from inventory success.

## Source may contain

- Go source files
- Python source files
- TypeScript/JavaScript source files
- Config examples
- Documentation
- Test files
- Safe scripts

## Source must not contain

- Wallet private keys
- Recovery words
- Real .env files
- OAuth tokens
- Bot tokens
- Chain runtime data
- Local DB files
- Cache files
- Logs
- Old executable backups
- Generated release packages
- Testnet backup state
- Media render outputs

## Release package rule

A user release package must be assembled separately from source and must contain only approved runtime files.

The signed manifest defines the exact artifact set. The candidate includes one
node, one Fyne wallet, one browser launcher, two matching miner client scripts,
and the SBOM and provenance metadata. See [Windows release provenance](RELEASE_PROVENANCE.md).

Build into a new directory outside the source tree. Use an explicit external
runtime data directory for each network profile. The canonical launcher reads
the verified package and a separate data directory. See
[CLI path resolution](CLI_PATHS_AND_OFFLINE_COMMANDS.md).

Run `python scripts/source_hygiene.py` to inspect the index and HEAD. The source
archive command validates both before archiving an immutable commit. Review
already tracked paths and staged removals before producing a source package.
The guard checks path policy and applies Git's committed export attributes.
Use the staged-blob checks as well when reviewing credentials in source changes.
