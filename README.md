# 🌌 QuantumCoin — AI + Blockchain + Gaming + Social Power

**“Doing what hasn't been done.”**

QuantumCoin combines a mineable Proof-of-Work blockchain, a desktop wallet and
miner, Telegram Space Miner, community incentive tools and AI-assisted monitoring.
Transactions, ownership and rewards are governed by the chain's validation rules.

**Development status:** active development and pre-production validation.
The source includes tested runtime components and deployment preparation tools;
this update does not announce a mainnet launch or a new binary release.

## What is included

| Component | Current capabilities |
|---|---|
| Blockchain and consensus | Canonical block/transaction validation, signed protocol activation, exact fee accounting, persisted chain state and replacement/reload checks. |
| Wallet and miner | Windows desktop wallet, send/receive contracts, shared backend startup, mining controls and canonical reward validation. |
| Network and API | API authentication, CORS and request limits, peer message validation, network identity checks and connection lifecycle controls. |
| Token framework | QC20/QC721 authorization and ownership controls, address validation, registry integrity and persistence contracts. |
| State and workers | Durable state adapters, idempotent work handling, retry/lease controls, and backup/restore safety policies. |
| Economy controls | Fixed-supply rules, founder vesting, burn accounting and chain-bound policy validation. |
| Telegram Space Miner | Cinematic game interface, mining-session flow, wallet panels, verified activity and authenticated administrative review. |
| Monthly NFT rewards | Wallet-control proofs, frozen candidate pools, externally witnessed commitments, verified randomness and canonical mint receipts. |
| AI and community tools | Analysis, telemetry and advisory tools, social content automation and separate TGWT/watch/community workflows. Availability depends on configuration and service setup. |

The repository contains both runtime implementations and preparation/policy
modules. A module's presence does not automatically enable its service or reward path.

## Verified monthly NFT reward flow

The monthly program links game activity to an auditable award process:

1. Authenticate the player and derive activity from server records.
2. Review the candidate and prove control of the receiving wallet with a signed challenge.
3. Freeze the eligible pool and publish its commitment before a fixed future randomness round.
4. Read the independent witness's public signed record and verify the pinned beacon's BLS signature.
5. Select the winner without modulo bias and persist the draw evidence.
6. Verify final artwork/metadata and separate authorized AI/admin approval signatures.
7. Submit the signed award and confirm the node's persisted canonical receipt.

Missing evidence prevents mint authorization. A missing receipt after a chain
change enters reconciliation; it does not automatically create a replacement mint.

The program covers **12 active award periods, one rare NFT per period**.
Actual periods derive from the verified program activation date.

**Ready for independent deployment:** the publication witness service, HTTPS
client, fixed-round beacon verification, publication recovery and deployment guide.
An independent operator/host, real keys and operational configuration are still required.

**Still awaiting final preparation:** the 12 actual NFT artworks/metadata and
their real authorized approval evidence. Existing catalog entries are drafts.
The retired automatic miner NFT/annual bonus paths remain disabled.

See [reward lifecycle](docs/NFT_REWARD_LIFECYCLE.md),
[final preparation](docs/NFT_FINAL_PREPARATION.md) and
[independent witness setup](nft/preproduction/INDEPENDENT-WITNESS.md).

## Wallet, gaming and community

The desktop wallet and miner provide the core mining and transfer workflow.
Telegram Space Miner adds the game interface, cinematic intro, moon map, mining
sessions and wallet/community panels. Further LUNA zone transitions, visual
polish and player-facing wallet-signing support remain development work.

TGWT, watch tasks and social incentives have their own verification and settlement
controls. Their ledgers and preparation tools do not authorize extra QC issuance.
AI supplies analysis and advisory evidence; monetary rules and ownership remain
subject to chain validation and configured signing authorities.

The socialbot/autopublisher modules support community content workflows. External
publishing services require their own accounts, credentials and operator setup.

## TGWT — The Green World Technology

TGWT is the community incentive token planned for the Telegram Game reward layer.
Its deployed token contract is on **BNB Smart Chain mainnet (chain ID 56)**:

**[`0xbc70225507b132e9926c9b09c14ff4f87f6d5451`](https://bscscan.com/token/0xbc70225507b132e9926c9b09c14ff4f87f6d5451)**

We are building with a small, early community. There is room for the game,
community participation and incentive features to develop together. Early players,
testers and contributors can help shape the experience through gameplay and feedback.

**Planned Telegram Game distribution:** users will be able to earn TGWT through
qualifying game activity and eligible community tasks without purchasing the
reward tokens. The game's verification and settlement controls are still being
prepared; the current BSC claim configuration remains disabled by default.
On-chain claims, when enabled, may require the user to pay BNB network gas.

TGWT operates on BNB Smart Chain and has a separate reward/claim layer from the
native QC blockchain. Game ledger entries are subject to verification and are not
automatically proof that a token transfer has occurred on BSC.

## Canonical economy

| Parameter | Current source rule |
|---|---|
| Coin | QC |
| Total supply | **25,500,000 QC** |
| Initial mining reward | **50 QC** |
| Coinbase maturity | **10 blocks** |
| Founder allocation | **5% — 1,275,000 QC** |
| Founder liquid allocation | **1% — 255,000 QC** |
| Founder locked allocation | **4% — 1,020,000 QC** |
| Founder vesting | **12-month cliff, then 48 monthly unlocks** |
| Monthly founder unlock after the cliff | **21,250 QC** |
| Remaining network allocation | **95%** |

The authoritative monetary constants live in
[`consensus/canonical_economy_policy_v1.go`](consensus/canonical_economy_policy_v1.go).
Founder binding, transaction denomination, burn rules and activation gates must
be interpreted with the corresponding consensus and runtime validation code.
Game, AI, token and automation modules do not independently expand the QC supply.

## Planned: Token Factory and token mining

Once the core infrastructure is ready, we plan to add a **Token Factory** for
creating project tokens and a connected **token mining/reward infrastructure**.

The planned development covers:

- Tools to create tokens using the supported token framework.
- Project-specific issuance permissions, supply limits and reward rules.
- Token mining/reward flows with validated activity and recorded distribution.
- Wallet and game integration for displaying token balances and eligible rewards.

These are future capabilities. The mining model, token creation workflow,
operational requirements and launch conditions will be defined and tested before
release. Existing token authorization and persistence components provide a
foundation; they are not a released public token-creation or token-mining service.
Project tokens will have their own rules and will not override QC's fixed-supply
and consensus restrictions.

## Repository structure

| Directory | Purpose |
|---|---|
| `consensus/`, `blockchain/`, `node/` | Protocol rules, canonical ledger and node runtime. |
| `wallet/`, `miner/`, `cmd/` | Keys/addresses, mining and desktop/CLI entry points. |
| `api/`, `p2p/`, `security/` | HTTP contracts, peer transport and runtime security controls. |
| `token/` | Token authorization, ownership and registry persistence. |
| `persistence/`, `queueworker/`, `backuprestore/` | State handling, durable work and recovery controls. |
| `nft/`, `telegram_game/` | Reward catalog, evidence verification and game integration. |
| `configs/` | Example configuration and policy files. |
| `release/` | Launch/check scripts, control-panel sources and existing release support files. |
| `socialbot/`, `autopublisher/` | Community content and distribution tools. |
| `docs/` | Feature, deployment and architecture documentation. |

## Build and test

The current Windows validation used **Go 1.24.5**. The module's Go declaration is
in [`go.mod`](go.mod). Desktop GUI builds require CGO, a working C compiler and
the Fyne/OpenGL build environment. Python services should use an isolated environment.

```powershell
git clone https://github.com/QuantumCoinNews/quantumcoin.git
cd quantumcoin
go build ./...
go test ./... -count=1
```

For the game and NFT evidence dependencies:

```powershell
python -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install -r telegram_game/requirements.txt -r telegram_game/requirements-nft-witness.txt
python scripts/monthly_nft_catalog_v1.py --check
python scripts/test_monthly_nft_catalog_v1.py
```

Configuration examples need operator-specific values before use. Keep real keys,
API credentials, wallet files and runtime databases outside source control.
The witness deployment command and required operator inputs are documented in
[`nft/preproduction/INDEPENDENT-WITNESS.md`](nft/preproduction/INDEPENDENT-WITNESS.md).

## Validation and next steps

The integrated runtime passed **1,088 Go tests across 24 tested packages** and
**23 isolated Python regression scripts**, including the game-to-Go-node
mint/receipt lifecycle. PowerShell launch/check sources were syntax-checked.
These are software test results, not live network adoption or launch approval.

Current preparation work includes independent witness hosting, final NFT
artwork/approvals, wallet challenge-signing UX, further game content, operational
testnet checks and release packaging. Explorer, broader community incentives,
staking, governance and mobile experiences remain separate development areas.

See [runtime integration](docs/RUNTIME_INTEGRATION.md) and
[test coverage](docs/NFT_RUNTIME_TEST_COVERAGE.md).

## License

MIT, as stated by the project.

**QuantumCoin — Mine. Earn. Transfer.**
