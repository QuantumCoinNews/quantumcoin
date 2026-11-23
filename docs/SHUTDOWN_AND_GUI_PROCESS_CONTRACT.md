# Shutdown and GUI backend lifetime

The canonical node runtime owns mining cancellation, P2P closure, HTTP drain and
the final chain save. Shutdown closes new mining admission before stopping
services. HTTP startup also checks the shutdown state while registering its
server; a runtime that has shut down cannot reopen an HTTP listener.

The CLI signal worker starts a shutdown barrier before closing listeners. If
HTTP/P2P serving returns while shutdown is in progress, the CLI waits rather
than returning from main and terminating active requests or final persistence.
Read-only `api` mode drains HTTP only and does not final-save or enable mining.
Full node mode keeps its existing shutdown error/exit handling. A failed startup
that never began shutdown cancels its unused signal waiter. Ctrl+C triggers the
existing interrupt handling; main.go remains the unchanged delegating entry.

HTTP Shutdown waits for active requests until its context deadline. A timeout is
reported as an error; it is not a successful graceful drain. Existing transport
timeouts and runtime mining cancellation remain in force. This contract does not
promise that failed storage hardware or arbitrary user handler code terminates
within the HTTP context budget.

GUI startup uses the shared `apiclient.EnsureLocalBackendV1` coordinator:

* All launchers use the canonical authenticated `/api/health` protocol and port.
* An OS-owned startup lease serializes concurrent launchers. Old file timestamps
  cannot steal a live lease; process exit releases ownership.
* An occupied port with an unrelated, unready or unauthorized service is an
  error. The GUI does not kill that process or start a duplicate on the port.
* Startup failure cleans up only the child started by that attempt, including
  waiting/reaping it and closing the wallet's log handle.
* A healthy backend is a shared, independent node service. Closing a GUI does
  not stop it or another GUI's node. The native wallet stops its own local mining
  worker on window close. Browser launchers hand off a healthy process; the
  native wallet keeps a background Wait to reap it and close its log when it exits.

Healthy persistence is intentional service lifetime, not a failed-start child
left running. Changing to a GUI-owned backend that stops with the last window
would require a separate multi-client ownership policy. No process-name-wide,
port-owner or unrelated-window termination is used here.

Verification includes real HTTP draining/deadline tests, node mining/autosave/
shutdown regression tests, Windows child-process cleanup and healthy reuse,
lease crash recovery and Windows GUI package compilation. Graphical desktop
interaction and signed release acceptance are separate from these code tests.
