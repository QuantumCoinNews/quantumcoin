# API credentials and bind policy

Protected node routes require a nonempty configured QC_API_TOKEN in every mode,
including dev, numeric loopback, local-release, testnet and mainnet. Credentials
are accepted only from X-QC-API-KEY or Authorization: Bearer headers. Query,
form, JSON and cookie values are not fallback credential sources; authentication
does not read the body. A valid header remains independent of URL/body content.

The root gate and local miner-control gate use the same header parser:

* Each supported header may occur once. Duplicate values, casing aliases and
  explicit empty/nil-valued headers are rejected, even if another credential is
  valid. Identical duplicate lines are also rejected.
* Authorization must use the Bearer scheme, case-insensitively. A malformed
  supplied Authorization header cannot be hidden behind a valid API-key header.
* Both distinct supported headers may be sent together only with the same token.
* The supplied token is compared with the configured token in constant time.
  Error responses never echo the credential. Private response no-store and Vary
  headers remain in force.

Only numeric loopback binds qualify for local CORS behavior. Wildcard IPv4/IPv6,
non-loopback IPs, hostnames and unparseable bind strings receive restrictive
policy regardless of network labels. Any public-testnet flag can tighten a
loopback policy. An exposed API cannot start without QC_API_TOKEN; this check
runs before Runtime.ServeHTTP. Bind syntax/availability is still validated by
the underlying listener, not by the credential check.

Exposed browser origins require an exact allowlist. A wildcard entry is ignored;
untrusted origins cannot read or mutate even with a valid credential. Native
clients may omit Origin but must authenticate. CORS preflight grants no token or
application permission. Existing local miner-control restrictions are preserved.

Headers prevent URL/body credential fallback; they do not encrypt a connection.
Remote credential-bearing traffic needs HTTPS or an appropriate TLS proxy setup.
This task deploys no public service/TLS proxy and rotates no credentials.

Verification covers dev/network/bind combinations, body-not-read tests,
duplicate headers over real loopback HTTP, public-policy CORS, health/miner/
local-wallet clients and unchanged read-authorization behavior. Existing Python
QC bridge sources send the normal single API-key header; no Python dependency
installation or live platform call is part of this Go parser change.
