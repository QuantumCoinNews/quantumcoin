package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStandaloneStartHTTPFailsClosed(
	t *testing.T,
) {
	Init(nil, nil, nil)

	for _, addr := range []string{
		"",
		"127.0.0.1:8082",
		"0.0.0.0:8082",
		":8082",
		"8082",
		"[::]:8082",
	} {
		err := StartHTTP(addr)

		if err == nil {
			t.Fatalf(
				"StartHTTP(%q) unexpectedly succeeded",
				addr,
			)
		}

		if err != ErrStandaloneHTTPRetiredV1 {
			t.Fatalf(
				"StartHTTP(%q) error = %v, want %v",
				addr,
				err,
				ErrStandaloneHTTPRetiredV1,
			)
		}
	}
}

func TestStandaloneHTTPArchitectureRemainsRetired(
	t *testing.T,
) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	apiDir := filepath.Dir(thisFile)

	removedFiles := []string{
		"cors_public_mode_v1.go",
		"cors_public_mode_v1_test.go",
		"dev_miner_endpoint_lock_v1.go",
		"dev_miner_endpoint_lock_v1_test.go",
		"http_security_qma0018_v1.go",
		"http_security_qma0018_v1_test.go",
		"http_server_default_bind_qma0016_v1_test.go",
		"http_server_listen_qma0017_v1_test.go",
		"http_server_runtime_policy_v1_test.go",
	}

	for _, name := range removedFiles {
		path := filepath.Join(apiDir, name)

		if _, err := os.Stat(path); err == nil {
			t.Fatalf(
				"retired standalone HTTP architecture file still exists: %s",
				name,
			)
		} else if !os.IsNotExist(err) {
			t.Fatalf(
				"stat %s: %v",
				name,
				err,
			)
		}
	}

	serverPath := filepath.Join(
		apiDir,
		"http_server.go",
	)

	data, err := os.ReadFile(serverPath)
	if err != nil {
		t.Fatalf(
			"read http_server.go: %v",
			err,
		)
	}

	source := string(data)

	for _, forbidden := range []string{
		"ListenAndServe",
		"ListenAndServeTLS",
		"http.Server",
		"http.NewServeMux",
		"HandleFunc(",
		"buildStartHTTPHandler",
		"withCORS",
		"apiProtected",
		"apiHTTPGuard",
		"apiEndpointLimiter",
		"resolveStartHTTPListenAddr",
		"/health",
		"/api/send",
		"/api/tx/send",
		"/api/miner/start",
		"/api/miner/stop",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(
				"retired standalone HTTP symbol/route remains in http_server.go: %q",
				forbidden,
			)
		}
	}
}
