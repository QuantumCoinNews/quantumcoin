# Mining reward contract

The implemented consensus pays the complete block subsidy plus validated
transaction fees to the miner's coinbase output. There is no automatic
70/10/10/5/5 split, staking payment, developer/community payment, or percentage
burn from a block reward. This change corrects configuration defaults, console
reporting and documentation; it does not alter historical blocks or activate a
new consensus rule.

The locked supply allocation is separate: 25,500,000 QC total; 5% founder allocation
(1% liquid and 4% subject to the 12-month cliff / 48-month vesting schedule) and
95% mining emission pool. See QC_IMMUTABLE_ECONOMY_LOCK_V1.md and the canonical
consensus economy constants. Fee and explicit burn rules retain their existing
consensus validation.

Legacy `reward_pct_miner`, `reward_pct_stake`, `reward_pct_dev` and `reward_pct_burn`
JSON/environment field names remain readable for source compatibility. Values
must each be in 0..100, total exactly 100, and match the implemented role contract.
Noncanonical proposals fail configuration validation before runtime startup; they
cannot redirect a coinbase payment. Defaults and the exported `Share*Pct` values describe the
implemented 100/0/0/0/0 recipient contract. `CommunityPercent()` returns zero.
The mining console states the recipient contract without recomputing an imaginary
float split or confusing whole QC with atom-denominated protocol amounts.

Any future split would require an explicit economic decision, deterministic
recipient/amount rules, validation and activation for all nodes. Configuration
percentages alone cannot activate it.
