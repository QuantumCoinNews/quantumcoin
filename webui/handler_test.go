package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerSeparatesAPIFromNavigationAndAssets(t *testing.T) {
	handler, err := Handler()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/api", "/api/", "/api/missing", "/a%70i/missing", "/api/../wallet", "/page/../api/missing"} {
		t.Run(target, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			var body struct {
				Success *bool  `json:"success"`
				Message string `json:"message"`
			}
			if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != "application/json" ||
				json.Unmarshal(rec.Body.Bytes(), &body) != nil || body.Success == nil || *body.Success || body.Message == "" {
				t.Fatalf("status=%d headers=%v body=%q", rec.Code, rec.Header(), rec.Body.String())
			}
		})
	}

	for _, target := range []string{"/index.backup.html", "/assets/index-BpysLhge.js", "/assets/missing", "/missing.mjs"} {
		t.Run(target, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("obsolete or missing asset returned %d", rec.Code)
			}
		})
	}

	for _, target := range []string{"/", "/wallet/receive"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		body := rec.Body.String()
		if rec.Code != http.StatusOK || !strings.Contains(body, "QuantumCoin Node") ||
			!strings.Contains(body, "/network-label.mjs") || strings.Contains(body, "<input") ||
			strings.Contains(body, "/api/dev/fastmine") || strings.Contains(body, "/api/tx/send") {
			t.Fatalf("unexpected public page at %s: %d %q", target, rec.Code, body)
		}
	}

	for _, target := range []string{"/network-label.mjs", "/webminer.mjs", "/canonical-pow-worker.mjs", "/canonical-pow.mjs"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
			t.Fatalf("module %s: status=%d headers=%v", target, rec.Code, rec.Header())
		}
	}
}
