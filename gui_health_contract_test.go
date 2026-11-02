package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readGUIHealthContractSource(
	t *testing.T,
	parts ...string,
) string {
	t.Helper()

	path := filepath.Join(parts...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(data)
}

func TestCanonicalGUIHealthContract(
	t *testing.T,
) {
	mainSource := readGUIHealthContractSource(
		t,
		"main.go",
	)
	if !strings.Contains(
		mainSource,
		`mux.HandleFunc("/api/health", protected(handleHealth))`,
	) {
		t.Fatal("canonical root /api/health route is missing")
	}

	apiSource := readGUIHealthContractSource(
		t,
		"api",
		"http_server.go",
	)
	if !strings.Contains(
		apiSource,
		"return ErrStandaloneHTTPRetiredV1",
	) {
		t.Fatal("retired package-api StartHTTP is not fail-closed")
	}
	if strings.Contains(apiSource, "ListenAndServe") {
		t.Fatal("retired package-api listener returned")
	}

	sources := []struct {
		name string
		path []string
	}{
		{
			name: "main GUI",
			path: []string{
				"cmd",
				"quantumcoin-gui",
				"main.go",
			},
		},
		{
			name: "apitest GUI",
			path: []string{
				"cmd",
				"apitest",
				"quantumcoin-gui",
				"main.go",
			},
		},
		{
			name: "wallet GUI",
			path: []string{
				"cmd",
				"wallet",
				"main_fyne.go",
			},
		},
	}

	for _, tt := range sources {
		source := readGUIHealthContractSource(
			t,
			tt.path...,
		)

		if !strings.Contains(source, "/api/health") {
			t.Fatalf(
				"%s does not use canonical /api/health",
				tt.name,
			)
		}

		if strings.Contains(source, `+"/health"`) ||
			strings.Contains(source, `+ "/health"`) ||
			strings.Contains(source, `"/health"`) {
			t.Fatalf(
				"%s still contains legacy /health client contract",
				tt.name,
			)
		}
	}
}

func TestWalletHealthDiscoveryUsesCanonicalEndpoint(
	t *testing.T,
) {
	source := readGUIHealthContractSource(
		t,
		"cmd",
		"wallet",
		"main_fyne.go",
	)

	required := []string{
		`canonicalAPIHealthPath = "/api/health"`,
		`canonicalAPIDefaultPort = "8082"`,
		`health := apiBase() + canonicalAPIHealthPath`,
		`"QC_API_LISTEN=127.0.0.1:"+apiPort`,
	}

	for _, item := range required {
		if !strings.Contains(source, item) {
			t.Fatalf(
				"wallet canonical health contract missing %q",
				item,
			)
		}
	}

	forbidden := []string{
		`return "8081"`,
		`health := apiBase() + "/health"`,
		`"http://127.0.0.1:" + p + "/health"`,
	}

	for _, item := range forbidden {
		if strings.Contains(source, item) {
			t.Fatalf(
				"wallet legacy health behavior remains: %q",
				item,
			)
		}
	}
}

func TestGUILaunchersRequireFinalHealthCheck(
	t *testing.T,
) {
	mainGUI := readGUIHealthContractSource(
		t,
		"cmd",
		"quantumcoin-gui",
		"main.go",
	)

	if strings.Contains(
		mainGUI,
		`_ = waitOK(port, 5*time.Second)`,
	) {
		t.Fatal("main GUI still ignores final health result")
	}
	if !strings.Contains(
		mainGUI,
		`if err := waitOK(port, 5*time.Second); err != nil {`,
	) {
		t.Fatal("main GUI does not fail closed on final health timeout")
	}

	apitestGUI := readGUIHealthContractSource(
		t,
		"cmd",
		"apitest",
		"quantumcoin-gui",
		"main.go",
	)

	if strings.Contains(
		apitestGUI,
		`_ = waitHealth(httpPort, 5*time.Second)`,
	) {
		t.Fatal("apitest GUI still ignores final health result")
	}

	if !strings.Contains(
		apitestGUI,
		"if err := waitHealth(",
	) {
		t.Fatal("apitest GUI does not validate final health")
	}

	if !strings.Contains(
		mainGUI,
		`Timeout: 500 * time.Millisecond`,
	) {
		t.Fatal("main GUI health request has no bounded client timeout")
	}

	if !strings.Contains(
		apitestGUI,
		`Timeout: 500 * time.Millisecond`,
	) {
		t.Fatal("apitest GUI health request has no bounded client timeout")
	}
}
