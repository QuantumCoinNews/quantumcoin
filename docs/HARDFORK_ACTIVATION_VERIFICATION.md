# Hardfork activation verification

Date: 2026-09-28
Work item: QMA-0065
Repository and working directory: C:\Projects\quantumcoin
Go module: quantumcoin
Status: QMA-0065 CLOSED for the requested implementation and regression scope. The final full normal and race runs passed. The first full race timeout and successful reproduction/retry results are retained below. Production activation remains intentionally disabled.

## Behavior

A verified signed hardfork decision now produces the existing ActivationPolicyV1 and ActivationPlan. NetworkID, LegacyGenesisHex, ActivationHeight and TargetProtocol are derived from the signed decision. The policy contains a private, value-only verification proof tied to that exact plan. An unsigned manually constructed policy, a changed plan, or a serialized policy without its proof cannot enable activation.

Verification checks the canonical reason allowlist, the trusted network and genesis, founder/admin authority sets, signatures and both quorums, and hashes of actual snapshot and executable bytes. Existing approval and artifact validators are reused. No parallel activation mechanism was introduced.

The existing blockchain binding remains one-time and retains its genesis hash check. ProtocolAtHeightV1 continues selecting the protocol. Integration tests exercise actual block acceptance before, at and after the signed height. Failed binding does not consume the binding slot. Runtime chain reload carries forward the verified policy; peer-chain replacement retains the local authoritative activation policy.

Public node startup obtains its trust only from CanonicalHardforkTrustV1. It verifies the release and runtime policy gate before chain loading/runtime side effects, then binds the verified policy to the loaded chain. Tests use a private injection seam in the same startup implementation. The public configuration does not select authority keys, quorum, activation height, manifest path or artifact directory.

## Production boundary

CanonicalHardforkTrustV1 currently returns nil. Production activation therefore remains disabled. No production key, quorum, network, genesis or height was invented, and no live fork was performed.

Once reviewed production trust is compiled into a release, the loader requires hardfork-release.json next to the running executable, containing only Decision and Approvals. Unknown manifest fields and trailing JSON are rejected. The snapshot is loaded from hardfork/<SnapshotSHA256>.snapshot in that directory. The release binary hash is checked against bytes read from os.Executable(), not a configurable binary path.

The manifest is detached because embedding a signature of the executable's hash inside that executable would be self-referential. Low-level verification accepts an explicit trust root for library use/tests; public node startup uses only the canonical compiled trust root. The default disabled production configuration means configured production authorities have not been exercised with real deployment material.

Existing public notice, rehearsal, rollback, upgrade-window and audit declarations are not converted into external attestations by this change. Existing unrelated mainnet gates remain in force.

## Determinism and mutation protection

Canonical signed-payload serialization is reused. Reason ordering produces the same derived plan. The returned policy retains no caller-owned slices or artifact buffers. Changes to input authorities, the decision, artifact buffers or local runtime configuration do not change an already verified/bound plan. Plan mutation invalidates the verification proof. Proof is not accepted from JSON, chain persistence or peer payloads.

## RED evidence

The following regressions were observed before their corresponding fixes:

```text
--- FAIL: TestHardforkBindingRejectsUnsignedActivationV1 (0.00s)
    hardfork_verified_binding_v1_test.go:17: unsigned manually-created plan enabled production blockchain activation
FAIL quantumcoin/blockchain 0.905s

--- FAIL: TestHardforkActivationRejectsUnsignedPlanV1 (0.00s)
    hardfork_activation_binding_v1_test.go:9: unsigned activation policy validated without decision, approvals or artifact verification
FAIL quantumcoin/consensus 1.148s

--- FAIL: TestHardforkRuntimeReloadPreservesVerifiedActivationV1 (0.08s)
    hardfork_activation_reload_v1_test.go:31: reload lost signed activation: got QC_LEGACY_PROTOCOL_V1
FAIL quantumcoin/node 1.086s
```

The new P2P integration test passed against existing P2P production behavior; it is confirmation of existing behavior, not a claimed RED-to-GREEN fix. p2p/network.go has the same SHA256 as the pre-task copy:
75D03F20A579DBC39F7CAE0B2B9DEB620D58832544145EF66EBDB5A774D63425.

## Negative coverage

- Signed decision field tampering: height, protocol, network and genesis.
- Validly signed decision versus altered activation plan: height, protocol, network, genesis and preservation flags.
- Validly signed decision for the wrong chain genesis.
- Actual snapshot and binary hash mismatch, including a re-signed wrong executable hash.
- Missing founder/admin quorum, unauthorized signer, duplicate signer/authority and wrong signature.
- Signed noncanonical/forbidden reasons.
- Unsigned manual policy and policy reconstructed through serialization.
- Missing/invalid detached manifest and attempted authority injection.
- Missing decision keeps activation disabled.
- Reload preserves the signed height and target protocol.

## Completed verification

All commands start with Set-Location 'C:\Projects\quantumcoin'. Go tests use a writable temporary build directory and approved Go cache access. Toolchain: go1.24.5 windows/amd64; go.mod language version: 1.21.

Target command:
```powershell
go test ./consensus ./blockchain ./node ./p2p -run '^TestHardfork(Activation|Binding|Startup|Runtime|Detached|PeerChain)' -count=1
```
PASS, exit 0:
```text
ok quantumcoin/consensus 2.394s
ok quantumcoin/blockchain 6.070s
ok quantumcoin/node 2.056s
ok quantumcoin/p2p 1.043s
```

Target race command:
```powershell
go test -race ./consensus ./blockchain ./node ./p2p -run '^TestHardfork(Activation|Binding|Startup|Runtime|Detached|PeerChain)' -count=1
```
PASS, exit 0:
```text
ok quantumcoin/consensus 2.039s
ok quantumcoin/blockchain 43.716s
ok quantumcoin/node 4.261s
ok quantumcoin/p2p 40.913s
```

Package regression command:
```powershell
go test ./consensus ./blockchain ./node ./p2p -count=1
```
PASS, exit 0:
```text
ok quantumcoin/consensus 0.782s
ok quantumcoin/blockchain 33.463s
ok quantumcoin/node 1.999s
ok quantumcoin/p2p 1.977s
```

## Change scope

This continuation changes 22 existing Go files and adds 12 Go files: 9 production files and 25 test files in total. Existing enabled-policy test fixtures were updated to obtain verified policies from test-only signatures. No production private keys were added.

The repository was already dirty. Its complete git diff is not the scope of this task. Original Go files were copied before edits under C:\Users\cem78\AppData\Local\Temp\quantumcoin-hardfork-binding-20260928. The task-scoped diff is hardfork-activation-scoped.diff in that audit directory. Existing source changes were preserved; no git reset, clean or restore was used.

gofmt -l over all 34 affected Go files returned no filenames. Repository file enumeration found zero qma-prefixed filenames.

## Package race regression

`go test -race ./consensus ./blockchain ./node ./p2p -count=1`: PASS, exit 0.

```text
ok quantumcoin/consensus 2.629s
ok quantumcoin/blockchain 214.949s
ok quantumcoin/node 5.008s
ok quantumcoin/p2p 21.636s
```

## Exact source scope for this continuation

- atom_api_exact_amount_v1_test.go
- atom_balance_api_display_v1_test.go
- runtime_policy_startup_contract_test.go
- blockchain/activation_policy_binding_v1_test.go
- blockchain/activation_policy_binding_v1.go
- blockchain/activation_policy_serialization_v1_test.go
- blockchain/atom_block_reward_v1_test.go
- blockchain/atom_fee_utxo_v1_test.go
- blockchain/burn_accounting_atom_overflow_v1_test.go
- blockchain/burn_accounting_atom_v3_test.go
- blockchain/canonical_adapter_denomination_v1_test.go
- blockchain/mining_protocol_gate_v1_test.go
- blockchain/peer_block_validation_v2_test.go
- blockchain/state_access_sync.go
- blockchain/utxo_denomination_v1_test.go
- consensus/activation_policy_v1_test.go
- consensus/activation_policy_v1.go
- consensus/canonical_activation_policy_v1.go
- consensus/target_protocol_activation_v1_test.go
- node/bootstrap.go
- node/chain_reload.go
- node/runtime_policy_gate_v1.go
- consensus/hardfork_activation_v1.go
- consensus/hardfork_runtime_v1.go
- consensus/hardfork_activation_fixture_test.go
- blockchain/hardfork_activation_fixture_test.go
- node/hardfork_activation_fixture_test.go
- p2p/hardfork_activation_fixture_test.go
- hardfork_activation_fixture_test.go
- consensus/hardfork_activation_binding_v1_test.go
- blockchain/hardfork_verified_binding_v1_test.go
- node/hardfork_activation_reload_v1_test.go
- node/hardfork_activation_startup_v1_test.go
- p2p/hardfork_peer_chain_v1_test.go

## Full repository regression

go test ./... -count=1: PASS, exit 0.

```text
ok  	quantumcoin	29.189s
?   	quantumcoin/ai	[no test files]
ok  	quantumcoin/api	0.551s
ok  	quantumcoin/backuprestore	0.541s
ok  	quantumcoin/blockchain	26.564s
ok  	quantumcoin/burncontrol	0.609s
?   	quantumcoin/cmd/apitest	[no test files]
?   	quantumcoin/cmd/apitest/quantumcoin-gui	[no test files]
?   	quantumcoin/cmd/miner	[no test files]
?   	quantumcoin/cmd/qc_deterministic_wallet	[no test files]
?   	quantumcoin/cmd/qc_recovery_init	[no test files]
?   	quantumcoin/cmd/qc_seed_canonical_check	[no test files]
?   	quantumcoin/cmd/qc_seed_manager	[no test files]
?   	quantumcoin/cmd/qc_wallet_create	[no test files]
?   	quantumcoin/cmd/qc_wallet_restore	[no test files]
?   	quantumcoin/cmd/quantumcoin-gui	[no test files]
ok  	quantumcoin/cmd/wallet	1.413s
ok  	quantumcoin/config	1.146s
ok  	quantumcoin/consensus	0.696s
?   	quantumcoin/consolefx	[no test files]
ok  	quantumcoin/economycontrol	0.776s
ok  	quantumcoin/foundervesting	0.627s
?   	quantumcoin/game	[no test files]
ok  	quantumcoin/hardforkpolicy	0.617s
?   	quantumcoin/i18n	[no test files]
ok  	quantumcoin/internal	1.188s
ok  	quantumcoin/miner	0.867s
ok  	quantumcoin/miningtransition	0.539s
ok  	quantumcoin/monitoring	0.537s
ok  	quantumcoin/networkfee	0.531s
ok  	quantumcoin/nft	1.912s
ok  	quantumcoin/node	2.493s
ok  	quantumcoin/p2p	3.227s
ok  	quantumcoin/persistence	5.412s
ok  	quantumcoin/queueworker	4.843s
?   	quantumcoin/release/qc_kontrol_paneli	[no test files]
ok  	quantumcoin/security	0.583s
?   	quantumcoin/tests	[no test files]
ok  	quantumcoin/token	3.830s
?   	quantumcoin/utils	[no test files]
ok  	quantumcoin/wallet	0.845s
?   	quantumcoin/webui	[no test files]
```

## Full race first attempt: failure retained for audit

The first `go test -race ./... -count=1` attempt reported:

```text
--- FAIL: TestAPIHTTPShutdownHelperGracefullyStopsServer (2.18s)
    server_gui_lifecycle_contract_test.go:192: graceful API shutdown: context deadline exceeded
FAIL quantumcoin 38.836s
```

The test and main.go are unchanged from the pre-task copies (SHA256 checked). The test allows 2 seconds for http.Server.Shutdown. This output is a timeout, not a race-detector warning. Cause is not established solely by this failure. Isolated reproduction and final outcome are recorded below; this first failure must not be omitted from verification history.

## Timeout reproduction

`go test -race . -run '^TestAPIHTTPShutdownHelperGracefullyStopsServer$' -count=5`: PASS, exit 0 (`ok quantumcoin 1.850s`). No source changes, timeout increases or test exclusions were made. The timeout was not reproduced in five isolated runs; its exact cause remains unproven. The first full race run exited 1; every other package passed. A second identical full race run follows.

## Final full repository race regression

go test -race ./... -count=1: PASS, exit 0, second full attempt, no source changes between attempts.

```text
ok  	quantumcoin	25.122s
?   	quantumcoin/ai	[no test files]
ok  	quantumcoin/api	2.493s
ok  	quantumcoin/backuprestore	1.712s
ok  	quantumcoin/blockchain	323.375s
ok  	quantumcoin/burncontrol	1.647s
?   	quantumcoin/cmd/apitest	[no test files]
?   	quantumcoin/cmd/apitest/quantumcoin-gui	[no test files]
?   	quantumcoin/cmd/miner	[no test files]
?   	quantumcoin/cmd/qc_deterministic_wallet	[no test files]
?   	quantumcoin/cmd/qc_recovery_init	[no test files]
?   	quantumcoin/cmd/qc_seed_canonical_check	[no test files]
?   	quantumcoin/cmd/qc_seed_manager	[no test files]
?   	quantumcoin/cmd/qc_wallet_create	[no test files]
?   	quantumcoin/cmd/qc_wallet_restore	[no test files]
?   	quantumcoin/cmd/quantumcoin-gui	[no test files]
ok  	quantumcoin/cmd/wallet	2.822s
ok  	quantumcoin/config	1.875s
ok  	quantumcoin/consensus	2.543s
?   	quantumcoin/consolefx	[no test files]
ok  	quantumcoin/economycontrol	1.880s
ok  	quantumcoin/foundervesting	2.130s
?   	quantumcoin/game	[no test files]
ok  	quantumcoin/hardforkpolicy	1.817s
?   	quantumcoin/i18n	[no test files]
ok  	quantumcoin/internal	2.218s
ok  	quantumcoin/miner	5.152s
ok  	quantumcoin/miningtransition	1.748s
ok  	quantumcoin/monitoring	1.384s
ok  	quantumcoin/networkfee	1.790s
ok  	quantumcoin/nft	1.654s
ok  	quantumcoin/node	5.115s
ok  	quantumcoin/p2p	8.159s
ok  	quantumcoin/persistence	8.571s
ok  	quantumcoin/queueworker	4.468s
?   	quantumcoin/release/qc_kontrol_paneli	[no test files]
ok  	quantumcoin/security	1.992s
?   	quantumcoin/tests	[no test files]
ok  	quantumcoin/token	5.583s
?   	quantumcoin/utils	[no test files]
ok  	quantumcoin/wallet	2.934s
?   	quantumcoin/webui	[no test files]
```
