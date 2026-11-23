package node

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPHeaderLimitOverTCP(t *testing.T) {
	for _, limit := range []int{0, 4096} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			options := DefaultHTTPServerOptionsV1()
			options.MaxHeaderBytes = limit
			var reached atomic.Int32
			server := newHTTPServerV1("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1); w.WriteHeader(204) }), options)
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() {
				_ = server.Close()
				if err := <-done; err != http.ErrServerClosed {
					t.Errorf("serve: %v", err)
				}
			})
			for _, over := range []bool{false, true} {
				size := 1024
				want := 204
				if over {
					// net/http allows a bounded 4096-byte parser buffer beyond MaxHeaderBytes.
					// Test safely above that allowance rather than assuming an exact byte cut.
					size = effectiveHTTPMaxHeaderBytesV1(limit) + 8192
					want = 431
				}
				conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
				if err != nil {
					t.Fatal(err)
				}
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				// A large header may cause a write error when the server closes early;
				// the response, not the write result, is the acceptance criterion.
				_, _ = fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nX-Padding: %s\r\nConnection: close\r\n\r\n", strings.Repeat("a", size))
				response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "GET"})
				if err != nil {
					_ = conn.Close()
					t.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
				_ = conn.Close()
				if response.StatusCode != want {
					t.Fatalf("over=%v status=%d want=%d", over, response.StatusCode, want)
				}
			}
			if reached.Load() != 1 {
				t.Fatalf("oversized header reached handler: calls=%d", reached.Load())
			}
		})
	}
}
