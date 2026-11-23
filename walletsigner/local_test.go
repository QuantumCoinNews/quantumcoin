package walletsigner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("QC_SIGNER_TEST_CHILD") == "1" {
		data, _ := io.ReadAll(os.Stdin)
		if !strings.Contains(string(data), "private-material") || strings.Contains(strings.Join(os.Args, " "), "private-material") {
			os.Exit(2)
		}
		fmt.Println(`SIGNED_TRANSACTION_V1={"transaction":{"ID":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","Sender":"public-sender","Inputs":[{}],"Outputs":[{}]}}`)
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func TestLocalSignerKeepsKeyOffArgumentsAndHTTP(t *testing.T) {
	t.Setenv("QC_SIGNER_TEST_CHILD", "1")
	exe, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	raw, err := SignLocal(ctx, exe, "from", "to", "0.000001", "private-material", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private-material") {
		t.Fatal("secret in envelope")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "private-material") || r.Header.Get("X-QC-API-KEY") != "token" {
			t.Error("unsafe submit request")
		}
		fmt.Fprint(w, `{"success":true}`)
	}))
	defer server.Close()
	if _, status, err := Submit(ctx, server.URL, "token", raw); err != nil || status != 200 {
		t.Fatal(status, err)
	}
}
func TestWalletSubmitRejectsCleartextRemoteAndDoesNotFollowRedirect(t *testing.T) {
	if _, _, err := Submit(context.Background(), "http://example.com/api/tx/send", "token", nil); err == nil {
		t.Fatal("cleartext remote wallet allowed")
	}
	reached := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer destination.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	if _, status, err := Submit(context.Background(), origin.URL, "token", nil); err != nil || status != 307 || reached {
		t.Fatal("wallet followed redirect", status, err)
	}
}
