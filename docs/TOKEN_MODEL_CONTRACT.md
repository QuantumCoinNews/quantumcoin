# Fungible token model contract

The canonical library contract is an explicit model, its supply policy, a common
read interface and model-tagged persistence. Two implementations are intentional:

| Implementation | Model | Supply policy | Intended role |
| --- | --- | --- | --- |
| `Token` | `TOKEN` | `CONTROLLED_EMISSION` | Local controlled-emission prototype and optional miner-token bridge |
| `QC20Token` | `QC20` | `FIXED_SUPPLY` | Initial issuance to an owner; subsequent minting disabled |

Neither model is native QC. Native QC uses the existing blockchain UTXO,
consensus, PoW, reward and halving rules. QC721 collections and BSC TGWT are
separate assets and are not selected through this fungible contract.

## Model selection and persistence

- `FungibleTokenReadV1` exposes locked reads and defensive snapshots. It does not
  provide a generic mint/transfer method that could erase policy differences.
- Generic export/import uses `MarshalFungibleTokenV1` and
  `UnmarshalFungibleTokenV1`: version 1, explicit `model`, and model-specific
  `state`. The writer selects only supported concrete types and validates its
  detached state before export; the loader validates the model and schema.
- Field case, required scalar fields, nested duplicate keys and null scalar
  values are checked. A zero cap is explicit; a missing/null cap cannot silently
  become zero. Both loaders enforce their existing supply/balance invariants.
- Historical raw JSON remains available for callers that already know the
  concrete type. A raw state from the other family is rejected. Legacy empty
  maps may be absent/null; canonical envelope map fields must be present.
- `Registry` and the optional miner-token bridge store controlled-emission
  `Token` objects. They are not generic QC20 registries; QC20 is not converted to
  the bridge model. The existing registry JSON format is preserved. The bridge
  follows the shared [mining startup options](MINING_OPTIONS_CONTRACT.md).
- A fixed-supply token stays fixed after transfer, burn and reload. Burn does
  not reopen mint capacity. No automatic cross-model conversion is implemented.

## Trust boundary

Address inputs follow the shared [token address contract](TOKEN_ADDRESS_CONTRACT.md).
In particular, legacy QC20 construction with an invalid owner does not issue
initial supply; error-returning constructors are available for untrusted inputs.

Model labels are unsigned data, not authorization or consensus evidence. A
well-formed local JSON document does not prove authentic issuance, wallet
ownership or chain identity. Replacing an entire document with another valid
definition requires an authenticated registry/consensus design to detect.

The Token Factory remains inactive. Existing mint authorization and token
network/instance-domain gaps remain tracked separately. This contract does not
enable mining rewards, change supply policies or approve new consensus rules.
See [Token Factory status and completion plan](TOKEN_FACTORY_STATUS_AND_COMPLETION_PLAN.md).
