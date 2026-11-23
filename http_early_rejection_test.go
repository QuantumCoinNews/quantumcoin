package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type earlyRejectionCountBody struct {
	io.Reader
	bytesRead int
}

func (b *earlyRejectionCountBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.bytesRead += n
	return n, err
}
func (b *earlyRejectionCountBody) Close() error { return nil }

type earlyRejectionDeadlineWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
	err      error
}

func (w *earlyRejectionDeadlineWriter) SetReadDeadline(deadline time.Time) error {
	w.deadline = deadline
	return w.err
}

func TestEarlyHTTPRejectionBodyAndDeadlineBudgets(t *testing.T) {
	for _, row := range []struct {
		name                                                         string
		length                                                       int64
		close, expect, chunked, canceled, unsupported, deadlineError bool
		status, wantRead                                             int
	}{
		{name: "small", length: 2, close: true, status: 403, wantRead: 2},
		{name: "exact_limit", length: earlyRejectionBodyLimitV1, close: true, status: 401, wantRead: int(earlyRejectionBodyLimitV1)},
		{name: "over_limit", length: earlyRejectionBodyLimitV1 + 1, close: true, status: 413},
		{name: "unknown", length: -1, close: true, status: 403},
		{name: "empty", length: 0, close: true, status: 403},
		{name: "chunked", length: 2, close: true, chunked: true, status: 403},
		{name: "expect_continue", length: 2, close: true, expect: true, status: 403},
		{name: "keep_alive", length: 2, status: 403},
		{name: "success", length: 2, close: true, status: 200},
		{name: "redirect", length: 2, close: true, status: 302},
		{name: "canceled", length: 2, close: true, canceled: true, status: 403},
		{name: "no_transport_deadline", length: 2, close: true, unsupported: true, status: 403},
		{name: "deadline_failure", length: 2, close: true, deadlineError: true, status: 403},
	} {
		t.Run(row.name, func(t *testing.T) {
			body := &earlyRejectionCountBody{Reader: strings.NewReader(strings.Repeat("x", int(earlyRejectionBodyLimitV1)+10))}
			req := httptest.NewRequest("POST", "http://localhost/denied", nil)
			req.Body = body
			req.ContentLength = row.length
			req.Close = row.close
			if row.expect {
				req.Header.Set("Expect", "100-continue")
			}
			if row.chunked {
				req.TransferEncoding = []string{"chunked"}
			}
			if row.canceled {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			writer := &earlyRejectionDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
			if row.deadlineError {
				writer.err = http.ErrNotSupported
			}
			var output http.ResponseWriter = writer
			if row.unsupported {
				output = writer.ResponseRecorder
			}
			withEarlyRejectionBodyDrainV1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeError(w, row.status, "expected test response") })).ServeHTTP(output, req)
			if body.bytesRead != row.wantRead || writer.Code != row.status || !json.Valid(writer.Body.Bytes()) {
				t.Fatalf("read=%d want=%d status=%d", body.bytesRead, row.wantRead, writer.Code)
			}
			if row.wantRead > 0 && (writer.deadline.IsZero() || writer.deadline.After(time.Now().Add(earlyRejectionReadBudgetV1))) {
				t.Fatal("missing or unbounded read deadline")
			}
		})
	}
}

func TestEarlyHTTPRejectionKeepsEarlierContextDeadline(t *testing.T) {
	req := httptest.NewRequest("POST", "http://localhost/denied", strings.NewReader("{}"))
	req.Close = true
	deadline := time.Now().Add(50 * time.Millisecond)
	ctx, cancel := context.WithDeadline(req.Context(), deadline)
	defer cancel()
	req = req.WithContext(ctx)
	writer := &earlyRejectionDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	withEarlyRejectionBodyDrainV1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeError(w, 403, "denied") })).ServeHTTP(writer, req)
	if !writer.deadline.Equal(deadline) {
		t.Fatal("earlier request deadline was extended")
	}
}

func TestEarlyHTTPRejectionSlowBodyAndExpectDoNotHoldConnection(t *testing.T) {
	t.Chdir(t.TempDir())
	server := httptest.NewServer(readOnlyMuxFixtureV1(t))
	defer server.Close()
	for _, expect := range []bool{false, true} {
		t.Run(fmt.Sprintf("expect=%t", expect), func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", strings.TrimPrefix(server.URL, "http://"), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
			extra := ""
			if expect {
				extra = "Expect: 100-continue\r\n"
			}
			started := time.Now()
			_, err = fmt.Fprintf(conn, "POST /api/mine HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Length: 2\r\nX-QC-API-KEY: %s\r\n%s\r\n", actualMuxTestTokenV1, extra)
			if err != nil {
				t.Fatal(err)
			}
			// Never send a body. An Expect client must not be invited to send it;
			// a stalled sender must not hold the rejection beyond the short budget.
			response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "POST"})
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != 403 || !json.Valid(body) || time.Since(started) > 1500*time.Millisecond {
				t.Fatalf("rejection stalled or solicited a body: status=%d elapsed=%s err=%v", response.StatusCode, time.Since(started), err)
			}
		})
	}
}

func TestEarlyHTTPRejectionPreservesRateLimitOverTCP(t *testing.T) {
	t.Chdir(t.TempDir())
	guard := NewHTTPGuard()
	for i := 0; i < httpMaxRequestsPerIPInWindow; i++ {
		if err := guard.allow("127.0.0.1"); err != nil {
			t.Fatal(err)
		}
	}
	var called atomic.Bool
	server := httptest.NewServer(withEarlyRejectionBodyDrainV1(guard.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called.Store(true); w.WriteHeader(204) }))))
	defer server.Close()
	conn, err := net.DialTimeout("tcp", strings.TrimPrefix(server.URL, "http://"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, err = io.WriteString(conn, "POST /denied HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Length: 2\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(25 * time.Millisecond)
	if _, err = io.WriteString(conn, "{}"); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "POST"})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 429 || !json.Valid(body) || called.Load() {
		t.Fatal("rate limit changed or response lost", err)
	}
}

// Header/body writes are deliberately separated. Python urllib uses a closing
// connection and can send a small body after the early rejection has started.
// A ResponseRecorder cannot detect the resulting Windows TCP reset.
func TestEarlyHTTPRejectionHandlesAPIAllowedSizedBodyOverTCP(t *testing.T) {
	t.Chdir(t.TempDir())
	server := httptest.NewServer(readOnlyMuxFixtureV1(t))
	defer server.Close()
	for _, size := range []int{4097, 64 << 10, int(apiMaxBodyBytes)} {
		t.Run(fmt.Sprintf("bytes=%d", size), func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", strings.TrimPrefix(server.URL, "http://"), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
			_, err = fmt.Fprintf(conn, "POST /api/mine HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Length: %d\r\nX-QC-API-KEY: %s\r\n\r\n", size, actualMuxTestTokenV1)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.WriteString(conn, strings.Repeat("x", size)); err != nil {
				t.Fatal(err)
			}
			response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "POST"})
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != 403 || !json.Valid(body) {
				t.Fatal("allowed-sized body lost rejection", err)
			}
		})
	}
}

func TestEarlyHTTPRejectionConnectionCloseDeliversJSONOverTCP(t *testing.T) {
	t.Chdir(t.TempDir())
	server := httptest.NewServer(readOnlyMuxFixtureV1(t))
	defer server.Close()
	for _, row := range []struct {
		path, token string
		status      int
	}{
		{"/api/mine", actualMuxTestTokenV1, 403},
		{"/api/tx/send", "", 401},
		{"/api/miner/start", actualMuxTestTokenV1, 403},
	} {
		t.Run(row.path, func(t *testing.T) {
			for attempt := 0; attempt < 5; attempt++ {
				conn, err := net.DialTimeout("tcp", strings.TrimPrefix(server.URL, "http://"), time.Second)
				if err != nil {
					t.Fatal(err)
				}
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				_, err = fmt.Fprintf(conn, "POST %s HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Length: 2\r\nContent-Type: application/json\r\nX-QC-API-KEY: %s\r\n\r\n", row.path, row.token)
				if err == nil {
					time.Sleep(25 * time.Millisecond)
					_, err = io.WriteString(conn, "{}")
				}
				if err != nil {
					conn.Close()
					t.Fatal("body delivery failed", err)
				}
				response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodPost})
				if err != nil {
					conn.Close()
					t.Fatal("early rejection lost instead of an HTTP response", err)
				}
				body, err := io.ReadAll(response.Body)
				response.Body.Close()
				conn.Close()
				if err != nil || response.StatusCode != row.status || !json.Valid(body) {
					t.Fatalf("status=%d expected=%d validJSON=%v err=%v", response.StatusCode, row.status, json.Valid(body), err)
				}
			}
		})
	}
}
