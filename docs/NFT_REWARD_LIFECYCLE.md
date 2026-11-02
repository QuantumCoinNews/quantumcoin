# Verified monthly NFT reward lifecycle

Player authentication and server-recorded activity precede candidate review.
A one-use secp256k1 wallet challenge binds the player, receiving address, network,
campaign, genesis and award period. Candidate identity, period and wallet evidence
are rechecked when the pool is frozen.

Before the selected future beacon round, the frozen pool commitment is recorded
by an independent HTTPS publication witness. The game persists an outbox before
network submission and rereads the signed public record. The witness accepts the
first commitment per network/campaign/genesis/period; a conflicting pool or round
cannot replace it. A lost reply recovers the same public record.

The client verifies pinned chain information and the exact beacon round, BLS
signature and signature-derived randomness. Supported schemes are
`pedersen-bls-unchained` and Quicknet `bls-unchained-g1-rfc9380`.
Winner selection uses a full-width digest and rejection sampling. An unsigned
fallback, manual winner override, different round or `latest` result cannot
substitute for the committed evidence. Multi-account human identity is not a KYC
guarantee; the checks establish the authenticated account/activity/wallet bindings.

The final catalog and separate authorized AI/admin signatures bind the candidate,
period, recipient, draw receipt and exact catalog/image/metadata bytes. A controlled
mint records its intent before contacting the node and confirms a durable canonical
receipt before showing the award as confirmed. A missing canonical receipt enters
reconciliation and does not automatically cause a second mint.

The program has twelve active award periods and one NFT per period. Activation
determines actual periods. The retired miner NFT and annual automatic bonus paths
stay disabled. Current ownership follows accepted chain state, including replacement
and reload; a game DB status alone is not current chain ownership.

See [independent witness deployment](../nft/preproduction/INDEPENDENT-WITNESS.md)
and [final preparation](NFT_FINAL_PREPARATION.md).
