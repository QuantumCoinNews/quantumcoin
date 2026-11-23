# QuantumCoin testnet

Use the canonical node config and a dedicated testnet data directory. The
repository's [release profile](../release/config.testnet.json) selects
`chain_data_testnet.dat`, the 18082 HTTP port, the 13001 P2P port and base
difficulty 8 and the `network-v4` genesis/signing profile. Use a new directory
for an independent testnet; existing chain data is never automatically migrated.

Both testnet and mainnet will start fresh. Historical balances are not carried
over. See [fresh network setup](../docs/FRESH_NETWORK_START.md) for separate
directories, the old-data inventory and the mainnet preparation boundary.

From the repository root, the following starts the read-only local observer:

```powershell
$env:QC_NETWORK = 'testnet-local'
$env:QC_NET_MAGIC = 'QC_TESTNET_V1'
$env:QC_API_LISTEN = '127.0.0.1:18082'
$testnetData = Join-Path $env:LOCALAPPDATA 'QuantumCoin/networks/testnet-v4'
$env:QC_DATA_DIR = $testnetData
$env:QC_CONFIG_FILE = (Resolve-Path .\release\config.testnet.json).Path
go run . --config .\release\config.testnet.json --data-dir $testnetData api
```

Check `http://127.0.0.1:18082/api/health` and `/api/network/label` for the
selected network and genesis. Use the [CLI command guide](../docs/CLI_PATHS_AND_OFFLINE_COMMANDS.md)
for writable node startup. Public testnet requires an explicit public
profile and the configured authentication and origin boundaries.

Launch the native wallet from the same configured environment when signing for
this testnet. Its local signer inherits `QC_CONFIG_FILE` and `QC_DATA_DIR`, so
it uses the same trusted genesis as the node. A signer using an older directory
or another network produces a signature that the V4 node will reject.

The adjacent `genesis.json` contains the verified code-derived fingerprint of
the fresh V4 testnet, with zero allocated QC. It is not a running chain or
evidence of a live launch. Faucet and peer files remain reference artifacts.
The node derives genesis from its validated runtime profile, rejects prefund
addresses or consensus-input drift, and keeps extra bonus minting disabled.
See the [QC bonus review](../docs/QC_BONUS_RESEARCH.md) for the distinction
between mined QC, funded future bonus transfers and separate TGWT rewards. A fresh
[network-v4 profile](../docs/NETWORK_GENESIS_AND_EMISSION.md) has its own
genesis, requires a dedicated directory and binds every spend signature to that
genesis. Public testnet startup rejects the older signing profiles. Keep an
old directory with its original config for local inspection; no balance or
chain migration is performed by changing this setting.

For an older testnet originally created with an empty genesis profile,
`release/config.testnet.legacy-local.example.json` retains the previous template.
Use it only with `QC_NETWORK=testnet-local`, the original data directory and
the original allocation/configuration inputs. V2/V3 chains instead need their
original matching profile. These compatibility profiles are rejected by public
testnet startup; they do not migrate balances into V4.
