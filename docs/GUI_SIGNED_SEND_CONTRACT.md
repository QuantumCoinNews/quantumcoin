# Native GUI signed QC transfers

The native wallet uses one signed transaction route: POST /api/tx/send.
The HTTP body contains only the public transaction and its signatures.
The private key is excluded from the serializable GUI intent.

Before opening the signing pipe, the GUI probes the authenticated canonical
GET /api/health contract. apiReadOnly must explicitly be false. An absent mode,
read-only node, invalid service or invalid API token stops submission. An
existing node is never silently stopped or switched to another mode.

When the native wallet needs to start its backend it uses quantumcoin run,
which starts the canonical writable node without automatic mining. The api
command remains read-only. The two browser launchers keep their read-only
startup behavior. Loopback binding and shared startup coordination remain.

Local signing invokes the trusted local QuantumCoin executable's sign-send
wallet command. The key travels through that child process's stdin only,
never argv, environment or HTTP. This is an offline signing command in the
same executable, not server-side HTTP signing. It loads and validates the
selected local chain view, creates and signs the exact amount, and emits a
public SIGNED_TRANSACTION_V1 envelope. It does not broadcast or save a transfer.
The existing validated executable/release provenance requirements still apply.

walletsigner validates the public envelope schema and extracts its 32-byte ID.
Duplicate keys (including case aliases), unknown fields, trailing documents,
oversized JSON and excessive nesting are rejected. The node remains the sole
authority for signature validity, ownership, UTXOs, network/genesis domain,
fees and admission. Client schema checks do not replace consensus checks.

HTTP acceptance requires status 200, success=true, a 64-digit hexadecimal ID
and an exact match to the locally signed transaction ID. The GUI displays the
accepted TxID. This is mempool admission, not block confirmation or finality.
HTTP failure, semantic failure, malformed receipt or a different ID is an error.

There is no automatic retry or fallback to a legacy send endpoint. A transport
failure can leave the admission outcome unknown; the error contains the public
signed TxID so the operator can check /api/tx/status before trying again.
An outdated local chain view may fail signing or admission and must be synced;
the wallet never substitutes another key, chain, amount or unsigned transfer.

QC PoW, reward, halving, supply and transaction authorization rules are unchanged.
The technical acceptance tests use temporary chains and synthetic wallets only.
