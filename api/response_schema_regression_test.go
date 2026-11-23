package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPResponseRegressionAdvisoryMethods(t *testing.T) {
	for _, row := range []struct {
		name    string
		handler http.HandlerFunc
	}{{"alerts", HandleAIAlerts}, {"telemetry", HandleTelemetry}} {
		t.Run(row.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			row.handler(w, httptest.NewRequest("POST", "/api/"+row.name, nil))
			var failure struct {
				Success *bool  `json:"success"`
				Message string `json:"message"`
			}
			if w.Code != 405 || json.Unmarshal(w.Body.Bytes(), &failure) != nil || failure.Success == nil || *failure.Success || failure.Message == "" || w.Header().Get("Allow") != "GET" {
				t.Fatalf("missing canonical method error: status=%d headers=%v body=%q", w.Code, w.Header(), w.Body.String())
			}
		})
	}
}
