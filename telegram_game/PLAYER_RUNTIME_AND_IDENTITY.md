# Telegram Game runtime, identity and reward boundaries

The canonical backend is `telegram_game.runtime:app`, launched from the outer
`telegram_game` directory. `dev_api.py` currently holds the route implementation;
`runtime.py` is its public composition root. The compatibility `api_server` import
exports the exact same app. It has no separate database, pool, worker or payout
confirmation loop. The old modular handlers expose no routes and constructing
the independent `database.Store` fails with an explicit retirement error.

`main_bot` is only a messaging adapter: `/start` opens the canonical WebApp.
It cannot accrue balances, run reward missions or confirm withdrawals in memory.
Its optional launch needs the bot dependencies in the outer requirements file.

## Running and existing data

```powershell
Set-Location C:\Projects\quantumcoin\telegram_game
# Set secrets through the deployment environment, never commit them.
$env:QC_GAME_DB = 'C:\path\to\existing\dev_api_state.db'
python -m uvicorn telegram_game.runtime:app --host 127.0.0.1 --port 8080 --no-proxy-headers
```

`QC_GAME_DB` must be an absolute file path. It selects one SQLite database for
players, ledger, mining runs, identity sessions and security queries, independent
of the working directory or entry point. Its parent is created if necessary.
All canonical connections use WAL and a five-second busy timeout.

For a fresh install without that setting, the default on Windows is
`%LOCALAPPDATA%\QuantumCoin\telegram-game\<local|testnet|mainnet>\state.sqlite3`.
On Unix it is `$XDG_DATA_HOME/quantumcoin/telegram-game/<profile>/state.sqlite3`
or `~/.local/share/quantumcoin/telegram-game/<profile>/state.sqlite3`.
An absolute `QC_GAME_DATA_DIR` overrides this base directory. The same network
mode resolver selects the profile as the node bridge. An explicit `QC_GAME_DB`
is an operator override and is not automatically split by network.

If a known old source-local/working-directory game DB is found without an
explicit selection, startup stops instead of creating an empty replacement.
Back it up, then set `QC_GAME_DB` to the intended existing absolute path.
The launchers delegate this decision to the same Python policy; none pins a
different implicit database. No records are automatically moved, deleted or
merged. Startup adds tables only to the explicitly selected existing database.

Back up existing databases before rollout. Old `qc_tgwt.sqlite3` or custom
`QC_TGWT_DB` files are preserved but never merged automatically: client-created
balances and simulated confirmations are not proof of a legitimate entitlement.
Inventory and reconcile them against independent evidence before any later
migration. `QC_TGWT_DB` alone stops startup with a retirement error; it cannot
select a second live backend. If `QC_GAME_DB` is explicit, the old setting is
ignored and its old file is preserved.

Set `TELEGRAM_BOT_TOKEN` (supported aliases remain available). The optional bot
adapter also needs an HTTPS `QC_GAME_WEBAPP_URL` pointing at the same `/game`.
Expose the game through HTTPS in deployment. Keep the node API private and its
`QC_API_TOKEN` on the server. Configure `TELEGRAM_GAME_ADMIN_TOKEN` independently;
do not distribute it in the WebApp or use it as a player credential.

## Player access

1. The WebApp submits Telegram's signed `initData` to
   `POST /api/telegram/identity/verify`.
2. The server validates Telegram's HMAC, unique query keys, integer user ID and
   authentication date (15 minute login age, at most 60 seconds future skew).
   Missing bot configuration fails closed. The authoritative actor is `tg_<id>`.
3. A successful login returns an opaque 256 bit bearer credential and expiry.
   Only its SHA-256 hash is stored, alongside actor, expiry and bot fingerprint.
   Sessions expire after one hour; at most eight remain per player. Logout
   deletes the current credential. Bot token rotation invalidates prior sessions.
4. Private requests carry `Authorization: Bearer <accessToken>`. The outermost
   ASGI boundary authenticates before route logic and legacy middleware. Every
   actor alias in the query/body must match the verified actor. Duplicate JSON
   keys and bodies over 256 KiB are rejected. Withdraw lookup by request ID also
   checks record ownership. Private responses and login responses use no-store.

The default for all new `/api/` routes is player authentication. The explicit
public exceptions are identity status/login and GET store, social links, watch
catalogue, mining mode and leaderboard. `/health` and static UI/assets remain
public. OPTIONS serves preflight only. No development flag disables identity.

Admin, settlement, security diagnostics, withdrawal operator actions, dev reset,
manual TGWT earn and verification require the separately configured operator
credential. A player's session never grants operator access. The existing NFT
admin evidence checks still apply after this outer boundary. NFT proof routes
and signed wallet submission retain their additional signed initData checks.

The UI waits for login, holds its credential only in memory and sends it only to
the game origin. It never treats localStorage or `initDataUnsafe` as identity.
Missing/failed/expired authentication locks the account UI and requires reopening
from Telegram. The former TGWT-only page points to `/game`. Signed transaction
request bodies remain byte-for-byte intact through the fetch and ASGI wrappers.

Validation follows Telegram's
[official Mini Apps authentication contract](https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app).

## Mining completion and verified task rewards

The game uses actual client-solved canonical block work. Game finish cannot ask
its node to search for a nonce, and cannot credit a second QC payment for a block
already rewarded on-chain. Owned server sessions, accepted proof and single-use
completion are required. Generic client-reported completion remains retired.

All TGWT task rewards require provider evidence. Each verified task earns exactly
1 TGWT from the existing shared 3,000,000 TGWT accounting pool. X follow and like,
YouTube subscription and Telegram membership use separate campaigns. Their
server-side adapters are implemented, but social rewards default off and are
blocked on testnet. Live acceptance still needs provider credentials/permissions
and mainnet claim-contract acceptance.

See [verified task rewards, X setup, real client mining and limits](TGWT_VERIFIED_REWARDS_AND_CLIENT_POW.md)
for the current protocol, provider API checks, accounting and deployment limits.

Wallet links now require a one-use public signature scoped to the Telegram
account and configured network/genesis. Withdraw requests reserve entitlement;
they do not mark it paid. See [wallet control and reserved withdrawals](WALLET_CONTROL_AND_RESERVED_WITHDRAWALS.md)
for the exact protocol, legacy data boundaries and user recovery migration.
