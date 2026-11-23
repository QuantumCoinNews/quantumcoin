package apiclient

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestBackendHealthyChildIsSharedAndExitsCleanly(t *testing.T) {
	if port := os.Getenv("QC_TEST_SHARED_HEALTHY_PORT"); port != "" {
		listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
		if err != nil {
			t.Fatal(err)
		}
		server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != CanonicalHealthPathV1 {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"ok":true,"service":%q,"healthVersion":%q}`, HealthServiceV1, HealthProtocolV1)
		})}
		done := make(chan error, 1)
		go func() { done <- server.Serve(listener) }()
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Error(err)
		}
		if err := <-done; err != http.ErrServerClosed {
			t.Error(err)
		}
		return
	}
	port := testFreeAPIPort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBackendHealthyChildIsSharedAndExitsCleanly$")
	cmd.Env = append(os.Environ(), "QC_TEST_SHARED_HEALTHY_PORT="+port)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	starts, stops, releases := 0, 0, 0
	childDone := make(chan error, 1)
	err = EnsureLocalBackendV1(ctx, port, "", func() (BackendHandleV1, error) {
		starts++
		if err := cmd.Start(); err != nil {
			return BackendHandleV1{}, err
		}
		go func() { childDone <- cmd.Wait() }()
		return BackendHandleV1{Stop: func() { stops++; _ = cmd.Process.Kill() }, Release: func() error { releases++; return nil }}, nil
	})
	if cmd.Process != nil {
		defer cmd.Process.Kill()
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureLocalBackendV1(ctx, port, "", func() (BackendHandleV1, error) {
		starts++
		return BackendHandleV1{}, fmt.Errorf("second GUI must reuse healthy child")
	}); err != nil {
		t.Fatal(err)
	}
	if starts != 1 || stops != 0 || releases != 1 {
		t.Fatalf("start/stop/release=%d/%d/%d", starts, stops, releases)
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-childDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("healthy child did not exit after controlled stop")
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatal("child retained port", err)
	}
	_ = listener.Close()
}
