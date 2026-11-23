# QuantumCoin Telegram Game

The canonical backend is **`telegram_game.runtime:app`** and the canonical UI is
**`/game`**. The bot opens this WebApp; it does not maintain a second economy.

The QC wallet section supports device-local creation, encrypted backup and
word-based recovery. Build its pinned Go assets with
`python ../scripts/build_wallet_crypto.py` before a manual server start.
See [QC wallet recovery](../docs/QC_WALLET_RECOVERY.md) for existing-wallet
compatibility and device-change recovery. This applies only to QC wallets.

## One reproducible installation

From this directory, using the pinned Python 3.14.10:

```powershell
python -m venv .venv
.\.venv\Scripts\python.exe -m pip install --require-hashes -r requirements.txt
.\.venv\Scripts\python.exe -m pip check
```

`requirements.in` defines direct dependencies; `requirements.lock` pins every
resolved package and its SHA-256 hashes. All older requirements files delegate
to this one installation, including bot and NFT verification dependencies.
The lock was installed and tested on Windows AMD64 / Python 3.14.10. Other platforms require separate acceptance.
The dependency security gate is documented in ../docs/DEPENDENCY_BOUNDARIES_AND_AUDIT.md.

To regenerate deliberately, use a separate tooling environment with
`pip-tools==7.6.2`, then run from the repository root:

```powershell
python -m piptools compile --generate-hashes --resolver=backtracking --strip-extras --no-emit-index-url --no-emit-trusted-host --output-file telegram_game/requirements.lock telegram_game/requirements.in
```

Reinstall with hashes and rerun tests after changing the lock. This file pins
the validated runtime versions; generating it does not certify their security.

## Configuration and startup

Process environment values take precedence over the optional outer `.env`.
The `.env` location is relative to this package, never the current directory.
`QC_GAME_LOAD_DOTENV=0` disables it, as the isolated test suite does. Keep bot,
operator, node and OAuth secrets out of source, browser storage and logs.

```powershell
$env:TELEGRAM_GAME_CHAIN_MODE = "testnet"
$env:QC_API_BASE = "http://127.0.0.1:18082"
$env:QC_GAME_MINING_MODE = "TESTNET_SIMULATED"
.\.venv\Scripts\python.exe -m uvicorn telegram_game.runtime:app --host 127.0.0.1 --port 8080 --no-proxy-headers
```

Supply `TELEGRAM_BOT_TOKEN`, `TELEGRAM_GAME_ADMIN_TOKEN` and `QC_API_TOKEN`
through your deployment environment. The WebApp needs HTTPS and signed current
Telegram initData; opening a plain browser does not create an authenticated
player. CORS is an explicit origin allowlist, with no wildcard default.

`REAL_CHAIN` mining requires device-solved node work and a verified receipt.
`TESTNET_SIMULATED` gives server progression without a real QC block or payout;
`DISABLED` rejects game mining. Testnet normally uses port 18082 and must not
silently use the local/main node. Existing PowerShell launchers use
`$PSScriptRoot` and the same ASGI runtime. Reload launchers are for development;
the no-reload launcher or the command above is suitable for supervised startup.
Activating a game mode does not enable mainnet payouts or a consensus hardfork.

## Database and migration

For an existing database, back it up while the service is stopped and set
`QC_GAME_DB` to its **absolute** SQLite file path before startup. Keep chain
identity and network configuration aligned. No launcher copies or merges DBs.
Known old source-local databases require explicit selection; startup refuses
to silently replace them with an empty database. Fresh installs use a
network-specific user data directory, independent of cwd/executable location.

Schema upgrades are additive. Legacy REAL QC/XP records are preserved for
review, without being certified as new integer-atom balances. Restore a tested
backup with a compatible application version if rolling back; do not combine
old and new DBs or delete working data. See the detailed persistence and exact
accounting documents linked below. Tests never migrate the user's database.

## Standard test discovery

Go and Node.js are required for the integration/PoW/browser-controller tests.
From this directory:

```powershell
.\.venv\Scripts\python.exe -m unittest discover -v
```

Or from the repository root:

```powershell
telegram_game\.venv\Scripts\python.exe -m unittest discover -s telegram_game -v
```

Discovery uses package `load_tests` hooks. It runs all 31 historical policy
scripts and the actual runtime/route/auth/DB/PoW/wallet/withdrawal and JavaScript
contracts in child processes. Each child has a separate temporary database and
working directory, test-only credentials, disabled `.env` loading and an
unreachable default node origin. A fresh Go test contract is generated locally;
no Codex-specific path or pre-existing artifact is required. Node tests are
included. Failures and child output propagate to unittest; tests are not skipped.

The historical scenario files are executable contracts, so use discovery from
the documented package root rather than importing individual scenarios in a
shared application process. Existing `scripts/nft_regression_v2.py` remains an
optional detailed-log runner for the Python NFT/runtime subset.

## Frontend ownership

`dev_ui/game.html` loads external deferred controllers once. Old index and
prototype pages redirect to `/game`. `player_session.js` owns the in-memory
verified identity; its compatibility getters cannot be overwritten by later
scripts. `account_wallet.js` owns account reads, ledger display, signed transfer
submission and decimal-string withdrawal reservations. `game_scene.js` retains
the graphics, translations and cosmetic views, using server snapshots for
QC/TGWT/XP. `mining_controller.js` listens to explicit scene enter/exit events;
it does not wrap scene functions after a timeout. Encrypted recovery and wallet
ownership proofs remain separate controllers with their existing safeguards.

Old local currency/XP bookkeeping and demo withdrawal handlers are removed.
Local preferences, cosmetic state and existing recovery material are preserved;
loading the page never deletes or displays a recovery phrase. Signed transaction
JSON is transported verbatim, including int64 values. A reservation is shown
separately from a paid BSC withdrawal, with the same nonce retained after an
uncertain retry.

## Remaining launch inputs

TGWT uses the fixed 3,000,000-token pool and 1 TGWT per separately verified task.
Telegram membership, YouTube subscription and X follow/like have server-side
verification adapters. Social rewards remain disabled on testnet and off by
default; they require real provider setup and the approved BSC mainnet claim
path. BSC claim/receipt acceptance remains a later mainnet task, and the user's
wallet pays BNB gas. Mock tests do not count as live provider acceptance. Final
NFT assets/metadata/approvals, independent witness operation and final dependency
and platform audit are also still open.

Further details:

- [Runtime, identity and data boundaries](PLAYER_RUNTIME_AND_IDENTITY.md)
- [Verified rewards, Telegram/YouTube/X setup and device PoW](TGWT_VERIFIED_REWARDS_AND_CLIENT_POW.md)
- [Wallet ownership, recovery and withdrawal reservations](WALLET_CONTROL_AND_RESERVED_WITHDRAWALS.md)
- [Runtime persistence and clean source delivery](RUNTIME_PERSISTENCE_AND_SOURCE_HYGIENE.md)
- [Exact accounting and node policy](EXACT_ACCOUNTING_AND_NODE_POLICY.md)
