# QuantumCoin Peer Transaction Validation

Status: QC_PEER_TRANSACTION_VALIDATION_V1

Date: 20260617_190932

Peer-provided transactions must pass explicit strict validation before they enter the canonical blockchain mempool path.

## Active rule

P2P transaction messages must call:

- blockchain.AddTransactionFromPeerStrictV1

## Checks

- non-nil blockchain and transaction
- basic transaction structure validation
- peer coinbase transactions are rejected
- transaction ID must exist
- normal transaction must have inputs and outputs
- transaction signature must verify
- UTXO set must be available or rebuilt
- referenced UTXOs must exist
- spent / empty UTXOs are rejected
- coinbase maturity must be respected
- input PubKey must unlock the referenced UTXO
- duplicate input spending inside the transaction is rejected
- pending mempool input conflicts are rejected by the canonical AddTransaction path

## Safety boundary

This validation layer does not mint QC, change balances directly, change total supply, or change mining reward logic.

It only rejects unsafe peer-supplied transactions before mempool acceptance.

## QC_10B repair: panic-safe peer Verify

Date: 20260617_191241

Peer transaction verification is now panic-safe.

Malformed peer fields, including invalid Base58 sender data that previously could panic during tx.Verify, are converted into reject errors.

Rule:
- peer input must never crash the node
- malformed peer tx must be rejected
- canonical AddTransaction still performs final validation again
