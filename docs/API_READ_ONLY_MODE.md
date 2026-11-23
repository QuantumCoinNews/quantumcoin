# Read-only API mode

`quantumcoin api` starts an observer API. Before serving requests the command
sets both the application mode and `node.Runtime` to read-only. It retains
chain reload from disk and HTTP-only shutdown, without autosave or final save.

The production mux applies a capability boundary before route handlers and
their mutation rate limiters. Only explicitly reviewed reads are allowed;
adding an API route does not automatically authorize it in observer mode.
The restriction is captured when the server is constructed and cannot be
downgraded by resetting compatibility globals. A runtime switched to read-only
after construction also causes the boundary to reject writes.

## Requests

- Authenticated mutation requests receive HTTP `403` and the existing JSON
  error envelope: `{"success":false,"message":"API mode is read-only"}`.
- Missing/invalid header credentials still receive `401`. A valid token does
  not grant write capability in observer mode.
- Transaction send/burn, mining/fastmine, work creation/submission, miner
  start/stop, game score, wallet generation, stake start and NFT mint are
  denied regardless of the HTTP method. This includes legacy mutating GETs.
- Explicitly listed GET reads retain their existing handlers and token rules:
  health, network label, telemetry, economy, NFT program, security views,
  existing wallet address/balance, transaction status, AI views, leaderboard,
  blocks/block details, capabilities, stake status and miner status. Stake
  status explicitly reports `501 FEATURE_UNAVAILABLE` rather than active staking.
- `/api/nft/monthly/receipt` retains its POST contract because signed proof is
  carried in the body. It performs receipt lookup, never minting. Existing
  evidence, content-type, authentication and rate-limit checks still apply.
- Other methods on read routes and unreviewed API paths fail closed. CORS
  preflight remains handled outside the capability boundary without invoking
  an application handler. Static frontend GET/HEAD requests remain available.

`/api/wallet/address` now only looks up an existing address. If none is
available, it returns `503` rather than generating a wallet and writing
`miner_address.txt`. Wallet creation is local only; the HTTP creation route is
retired in writable node mode too. Existing CLI address generation keeps its
behavior.

Normal node modes keep their existing authenticated write routes. Read-only
mode does not turn observability into a storage sandbox: normal access logs,
security monitoring and in-memory chain reload can still operate. It denies
application mutations to the ledger, wallet files, scores and mining state.

## Regression coverage

`api_read_only_boundary_test.go` exercises the actual mux across mutating paths
and HTTP methods, header authentication, readable routes, signed receipt lookup,
unknown/disguised paths, mode capture, runtime mode changes and CORS preflight.
It compares chain bytes and application snapshots before/after denied requests,
checks for newly created files and verifies missing-wallet lookup has no write
side effect. Existing writable-route and lifecycle tests remain in place.

Run `go test . ./node -race -count=1` for the affected application/runtime suite.
