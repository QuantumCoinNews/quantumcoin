# Mandatory chain validation at startup

Canonical node startup, observer API startup and the standalone local miner
validate the complete chain before returning a runtime or starting services.
Validation is mandatory for loaded history and newly created genesis.

`QC_STRICT_CHAIN` no longer controls whether startup validates history. Empty,
`0` and `1` all retain full validation. `BootstrapOptions.StrictChain` and the
boolean argument of `BootstrapWithConfig` remain for source compatibility;
their false value cannot disable the safety boundary. Default options and
production command adapters explicitly request validation.

Startup now waits for complete history validation before services become ready;
the work grows with chain history. There is no environment shortcut around it.

## Admission sequence

1. Validate canonical configuration/network policies and verified activation.
2. Create the trusted local genesis and bind local activation/maturity before
   decoding persisted history. For NFT activation, also verify and bind the
   local authorities and monthly program before loading the file.
3. Use `LoadBlockchainFromFileWithLocalContextV1` for every persisted runtime
   chain. Reject decode failures, trailing data, supply differences, genesis
   differences and invalid canonical block replay. Replay includes block
   structure, parent links, difficulty/PoW, transactions/UTXO, reward/supply,
   vesting and active NFT/consensus requirements.
4. Rebuild UTXO and transaction indexes from validated history. Serialized
   derived state is not an authority for balances.
5. Run `ValidateBlockchainSynchronized` unconditionally before `NewRuntime`.

The old context-free `LoadBlockchainFromFile` compatibility function is no
longer used by node startup. Its decoder behavior cannot authorize a runtime.
Observer reload continues through the same trusted-context loader.

## Failure behavior

An existing invalid or unreadable chain causes an error. Startup publishes no
runtime, opens no HTTP/P2P listener and does not overwrite/reset the chain.
Only a genuinely missing file permits a new local genesis; this remains in
memory until the existing explicit persistence lifecycle writes it.

A rejected reload preserves the healthy runtime and does not mark the bad file
as successfully loaded. A later valid repair can be retried. Validation does
not automatically repair, delete, truncate or replace evidence on disk.

This is a local integrity and consensus admission check. It is not a freshness
oracle: a valid historical prefix can still be behind the network. There is
no new rollback-selection or peer synchronization policy in this change.

## Regression coverage

`node/mandatory_chain_startup_validation_test.go` covers malformed/truncated gob,
empty history, altered genesis, supply, block hash, parent, height, difficulty,
transactions and coinbase; garbage/second objects after the encoded chain;
derived-state forgery; mandatory validation across legacy flag values; fresh
genesis; rejected reload preservation/retry; and signed NFT history with the
strict option disabled.

An architecture contract also requires an unconditional validation call before
runtime construction and the trusted-context loader in the startup path.
Existing hardfork, NFT authorization, persistence, miner, observer and HTTP
authorization suites remain applicable.
