package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"quantumcoin/apiclient"
	"quantumcoin/blockchain"
)

func TestMain(m *testing.M) {
	if os.Getenv("QC_GUI_SEND_CHILD_V1") == "1" {
		key, _ := io.ReadAll(io.LimitReader(os.Stdin, 1025))
		if strings.TrimSpace(string(key)) != "gui-private-fixture" || strings.Contains(strings.Join(os.Args, " "), "gui-private-fixture") {
			os.Exit(2)
		}
		fmt.Println("SIGNED_TRANSACTION_V1=" + os.Getenv("QC_GUI_SEND_ENVELOPE_V1"))
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestGUISignedSendTransportBindsReceiptAndNeverPostsKey(t *testing.T) {
	tx := &blockchain.Transaction{ID: bytes.Repeat([]byte{1}, 32), Sender: "public-owner", Inputs: []blockchain.TransactionInput{{}}, Outputs: []blockchain.TransactionOutput{{Amount: 1}}}
	envelope, _ := json.Marshal(map[string]any{"transaction": tx})
	t.Setenv("QC_GUI_SEND_CHILD_V1", "1")
	t.Setenv("QC_GUI_SEND_ENVELOPE_V1", string(envelope))
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	expected := hex.EncodeToString(tx.ID)
	for _, mode := range []string{"accepted", "different-id", "rejected"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("X-QC-API-KEY") != "gui-token" {
					t.Error("missing authentication")
				}
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == apiclient.CanonicalHealthPathV1 {
					fmt.Fprintf(w, `{"ok":true,"service":%q,"healthVersion":%q,"apiReadOnly":false}`, apiclient.HealthServiceV1, apiclient.HealthProtocolV1)
					return
				}
				body, _ := io.ReadAll(r.Body)
				if r.URL.Path != "/api/tx/send" || r.Method != http.MethodPost || !bytes.Equal(body, envelope) || bytes.Contains(body, []byte("gui-private-fixture")) {
					t.Error("unsafe or noncanonical GUI submission")
				}
				if mode == "rejected" {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, `{"success":false}`)
					return
				}
				id := expected
				if mode == "different-id" {
					id = strings.Repeat("b", 64)
				}
				fmt.Fprintf(w, `{"success":true,"txid":%q}`, id)
			}))
			defer server.Close()
			id, err := sendWalletRequestV1(canonicalSendRequest{From: "public-owner", To: "public-recipient", Amount: "1", PrivHex: "gui-private-fixture"}, executable, server.URL+"/api/tx/send", "gui-token", 3*time.Second)
			if mode == "accepted" {
				if err != nil || id != expected {
					t.Fatalf("valid GUI receipt rejected: %v", err)
				}
			} else if err == nil || id != "" {
				t.Fatal("failed or unrelated receipt shown as successful")
			}
			if calls.Load() != 2 {
				t.Fatalf("unexpected retry or missing preflight: %d calls", calls.Load())
			}
		})
	}
}

func TestGUISendRejectsReadOnlyUnknownAndUnauthorizedBeforeSigning(t *testing.T) {
	for _, mode := range []string{"read-only", "unknown", "unauthorized"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != apiclient.CanonicalHealthPathV1 {
					t.Error("submission occurred before preflight succeeded")
				}
				w.Header().Set("Content-Type", "application/json")
				if mode == "unauthorized" {
					w.Header().Set("WWW-Authenticate", apiclient.HealthAuthRealmV1)
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, `{"success":false}`)
					return
				}
				extra := ""
				if mode == "read-only" {
					extra = `,"apiReadOnly":true`
				}
				fmt.Fprintf(w, `{"ok":true,"service":%q,"healthVersion":%q%s}`, apiclient.HealthServiceV1, apiclient.HealthProtocolV1, extra)
			}))
			defer server.Close()
			id, err := sendWalletRequestV1(canonicalSendRequest{PrivHex: "gui-private-fixture"}, filepath.Join(t.TempDir(), "does-not-exist.exe"), server.URL+"/api/tx/send", "", time.Second)
			if id != "" || err == nil {
				t.Fatal("unsafe mode accepted")
			}
			if mode == "unauthorized" {
				if !errors.Is(err, apiclient.ErrHealthUnauthorizedV1) {
					t.Fatal(err)
				}
			} else if !strings.Contains(err.Error(), "read-only") {
				t.Fatal("signing was attempted before mode validation")
			}
		})
	}
}

func TestWalletBootstrapStartsWritableNodeWithoutMining(t *testing.T) {
	dir := t.TempDir()
	command := newWalletBackendCommandV1(filepath.Join(dir, "quantumcoin.exe"), dir, "8082")
	if len(command.Args) != 2 || command.Args[1] != "run" || command.Dir != dir {
		t.Fatal("wallet started a read-only API or automatic miner")
	}
	if !strings.Contains(strings.Join(command.Env, "\n"), "QC_API_LISTEN=127.0.0.1:8082") {
		t.Fatal("wallet backend is not explicitly loopback-bound")
	}
	raw, _ := json.Marshal(canonicalSendRequest{PrivHex: "gui-private-fixture"})
	if bytes.Contains(raw, []byte("gui-private-fixture")) {
		t.Fatal("GUI intent serialized private key")
	}
}
