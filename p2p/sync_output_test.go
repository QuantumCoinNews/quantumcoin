package p2p

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type syncOutputErrorReader struct{ err error }

func (r syncOutputErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestSyncNodeOutputScannerReportsReadFailure(t *testing.T) {
	failure := errors.New("test-only stdout failure")
	ready := make(chan string, 1)
	output := io.MultiReader(strings.NewReader("READY 127.0.0.1:1234\n"), syncOutputErrorReader{failure})
	if err := scanSyncNodeReadiness(output, ready); !errors.Is(err, failure) {
		t.Fatal("scanner hid child output failure", err)
	}
	if address := <-ready; address != "127.0.0.1:1234" {
		t.Fatal("readiness was lost before failure", address)
	}
}
