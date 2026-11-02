# QuantumCoin Peer Block Validation

Status: QC_PEER_BLOCK_VALIDATION_V2

Date: 20260617_185947

Peer-provided blocks must pass strict validation before append.

## Active rule

P2P block messages must call:

- blockchain.AddBlockFromPeerStrictV2

## Checks

- non-nil blockchain and block
- canonical next block index
- previous hash matches local best block
- timestamp does not go backwards
- block difficulty must be within allowed peer range
- stored block Hash must match computed PoW hash for the block data and nonce
- PoW must validate
- exactly one coinbase transaction
- coinbase must be first transaction
- coinbase reward must not exceed current reward
- total supply cap must not be exceeded
- normal transaction signatures must validate
- normal transaction inputs must exist in the current UTXO set
- coinbase maturity must be respected
- input pubkey must unlock the referenced UTXO
- duplicate input spending inside the same block is rejected

## Safety boundary

This validation layer does not mint QC, change balances directly, change total supply, or change mining reward logic.

It only rejects unsafe peer-supplied blocks before append.
