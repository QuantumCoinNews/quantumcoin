package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readLegacySendContractSource(
	t *testing.T,
	parts ...string,
) string {
	t.Helper()

	path := filepath.Join(parts...)
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

func TestLegacySendRouteRemainsRetired(
	t *testing.T,
) {
	source := readLegacySendContractSource(
		t,
		"main.go",
	)

	for _, forbidden := range []string{
		`mux.HandleFunc("/api/send"`,
		`mux.Handle("/api/send"`,
		`HandleFunc("/api/send"`,
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(
				"legacy transfer route returned: %q",
				forbidden,
			)
		}
	}

	for _, required := range []string{
		`"/api/tx/send",`,
		`func newHTTPAPIHandlerV1(`,
		`"tx_send",`,
		`protected(handleSendTx),`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf(
				"canonical transfer route contract missing %q",
				required,
			)
		}
	}
}

func TestCanonicalSendRetainsSecurityGuards(
	t *testing.T,
) {
	source := readLegacySendContractSource(
		t,
		"main.go",
	)

	required := []string{
		`if r.Method != http.MethodPost {`,
		`if req.From == "" || req.To == "" || strings.TrimSpace(req.Amount.String()) == "" {`,
		`qconsensus.ParseQCToAtomsV1(value)`,
		`wallet.ImportPrivateKeyHex(privHex)`,
		`tx.Sign(priv)`,
		`tx.Verify()`,
		`bc.AddTransaction(tx)`,
		`nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))`,
		`qnode.DefaultHTTPServerOptionsV1()`,
	}

	for _, item := range required {
		if !strings.Contains(source, item) {
			t.Fatalf(
				"canonical send safety contract missing %q",
				item,
			)
		}
	}
}

func TestWalletHasNoLegacySendFallback(
	t *testing.T,
) {
	source := readLegacySendContractSource(
		t,
		"cmd",
		"wallet",
		"main_fyne.go",
	)

	if strings.Contains(source, "/api/send") {
		t.Fatal(
			"wallet production source contains legacy /api/send reference",
		)
	}

	required := []string{
		`apiBase()+"/api/tx/send"`,
		`validateCanonicalSendResponse`,
		`if !out.Success`,
	}

	for _, item := range required {
		if !strings.Contains(source, item) {
			t.Fatalf(
				"wallet canonical transfer contract missing %q",
				item,
			)
		}
	}
}

func TestStandaloneAPIHasNoTransferRoutes(
	t *testing.T,
) {
	source := readLegacySendContractSource(
		t,
		"api",
		"http_server.go",
	)

	if !strings.Contains(
		source,
		"return ErrStandaloneHTTPRetiredV1",
	) {
		t.Fatal(
			"standalone package-api is not fail-closed",
		)
	}

	for _, forbidden := range []string{
		`"/api/send"`,
		`"/api/tx/send"`,
		"ListenAndServe",
		"HandleFunc(",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(
				"retired package-api behavior returned: %q",
				forbidden,
			)
		}
	}
}

func TestNoRuntimeLocalCLITransferBridge(
	t *testing.T,
) {
	roots := []string{
		"cmd",
		"api",
	}

	re := regexp.MustCompile(
		`(?s)exec\.Command(?:Context)?\([^)]*"(?:send|sendtx|tx-send|transfer)"`,
	)

	for _, root := range roots {
		err := filepath.WalkDir(
			root,
			func(
				path string,
				entry os.DirEntry,
				walkErr error,
			) error {
				if walkErr != nil {
					return walkErr
				}

				if entry.IsDir() {
					switch entry.Name() {
					case "vendor", "node_modules", ".git":
						return filepath.SkipDir
					}

					return nil
				}

				if !strings.HasSuffix(path, ".go") ||
					strings.HasSuffix(path, "_test.go") {
					return nil
				}

				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				if re.Match(data) {
					t.Errorf(
						"runtime local CLI transfer bridge found in %s",
						path,
					)
				}

				return nil
			},
		)

		if err != nil {
			t.Fatalf(
				"scan %s: %v",
				root,
				err,
			)
		}
	}
}
