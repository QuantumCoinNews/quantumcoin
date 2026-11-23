# Token Factory: implementation status and completion plan

Reviewed on 15 November 2026. This document is a proposal and an inventory,
not an activation or a new consensus specification.

## Current implementation

The two fungible library implementations follow the explicit
[token model contract](TOKEN_MODEL_CONTRACT.md). This separates their existing
supply policies and persistence formats without activating the Factory.

There is a `token/` library, not a production Token Factory service. Root node,
native wallet, standalone miner, launcher and API test-command dependency graphs
do not import `quantumcoin/token`. The canonical endpoint inventory has no native
Token Factory creation/mint/transfer routes. `blockchain.Transaction` and the QC
block acceptance engine do not replay the library's token state transitions.

| Component | Implemented behavior | Boundary |
| --- | --- | --- |
| `token/token.go` | Initial allocations, cap, owner, balances; public post-creation mint is disabled | Optional miner-token bridge remains a separate prototype and can issue only when explicitly enabled |
| `token/qc20.go` | Fixed supply assigned to an owner at construction | Later minting is deliberately disabled; constructors are local object creation, not on-chain issuance |
| `token/qc721.go` | Initial collection items, ownership and metadata URI | Public mint methods are disabled after collection creation; constructors are local, not on-chain issuance |
| `token/authorization_*.go` | Signed transfer/burn, signed allowance/transfer-from, nonce replay checks | Scoped constructors bind network magic, genesis hash and a persisted instance ID; legacy/unscoped state cannot authorize a signed operation |
| `token/storage.go`, `fungible_persistence_v1.go` | Local JSON persistence and integrity validation | A local registry is not consensus-protected registration |
| `token/miner*.go` | Optional block-aware prototype reward bridge | Disabled by default, outside root runtime; local token records are not replicated token consensus |

Signed token self-transfers previously increased balances. The reviewed fix keeps
balances and total supply unchanged while consuming the authorization nonce;
delegated self-transfers also consume allowance. It does not create a new mint
policy, enable the Factory, or change QC economics.

Native QC transfers are a separate, consensus-validated UTXO system. Accepted
monthly NFT records in `blockchain/` are also a separate existing chain feature;
they do not turn `QC721Token` objects into on-chain Factory collections. TGWT's
BSC accounting/payout layer is separate from all of these.

## Answers to the product questions

1. **Can users create tokens?** Developers can create local Go token objects.
   Users cannot currently submit canonical native-token creation to the QC chain.
2. **Is creation durable and verifiable on-chain?** No. Local JSON save/reload is
   implemented, but nodes do not validate or replicate token creation/state.
3. **Is there a separate Factory module?** No active Factory module. Existing
   functionality belongs to `token/` and its persistence/miner adapters.
4. **Is Factory behavior scattered inside the QC blockchain?** No generic
   fungible-token state machine is wired there. QC transactions and monthly NFT
   validation must be distinguished from prototype token functionality.
5. **Do operations work?** Library transfer, burn and allowances work under their
   signed contracts, with the self-transfer correction. Public TOKEN and QC721
   post-creation mint methods now fail closed; initial balances/items are supplied
   at construction. QC20 remint rejection is an intentional fixed-supply rule.
6. **Where is authorization enforced?** In library methods for the supported
   signed operations. There is no active Factory API or token-consensus path.
7. **Is Factory complete?** No: a partly implemented library plus design work;
   not a deployable on-chain feature.

## Proposed completion sequence — approval required before policy changes

### 1. Decide the asset and authority contract

Retain the distinction between fixed-supply QC20, controlled-emission tokens and
QC721 collections until the owner chooses which profiles may be exposed. Decide
creation ownership, later mint permission, bounded mining-program authorization,
metadata mutability and whether a zero/unlimited cap is permitted. No new cap,
creator privilege, fee amount, or emission schedule is selected by this review.

### 2. Bind identity and signatures to the existing QC network

Propose an immutable token identifier derived from the QC genesis, creator,
creation nonce and definition commitment. Bind signed actions to that identity,
the operation, all state-changing arguments and a replay nonce. A definition's
name/symbol/owner alone cannot separate independent networks or instances.
Wallets sign locally; neither node tokens nor caller address strings substitute
for ownership signatures. An internal mining mint authority must have a separately
approved, bounded verifiable program rather than impersonating an owner.

### 3. Specify activation and canonical token state

Define the operation encoding and activation through the existing versioned
consensus/release process. Use the current QC chain and block validation, one
canonical state owner, deterministic arithmetic, supply conservation, durable
commit and reorg undo/replay. All nodes must derive token state from accepted
chain data. Do not create a second blockchain or use local callback JSON as the
source of token truth. QC PoW, subsidy, halving and total-supply rules stay intact.

### 4. Reuse the current implementation behind that contract

Adapt existing arithmetic, signed operation, allowance and validation helpers.
Avoid rewriting a second token library. Choose a migration/archive procedure for
prototype JSON without representing it as accepted testnet/mainnet ownership.
Creation costs, if approved, use the existing QC denomination and fee accounting;
this plan does not choose a price or alter QC supply.

### 5. Add the client/API boundary after acceptance is specified

Use the canonical node router and endpoint schemas, with feature activation
defaulting off until approved. Keep backend API authentication and asset-owner
authorization separate. The GUI signs and broadcasts public signed operations.
Telegram uses the same canonical service; neither receives private keys.

### 6. Require adversarial integration acceptance

Require signed creation/mint/burn/transfer/allowance proofs, wrong-owner rejection,
tampered arguments, cross-network/instance replay rejection, cap/overflow bounds,
self-transfer conservation, nonce atomicity, concurrent requests, mempool/block/
P2P/reorg equivalence, persistence failure rollback and restart replay. Preserve
the negative audit probes until the approved model passes them; do not label the
feature live based only on its current library tests.

The public TOKEN/QC721 mint and scoped authorization-domain gaps have local
library fixes. QMA-0005 is closed for the current testnet scope: the bridge gate
rejects resolved testnet environment profiles, and no token state/identity path
is connected to root runtime or QC consensus. The optional bridge can still
issue local rewards on an explicitly enabled non-testnet profile. No Factory
route or new token consensus operation was activated by this review; the active
runtime does not currently import this package.
