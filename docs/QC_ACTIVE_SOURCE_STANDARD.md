# QuantumCoin active source standard

Document ID: `QC_ACTIVE_SOURCE_STANDARD_V1`\
Evidence stage: **Source verification**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

| Classification | Current scope |
| --- | --- |
| Node runtime | The root Go package and its transitive imports. The actual HTTP routes are registered in `node_app_router.go`. |
| Embedded node WebUI | The explicit file list in `webui/embed.go` and its Go handler. The landing and operator miner ship with the node. |
| Native GUI and wallet | `cmd/quantumcoin-gui` and `cmd/wallet` are separate desktop targets. `cmd/gui-smoke` is an optional diagnostic target. |
| Separate browser application | `webwallet/src`, the HTML/Vite entrypoint and the npm manifest/lockfile build a read-only balance viewer. The resulting dist is generated. |
| Separate services | The Telegram Game uses its documented Python entrypoint and data directory. Autopublisher uses its own documented command, configuration and package lock. |
| Verification and tooling | Go `*_test.go`, actual Python and Node test entrypoints, `scripts/`, and audit/release `tools/` are maintained for their respective checks. |
| Retired and historical | Root `src/` was an orphan game prototype. The old `gui/`, `ui/` and binary-only `qc-desktop/` layouts are retired. The old i18n loader and three top-level demonstration scripts have `.go.disabled` suffixes. The language JSON files are historical references. |
| Generated and local runtime | `bin/`, `node_modules/`, `webwallet/dist/`, packaged executables, logs, chain data and databases are local outputs or release artifacts. |
| Documentation | `docs/` is maintained reference material. Policy evidence and deployment boundaries are tracked separately in [the policy register](POLICY_EVIDENCE_REGISTER.md). |

Use `go list -deps .` to inspect the node import graph and `go list ./...` to
inspect buildable Go packages. Package presence is separate from a runtime import.
Use each service's launch command and router to verify its active entrypoints.
Actual faucet policy, testnet profile and transaction signature regressions
live in the `config` and `blockchain` Go test packages.

Store actual environment files, wallet keys, recovery words, API credentials and
chain/database files in local custody. Example environment templates contain
placeholders. Release candidates require a manifest verified against an external
trusted signer and the matching clean source commit.

Historical runtime directories include `old_working_release/`,
`_legacy_runtime_*/`, `_wm_review/`, and local checkpoints. Keep them outside
source/release inventories. See [legacy runtime custody](LEGACY_RUNTIME_CUSTODY.md)
for snapshot and recovery handling.
