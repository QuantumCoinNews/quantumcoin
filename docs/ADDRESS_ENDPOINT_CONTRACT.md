# Wallet address endpoints

The canonical root API owns wallet routes and requires header authentication.
It never creates or exports wallet keys over HTTP.

| Request | Response |
| --- | --- |
| GET /api/wallet/address with a valid configured local wallet | 200 JSON address |
| GET /api/wallet/address with no usable local address or an invalid selected address | 503 JSON error; no key, wallet or address file generated |
| GET /api/wallet/balance/ADDRESS with canonical QC Base58Check address | 200 JSON exact atom/QC balance; a valid unused address has zero balance |
| Balance lookup with empty, malformed, whitespace-padded, checksum-invalid or wrong-version address | 400 JSON error, before chain lookup |
| Balance lookup with a valid address but unavailable runtime/chain | 503 JSON error |
| Non-GET wallet address/balance request with valid auth in writable mode | 405 JSON error with Allow: GET |

Read-only mode preserves these reads. Missing authentication, CORS rejection or
the capability/rate/body gates can take precedence over route validation.

`wallet.ValidateAddress` checks the existing version 0x00, 25 decoded bytes and
double-SHA256 checksum. It rejects empty or over-35-byte strings before Base58
big-integer decoding: a 25-byte Base58Check payload cannot require more than 35
encoded characters. This bounds work without changing valid address derivation,
seed recovery, network signing domains or balances. Whitespace is not part of a
path address and is rejected rather than silently trimmed.

The old `api.RegisterWalletRoutes` is a compatibility retirement marker only:
GET returns 410 JSON, other methods 405. It never invokes `AddressProvider` or
returns a wallet address. `api.StartHTTP` remains retired and opens no listener.
All live clients use the authenticated canonical root API.
