# Runtime integration

The current source integrates chain validation and persistence, token authorization,
wallet/miner startup, authenticated APIs, exact fee accounting, supply/burn/vesting
rules, durable worker controls and evidence-based game rewards.

The ledger and node are authoritative for QC balances, accepted mint records and
NFT ownership. Advisory AI output, a game database flag or an administrator's
request cannot replace consensus validation and configured signing authority.

The wallet GUI shares backend startup coordination and canonical API contracts.
Network validation checks peer input and chain context. Public configuration
examples are in `configs/`; launch/check scripts and control-panel sources are in
`release/`. Real operator settings and credentials are supplied separately.

For rewards, see [NFT lifecycle](NFT_REWARD_LIFECYCLE.md),
[final preparation](NFT_FINAL_PREPARATION.md), [catalog](NFT_CATALOG.md) and
[verification coverage](NFT_RUNTIME_TEST_COVERAGE.md).

This source delivery does not activate mainnet, publish a binary release or create
actual NFT artwork. Staking, explorer and broader governance/product features
remain separate development areas.
