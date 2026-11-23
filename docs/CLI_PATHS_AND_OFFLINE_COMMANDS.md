# CLI paths and offline commands

The root executable preserves the caller's working directory. It never changes
to its executable directory. This makes `go run . help`, installed binaries and
service launches use the same explicit path rules rather than a temporary Go
build directory or installation directory.

Global path options precede the command:

```text
quantumcoin [--data-dir DIR] [--config FILE] COMMAND [arguments]
quantumcoin --data-dir C:\QuantumCoinData run
quantumcoin --config C:\QuantumCoinConfig\node.json --data-dir C:\QuantumCoinData api
quantumcoin --data-dir C:\QuantumCoinData --wallet-file wallets\owner.json newaddr
quantumcoin help
go run . help
```

## Resolution and migration

| Input | Resolution |
| --- | --- |
| Data directory | `--data-dir`, then `QC_DATA_DIR`, then caller's working directory. Relative directories are caller-relative and converted to absolute paths. |
| Node config | `--config`, then `QC_CONFIG_FILE`, then `config.json` in the selected data directory. Explicit relative config paths are caller-relative. |
| Node chain, bonus and wallet storage | Existing config/default values with existing environment precedence (`QC_CHAIN_FILE`, `QC_BONUS_FILE`, `QC_WALLET_FILE`). Relative storage paths are anchored to the selected data directory; absolute paths are preserved. |
| Offline wallet creation | `--wallet-file`, then `QC_WALLET_FILE`, then `wallet_data.json`, anchored to the selected data directory unless absolute. |

Path resolution itself creates no directories or files. Loading config publishes
the same resolved storage paths to both the runtime and wallet package. Node
configuration validation, network separation and mandatory trusted chain
validation remain enforced before starting services. Existing library callers
that omit `BootstrapOptions.DataDir` keep their previous path semantics.

There is no automatic migration, executable-directory fallback or silent choice
between multiple chain/wallet files. For an existing portable installation,
specify its directory with `--data-dir` or set the service working directory
there. For a separated installation, explicitly choose the data directory and
config path. Check existing config and environment overrides before switching.
No new platform user-data directory is selected automatically.

`--data-dir` anchors the config-backed storage files and legacy miner-address
lookup. It does not relocate every auxiliary log/cache path in other packages;
those retain their own configuration or caller-relative behavior. Detached
consensus release artifacts remain bound to the running executable and compiled
trust root; a data directory cannot replace their authorities.

## Commands that do not bootstrap a node

`help`, `--help`, `-h`, unknown commands, unknown global options and malformed
command arity return before config/chain loading or saving. No arguments still
means explicit normal node startup without implicit mining, as previously
documented. Global options must come before the command; signing's
`--key-stdin` remains a command-specific trailing option.

`newaddr` and `newaddr-priv` create and save a local wallet without reading node
config, chain or release manifests. In particular, they do not obtain
`wallet_file` from config.json: use `--wallet-file` or `QC_WALLET_FILE` when the
node uses a custom store. The selected absolute save path is printed. A failed
save does not report a new address or private key and does not overwrite a
malformed existing store. `newaddr-priv` additionally prints the private key
only after successful persistence, because the command explicitly requests
export. Store backups and filesystem access controls remain the operator's
responsibility.

`--wallet-file` is restricted to these offline creation commands. Node and
signing commands use the node's configured wallet store. Local transaction
signing still requires the trusted chain to determine spendable outputs and
active protocol rules.

The dispatcher has no blanket final chain save. Read-only `print`, help,
invalid commands and wallet creation never save the chain. Mining keeps its
explicit persistence, and service modes retain their autosave/shutdown
contracts. CLI send admission is still a local, non-durable mempool result.

Regression tests run the actual dispatcher in child processes with broken
config/chain files, verify the working directory remains unchanged, restore
generated wallet keys, and check that invalid/help commands leave files intact.
