# Permanent burn and proposed recycling reserve

The active burn policy is PERMANENT_BURN. Owner-signed canonical burn transactions
send value to an unspendable sink. Issued supply is not reset by a burn, circulating
supply decreases by burned atoms, and the fixed maximum supply does not change.
A burn does not create new mining emission capacity or a replacement mint.

A RECYCLING_RESERVE_PROPOSAL is modeled separately and remains disabled. A future
reserve would hold spendable existing QC under a defined owner's signing authority.
Those funds would be reserve holdings, not burned supply. Any transfer to miners
would debit existing reserve funds and would not mint QC. Burn sink outputs cannot
be recovered into a reserve. The policy validator rejects activation, replacement
minting, counting reserve funds as burned, and spending the burn sink.

The distinction is visible in the Go policy returned through the canonical policy
bridge. This change implements no reserve transfer, automatic payout or new
consensus rule. A future reserve needs explicit rules and separate activation.
