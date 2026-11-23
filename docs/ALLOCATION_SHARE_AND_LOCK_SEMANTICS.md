# Allocation share and lock semantics

The V1 economy policy has a closed allocation set:

| Allocation | Share | Locked | Liquid | Meaning |
| --- | --- | --- | --- | --- |
| founder_liquid_operations | 100 basis points (1%) | false | true | Founder liquid allocation; ordinary ownership/spend rules still apply |
| founder_locked_vesting | 400 basis points (4%) | true | false | Founder allocation subject to canonical genesis-bound vesting |
| non_founder_network_supply | 9500 basis points (95%) | false | false | Non-founder mining emission budget; not an immediately spendable genesis treasury balance |

Founder liquid plus locked is 5%; the complete allocation set totals 100%.
Each name must retain its exact share and flags. Every allocation has
`MintsNewQC=false`: this declaration does not itself issue or transfer coins.
Unknown names, ambiguous whitespace, duplicates, missing allocations, shifted
shares, contradictory flags and overflow attempts are invalid.

The canonical model has no separately reserved QC bonus/NFT issuance pools.
Historical proposed sub-pools must not be reintroduced as extra allocations.
Any future allocation requires an explicitly revised policy and compatible
consensus/launch review, rather than an unchecked extra JSON row.

`Locked`/`Liquid` metadata does not create a cryptographic lock or bypass one.
Genesis outputs and blockchain founder-vesting validation provide enforcement.
The existing cliff/schedule is unchanged: first tranche at month 13, 48 monthly
tranches completed by month 60. Final founder addresses/genesis launch approval
remain separate work. A node/operator cannot unlock coins by editing this policy.
