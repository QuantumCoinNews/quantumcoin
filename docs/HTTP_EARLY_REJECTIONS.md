# Early HTTP rejection delivery

The canonical root router wraps the complete guard/CORS/capability stack with
`withEarlyRejectionBodyDrainV1`. Authentication, capability and rate-limit
decisions happen first. The wrapper preserves their status and JSON response;
it does not dispatch a rejected action or retry any transaction/mining request.

For an HTTP/1 connection marked `Connection: close`, Windows can otherwise
discard an early error response when unread request bytes cause a TCP reset.
Before committing a final error status, the wrapper discards an explicitly
sized body only when it is within the existing API cap of 1 MiB. Bytes go to
`io.Discard`; there is no body-sized buffer, secret logging or persistence.

The transport read deadline is at most 200 ms after handler dispatch and is
shortened to an earlier request-context deadline when present. The deadline
is not cleared. This path applies only to closing connections, so it does not
change the next request's deadline on a keep-alive connection. The production
server retains its normal header/read/write deadlines and body/rate limits.

Unknown-length/chunked bodies, oversized bodies, canceled requests, writers
without transport-deadline support and `Expect` handshakes do not enter this
discard path. In particular, rejecting `Expect: 100-continue` does not solicit
a body with an interim 100 response. A stalled, malformed or out-of-budget
sender cannot obtain an unlimited read; delivery after such a transport
failure is best effort. None of these cases makes a denied action executable.

Real TCP regression tests separate header/body writes to reproduce the old
reset, cover 401/403/429, verify the exact API body-size cap and check stalled
senders/Expect behaviour. Unit tests verify skip conditions and bounded reads.
The rebuilt executable is also checked with Python urllib's closing
connections without automatic retries and with a keep-alive client.
