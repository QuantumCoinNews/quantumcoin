# QMA-0070: Identity checks for monthly candidate selection

Status: **partial implementation; QMA-0070 remains OPEN**.

The existing admin `select-winner` endpoint still selects a supplied candidate
ID. It now rechecks server-recorded evidence and linked wallet identities at
selection time, within one SQLite `BEGIN IMMEDIATE` transaction. This is a
manual selection guard, not an implementation of the required random draw.

The selected candidate must have one canonical user identity in that month,
one valid Base58Check wallet (using the mint bridge's existing decoder), AI
PASS/LOW and admin approval. Decoding linked wallets detects aliases with
surrounding whitespace and legacy duplicate links across players. Ambiguous
identity is rejected; the code does not silently choose one duplicate.
Activity must be server-verified, labelled with the award period, timestamped
inside that UTC month and completed no later than the current server time.
Client counters and score cannot substitute for that evidence.

One selected or mint-locked candidate occupies the month regardless of NFT
type. A partial unique index on period backs the transactional guard. Unknown
legacy mint states also occupy the month. Conflicting legacy winners prevent
index creation and cause a closed failure; no existing rows are deleted or
automatically repaired. A separate review is needed for such legacy data.
The index is created only on actual selection; installation itself does not
open or migrate the application's DB.

Admin review cannot clear selected/mint-locked state. Its conditional SQL
update also rejects a stale review that races with a successful selection in
another worker. Selection does not mint, write the chain or change balances.
The same player and wallet may win another month; the cap is one NFT per
month, not one NFT for the player's lifetime.

The unused Go placeholder selector separately rejects duplicate eligible
users/wallets, including whitespace aliases. Its default/public seed and
first-byte modulo selection remain unchanged and are NOT suitable evidence
of a fair production draw.

Tests exercise the real FastAPI admin routes and SQLite, with an explicitly
test-only activation clock and preapproved candidate fixtures. Sixteen new
Python cases cover auth, duplicate identities, malformed wallets, current
activity, programme periods, per-month limits, review races, DB rollback and
eight competing processes. Four Go tests exercise duplicate-weight rejection.
An old-route regression proves that duplicate wallets were previously accepted;
old Go code also fails the three new duplicate-rejection cases.

## Remaining work for this group

- QMA-0070: a verifiable random selection and entropy/roster commitment; wallet
  possession proofs and Sybil policy; immutable binding across selection and
  mint. This patch does not freeze wallet association after selection.
- Approval integration: the current local AI scorer reads older
  `miningSessions`/`realBlockFound` activity fields, whereas new candidates store
  `verifiedCompletedGameSessions`. Real evidence-compatible AI review is still
  required. The new tests seed approvals to isolate identity validation; they
  do not prove the complete AI approval workflow.
- QMA-0071: final artwork/metadata and commitments, uniqueness evidence,
  AI/admin approval flow and an actually mint-ready gate.
- QMA-0066: one complete runtime award flow with immutable programme/selection
  binding, preparation, approvals and confirmed canonical mint receipt.
- QMA-0073: end-to-end acceptance plus the remaining economic, burn,
  founder/network split, hardfork/quorum and persistence regressions.

Closed items remain 0067, 0068, 0069 and 0072 (draft catalogue scope). This is
four of eight items by count, not a claim that half the remaining effort is
complete or that one further patch will close the group. The programme rule
is one randomly awarded NFT per month for twelve months, twelve total.

Run the focused tests from the repository root:

```powershell
python telegram_game/tests/test_nft_selection_identity_v1.py
go test ./nft -race -count=1
```

The guarded updater additionally runs the existing catalogue checks, real Go
node contracts and Telegram regression scripts. Windows verification remains
pending until the user's Apply report is checked. No activation, keys, chain
data, real application DB or GitHub remote is changed by installation.
