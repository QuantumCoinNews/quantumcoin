# Monitoring and live security evidence

Canonical bootstrap loads a monitoring engine at
`<absolute chain filename>.monitoring/state.json`. Startup reads without creating
files. Runtime events, rejected HTTP requests and autosave security polling feed
the sink. API read-only mode retains observations in memory without writing the
monitoring file; operator mutations are forbidden. Bare runtime adapters have an
in-memory engine only.

Events require a nonempty ID, a valid RFC3339 timestamp, source, category, message
and a recognized severity. Documented aliases are normalized; unknown severities
remain invalid and are rejected. Invalid events classified directly are flagged
critical for review, never silently downgraded to INFO.

Incidents are grouped by source, category and subject. Warning/critical events
open incidents. Repeated observations increment occurrences without creating a
notification storm. Operators acknowledge or resolve incidents; a new occurrence
after resolution reopens them. Escalation from warning to critical clears the
earlier acknowledgement. Actor, note and timestamps record operator actions.
Runbook instructions and the advisory boundary accompany each incident.

The local notification outbox stores opening, escalation, acknowledgement,
resolution and recurrence transitions. It is available through the protected
monitoring API for an operator/dashboard to poll. Consuming a notification means
the operator read it; it does **not** assert delivery to email, Telegram, Slack or
another external provider. No external provider is configured or contacted.

## API

- `GET /api/monitoring`: bounded event/incident/outbox snapshot, error visibility.
  Requires the existing API header credential.
- `POST /api/monitoring/action`: body
  `{"id":"incident-or-notification-id","action":"ack|resolve|consume","note":"..."}`.
  Requires both the normal API credential and an additional
  `X-QC-MONITORING-ADMIN` header matching server-side `QC_MONITORING_ADMIN_TOKEN`.
  An unset operator token disables mutations. Actor is assigned by the server
  as `node-operator`; the body cannot claim an arbitrary identity.

Actions accept at most 4 KiB, reject unknown fields and trailing JSON/data, and
are rate limited. Acknowledgement is separate from resolution. `consume` applies
to a notification ID. These operations never authorize funds, mining or bans.

## Persistence and bounds

The owning engine serializes mutations, writes a unique temporary file, flushes
and closes it, then replaces the state file. Failed persistence leaves the last
committed state intact. Corrupt/oversized startup state blocks writes until the
operator preserves/repairs it and restarts; it is never silently overwritten.
Operate one owning runtime per chain; this store is not a distributed database.

There are at most 1,000 events, 256 incidents and 512 notifications. Dropped event
history is counted. Resolved incidents and consumed notifications can be evicted;
open incidents and unconsumed notifications are not silently discarded. Capacity
errors are reported, and the proposed mutation does not partially commit. Input
lengths/metadata and an 8 MiB state read/write limit bound allocation. This is a
bounded operational store, not an immutable or unlimited audit archive.

Event IDs are deduplicated within retained history and the latest event of each
retained incident. HTTP failure observations debounce by actual client IP,
category and minute. Analysis polling uses process epoch plus evidence revision
so unchanged evidence does not reopen a resolved incident merely on the next
poll. A new process or new evidence has a distinct identity.

## Actual runtime evidence

HTTP observations wrap the full canonical handler, including authentication and
rate-limit rejections. Client identity comes from `RemoteAddr`, never a supplied
forwarding header. The bounded 60-second sample holds at most 2,000 requests.
Action-burst, IP/API reputation and API-surface guardrail reviews consume it.
No request body, token, private key, query string or untrusted GUI claims are
stored. GUI/client internals are not inferred from server requests.

Canonical transaction rejections from API, CLI, peer and direct entry paths feed
the last 256 spend-conflict observations. Miner lifecycle/error events and the
actual stop-channel/running state feed miner-safety review. Node-sync drift,
admin alerts and the existing wallet/mempool analyses are included in shared
dashboard/admin/AI report construction; sections are no longer omitted.

These analyzers are advisory reviews. Existing signature/UTXO/duplicate checks,
HTTP rate limiting/auth and P2P security remain the enforcing mechanisms. Review
maps do not introduce economic thresholds, automatic bans or fee changes.

## Network view

Peer count comes from the live P2P registry. Separate fields show completed
handshakes and observed peer chains. Heights are recorded only after peer-chain
validation against local policy, never from an unauthenticated height claim.
Successfully validated/accepted peer blocks also refresh their sender's height.
Disconnected observations disappear; heights older than ten minutes expire.

The highest fresh validated peer-chain height is a local observed network view,
not a global quorum or finality proof. If unavailable, `networkHeightKnown=false`
and node-health height/drift fields are null. No synthetic zero or substitution
of local height is presented as a measured network height. A connected peer with
unknown height is reported as unknown, not isolated. Finality also exposes the
availability of peer-height evidence. Genesis height zero can be valid evidence.

Security dashboard, admin report, AI report, finality and node-health use this
runtime view. A stale head or insufficient depth may still legitimately produce
WATCH even when peers are connected.
