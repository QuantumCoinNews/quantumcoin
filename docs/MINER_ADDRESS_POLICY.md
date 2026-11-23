# Miner address selection

The node and native wallet's visible-miner launcher use
`wallet.ResolveMinerAddress`. Selection reads existing public address sources;
it never creates a key, rewrites an address file, opens the configured private
wallet store or signs a transaction. Selecting an address does not prove key
ownership. Back up the owning wallet before mining.

## Priority

1. An explicit operation address, including the native launcher's explicit
   `ADDR` override. CLI mining continues to require an address argument.
2. The loaded `Config.Miner.Address`. Node configuration loads `miner.address`
   first and applies `QC_MINER` last. The chosen value/source is captured at
   startup. Existing `Miner` / `Address` JSON casing remains accepted.
3. `miner_address.txt` in the selected data directory.
4. `wallet_address.txt` in that directory, for existing portable clients.
5. The `address` field in the legacy `%APPDATA%/QuantumCoin/wallet.json` record.

Each selected value must pass QuantumCoin Base58Check validation. An invalid,
empty, oversized or unreadable existing candidate fails resolution; it does
not silently redirect rewards to another wallet. Missing legacy files permit
the next fallback. Text files are bounded to 256 bytes and the legacy JSON
record to 4 KiB. A missing address returns an actionable error.

Use explicit configuration for predictable deployments:

```json
{
  "miner": {
    "address": "YOUR_VALID_QUANTUMCOIN_ADDRESS"
  }
}
```

The placeholder above must be replaced with a valid address. `QC_MINER` takes
precedence over this field. A node already running keeps its loaded value;
editing config.json or changing process environment does not hot-switch its
configured reward recipient. Restart to apply config changes. Legacy address
files are read when a fallback is requested, so changes to those files affect
later fallback resolutions.

`wallet_file` / `QC_WALLET_FILE` select the private wallet store used by local
wallet/signing commands. Miner selection does not inspect that store. After
creating a wallet, explicitly configure its address or export its public
address through the existing portable wallet flow.

The native launcher honors `QC_CONFIG_FILE` and `QC_DATA_DIR`; relative values
are resolved against the release directory, matching its child process's
working directory. Its selected address is passed explicitly to the miner.
An explicit operation override does not rewrite the default config or wallet
address. Use miner status for the running miner's actual address.

`/api/wallet/address` and the miner-start fallback share the same default
selection and propagate errors. The former never generates a wallet. Legacy
internal compatibility wrappers delegate to this policy. The old loose
config.json reread and duplicate environment selection have been removed.

The visible launcher no longer performs redundant address-file writes during
startup. Wallet creation and explicit backup/export operations retain their
own checked persistence contracts. See [CLI paths](CLI_PATHS_AND_OFFLINE_COMMANDS.md)
and [local transaction signing](CLI_SIGNED_TRANSACTIONS.md).
