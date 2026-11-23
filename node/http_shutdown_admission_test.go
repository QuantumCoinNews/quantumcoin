package node

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestShutdownRejectsLateHTTPStart(t *testing.T) {
	r := NewRuntime(nil, nil)
	_ = r.Shutdown(context.Background()) // No chain is expected; admission must still close.
	done := make(chan error, 1)
	go func() {
		done <- r.ServeHTTP("127.0.0.1:0", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), DefaultHTTPServerOptionsV1())
	}()
	select {
	case err := <-done:
		if !errors.Is(err, ErrRuntimeShuttingDownV1) {
			t.Fatal("late HTTP start not rejected", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("HTTP listener started after runtime shutdown")
		if server := r.HTTP(); server != nil {
			_ = server.Close()
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("test listener did not close")
		}
	}
}
