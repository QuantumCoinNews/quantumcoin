# Testnet preparation and acceptance

Review date: 2026-11-11. This is a preparation checklist, not launch approval
or evidence that a public network is running. The active external tracker holds
the item-level status. Existing open findings retain their identifiers.

## Before starting a testnet

| Order | Work | Required evidence | Existing findings |
| --- | --- | --- | --- |
| 1 | Verify the four source ZIP manifests and exact source bytes; retain public recovery controllers and the Python requirements.lock while excluding actual custody material. | ZIP hash receipt, locked dependency restoration, clean-source frontend tests, source hygiene regression. | 0278, 0003, 0224, 0250 |
| 2 | Replay candidate branches in their own state, select validated cumulative work, serialize coherent snapshots. | Common spent history, conflicting fork, shorter higher-work and longer lower-work chains, equal-work stability, invalid claimed work, concurrent serialization and race checks. | 0279, 0276, 0025 |
| 3 | Replace full-history single-message catch-up with bounded resumable synchronization. | Actual two-node synchronization of more than 1 MiB; interruption, reconnect, divergent histories, malformed chunks and resource limits. A raised message cap alone is insufficient. | 0277 |
| 4 | Freeze a fresh testnet genesis and protocol activation profile. | Identical genesis across test nodes; different mainnet genesis; transaction signature domain replay rejection; timestamp/difficulty boundary tests; rejection of legacy chain files under the new profile. | 0144, 0268, 0270, 0271 |
| 5 | Validate node, game and wallet contracts against that profile. | Token enforcement for every client, public-bind/CORS checks, observer write rejection, local signing, ownership proof, authenticated game actions, integer ledger and rollback integration tests. Audit the unresolved historical records before calling this complete. | 0257, 0262, 0263; linked historical findings |
| 6 | Complete the clean build and security gate. | Locked dependency restore; full Go/Python/Node tests; vet, race and Staticcheck findings resolved or explicitly dispositioned; current dependency scan; pinned build tools and matching source manifest. | 0094, 0247, 0251, 0263 |
| 7 | Prepare new test-only custody and recovery. | No reuse of exposed historical keys; encrypted backup; independently stored copy; restore into an empty temporary data directory; source-history and index inventory reviewed. | 0170, 0198, 0239, 0206, 0224, 0250 |
| 8 | Run a local rehearsal before public access. | Two isolated processes, authenticated API, signed transfer/balance agreement, mempool propagation, mining, fork recovery, restart, disk corruption rejection, backup restore and reconnect catch-up. Then produce a signed candidate from a clean reviewed commit. | 0192, 0242, 0263 |

All reproducible failure, replay, authentication and recovery scenarios above
belong before public launch. Test receipts from a previous source snapshot do
not certify later edits. The four review ZIPs contain uncommitted working-tree
source; they are not signed executable releases.

## Checks that require an operating testnet

Run the scenarios locally first, then repeat on independently operated hosts:
external peer discovery, firewall and TLS deployment, long-running stability,
real latency/loss, catch-up after downtime, telemetry/alerts and independent
backup recovery. Define a measured observation window and record failures;
elapsed uptime by itself is not acceptance. Publish only test-only network and
faucet instructions after the prelaunch gates pass.

The local rehearsal must use new temporary storage, loopback listeners and
test-only credentials. Keep API tokens out of command arguments and logs.
Existing release launchers require a verified signed package; preserve that
boundary for public distribution.

## Product integrations and mainnet preparation

| Scope | Handling |
| --- | --- |
| Final NFT art, metadata and real authorization | Remain open until the final product stage. Disabled mint routes must continue to reject mint requests. 0071. |
| Real BSC payout and receipt confirmation | Deferred to the final mainnet stage by the user. Keep actual payout disabled. Never simulate a confirmed payment or increment withdrawn before proof. 0107. |
| Social task rewards | X, Telegram and YouTube server-side verification paths are prepared; social campaigns and credits are hidden/blocked on testnet and default off. Live provider credentials, Telegram bot admin access, final X campaign account ID, YouTube privacy/consent materials and BSC mainnet claim acceptance remain before activation. Preserve the shared 3,000,000 TGWT pool and exactly 1 TGWT per separately verified task. QMA-0098. |
| Autopublisher providers and AI | Independent from node-network launch when publishing is disabled. Live provider receipts are required before claiming a platform operational. 0164, 0165. |
| Economics | Preserve the user's selected whole-QC series. Do not silently adopt fractional rewards, a new split or a ten-year emission curve. The current early cap, reserve allocation, end-of-emission and future atom model remain explicit mainnet design/acceptance work. A testnet can reproduce the current rule but must disclose it and test cap boundaries. 0265, 0266, 0267, 0272. |
| Production custody, licensing and provenance | Require confirmed ownership/licensing, actual credential rotation/custody evidence, off-host restore and signed production provenance. Local test fixtures do not close these findings. 0170, 0198, 0239, 0222, 0192, 0242, 0206. |

Do not postpone transaction-domain separation, fork choice, catch-up, amount
integrity or authentication to mainnet. They must be exercised in the testnet
protocol before public exposure.

## Existing verification entrypoints

Run from the repository root. Receipts must be written outside the checkout.

```powershell
python scripts/verify_clean_stack.py --output C:/Temp/qc-clean-stack.json
python scripts/go_source_audit.py --core-only --output C:/Temp/qc-core-audit.json
python scripts/source_hygiene.py --cleanup-plan
python scripts/runtime_config_preflight.py
./scripts/New-SourceZips.ps1 -PlanOnly
```

The source hygiene inventory is read-only. It does not remove files from the
index, rewrite Git history, move funds or rotate external credentials. A clean
fixture Git repository used by integration tests is separate from an actual
clean release commit.
