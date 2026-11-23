# API, wallet and service contracts

## Local signing and transaction submission

Private keys stay in the wallet process or its trusted local signer. Neither
`POST /api/tx/send` nor `POST /api/tx/burn` accepts a key, mnemonic, sender/amount
intent or unsigned transaction. Their sole request field is `transaction`,
containing the signed canonical transaction object. Unknown fields, malformed
envelopes, invalid signatures and requests above 1 MiB are rejected. Send and
burn routes enforce their respective transaction kind before canonical mempool
admission. Admission does not imply confirmation or durable mempool storage.

Use the [local CLI signer](CLI_SIGNED_TRANSACTIONS.md) or the native wallet.
The native wallet invokes an absolute trusted local node executable, supplies
an explicitly entered key through stdin and submits only the signed result.
The signer validates its local chain through normal mandatory startup checks;
it needs an appropriate, current local chain/configuration. It does not trust
an unsigned transaction proposal from an arbitrary remote server.

For the fresh `network-v4` testnet, the signer and node must use the same trusted
genesis. Every spend signature commits that genesis; a signature from another
network or from the historical signing format is rejected. The public JSON
envelope stays unchanged. See [network profiles](NETWORK_GENESIS_AND_EMISSION.md)
for the explicit fresh-directory transition and legacy-chain compatibility.

The Telegram interface accepts an already signed envelope. Preserve its JSON
bytes because JavaScript number parsing and reserialization can change signed
integer values above 2^53. The Telegram bridge parses the original JSON in Python
without converting amounts to floats. Submission requires verified initData and
a sender matching the linked QC address. The old Python private-key send helper
fails locally without making a network request.

The embedded node landing displays public network information. Wallet operations
use the local signer and the authenticated API. End-user Telegram flows use
Telegram identity and wallet ownership, with node API credentials held by the
backend. The separate browser mining tool is for node operators. It sends a
per-run operator token in the header for the two mining endpoints.

The wallet SDK and Telegram signed-submit bridge require HTTPS for non-loopback
connections and do not follow redirects. The node itself does not provide TLS;
public deployments must configure their transport separately. API credentials
are sent in `X-QC-API-KEY`, never URL query parameters.

## Wallet creation and health

`/api/wallet/new` no longer creates keys. GET returns 405; POST returns 410 with
`WALLET_CREATION_LOCAL_ONLY`. Use local CLI `newaddr`, which saves the wallet
before reporting its address, or local wallet creation and backup facilities.

`/api/health` remains protected. Native wallet and launcher probes send the
configured `QC_API_TOKEN`. A recognized QuantumCoin authentication challenge
means the backend is reachable but authentication is required; it does not
authorize application operations. Unrelated 401s, malformed bodies and redirects
are not accepted as healthy node responses.

## Bounded reads and advisory analysis

Block pages are capped at 200 and copy summaries without copying every
transaction. AI/security analysis samples at most 200 recent blocks and 2,000
visited transactions with a conservative 4 MiB copying budget. Oversized
transactions are skipped before cloning. Pending analysis is also bounded.
These responses are advisory samples, not full-chain audits.

Analysis responses carry `X-Analysis-Scope`, `X-Analysis-Max-Blocks`,
`X-Analysis-Max-Transactions` and `X-Analysis-Truncated` headers. Browser access
to custom response headers remains subject to the node's CORS policy.
Heavy analysis routes share a per-IP budget of four requests per ten seconds
and a global two-request concurrency budget. Exhaustion returns 429. Security
event reads are capped at 1,000 events with a separate ten-per-ten-second rate
budget and the same concurrency guard. Event files above 8 MiB are not loaded
or overwritten; an oversized file produces a truncated read result.

## Canonical game backend and retired root routes

The authenticated Telegram backend and its SQLite ledger own gameplay, identity,
sessions and reward evidence. Its leaderboard is `/api/leaderboard/top` on the
Telegram service. Score, activity and jackpot belong to that service boundary.
The Go node provides chain, mining and transfer services and does not initialize
a scoreboard or link the gameplay package.

The former `/api/game/score` and `/api/game/leaderboard` registrations were
removed after reviewing the maintained clients. In writable mode the generic
API fallback returns JSON 404. The observer applies its regular authenticated
capability boundary. The Telegram leaderboard path is
`GET /api/leaderboard/top` on the game service.

The signed score service/vector and Python signing helper remain compatibility
utilities for historical advisory projections, not connected runtime gameplay or
reward authority. Preserve any existing score cache for review; do not import its
scores into Telegram balances. Unsigned `AddScore`/`HandleTelegramScore` and demo
`GiveReward`/`EvaluateAndReward` explicitly return an unavailable error. They do
not print payments or maintain pretend claim records. A future real milestone
feature must be implemented in canonical task evidence and ledger accounting.

## Desktop compatibility

The maintained desktop entry point is `./cmd/quantumcoin-gui`. It starts the
node with the `api` command and checks `GET /api/health` on the configured local
port, which defaults to 8082. The retired `gui/` and `ui/` trees contained
historical proxy and wallet implementations and have been removed from the
working source. Use these current node contracts for integrations:

- Wallet balance uses `GET /api/wallet/balance/{address}`.
- Transaction submission uses `POST /api/tx/send` with the locally signed
  `transaction` envelope.
- Recent blocks use `GET /api/blocks?limit=10`.
- CLI mining uses `quantumcoin mine <miner-address>`.

## Capability discovery and unavailable staking

Authenticated `GET /api/capabilities` reports local signing, signed transaction
submission, disabled HTTP wallet creation, retired root score submission and
`staking: false`. Observer mode permits this read.

`POST /api/stake/start` and `GET /api/stake/status` return 501 with
`FEATURE_UNAVAILABLE`, `available: false` and `success: false`. They never create
a stake. Other methods return 405 in writable mode; observer authorization
continues to deny mutations. Staking itself is not implemented.

Real NFT artwork, production approvals and independent witness deployment
remain separate release prerequisites.

See [endpoint ownership and schema evidence](ENDPOINT_OWNERSHIP.md) for the
per-route inventory and transport ownership.
