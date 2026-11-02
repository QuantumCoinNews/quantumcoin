package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	qnode "quantumcoin/node"
)

func readLifecycleSource(
	t *testing.T,
	path string,
) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read %s: %v",
			path,
			err,
		)
	}

	return string(data)
}

func TestAPICommandUsesDedicatedGracefulShutdown(t *testing.T) {
	source :=
		readLifecycleSource(
			t,
			"main.go",
		)

	start :=
		strings.Index(
			source,
			`case "api":`,
		)

	end :=
		strings.Index(
			source,
			`case "connect":`,
		)

	if start < 0 ||
		end <= start {
		t.Fatal(
			"api command region not found",
		)
	}

	region := source[start:end]

	for _, required := range []string{
		"apiReadOnlyMode = true",
		"go trapAPIShutdown()",
		"startHTTPAPI()",
	} {
		if !strings.Contains(
			region,
			required,
		) {
			t.Fatalf(
				"api command missing %q",
				required,
			)
		}
	}

	if strings.Contains(
		region,
		"go trapAndShutdown()",
	) {
		t.Fatal(
			"api command uses full node shutdown trap",
		)
	}

	if strings.Index(
		region,
		"go trapAPIShutdown()",
	) > strings.Index(
		region,
		"startHTTPAPI()",
	) {
		t.Fatal(
			"API shutdown trap starts after HTTP serving",
		)
	}
}

func TestAPIHTTPShutdownHelperGracefullyStopsServer(t *testing.T) {
	oldRuntime := nodeRuntime

	runtime := &qnode.Runtime{}
	server := &http.Server{
		Handler: http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.WriteHeader(
					http.StatusNoContent,
				)
			},
		),
	}

	listener, err :=
		net.Listen(
			"tcp",
			"127.0.0.1:0",
		)
	if err != nil {
		t.Fatalf(
			"listen: %v",
			err,
		)
	}

	runtime.SetHTTPServer(server)
	nodeRuntime = runtime

	defer func() {
		nodeRuntime = oldRuntime
		_ = listener.Close()
	}()

	serveDone := make(chan error, 1)

	go func() {
		serveDone <- server.Serve(
			listener,
		)
	}()

	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	url :=
		"http://" +
			listener.Addr().String() +
			"/"

	readyDeadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for {
		resp, requestErr :=
			client.Get(url)

		if requestErr == nil {
			_ = resp.Body.Close()
			break
		}

		if time.Now().After(
			readyDeadline,
		) {
			t.Fatalf(
				"test HTTP server never became ready: %v",
				requestErr,
			)
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
	defer cancel()

	if err :=
		shutdownAPIHTTPServer(ctx); err != nil {
		t.Fatalf(
			"graceful API shutdown: %v",
			err,
		)
	}

	select {
	case err := <-serveDone:
		if !errors.Is(
			err,
			http.ErrServerClosed,
		) {
			t.Fatalf(
				"Serve returned %v, want ErrServerClosed",
				err,
			)
		}

	case <-time.After(
		2 * time.Second,
	):
		t.Fatal(
			"HTTP server did not stop after graceful shutdown",
		)
	}
}

func TestBrowserGUILaunchersSerializeBackendStartup(t *testing.T) {
	paths := []string{
		filepath.Join(
			"cmd",
			"quantumcoin-gui",
			"main.go",
		),
		filepath.Join(
			"cmd",
			"apitest",
			"quantumcoin-gui",
			"main.go",
		),
	}

	required := []string{
		"/api/health",
		"os.O_CREATE|os.O_EXCL|os.O_WRONLY",
		"quantumcoin-api-",
		".start.lock",
		"Process.Kill()",
		"cmd.Wait()",
		"Process.Release()",
		"ensureBackend(",
	}

	for _, path := range paths {
		source :=
			readLifecycleSource(
				t,
				path,
			)

		for _, contract := range required {
			if !strings.Contains(
				source,
				contract,
			) {
				t.Fatalf(
					"%s missing lifecycle contract %q",
					path,
					contract,
				)
			}
		}

		if strings.Contains(
			source,
			"return cmd.Start()",
		) {
			t.Fatalf(
				"%s still abandons raw cmd.Start child",
				path,
			)
		}
	}
}

func TestWalletGUIUsesSharedStartupLeaseAndFailedChildCleanup(
	t *testing.T,
) {
	source :=
		readLifecycleSource(
			t,
			filepath.Join(
				"cmd",
				"wallet",
				"main_fyne.go",
			),
		)

	required := []string{
		`canonicalAPIHealthPath = "/api/health"`,
		"os.O_CREATE|os.O_EXCL|os.O_WRONLY",
		"quantumcoin-api-",
		".start.lock",
		"cmd.Process.Kill()",
		"_ = cmd.Wait()",
	}

	for _, contract := range required {
		if !strings.Contains(
			source,
			contract,
		) {
			t.Fatalf(
				"wallet GUI missing lifecycle contract %q",
				contract,
			)
		}
	}
}

func TestControlPanelHealthMessageUsesCanonicalPort(t *testing.T) {
	source :=
		readLifecycleSource(
			t,
			filepath.Join(
				"release",
				"qc_kontrol_paneli",
				"index.html",
			),
		)

	if strings.Contains(
		source,
		"8080 &#252;zerinde /api/health",
	) {
		t.Fatal(
			"control panel still reports legacy health port 8080",
		)
	}

	if !strings.Contains(
		source,
		"8082 &#252;zerinde /api/health",
	) {
		t.Fatal(
			"control panel canonical health port 8082 is missing",
		)
	}
}
