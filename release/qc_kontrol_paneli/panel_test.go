package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quantumcoin/monitoring"
)

func TestPanelUsesLiveTelemetryAndMonitoringLifecycle(t *testing.T) {
	engine, err := monitoring.Open("")
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Record(monitoring.NewEvent("fixture", "node", "peer_drift", monitoring.SeverityWarning, "test"), false); err != nil {
		t.Fatal(err)
	}
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("X-QC-API-KEY") != "fixture" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "networkIdentity": map[string]string{
				"network": "local-release", "genesisHash": strings.Repeat("a", 64),
			}})
		case "/api/telemetry":
			_ = json.NewEncoder(w).Encode(map[string]int{"height": 42, "block_count": 43, "peers": 2})
		case "/api/monitoring":
			_ = json.NewEncoder(w).Encode(engine.Snapshot())
		default:
			t.Errorf("unexpected node path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer node.Close()
	game := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" || r.Header.Get("X-QC-API-KEY") != "" {
			t.Error("unexpected game request or forwarded credential")
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer game.Close()
	s := source{node: node.URL, game: game.URL, token: "fixture", client: node.Client()}
	h := s.handler("127.0.0.1:8181", "/fixture-path")
	render := func() string {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8181/fixture-path", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Body.String()
	}
	html := render()
	for _, text := range []string{"Yükseklik 42", "Blok 43", "Eş 2", "local-release", `class="warning">Uyarı · peer_drift`, "Telegram Game · Çevrim içi"} {
		if !strings.Contains(html, text) {
			t.Fatalf("expected %q in rendered live data", text)
		}
	}
	if err := engine.Transition(engine.Snapshot().Incidents[0].ID, "resolve", "test", "fixed"); err != nil {
		t.Fatal(err)
	}
	if html := render(); strings.Contains(html, "peer_drift") || !strings.Contains(html, "Açık uyarı yok.") {
		t.Fatal("resolved alert remained on the live panel")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://foreign.test/fixture-path", nil))
	if w.Code != http.StatusNotFound {
		t.Fatal("foreign host could read the panel")
	}
}
