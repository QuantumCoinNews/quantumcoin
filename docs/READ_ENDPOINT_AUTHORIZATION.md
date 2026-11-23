# Read endpoint authorization

The root router is the canonical node API. Its private reads require a configured
QC_API_TOKEN and a matching X-QC-API-KEY or Bearer header in every bind/network
mode, including loopback and read-only API mode. Missing configuration, absent
credentials, incorrect credentials and unsupported Basic auth fail closed with
JSON 401 and WWW-Authenticate. Read-only is a capability boundary, not an auth
bypass; a valid token does not enable mutations there.

Private reads include health/capabilities, telemetry, monitoring, economy/runtime
policy, NFT program/receipts, security/admin analysis, wallet/address/balance,
transaction status, AI reads, explorer blocks, stake status, mining work/tip/result
and miner status. Existing dev-only/local-host constraints still apply to miner
status. An unsupported feature or unavailable context can return its own error
after successful authentication.

For limited read routes, authentication runs before the endpoint/analysis quota
and analysis work. Invalid credentials do not spend an authenticated read quota.
The outer anonymous IP rate/capacity guard still applies to all traffic; this is
intentional resource protection, not an authentication bypass.

Token-checked responses and denials use Cache-Control: no-store and vary on the
credential headers, preserving any existing Origin variation. This explicitly
protects the custom API-key case from shared/browser cache reuse. These headers
do not replace authentication or TLS/bind policy.

The public API read exception is GET /api/network/label: only network labeling
and public identity data are returned, with no-store. Its response omits the
local chain_file filesystem path. The internal config label model can retain
that path for local use. Public landing/assets and CORS preflight are separate
from protected node data; a successful preflight grants no read permission.

Legacy /api/mempool, /api/utxo and /api/address/balance are not live data routes:
the writable router returns JSON 404; the read-only capability boundary may
reject earlier with 401/403. The old package-api server remains retired and
cannot open a listener. Retained in-process advisory helper functions are not
independent HTTP servers or authenticated routes.

The QC API token authenticates a trusted service/client to this node. Telegram
player identity and wallet signing/ownership remain separate contracts; this
change does not create per-player node tokens or change consensus/transfer rules.
