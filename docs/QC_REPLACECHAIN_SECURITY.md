# QuantumCoin ReplaceChain Security

Status: QC_REPLACECHAIN_STRICT_VALIDATION_V1

Date: 20260617_192821

Peer-provided longer chains must pass strict validation before replacing the local chain.

## Active rule

P2P chain messages must call:

- blockchain.ReplaceChainFromPeerStrictV1

## Checks

- local and incoming chain must be non-empty
- incoming chain must be strictly longer
- incoming genesis hash must match local genesis hash
- each block must be non-nil
- each block index must match its position
- each non-genesis block must pass basic block validation
- each non-genesis block PrevHash must match the previous block Hash
- each non-genesis block stored Hash must match computed PoW hash
- each non-genesis block PoW must validate
- each block must have exactly one coinbase transaction
- coinbase must be the first transaction
- coinbase outputs must be positive
- total minted amount must not exceed TotalSupply
- normal transactions must verify without panic
- normal transaction spends must be valid against the incoming chain UTXO state
- duplicate spends inside the same replacement block are rejected

## Safety boundary

This layer does not mint QC, change total supply, change rewards, or use AI to select a chain.

It only rejects unsafe peer-supplied replacement chains before the canonical ReplaceChain path.
