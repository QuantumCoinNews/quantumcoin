# Signed hardfork and quorum acceptance

Activation is derived from a signed decision, not approval booleans. The
canonical payload binds network, genesis hash, activation height, target
protocol, canonical reason set and snapshot/binary hashes. Founder and admin
signer sets have independent quorums; unauthorized keys, duplicate authorities,
duplicate approvals and cross-role identity overlap are invalid. Keys have one
accepted uncompressed encoding.

Cryptographic signature validation alone does not authorize activation. The
artifact-aware verifier also checks the trusted network/genesis, exact reason
allowlist, both quorums and actual artifact bytes. Node startup hashes the
running executable. A detached manifest cannot supply its own authorities or
quorums; the trust source is compiled/reviewed. The verified plan is bound once
to the chain and retained across trusted disk reload. Peer block validation uses
the signed protocol-selection height.

No production hardfork trust root is finalized: `CanonicalHardforkTrustV1`
currently returns nil and activation stays disabled. Test authorities, including
2-of-3 role examples, are ephemeral test data, not selected production signers.
Final authority identities/quorums, release signatures, snapshot custody, rehearsal,
rollback evidence and coordinated deployment remain release acceptance work.

Initial testnet explicitly keeps activation disabled. Authority identities and
quorums will be selected before the first hardfork; this is not a blocker that
requires enabling a fork for a fresh testnet. Release artifact signing uses a
separate owner-controlled key and does not populate this consensus trust root.

This technical verification does not approve a real fork, change an activation
height or publish a release. Existing supply, burn, vesting and mining rules
remain unchanged by this item.
