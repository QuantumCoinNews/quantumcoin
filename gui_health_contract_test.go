package main

import (
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
	data, err := readNodeContractSourceV1(path)
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

		if !strings.Contains(source, "apiclient.EnsureLocalBackendV1(") {
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
		`apiclient.EnsureLocalBackendV1(`,
		`apiclient.ConfiguredAPIPortV1()`,
		`apiclient.NormalizeAPIPortV1(`,
		`"QC_API_LISTEN=127.0.0.1:"`,
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

func TestGUILaunchersRequireFinalHealthCheck(t *testing.T) {
	for _, parts := range [][]string{{"cmd", "quantumcoin-gui", "main.go"}, {"cmd", "apitest", "quantumcoin-gui", "main.go"}} {
		source := readGUIHealthContractSource(t, parts...)
		for _, required := range []string{"apiclient.EnsureLocalBackendV1(", "context.WithTimeout(context.Background(), 20*time.Second)", "stopStartedBackend(cmd)", "releaseStartedBackend(cmd)"} {
			if !strings.Contains(source, required) {
				t.Fatalf("%s is missing shared health/process ownership contract %q", filepath.Join(parts...), required)
			}
		}
		if strings.Contains(source, "time.Since(st.ModTime())") {
			t.Fatal("GUI still steals startup leases by age")
		}
	}
}
