# QC bonus review and testnet startup

Reviewed 14 November 2026 against active code, not historical feature claims.

| Name | Current behaviour |
| --- | --- |
| Annual 100 QC mining bonus | Cancelled earlier; amount is zero and ENV cannot enable it. |
| AI QC bonus | Compatibility methods are inert; AI cannot mint or change balances. |
| internal.GiveBonus | Records pending_settlement; creates no UTXO, signed transfer or spendable QC. No active production caller was found. |
| Daily QC / 500 QC jackpot in config/economy.json | Historical game prototype data; not the active Telegram economy or a funded payment mechanism. |
| 500,000 QC bonus / 100,000 QC NFT reserves | Historical proposals; current consensus does not reserve or fund these pools. |
| Telegram mining | Real PoW pays the chain wallet through coinbase. Game completion adds XP and records zero secondary QC; it cannot pay the block reward twice. |
| Social task reward | Separate TGWT system: verified tasks earn 1 TGWT from the 3,000,000 TGWT pool. This does not mint QC. |

The canonical mainnet constants describe 25,500,000 QC total, 5% founder
allocation and the remainder for mining. Actual genesis allocation requires
the appropriate approved configuration. The official fresh testnet has no
funded genesis outputs, so there is no spendable founder or bonus allocation
at its start. Testnet QC comes from mined, validated blocks.

The testnet therefore starts without extra bonus minting or a speculative
reserve. This does not cancel a future game bonus product. A payable QC bonus
should have an explicitly funded QC wallet, bounded budget, authenticated
eligibility proof, unique reward ID/idempotency, an owner-signed transfer and
confirmed-chain reconciliation. Insufficient funds must leave a claim unpaid.
A game/AI/admin record alone is never spendable money. Founder locked funds
cannot be spent before vesting. No live payment service is added by this review.

No bonus amount, task entitlement, treasury owner or recurring schedule was
invented here. Mainnet funding and any proposed reserve allocation remain an
explicit economic decision under the mainnet allocation work item. A new
reserve cannot silently reduce the canonical mining pool.

## Genesis verification finding

Independent processes showed that unrelated prior Gob encoding changed the
old transaction IDs and genesis hash. Same-process tests had missed this.
The fresh, not-yet-launched network-v4 profile now uses fixed field order,
fixed-width numbers, length-prefixed data, UTC instants and explicit hash
domains for transaction content/input digests. Legacy wire fields and 32-byte
ID size remain. Chain context enforces the active codec; structural format
recognition is not an admission fallback.

Cold and warmed processes now produce the same genesis, transaction ID and
signing digests and verify each other's signed transactions. An independent
Python binary-format vector verifies the content hash. Official testnet
startup rejects hidden prefund addresses and drift in its consensus inputs.

testnet/genesis.json is now a code-derived fingerprint, replacing the stale
placeholder date/halving example. It is not an on-disk chain or a launch.
Mainnet genesis date, allocations, funding and final fingerprint remain
unfinalized; old balances and testnet QC will not migrate into mainnet.

Sources: consensus/canonical_economy_policy_v1.go,
consensus/no_reserved_bonus_nft_supply_v1_test.go, config/config.go,
miner/miner.go, ai/bonus.go, internal/reward_system.go,
telegram_game/telegram_game/dev_api.py and
blockchain/transaction_encoding_v2.go with its independent-process tests.
