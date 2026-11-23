package main

import (
	"io"
	"net/http"
	"time"
)

// Reuse the existing API cap so every explicitly sized, allowed payload can
// take the bounded compatibility path without a second, smaller size policy.
const earlyRejectionBodyLimitV1 int64 = apiMaxBodyBytes
const earlyRejectionReadBudgetV1 = 200 * time.Millisecond

// A closing HTTP/1 connection may otherwise be reset with unread request bytes,
// hiding an early 401/403/429 response on Windows. Only a small, explicitly
// sized body is discarded, after the rejection decision. Nothing is replayed.
// Unknown/large bodies and Expect handshakes retain the normal fail-closed path.
func withEarlyRejectionBodyDrainV1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline := time.Now().Add(earlyRejectionReadBudgetV1)
		if contextDeadline, ok := r.Context().Deadline(); ok && contextDeadline.Before(deadline) {
			deadline = contextDeadline
		}
		next.ServeHTTP(&earlyRejectionWriterV1{ResponseWriter: w, request: r, readDeadline: deadline}, r)
	})
}

type earlyRejectionWriterV1 struct {
	http.ResponseWriter
	request      *http.Request
	readDeadline time.Time
	wroteHeader  bool
}

func (w *earlyRejectionWriterV1) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *earlyRejectionWriterV1) WriteHeader(status int) {
	if !w.wroteHeader && (status >= 200 || status == http.StatusSwitchingProtocols) {
		w.wroteHeader = true
		if status >= 400 {
			w.discardSmallClosingBody()
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *earlyRejectionWriterV1) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *earlyRejectionWriterV1) discardSmallClosingBody() {
	r := w.request
	if r == nil || r.ProtoMajor != 1 || !r.Close || r.Body == nil ||
		r.ContentLength <= 0 || r.ContentLength > earlyRejectionBodyLimitV1 ||
		len(r.TransferEncoding) != 0 || r.Header.Get("Expect") != "" || r.Context().Err() != nil {
		return
	}
	// Do not clear the transport read deadline or affect a reusable connection.
	// Capture the short deadline at dispatch, not after an expensive handler.
	// Writers without transport deadline support must never cause a blocking read.
	if err := http.NewResponseController(w.ResponseWriter).SetReadDeadline(w.readDeadline); err != nil {
		return
	}
	_, _ = io.CopyN(io.Discard, r.Body, r.ContentLength)
}
