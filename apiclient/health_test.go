package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthProbeAuthenticatesAndRecognizesProtectedReachableNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("X-QC-API-KEY") != "secret" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="QuantumCoin API"`)
			w.WriteHeader(401)
			fmt.Fprint(w, `{"success":false,"message":"unauthorized"}`)
			return
		}
		fmt.Fprint(w, `{"ok":true,"service":"QuantumCoin API","healthVersion":"QC_NODE_HEALTH_V1"}`)
	}))
	defer server.Close()
	for _, token := range []string{"secret", "", "wrong"} {
		state, err := ProbeHealth(context.Background(), server.URL+CanonicalHealthPathV1, token)
		if err != nil || !state.Reachable || state.Authenticated != (token == "secret") {
			t.Fatal(token, state, err)
		}
	}
}
func TestHealthProbeRejectsUnrelatedServerAndRedirect(t *testing.T) {
	for _, status := range []int{200, 401, 302} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			fmt.Fprint(w, `{"unrelated":true}`)
		}))
		if _, err := ProbeHealth(context.Background(), s.URL+CanonicalHealthPathV1, ""); err == nil {
			t.Fatal("unrelated service accepted", status)
		}
		s.Close()
	}
}
