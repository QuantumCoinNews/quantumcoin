# Browser mining and safe node startup

Running `quantumcoin` without arguments starts the node, HTTP API and P2P
services. Mining requires an explicit command or authenticated request.
Use `quantumcoin run-mine <wallet-address>` for a node with continuous mining,
`quantumcoin mine <wallet-address>` for one block, or
`quantumcoin mine-forever <wallet-address>` for the foreground miner.
The address must belong to a wallet whose private key the operator controls
and has backed up. Supplying an address does not prove private-key possession.

Missing miner configuration produces an error when an address is required.
Address lookup never generates a wallet or writes `miner_address.txt`.
The premine/founder address is not a default mining address. Existing explicitly
configured miner addresses remain usable. `newaddr` saves the generated wallet,
including its private key, before displaying its address; a storage error stops
the command without announcing a usable new wallet. The existing wallet store
format and its backup/access-control requirements still apply.

## Browser protocol

Open `/webminer.html` on a writable node using HTTPS or localhost. Enter the
wallet address and operator API token, then request mining. The input is cleared
on start. The worker receives only the public block job, and requests carry the
token in the header for that run. It mines one block per explicit start in a worker;
Stop terminates the worker and cancels outstanding browser requests. Cancellation
cannot undo a block that the node has already committed.

Authenticated `GET /api/mine/job?address=<address>` returns:

- `protocol`: `QC_WEB_BLOCK_POW_V1`;
- `job_id`: a cryptographically random 256-bit identifier;
- `header`: the actual canonical block header with nonce zero, encoded as hex;
- `nonce_offset`, `nonce_encoding`: the final eight bytes, unsigned big-endian;
- `max_nonce`: `9223372036854775807`, as decimal text;
- `difficulty`, `miner`, candidate `height`, and Unix-second `expires`.

The header is serialized by the blockchain's bound activation policy. Historical
V1/atom blocks use the historical encoding; activated NFT V4 blocks use its
domain-separated header and NFT commitment. The browser only replaces the nonce
bytes and computes one SHA-256. A valid hash must be strictly below
`2^(256-difficulty)`. It never solves a separate random challenge.

Submit authenticated `POST /api/mine/submit` with:

```json
{"job_id":"<issued-id>","address":"<issued-miner>","nonce":"<decimal-integer>"}
```

Use a string nonce to preserve integers above JavaScript's exact Number range.
The old `challenge + nonce` protocol fails closed and must be replaced in external
clients. The included page and worker implement the new protocol.

Each job holds a detached block template built under the chain lock using the
canonical subsidy, atom denomination, verified mempool fees and coinbase rules.
Different miners' jobs coexist. Issuing a job does not replace another job.
Capacity is 256 live jobs overall and eight per receiving address, with a
60-second lifetime; capacity errors return HTTP 429 without evicting live jobs.
Expired jobs are removed during issuance or submission.

Submission admission, commit and consumption are serialized. The runtime pins
its chain and checks read-only/shutdown state. Canonical block validation checks
the current parent, height, difficulty, proof of work, transactions, fees and
activation rules. The submitted nonce becomes the stored block nonce. There is
no server-side nonce search or background `MineBlock` after acceptance.

Only a successful atomic chain-file installation produces HTTP 200 with
`accepted: true`, the committed block hash and height. The job is then consumed;
replays return HTTP 409 and cannot generate another block. Expired/stale jobs,
invalid proofs and persistence failures never produce acceptance. An invalid
attempt or failed persistence leaves the original template available for a valid
retry until expiry; a chain-tip change still makes it stale. Confirmed mempool
transactions are removed after commit, while later unrelated admissions remain.
If the success response is lost, inspect the block/transaction explorer rather
than treating a replay rejection as proof that the first request failed.

Acceptance means a local validated disk commit. P2P propagation follows commit;
it does not promise finality or that the block will survive a competing fork.
Use one authoritative writable runtime per chain file, as documented in
[chain persistence](CHAIN_CONCURRENCY_AND_PERSISTENCE.md).

## Miner lifecycle

On Windows, the root `run_miner.cmd` delegates to `release/run_miner.cmd`, which
runs the PowerShell 7 full-block miner client
against an already running writable isolated local-release node. Start the
node with `run`, the isolated local configuration and its own data directory.
Supply `QC_API_TOKEN` in the process environment and `ADDR` as the reward
address. The client falls back to `QC_ADDR` and then the local runtime's address
files. `QC_API_BASE` defaults to
loopback port 28082.

The client requires writable mode and the isolated genesis domain in the
authenticated `GET /api/health` response. It calls
`POST /api/mine/local-release` once per requested block and sends
`X-QC-Expected-Genesis`. The runtime checks that hash under its mining lock
before producing a block. The isolated request succeeds after the validated
block is installed on disk. The node chooses difficulty and the client controls
only the delay between completed blocks. An observer, legacy node, mainnet,
identity mismatch, authentication failure, redirect or connection error stops
the client. The operator starts and stops the node independently.

The canonical node miner captures its own stop/done channels. Stop closes its
channel once under the runtime mutex; ownership remains registered until that
specific loop exits. Start refuses a second loop while the first is stopping.
Concurrent shutdown/stop cannot double-close the channel or attach an old loop
to a replacement channel. The compatibility `miner.Start`/`miner.Stop` path also
serializes lifecycle changes and captures per-run completion channels.

HTTP start registers mining synchronously before replying. HTTP stop reports
`running: true, stopping: true` until the active operation finishes; status turns
false when the loop exits. An already-started PoW calculation is allowed to
finish, so Stop is a request and shutdown waits for completion within its context.

## Verification

The standalone miner uses `-mode node` by default. Set the operator token in
the process environment and pass the node origin with `-api` or `QC_API_BASE`.
It retrieves `GET /api/mine/job`, solves the exact header and submits the
result to `POST /api/mine/submit`. Telegram proof-of-work and the browser
operator miner use the same endpoints. The Windows release wrapper requests
an isolated full block through the node and checks the expected genesis.
Configured node mining acknowledges only after durable chain installation.

`go test -race ./... -count=1` covers real mux submission, concurrent replay,
separate miners, stale/expired work, address binding, invalid proofs, storage
failure/rollback/retry, startup address lookup, shared fee/mempool behavior,
activated NFT V4/atom rewards and concurrent stop/restart/shutdown.

For cross-language header checks, set `QC_WEB_MINING_VECTOR_FILE` to an absolute
temporary filename, run
`go test ./blockchain -run TestExternalMiningBrowserVectors -count=1`, then run
`node --test webui/canonical-pow.test.mjs` with the same environment variable.
The vectors compare actual Go V1/V4 serialization with JavaScript at nonce zero,
above `2^53`, and at the consensus maximum. All generated wallets and chain files
in these tests are isolated test fixtures.
