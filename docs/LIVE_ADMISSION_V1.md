# QMA-0068 live NFT admission core

Status: VERIFICATION PENDING - a test/build/source check failed. Review packet. QMA-0068 remains OPEN.

## Patch transport correction

The first live-admission attempt stopped on a multiline patch anchor before staging/gofmt, before adding a test and before applying runtime sources. All thirty-two source hashes match the reviewed PASS baseline.
Evidence: QMA-0068-live-admission-result-20261005-203836-9232979.zip
SHA256: B224D2D91E1B9D1A698DA2A320B9766330EF742F5F52BB4645B5448FFD190CA3

The intended literals each match the uploaded exact source once. V2 transports patch strings as JSON with escaped tabs and newlines so terminal indentation cannot change their byte content. It records each named anchor count before application. The five intended patches, runtime implementation, nine test functions, source identity guards and RED/GREEN sequence are unchanged.

The NFT distribution rule is twelve active award months, one user and one rare NFT per month, twelve total. All numeric test counts in previous verification summaries refer to software tests only.

## Resume after Windows copy lock

V2 verified the specific expected RED on 2026-10-05: TestNFTLivePeerAdmissionAcceptsLocallyApprovedAwardV1 failed because QC_NFT_MINT_AUTHORIZATION_V1 rejected the locally approved record. After RED, Copy-Item could not overwrite blockchain.go because Windows reported an open mapped file section. GREEN tests and runtime race/build were not reached.
Initial backup: C:\Projects\quantumcoin-backups\QMA-0068-live-admission-v2-before-20261005-204326-0107753

This resume verifies all original backup identities, independently reconstructs and formats the five intended Go files, compares the existing staged files with those exact bytes, and accepts current runtime sources only if they equal the old baseline or the reviewed staged version. It uses same-directory temporary files and atomic File.Replace/File.Move, with bounded retries, instead of Copy-Item overwriting the mapped file. Current files are backed up again. It preserves the existing new test and does not repeat the verified RED. Every failure produces a result packet for review.

## Last verified checkpoint

Monthly policy verification PASS on 2026-10-05.
Evidence: QMA-0068-monthly-verify-result-20261005-201945-3482061.zip
SHA256: 2E7BD0E23D83454F6624C2CC1FD80AC811BCFE18AC83C62EE863A5EE9CE32F9A
Nine new monthly policy/preflight tests, 69 NFT/Monthly race tests, three annual bonus retirement regressions and full build passed. All four exit codes were zero. Thirty source identity and thirty preservation checks were true. Test counts are not NFT award quantities.
QMA-0069 and QMA-0047/S1 remain CLOSED/PASS (annual 100 QC feature cancelled).

## Product rule and authority boundary

Telegram Game: one rare NFT for one randomly selected active verified user per month, twelve months and twelve total. Selection, AI review and administrator approval remain required product steps. No actual game activation date, real administrator keys or mainnet activation is configured by this patch.

Existing period generation remains twelve complete UTC calendar months following supplied Telegram Game activation. Award serial remains the position 1-12 in that window. This wave additionally requires each mint record's award period to equal its containing block's UTC month. This is a deterministic block timestamp rule; existing basic validation still bounds future timestamps.

Live admission requires the existing verified signed NFT V4 activation at candidate height, private local NFT administrator trust, a private locally bound monthly program, matching network scope, correct record/proof pairs, local signer quorum, signed actual recipient/metadata digest, V4 PoW commitment and one award per network/month across the candidate history. Program shape validation is trusted local configuration, not a cryptographic program release approval. No block can configure these private policies.

The context-free compatibility authorization guard and existing context-free file loader continue rejecting all nonempty NFT data. No public default policy is enabled. A valid signature does not itself prove Telegram activity, AI review or fair selection; these remain separate runtime/selection scopes.

## This wave

One shared chain-state NFT validator delegates to the already tested monthly history/signature engine. The canonical block envelope and IsValidChain use it with the appropriate history prefix. Replacement scratch states inherit only local private activation, NFT authority and program state; uniqueness uses the incoming history, not a union with an abandoned branch.

The actual peer and replacement writer boundaries detach submitted data and repeat strict validation while owning the write lock. Accepted peer evidence cannot be modified through the caller's pointers. Concurrent submits cannot both accept an award. Canonical QC fee, subsidy, founder vesting and UTXO validators remain authoritative; no new monetary algorithm is introduced.

Files modified: blockchain/blockchain.go (three exact patches), blockchain/canonical_block_validation_adapter_v1.go (one call replacement), blockchain/chain_replacement_validation_v1.go (two private context fields in scratch). Two Go files and this checkpoint are added. Existing NFT proof/trust/program files, context-free rejection guard, economy, miner bonus retirement, P2P handler and node bootstrap/reload are protected from modification in this wave.

## Tests and remaining work

The first new test must show the expected RED: a fully signed locally approved award with valid V4 PoW and canonical coinbase is rejected by the old context-free authorization call. Runtime sources are applied only after that specific RED. Nine new live tests cover successful actual peer append, missing local policy, activation boundary/network, validly signed wrong period/campaign/serial, untrusted signers/quorum, duplicate month, accepted pointer detachment, canonical QC excess coinbase rejection, valid replacement/alternate-history reorg, bad replacement atomicity, commit revalidation, serialization without imported runtime policy, context-free load rejection and concurrent duplicate submit.

Remaining QMA-0068: approved local mining using the same canonical economic path; actual P2P full-chain handler context binding; trusted load/bootstrap/reload with full candidate chain validation; persisted/rebuilt/queryable ownership across restarts and reorgs. The old loaders intentionally cannot yet load an NFT-bearing chain. IsValidChain retains its historical PoW/link plus NFT check role; full transaction/economic validation stays in strict admission. This patch alone does not close QMA-0068. QMA-0066, QMA-0070-0073 remain separate open scopes.

Software verification results (not NFT distribution counts):
live-admission-tests exit-code=1 software-tests-passed=8 expected-count-check=False
nft-monthly-runtime-race exit-code=1 software-tests-passed=118 expected-count-check=True
bonus-retirement-regression exit-code=0 software-tests-passed=3 expected-count-check=True
build exit-code=0 software-tests-passed=0 expected-count-check=True
