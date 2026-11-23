# Source and runtime ownership

Source membership has one authority: `source_hygiene.working_source_inventory`.
Both the isolated Go/clean-stack snapshot and dependency-security snapshot use
it. A local working snapshot is review/test evidence, not a committed release.

| Owner | Source/build inputs | Working data and custody |
| --- | --- | --- |
| Go node | Root Go package, imported packages, reviewed profiles and config examples | Explicit `--data-dir` / `QC_DATA_DIR`; config-selected chain/wallet paths. Fresh testnet/mainnet workspaces use separate external directories via `scripts/network_workspaces.py`. |
| Wallet | Canonical wallet/signer code and recovery UI | User-selected wallet store and recovery custody; never included in a source snapshot. |
| Telegram Game | Python backend, browser controllers, hashed requirements | `QC_GAME_DB` or profile-specific `QC_GAME_DATA_DIR`; default Windows location is `%LOCALAPPDATA%/QuantumCoin/telegram-game/<profile>/state.sqlite3`. Legacy DB detection requires explicit selection; there is no automatic replacement. |
| Autopublisher | TypeScript, public configuration, editorial topic input, npm lock | `QC_PUBLISHER_STATE_DIR`, default `%LOCALAPPDATA%/QuantumCoin/autopublisher`; ledger, renders, media pool, logs and credential inputs remain local. |
| Browser applications | `webwallet/src` and its build inputs; canonical `webui/dist` operator files | `webwallet/dist`, publisher `dist/.build`, generated wallet WASM and dependency trees are build outputs. `webui/dist` is intentionally maintained source embedded by Go; a blanket `dist` exclusion is incorrect. |
| Release | Reviewed release scripts/templates, builder, verifier | Candidate binaries, signed package, runtime data and backups are separate external directories. The trusted signed manifest defines the exact distributable payload. |
| Operations/audit | Maintained tools and public runbooks | `_local_audit_reports`, caches, local environment files, machine metadata and historical custody archives are excluded from source. |

`config.json` at the checkout root is local configuration. Public defaults live
in `config.json.example` and reviewed `configs/` profiles. `.env.example`,
`package-lock.json`, `go.sum` and `requirements.lock` are source inputs; actual
environment/credential files and runtime lock files are not.

## Inventory and acceptance

```text
python scripts/source_hygiene.py --inventory
python scripts/source_hygiene.py --layout
python scripts/source_hygiene.py --cleanup-plan
python scripts/source_hygiene.py
```

`--layout` reads paths and filesystem metadata only, never runtime/credential
file contents. Large excluded directory trees appear as directory records, not
individual file totals. Its successful inventory result is separate from the
`sourceGate.ok` result: an inventory can succeed while index/HEAD are unsafe.

The last command validates the **real index and selected HEAD**. Ignoring a file,
deleting a working copy, or preparing an alternative index does not erase its
tracked/committed version. Source archiving and the clean-release CI gate must
continue to reject that state. Git history cleanup and secret rotation are
separate operations; path classification cannot prove that a source file has
no embedded secret.

## Existing installations

The established CLI caller-relative fallback remains supported. This change
does not silently select a new wallet/chain, change the working directory,
import old balances, stop services, or move/delete existing data. For separated
operation, explicitly use the prepared external network directory and config.
See [CLI paths](CLI_PATHS_AND_OFFLINE_COMMANDS.md),
[legacy custody](LEGACY_RUNTIME_CUSTODY.md), and
[source-clean standard](QC_SOURCE_CLEAN_STANDARD.md).

Local ignored `node_modules` and caches may be restored for development. They
are not source or release inputs. Old chain/wallet/DB material needs a reviewed
custody and migration decision before physical relocation; a generic filename
cleanup must never move a live wallet or SQLite database.

Legacy `bin/`, `build/`, `release/dist/` and `release_clean_test*` directory
summaries use `mixed-custody-review`: a pruned subtree may contain both old
binaries and wallet/chain data. This is a review category, never permission to
delete the directory. Inventory descendants, verify active references and
recovery custody before selecting individual paths for relocation. Audit and
diagnostic snapshots remain archives; they are not current build evidence.

Working-source line endings follow `.gitattributes`: script/web/Go/JSON sources,
public examples and Go module files use LF; PowerShell and batch files use CRLF.
Check the entire working diff with `git diff --check`. Normalize only reviewed
source files, preserve a local pre-change copy, and compare content after
removing line-ending differences. Never apply text normalization to runtime or
secret files. This does not stage changes or erase committed history.
