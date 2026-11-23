# Genesis-bound founder vesting

Existing-chain vesting resolves its locked key from the trusted genesis
allocation. Runtime founder config and `QC_MAIN_ADDRESS` are creation/readiness
inputs, not authority to redirect or disable an existing genesis lock. The
schedule uses the chain's genesis timestamp. Final mainnet founder addresses
and launch approval remain separate work.

A missing/invalid genesis or ambiguous matching locked allocation fails closed.
A valid network/profile with no founder allocation remains valid; this does not
authorize removing an allocation from an existing chain, because the trusted
genesis hash and full replay reject that alteration.

Candidate block accounting resolves both confirmed outpoints and outputs created
earlier in that same block. Forwarding locked value through another transaction
cannot hide its net release from the cliff/tranche limit. Existing legacy/atom
normalization remains in use, using the origin and accounting heights.

The allocation and schedule are unchanged: first tranche at month 13, 48 monthly
tranches ending at month 60. Signature, ownership, maturity, UTXO conservation,
PoW and branch validation remain separate mandatory checks. Standalone vesting
validation is not a replacement for those checks.

Acceptance covers red-before/green-after ambiguous genesis and intra-block change
cases; real peer rejection before cliff and acceptance after unlock; trusted disk
replay after local config/environment changes; higher-work fork replacement and
removal of orphan release accounting; and existing early/excess release and
denomination tests. Historical component-only fixtures are identified as such.

These are consensus validation tightenings. Updated participants must use the
same rules; no release deployment, migration or mainnet activation was performed.
Previously accepted invalid history is not silently grandfathered or rewritten.
