# Actual mux security acceptance

Tests construct the production root router and its real middleware; they do not
replace application handlers with success stubs. Network tests listen only on
temporary loopback ports and use temporary chains, test wallets and fixture
credentials. An exposed-bind string tests policy, not an exposed deployment.

| Contract | Acceptance evidence |
| --- | --- |
| Auth, headers-only tokens and public CORS | Existing actual-mux, read-authorization and token-bind matrices; native/browser requests remain subject to authentication. |
| Signed numeric validation | A current network-bound transaction envelope; a negative output re-signed by its real fixture owner passes the signature check but is rejected by admission. Nested raw/quoted NaN and float overflow are rejected without state changes. |
| Middleware composition | Signed-body TCP requests independently fail auth, non-preflight CORS, method, known/chunked body size and read-only gates. Chain state, pending transactions and chain file remain unchanged. |
| Concurrent admission | Four valid duplicate submissions, four incorrect credentials and four rejected browser origins race through one actual router. At least one valid receipt is obtained; only one transaction remains pending, with no block or disk mutation. Existing rate-limit responses remain valid denials. |
| Transport deadlines | Runtime.ServeHTTP owns the actual mux server. Test-specific 100 ms header/300 ms read budgets close incomplete headers and produce an application JSON error for stalled bodies. Client-deadline-only errors fail the test. The server still serves authenticated reads afterwards. |
| Pagination and navigation | Existing bounded nonempty-page, overflow/offset, unknown API JSON and SPA/asset tests. Projection-only pagination fixtures do not claim consensus-valid history. |
| Compatibility | Node HTTP, retired API, client health/mining/signing, native wallet and embedded WebUI tests run with the race detector. |

Successful output DTOs and consensus/PoW/reward rules are unchanged. Existing
legacy send-payload rejection tests are retained; the new signed-schema numeric
tests provide the stronger current-contract evidence. Production timeout values
are unchanged; shortened test deadlines exercise the same runtime path without
waiting the full production duration.

The command list and exact package/test results live in the accompanying local
acceptance report and receipt. No interactive GUI, live external network, signed
release, or broad blockchain race acceptance is implied by this HTTP test scope.
