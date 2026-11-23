# Exact accounting and node policy

The game uses integer QC atoms (1 QC = 1,000,000 atoms) and integer XP units.
This is the canonical denomination defined in Go consensus. Legacy and
continuity transaction wire integers still mean whole QC; AtomV3 and NFTV4
transaction integers mean atoms. The game forwards signed wire integers
unchanged. This change does not activate a hardfork.

## Database transition

On startup of an explicitly selected database, the schema adds `qc_atoms` and
`xp_units` to players, and `amount_atoms`, `xp_units`, `block_reward_atoms` to
reward records. Nonnegative SQLite integer checks and application overflow
checks protect new totals. `numeric_version=1` identifies new records.

Existing REAL columns and rows remain intact as audit history. Historical
client-reported QC/XP are **not automatically certified or converted into
spendable balances**. The wallet flags nonzero legacy totals with
`legacyNumericReviewRequired`; historical reward rows remain version 0.
Back up the selected database before rollout. Reconcile historical entries
against independently verified events before any separate, audited import.
Do not discard the original data or infer entitlements by rounding it.

Mining completion awards bounded server-calculated integer XP, with zero
secondary QC or TGWT. The real node coinbase already pays the linked wallet.
Old float-based QC settlement and mock payout mutations return HTTP 410,
including when development flags are enabled. Historical reports remain
readable for audit; their old amounts/statuses are not executable entitlements.
This retirement is separate from the existing TGWT reserve/release ledger.

The node wallet balance response now includes decimal-string `balanceAtoms`,
`spendableAtoms`, `balanceQc`, `spendableQc`, and `atomsPerQc`. Old numeric fields
remain display compatibility fields. Use the exact fields for reconciliation.
The game requires the exact denomination for its real-wallet balance status.
Transaction confirmation requires a node transaction receipt; a balance
increase is never confirmation of a particular transaction.

TGWT uses the same six-decimal integer scale, its existing 3,000,000-token pool,
and 1 TGWT per separately verified task. Decimal withdrawals are converted by
their digits/exponent, without ambient Decimal-context rounding. BSC payment
verification remains disabled until the final mainnet stage; users pay BNB gas.

## Policy authority

`GET /api/runtime/policy` is an authenticated, read-only Go node endpoint. It
returns chain height, tip/genesis identity, next-block protocol and denomination,
next subsidy in protocol units, and the bound activation policy. All chain
context is observed within one read lock.

The six game admin economy/network policy routes obtain this snapshot using
the server-side node token. They return `source`, `context`, `policy`, and
`authority`. Unavailable, incompatible or unauthorized node responses cause
HTTP 503; no Python policy default is substituted. Go policy sections are
startup declarations, not proof that all declared capabilities are active.
Effective protocol and hardfork state are reported separately in `context`.
Standalone Python policy modules remain offline reference material only.

## Bot, NFT and idempotency boundaries

The bot registers only `/start`, opening the canonical WebApp. It maintains no
random-reward economy, balances or mining JobQueue.

NFT activity comes from verified server game sessions, with wallet/identity
proofs and controlled node minting. Client counters and `realBlockFound` do not
create eligibility. Final artwork, metadata and approvals remain launch inputs.

The legacy `(user_id, nonce)` table is audit history only; its unused writer is
removed. Active task rewards deduplicate authenticated provider events and
campaign/player/subject identities; mining deduplicates persisted sessions;
withdrawals deduplicate their own player/nonce/fingerprint records. Reusing a
client nonce across distinct actions does not consume another action's key.
Changing a nonce never bypasses the underlying proof or session uniqueness.
