# QMA-0070/0066: verified activity review integration

This is a bounded second step after the identity guard, not closure of QMA-0070,
0066, 0071 or 0073. Closed items remain 0067, 0068, 0069 and 0072 (draft scope).

The existing local `ai-review` endpoint now reads fresh server evidence from
`monthly_nft_verified_game_sessions`, using the same counter as candidate
creation. A record must match the user, exact period label and UTC month, be
verified, and not have a completion time in the future. Old mining counters,
`realBlockFound`, source labels and cached candidate activity never raise the
review score. The old activity schema mismatch is removed.

The local deterministic advisory rule applies the existing active-user minimum:
one verified completed game session. It returns PASS/LOW and score 0.65 for a
valid rare candidate with that evidence, otherwise FLAGGED/HIGH and zero.
Additional sessions do not increase this score or draw weight. PASS/LOW only
describes this activity check: it is not a wallet possession, Sybil, fraud or
artwork approval certificate, and no external AI service is called.

The review requires the verified node programme and an in-program period.
It runs in a SQLite immediate transaction and refuses selected or mint-locked
candidates. Every completed review replaces cached activity with the current
server count and requires fresh admin approval; an earlier admin approval is
not silently reused. Another process cannot select between the evidence read
and the review update. Database failures roll back and close the connection.
No winner selection, mint or balance change is performed by review.

Thirteen new tests use real FastAPI routes and temporary SQLite databases. The
positive flow signs a test Telegram identity, starts a game session, records a
real game action, completes it, creates a candidate, performs local review and
admin approval, then manually selects the candidate. It never seeds approval
statuses. The programme and clock are explicitly test-only; wallet linking is
a fixture and is not claimed as possession proof. Other tests reject cached
or client evidence, future/wrong-period records, missing evidence and programme,
unauthenticated review, reused admin approval, selected/mint-locked changes,
competing selection and failed DB updates.

Two earlier period tests advance their isolated clocks before admitting future
months; the node programme test now also confirms a future completion is
rejected before that advance. The production time check is never weakened to
accommodate the old fixtures.

Remaining work: verifiable random drawing with committed roster/entropy,
wallet possession and Sybil policy, immutable selection-to-mint binding,
final artwork/metadata commitments and approval gate, then complete award-flow
and the remaining economic/consensus regressions. The production selection
endpoint is still manual. Exactly one randomly awarded NFT per month over
twelve months remains the intended final rule.

The guarded installer modifies source/test files only, preserves the preceding
identity/catalogue/bridge sources, and runs existing Go race and Python checks.
It does not enable activation or mint, change keys/config, open the real app DB,
restart services or push to GitHub. Windows verification remains pending until
the user's Apply result is reviewed.
