package apiclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestBackendCoordinatorCleansOwnedFailedProcess(t *testing.T) {
	if port := os.Getenv("QC_TEST_GUI_FAILED_PORT"); port != "" {
		listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
		if err != nil {
			t.Fatal(err)
		}
		server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
		})}
		defer server.Close()
		_ = server.Serve(listener)
		return
	}
	port := testFreeAPIPort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	var cmd *exec.Cmd
	stopped, released := false, false
	err := EnsureLocalBackendV1(ctx, port, "", func() (BackendHandleV1, error) {
		cmd = exec.Command(os.Args[0], "-test.run=^TestBackendCoordinatorCleansOwnedFailedProcess$")
		cmd.Env = append(os.Environ(), "QC_TEST_GUI_FAILED_PORT="+port)
		if err := cmd.Start(); err != nil {
			return BackendHandleV1{}, err
		}
		return BackendHandleV1{Stop: func() { stopped = true; _ = cmd.Process.Kill(); _ = cmd.Wait() }, Release: func() error { released = true; return cmd.Process.Release() }}, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) || !stopped || released || cmd == nil || cmd.ProcessState == nil {
		t.Fatalf("failed child was not reaped: %v stopped=%t released=%t", err, stopped, released)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatal("owned failed child retained port", err)
	}
	_ = listener.Close()
}

func TestStartupLeaseSurvivesAgeAndReleasesAfterProcessExit(t *testing.T) {
	if dir := os.Getenv("QC_TEST_GUI_LEASE_DIR"); dir != "" {
		lease, err := acquireStartupLeaseAtV1(context.Background(), dir, "8082")
		if err != nil {
			t.Fatal(err)
		}
		defer lease.Release()
		fmt.Println("LEASE_READY")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStartupLeaseSurvivesAgeAndReleasesAfterProcessExit$")
	cmd.Env = append(os.Environ(), "QC_TEST_GUI_LEASE_DIR="+dir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "LEASE_READY\n" {
		t.Fatalf("child lease readiness %q %v", line, err)
	}
	old := time.Now().Add(-24 * time.Hour)
	if err = os.Chtimes(filepath.Join(dir, "quantumcoin-api-8082.start.guard"), old, old); err != nil {
		t.Fatal(err)
	}
	waiting, cancelWait := context.WithTimeout(ctx, 150*time.Millisecond)
	lease, err := acquireStartupLeaseAtV1(waiting, dir, ":08082")
	cancelWait()
	if lease != nil {
		_ = lease.Release()
		t.Fatal("live lease was stolen by age")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock wait: %v", err)
	}
	if err = cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	lease, err = acquireStartupLeaseAtV1(ctx, dir, "8082")
	if err != nil {
		t.Fatalf("crashed owner retained lease: %v", err)
	}
	if err = lease.Release(); err != nil {
		t.Fatal(err)
	}
	if err = lease.Release(); err != nil {
		t.Fatal("release is not idempotent", err)
	}
}
