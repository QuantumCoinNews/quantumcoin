# Fee recipient and post-emission blocks

The accepted rule is: the block producer receives subsidy plus verified
transaction fees. Ordinary node operation, founder ownership or platform choice
does not create a separate entitlement to these fees.

At the cap, subsidy becomes zero. Blocks continue even when they contain no
fee-paying transactions; the single coinbase then has a zero-valued payout.
If verified fees are present, coinbase pays exactly those fees. Fees transfer
existing value and never increase issued supply. Negative subsidy/fees and
incorrect coinbase totals remain invalid. Genesis allocation rules remain
unchanged. A chain already above its cap cannot be mined further.

The final partial subsidy still uses the remaining supply. Legacy QC and
atom-denominated protocols use their existing wire units; a zero subsidy is zero
in either denomination. The whole-QC halving schedule and canonical cap do not
change. Normal signed transactions do not gain permission to spend zero/negative
amounts or bypass UTXO checks.

Both local mining and external templates use canonical reward composition.
The compatibility QC-local miner now delegates template construction to that
same blockchain boundary instead of building a separate subsidy-only coinbase.
Current template selection, PoW, signature, expected difficulty and persistence
validation remain mandatory.

This is a consensus behavior change at completed emission: older binaries reject
these blocks. Release/network participants must use the same upgraded rule before
that boundary. No deployment, existing-chain migration or mainnet activation was
performed. The independent final fee-parameter and release gates remain in place.

Acceptance includes red-before/green-after reward tests, a small-cap deterministic
legacy fixture exercising final partial reward plus fee, fees-only payout, empty
external commit, repeated empty local blocks and trusted disk replay; protocol
unit checks for Legacy/AtomV3/NFTV4; and full consensus, networkfee and miner tests.
The small fixture cap is not a canonical QC economy change.
