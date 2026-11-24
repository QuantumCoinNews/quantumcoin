# QMA closure evidence — 2026-11-24

## QMA-0251 — Fixed Windows toolchain and local CI acceptance

**Closed.** Exact locked Windows gate passed on clean source commit `c129c98772b3a0a79847c650c0bf2a2117e1671a` using Go 1.26.11, Node 22.23.5/npm 10.9.9, Python 3.14.10, FFmpeg 9.0.2, and MinGW 15.1.0. The complete clean-stack stages passed, including `go test ./...`, `go vet ./...`, `go test -race ./...`, Staticcheck, npm audit for root/WebWallet/Autopublisher (zero findings), govulncheck, pip-audit, and Telegram Game 77/77. Three Windows release targets built byte-identically from two independent clean checkouts. The gate receipt returned `ok=true`.

GitHub hosted required checks and signing acceptance remain tracked under QMA-0263 and QMA-0192; they are outside this local toolchain/CI closure.

## QMA-0242 — Source-to-binary provenance

**Closed.** The same clean commit and locked toolchain produced `quantumcoin.exe`, `wallet-gui.exe`, and `quantumcoin-launcher.exe` with identical bytes in two independent clean checkouts. The release manifest, provenance, and SBOM were generated and validated; the unsigned candidate CI gate returned `ok=true`.

Production release signing and hosted signed-release acceptance remain tracked under QMA-0192 and QMA-0263. This closure confirms reproducible source-to-binary provenance, not a signed public release.

## QMA-0270 — Fresh testnet/mainnet genesis isolation

**Closed.** Added `blockchain/network_v4_genesis_isolation_test.go` and pushed test-only commits `86202263053df58fced9675cf849095bbce6122c` and `2f24aca` to `codex/qma-interim-20261123`. `TestNetworkV4FreshPublicGenesisIsolationAndRestart` passed on `2f24aca`; the full `go test ./blockchain -count=1` passed on the same test implementation before a final deterministic-identity assertion was added. The test verifies public V4 testnet and mainnet each start at height 0 with zero issuance, same-profile genesis identities are reproducible, the two networks have distinct genesis identities, cross-network chain-file loading fails without changing the source file, and each network reloads its own genesis after restart.

Mainnet founder/reserve allocation and final launch fingerprint remain under QMA-0266/economy acceptance; they are not required to prove that testnet and mainnet cannot reuse each other’s chain history.

## QMA-0267 — Emission end rule

**Closed.** The accepted emission end is supply-based, not a separate `MiningPeriodSecs` cutoff: whole-QC subsidy continues by block height (including the 1 QC floor) until the fixed cap; at the cap subsidy is zero and fee-only or zero-payout blocks continue. `consensus.TestCanonicalEmissionProjectionV1` and `blockchain.TestPostEmissionPartialFeesEmptyBlocksAndTrustedReplay` both passed on the locked Go toolchain. The projection test verifies 484,499 last-subsidy height for 1,275,050 QC initial/genesis issuance, before first halving at height 2,102,401, and exact cap accounting. On 2026-11-24 the user confirmed that existing rules remain and the cap takes precedence over the illustrative 10-year target; this closes QMA-0265 without changing consensus values.

## QMA-0265 — Cap-first emission schedule decision

**Closed.** The user explicitly chose to preserve the existing whole-QC reward, 25.5M cap, 30-second target and height-based halving rules, accepting that the cap may be reached before the illustrative 10-year target. `consensus.TestCanonicalEmissionProjectionV1` and `blockchain.TestPostEmissionPartialFeesEmptyBlocksAndTrustedReplay` passed on Go 1.26.11. The projection confirms cap exhaustion at height 484,499 with the test fixture's 1,275,050 QC initial/genesis issuance; a fresh V4 founder-only launch projects 484,500 50-QC reward blocks. Both projections are about 168.23 target days and precede the first halving at height 2,102,401. The documentation was clarified to call 10 years an illustrative horizon. No consensus code or reward values changed.
