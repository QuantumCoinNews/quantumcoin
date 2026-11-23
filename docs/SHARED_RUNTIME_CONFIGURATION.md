# Shared runtime configuration

The declaration source is [runtime-environment.schema.json](../config/runtime-environment.schema.json).
[The generated catalog](runtime-environment-catalog.json) maps each key to its
semantic type, owning service, alias, lifecycle, secret classification and current
source references. It contains no process values or credentials. It includes
test-only controls and explicitly ignored historical CMD markers; the number of
entries is not a count of enabled product features.

Environment values are strings. Project validators trim for shape checks and use
the schema's `x-kind`, bounds, enum/case rules, owners and alias groups. These
extensions are normative for semantic/cross-field checks; a generic JSON Schema
validator alone does not implement alias conflicts, network rules or startup
policy. Literal `boolean-01` consumers require exact `0`/`1`, without whitespace.
The schema does not replace native path/provider validation or Go consensus.

| Owner | Runtime boundary | Precedence |
| --- | --- | --- |
| Node | `config.Load` validates before publishing configuration; canonical node bootstrap also validates callers supplying Config directly, before opening chain state. | Go defaults → explicit JSON file → environment → caller-selected data directory anchoring → typed config, effective bind and existing runtime policy checks. CLI selects file/data paths. |
| Telegram Game | `telegram_game.config` validates after optional dotenv loading and before canonical DB selection/ASGI composition. | Inherited environment wins; optional game `.env` fills only absent keys. Bot aliases and node-origin aliases must agree. One dynamic network/origin resolver serves every getter. |
| Autopublisher | `configuration` validates the resolved settings before work/provider execution. | Process environment → explicitly selected env/encrypted secret source → whitelisted public settings → explicit CLI overrides. Plaintext credential files remain forbidden for live publish. Credential import is separate from publishing. |
| PowerShell/CMD | Supported wrappers delegate to canonical node/miner/publisher commands. Deployment tooling can invoke the preflight below. | Wrappers do not become another config/consensus authority. Existing service validators still enforce the effective network/genesis and file policy. |

## Rejected combinations

- Invalid numeric/boolean/enum spelling, integer overflow, malformed origin or
  bind, relative paths for the explicitly absolute game/publisher settings.
  Node chain/wallet paths retain their existing caller/data-directory anchoring.
- Disagreeing aliases, contradictory local/testnet/mainnet signals and magic,
  testnet bridge origins using main/local API ports without the existing explicit
  compatibility assertion, and main/local origins using the testnet port.
- An explicit HTTP bind that disagrees with the effective config port; HTTP_PORT
  is an assertion of QC_HTTP_PORT, not a second override.
- Non-loopback binds without a token, obvious template tokens on those binds,
  remote plaintext bridge origins, or publisher monitor origins outside numeric
  loopback HTTP / without their separate credential.
- Retired alternate TGWT database selection. Go JSON files additionally reject
  duplicate keys at any nesting level, non-object roots and the existing unknown
  fields/trailing-data cases. Invalid config is not published.
- Unknown `QC_` keys. `QC_TEST_` is reserved for isolated test controllers; the
  schema declares other known test controls explicitly. Other OS namespaces are
  left alone. Old CMD markers such as QC_OVERWRITE_RUN_MINER are explicitly
  ignored and cannot revive overwrite/process-launch behavior. Tests prevent
  active readers for those historical markers from being reintroduced silently.

Inactive service settings are not mistaken for that service's own settings.
Known publisher settings therefore do not become node economic policy. Secrets
are never defaults in the schema and validation diagnostics contain key names
and fixed codes, not their values. A passed shape check does not establish valid
credentials, secret rotation, wallet custody or release readiness.

## Verification

Using the locked Python environment:

```text
python scripts/runtime_config_preflight.py --service tooling
python scripts/export_runtime_environment.py --check
python -m unittest discover -s scripts -p test_shared_runtime_environment.py -v
go test -race ./config -count=1
```

`--allow-unknown` is an explicit tooling compatibility switch, not a production
runtime bypass. Source changes consuming a new setting must add a reviewed
declaration and regenerate the catalog; the clean-stack gate checks for drift.

Go, Python and TypeScript run the same independently specified
[conformance cases](../config/runtime-environment.cases.json). They cover valid
settings, alias/network conflicts, numerical edge cases, origin/bind handling,
service isolation and ignored/unknown controls. Tests use generated credentials
and temporary paths, never operational DBs or payouts.

The publisher build manifest binds the shared schema hash. Changing only the
schema invalidates a previously built publisher before compiled code runs;
rebuild using the existing publisher build command. Source bundles must retain
the repository-level `config` directory; an isolated copied publisher directory
without its canonical schema fails closed. Runtime files remain outside source.

Final mainnet genesis/economics, NFT assets, BSC verification, live social/provider
acceptance, secret rotation, signed release and hosted CI are their separate open
gates. This schema does not enable them or change block reward, supply or PoW.
