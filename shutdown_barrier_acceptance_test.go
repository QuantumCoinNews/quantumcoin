package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	qnode "quantumcoin/node"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestShutdownCLIExitBarrierInstalled(t *testing.T) {
	source := readLifecycleSource(t, "node_app_cli.go")
	if !strings.Contains(source, "defer shutdownBarrier.Wait()") {
		t.Fatal("CLI may return when a listener closes before graceful shutdown completes")
	}
	if strings.Contains(source, "go trapAPIShutdown()") || strings.Contains(source, "go trapAndShutdown()") {
		t.Fatal("uncoordinated signal trap remains")
	}
}

func TestShutdownCLIWaitsForActiveHTTPResponse(t *testing.T) {
	previous := nodeRuntime
	defer func() { nodeRuntime = previous }()
	r := qnode.NewRuntime(nil, nil)
	nodeRuntime = r
	entered, release, closing := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = io.WriteString(w, "completed")
	}))
	server.Config.RegisterOnShutdown(func() { close(closing) })
	r.SetHTTPServer(server.Config)
	server.Start()
	defer func() { releaseOnce.Do(func() { close(release) }); server.Close() }()
	responseDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		res, err := client.Get(server.URL)
		if err == nil {
			defer res.Body.Close()
			var data []byte
			data, err = io.ReadAll(res.Body)
			if err == nil && (res.StatusCode != 200 || string(data) != "completed") {
				err = errors.New("response truncated")
			}
		}
		responseDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("request never entered")
	}
	b := newCLIShutdownBarrierV1()
	if !b.Begin() {
		t.Fatal("shutdown not admitted")
	}
	shutdownResult := make(chan error, 1)
	go func() {
		defer b.Finish()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		shutdownResult <- shutdownAPIHTTPServer(ctx)
	}()
	select {
	case <-closing:
	case <-time.After(time.Second):
		t.Fatal("listener did not begin shutdown")
	}
	cliReturned := make(chan struct{})
	go func() { b.Wait(); close(cliReturned) }()
	select {
	case <-cliReturned:
		t.Fatal("CLI returned before active response drained")
	case <-time.After(40 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	if err := <-responseDone; err != nil {
		t.Fatal(err)
	}
	if err := <-shutdownResult; err != nil {
		t.Fatal(err)
	}
	select {
	case <-cliReturned:
	case <-time.After(time.Second):
		t.Fatal("CLI did not finish after drain")
	}
}

func TestShutdownBarrierCancellationAndConcurrentSignal(t *testing.T) {
	b := newCLIShutdownBarrierV1()
	b.Wait()
	b.Wait()
	if b.Begin() {
		t.Fatal("failed startup admitted late shutdown")
	}
	select {
	case <-b.stop:
	default:
		t.Fatal("unused trap not canceled")
	}
	b = newCLIShutdownBarrierV1()
	var winners atomic.Int32
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			if b.Begin() {
				winners.Add(1)
				b.Finish()
			}
		})
	}
	workers.Wait()
	b.Wait()
	if winners.Load() != 1 {
		t.Fatal("multiple shutdown owners", winners.Load())
	}
}

func TestShutdownAPIContextDeadlineIsReported(t *testing.T) {
	previous := nodeRuntime
	defer func() { nodeRuntime = previous }()
	entered, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { close(entered); <-release }))
	defer server.Close()
	defer close(release)
	nodeRuntime = qnode.NewRuntime(nil, nil)
	nodeRuntime.SetHTTPServer(server.Config)
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		res, err := http.Get(server.URL)
		if err == nil {
			_ = res.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("request not active")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := shutdownAPIHTTPServer(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("deadline hidden", err)
	}
}
