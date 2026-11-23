# One network and solo proof of work

Telegram, the native Windows wallet and the standalone node miner are clients of
one selected QuantumCoin network. They do not own separate chain files, heights,
difficulties or QC reward ledgers.

The miner address in the accepted canonical block receives the miner reward.
Before the existing first halving the subsidy is 50 QC, independent of the number
of participants. Existing halving, fee, cap, coinbase maturity and chain selection
rules apply. A local acceptance receipt is not finality: reorganizations can
remove it and immature coinbase outputs cannot be spent.

## Runtime ownership

| Path | Work and acceptance |
| --- | --- |
| Native wallet GUI | In-process `miner.Worker`, pinned to locally derived genesis; device hashes the node-issued canonical header |
| Telegram Game | Verified Telegram session and proved linked QC address; browser worker hashes the same node-issued header; Game relays a nonce once |
| Standalone `cmd/miner` | Node backend and the same `miner.Worker`; no independent chain writer |
| Node CPU mining | Detached canonical template on fresh V4 networks; same prepare/commit service as external PoW |

The common acceptance boundary is `node.Runtime.CommitExternalMining` and the
strict blockchain durable writer. The Game database does not award another
50 QC. Game completion can record activity/XP after a proof; its QC credit is zero.
TGWT social rewards remain a separate system.

`GET /api/mine/tip` returns a small height/hash/genesis snapshot. A changed hash
invalidates old work even at equal height. Clients also respect job expiry.
`GET /api/mine/result?height=...&hash=...&address=...` checks the current canonical
chain to reconcile a lost submission response. These reads require the node API
token; they cannot mine or credit rewards. Both work issue and submit remain
disabled in the node's read-only `api` mode.

## Operator setup

Start one writable node with the selected network's config and an isolated data
directory (`quantumcoin --data-dir DIR --config FILE run`). GUI and Game must use
that same node origin and genesis. Set `QC_API_TOKEN` locally, never in client
bundles or chat. For the fresh testnet use `release/config.testnet.json`,
`QC_NETWORK=testnet-local`, and the actual testnet origin (normally loopback
port 18082). Remote origins must satisfy the existing HTTPS and authorization
policy. Existing `api`-only wallet services are read-only: stop/reconfigure them
through normal operator controls before starting the writable node on that port;
the GUI never kills or silently upgrades another node process.

The GUI needs the same `QC_CONFIG_FILE`/`QC_DATA_DIR` selection as its node and
the public reward address from a created/restored QC wallet. `QC_API_BASE` can
select the node origin. Start checks the expected genesis before device mining.
Stop cancels only the GUI's own worker. The old local-release batch tools remain
isolated local tools and are not launched by either GUI mining button.

Telegram needs `QC_GAME_MINING_MODE=REAL_CHAIN`, the same network/origin and a
server-side node credential. Its normal authenticated session and QC address
ownership checks remain mandatory. A mining session can accept multiple blocks;
the durable receipt archive retains proof when the next job replaces the old
one. Leaving/stopping mining terminates the browser worker. No private key is
sent with work or nonce submissions.

Mainnet needs its separately finalized genesis configuration/fingerprint before
activation. Testnet history and balances are never imported. Mainnet genesis and
economics acceptance remain separate open work.

## Capacity and validation boundary

Reward ownership is independent of miner count. Admission still enforces bounded
jobs, authentication and rate limits; temporary work admission failures back off.
Telegram coalesces tip polling across players. These are resource limits, not
proportional reward rules. A 100-participant functional test is not a benchmark
for 100,000 simultaneous miners; production capacity requires measured deployment
sizing. No share accounting or proportional pool payout is implemented by this
contract.
