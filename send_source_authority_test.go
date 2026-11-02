package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalSendSourceAuthorityHasNoParallelArtifacts(
	t *testing.T,
) {
	for _, relative := range []string{
		filepath.Join("api", "tx", "send"),
		filepath.Join(
			"api",
			"http_server.go.bak_api_route_fix",
		),
	} {
		_, err := os.Stat(relative)

		if err == nil {
			t.Fatalf(
				"stale send source artifact exists: %s",
				relative,
			)
		}

		if !errors.Is(
			err,
			os.ErrNotExist,
		) {
			t.Fatalf(
				"stat %s: %v",
				relative,
				err,
			)
		}
	}
}

func TestCanonicalSendSourceAuthorityRemainsRootOwned(
	t *testing.T,
) {
	mainBytes, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf(
			"read main.go: %v",
			err,
		)
	}

	mainSource := string(mainBytes)

	if got := strings.Count(
		mainSource,
		`"/api/tx/send",`,
	); got != 1 {
		t.Fatalf(
			"canonical route registrations=%d want=1",
			got,
		)
	}

	if got := strings.Count(
		mainSource,
		"func handleSendTx(",
	); got != 1 {
		t.Fatalf(
			"canonical send handlers=%d want=1",
			got,
		)
	}

	if !strings.Contains(
		mainSource,
		"protected(handleSendTx)",
	) {
		t.Fatal(
			"canonical send route is not bound to protected handleSendTx",
		)
	}

	apiBytes, err :=
		os.ReadFile(
			filepath.Join(
				"api",
				"http_server.go",
			),
		)

	if err != nil {
		t.Fatalf(
			"read api/http_server.go: %v",
			err,
		)
	}

	apiSource := string(apiBytes)

	if !strings.Contains(
		apiSource,
		"return ErrStandaloneHTTPRetiredV1",
	) {
		t.Fatal(
			"alternate package API is no longer fail-closed retired",
		)
	}

	for _, forbidden := range []string{
		`"/api/tx/send"`,
		"ListenAndServe",
		"ListenAndServeTLS",
		"http.NewServeMux",
	} {
		if strings.Contains(
			apiSource,
			forbidden,
		) {
			t.Fatalf(
				"retired alternate API contains active send/server fragment %q",
				forbidden,
			)
		}
	}
}
