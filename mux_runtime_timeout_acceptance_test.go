package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	qnode "quantumcoin/node"
)

func muxSecurityRuntimeHTTPV1(t *testing.T) string {
	t.Helper()
	externalMiningMux(t)
	runtime := nodeRuntime
	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reserved.Addr().String()
	_ = reserved.Close()
	options := qnode.DefaultHTTPServerOptionsV1()
	options.ReadHeaderTimeout = 100 * time.Millisecond
	options.ReadTimeout = 300 * time.Millisecond
	options.WriteTimeout = 2 * time.Second
	options.IdleTimeout = time.Second
	done := make(chan error, 1)
	h := newHTTPAPIHandlerV1(NewHTTPGuard(), NewEndpointLimiter())
	go func() { done <- runtime.ServeHTTP(address, h, options) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := runtime.Shutdown(ctx); err != nil {
			t.Error("test runtime cleanup", err)
		}
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-ctx.Done():
			t.Error("test HTTP server did not exit")
		}
	})
	client := &http.Client{Timeout: 300 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for {
		req, err := http.NewRequest("GET", "http://"+address+"/api/health", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
		response, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatal("runtime health", response.StatusCode)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("runtime HTTP not ready", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	server := runtime.HTTP()
	if server == nil || server.ReadHeaderTimeout != options.ReadHeaderTimeout || server.ReadTimeout != options.ReadTimeout || server.MaxHeaderBytes != options.MaxHeaderBytes {
		t.Fatal("production runtime did not apply test transport budgets")
	}
	return address
}

func TestMuxSecurityRuntimeReadTimeoutsOverTCP(t *testing.T) {
	address := muxSecurityRuntimeHTTPV1(t)
	chain, _ := nodeRuntime.Snapshot()
	disk, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"slow-header", "slow-body"} {
		t.Run(kind, func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", address, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			if kind == "slow-header" {
				_, err = io.WriteString(conn, "GET /api/blocks HTTP/1.1\r\nHost: localhost\r\nX-Incomplete: ")
			} else {
				_, err = fmt.Fprintf(conn, "POST /api/tx/send HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: application/json\r\nX-QC-API-KEY: %s\r\nContent-Length: 64\r\n\r\n{\"tra", actualMuxTestTokenV1)
			}
			if err != nil {
				t.Fatal(err)
			}
			response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "POST"})
			if err != nil {
				var networkError net.Error
				if errors.As(err, &networkError) && networkError.Timeout() {
					t.Fatal("client deadline fired; server timeout was not observed", err)
				}
				if kind == "slow-body" {
					t.Fatal("application body timeout lost its JSON response", err)
				}
				return // Server closed an incomplete header before dispatch.
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "slow-body" {
				assertRoutingJSONErrorV1(t, response.StatusCode, response.Header, body, 400)
			} else if response.StatusCode != 400 && response.StatusCode != 408 {
				t.Fatal("incomplete header reached application", response.StatusCode)
			}
		})
	}
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, after) || chain.BestHeightSnapshot() != 0 || chain.PendingTransactionCountSnapshot() != 0 {
		t.Fatal("timed-out request mutated chain")
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://"+address+"/api/blocks", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("healthy request failed after timeouts", response.StatusCode)
	}
}
