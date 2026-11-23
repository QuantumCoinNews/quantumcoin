# QMA-0068 canonical twelve-month NFT program policy

Status: PASS - nine monthly policy/preflight tests, NFT/Monthly race, three bonus retirement regressions, full build, exact source identities and preservation. QMA-0068 remains OPEN for runtime admission and ownership.

## Verification-only correction

V2 applied all six Go sources and gofmt succeeded, then an incorrect whitespace-only scope check treated the sorted wrapper imports as an unexpected change. It stopped before monthly tests, race checks and build. The uploaded packet was reviewed: all five helper/policy/test sources match the prepared code after gofmt transformations, Blockchain differs only by one private program field plus formatting, and all twenty-four protected sources are unchanged.
Evidence QMA-0068-monthly-policy-v2-result-20261005-200937-6972104.zip
SHA256 39C4E2382A873852E2A942CC1A12F7BCDCCB8BF718D691F6D40C9808627783D3

This verification-only script checks exact SHA256 identities for the six reviewed applied sources and twenty-four protected sources, backs them up, runs missing tests and build, and checks that sources remain unchanged. It writes no Go source and does not rerun gofmt. Run outcomes and exit codes are packaged separately; this note is marked PASS only after all checks pass.

## Verified checkpoint

Block approval commitment PASS on 2026-10-05.
Evidence QMA-0068-block-proof-result-20261005-194914-2256131.zip
SHA256 F2BC6421B4D1A06471A51041B66324DC5CF4594D81B2A2754836C3D267ED865A
Nine new proof block tests, 60 NFT/Monthly race tests, three annual bonus regressions and build passed. Every recorded exit code was zero. All five source-scope checks were true; all 19 protected sources were unchanged. These are test counts, not NFT quantities.
QMA-0069 and QMA-0047/S1 remain CLOSED / PASS (annual 100 QC feature cancelled).

## Product rule

Telegram Game: one random active verified winner each month, one rare NFT per month, total twelve NFTs. AI review and administrator approval remain required. No fixed activation date or mainnet launch coupling is chosen.

The existing schedule calculation is retained: the twelve complete UTC calendar months following supplied game activation. The consensus period generator is the single source; the old nft API delegates to it. The compatibility error name is retained but its message now correctly refers to Telegram Game activation instead of mainnet.

## This wave

Canonical policy has local network, campaign and first award period. Award serial is the one-based period position, 1 through 12. Only the twelve consecutive periods are accepted. The policy factory derives its first period from explicit trusted game activation, not time.Now or a built-in calendar date.

Blockchain privately binds this immutable program once, after local authority trust is bound, and checks network scope. Gob cannot import the private program. The monthly evidence preflight checks every prior award and the candidate together: V4 proof commitment, signatures against local trust, campaign/network, period/serial, total twelve and existing one-award-per-network/month uniqueness. Counts are derived from chain history, not a separate mutable RAM counter. Tests cover twelve valid periods, thirteenth period, multiple recipients in one month, duplicates within a block, validly signed wrong program data, local rebind and concurrent access.

Two existing Go files are modified: Blockchain gains one private field and nft/monthly_program_periods_v1.go becomes a compatibility wrapper. Four Go files are added. The guard/rejected mint/peer/load paths and QC economics are not changed.

## Limits and next step

This is preflight evidence validation, not runtime block admission. Valid signatures and PoW do not prove Telegram activity, AI review or unbiased random selection. The local bootstrap caller must supply trusted program configuration; policy shape validation is not a cryptographic program release approval.

Remaining QMA-0068: unify actual approved NFT admission across mining, peer, replacement and locally trusted load/reload; enforce activation/timing with verified program configuration; persist/rebuild/query ownership. Mainnet activation, real administrator keys, actual launch dates, award runtime and catalog are not configured here. All nonempty NFT data remains rejected by existing runtime admission gates until that integration is completed. QMA-0066 and QMA-0070-0073 remain separate open scopes.

Verification results (test counts, not NFT quantities):
monthly-policy-tests exit-code=0 top-level-passed-tests=9 expected-count-check=True
nft-monthly-race exit-code=0 top-level-passed-tests=69 expected-count-check=True
bonus-retirement-regression exit-code=0 top-level-passed-tests=3 expected-count-check=True
build exit-code=0 top-level-passed-tests=0 expected-count-check=True
