# Canonical economy constants

`consensus/canonical_economy_policy_v1.go` owns supply, initial reward, maturity,
founder allocation, vesting and reserved address labels.
`consensus/canonical_runtime_parameters_v1.go` owns the existing block-time,
halving and minimum reward parameters. These are parts of one consensus authority,
not independently editable economics in each service.

Config defaults/immutable profiles, blockchain compatibility names and the
burn/economy/vesting/fee/mining-transition policy adapters reference that authority.
Node startup rejects config/policy drift before opening the chain. Chain runtime
context binds maturity and network/profile choices; test constructors and retired
mutable config shims do not define production consensus.

The current whole-QC reward series remains 50, 25, 12, 6, 3, 1 QC, subject to the
existing hard-cap checks. Founder allocation remains separate from the miner's
coinbase recipient. No staking emission, replacement burn mint, additional reserved
bonus/NFT supply or new reward split is activated by this verification.

Checked-in JSON examples repeat values for operators and are checked against
their typed policy validators. They are not a second source of authority.
Fresh minimal profile examples deliberately inherit economic defaults; network
genesis dates/identities and difficulty overrides remain profile-specific.

Tests cover cross-package compatibility aliases, canonical values and derivations,
percent/basis-point agreement, whole-QC halving, policy JSON examples, config
validation, and runtime drift rejection. Final emission-duration/economy/mainnet
acceptance remains separately tracked. Single-source consistency does not by
itself resolve those future economic decisions.
