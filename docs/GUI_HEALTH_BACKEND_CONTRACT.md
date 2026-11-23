# GUI health and backend startup

The canonical node route is GET /api/health. The retired package-api server
must not be started as a fallback. GUI clients share apiclient.ProbeHealth
and apiclient.EnsureLocalBackendV1.

A ready node returns application/json, HTTP 200, ok=true,
service="QuantumCoin API" and healthVersion="QC_NODE_HEALTH_V1".
Existing health fields remain available. A node without a chain returns 503.
The GUI sends QC_API_TOKEN as X-QC-API-KEY. A canonical 401 with
WWW-Authenticate: Bearer realm="QuantumCoin API" and success=false means
authentication is required: report the error without spawning another node.
These protocol markers prevent accidental service mismatch; they are not
cryptographic server attestation.

Health responses have a 64 KiB limit. Duplicate top-level fields, conflicting
status, case aliases, non-JSON content, redirects and unsupported versions
are rejected. Redirects never forward the API token. Individual health
requests and the complete startup operation have finite deadlines.

The configured loopback port is preserved; an unhealthy or invalid setting
does not silently select another node. Default port is 8082. The GUI uses
QC_HTTP_PORT; a conflicting legacy HTTP_PORT is rejected rather than used
as a second authority. Startup uses
an OS-owned exclusive lock shared by all three Windows GUI entry points.
The guard file in the operating-system temporary directory is retained.
Its age does not permit another process to steal the lock. The OS releases
ownership on process exit, including a crash. Port normalization gives
equivalent values such as :08082 the same lock key.

Availability is checked while holding the startup lock. Existing unrelated
or unready listeners cause an error, and are never killed. A backend is
started only when TCP connection refusal establishes an available port.
Successful authenticated readiness releases the started process to continue;
failure or timeout stops and reaps only the process started by that call.
The GUI never uses process-name-wide termination for this operation.

The GUI can reuse an authenticated node without a local executable. If it
must start a node, the normal executable lookup applies. Launched node
processes bind explicitly to loopback. The native wallet starts the canonical
run node (without mining), because signed transfers require a writable node.
The two browser launchers retain the read-only api command. Wallet send checks
authenticated apiReadOnly=false before signing and never restarts an existing
read-only node. Unknown mode is insufficient for a transfer.
Windows launch is hidden. Wallet API logs remain local to the node folder.

Deploy the GUI and node from the same validated source/release set. Old
binaries without the health version marker are reported as incompatible;
they are not automatically replaced or restarted. Source acceptance does
not establish release provenance or update an existing deployed binary.
