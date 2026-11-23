package main

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"quantumcoin/ai"
	"quantumcoin/api"
)

func TestCanonicalAdvisoryAPIsShareAlertsAndLiveTelemetry(t *testing.T) {
	h := readOnlyMuxFixtureV1(t)
	before := ai.Enabled()
	ai.SetEnabled(true)
	defer ai.SetEnabled(before)
	ai.PushAlert(ai.AlertWarning, "ADVISORY_API_TEST", "visible from both adapters", "test")
	w := readOnlyRequestV1(h, "GET", "/api/ai/alerts", actualMuxTestTokenV1)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var rootAlerts []ai.AIAlert
	if err := json.Unmarshal(w.Body.Bytes(), &rootAlerts); err != nil {
		t.Fatal(err)
	}
	compat := httptest.NewRecorder()
	api.HandleAIAlerts(compat, httptest.NewRequest("GET", "/api/ai/alerts", nil))
	var packageAlerts []ai.AIAlert
	if err := json.Unmarshal(compat.Body.Bytes(), &packageAlerts); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rootAlerts, packageAlerts) || len(rootAlerts) == 0 || rootAlerts[len(rootAlerts)-1].Code != "ADVISORY_API_TEST" {
		t.Fatal("split alert authorities")
	}
	w = readOnlyRequestV1(h, "GET", "/api/telemetry", actualMuxTestTokenV1)
	var telemetry TelemetryResponse
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &telemetry); err != nil {
		t.Fatal(err)
	}
	if telemetry.AI.Timestamp == 0 || telemetry.AI.PeerCount != nodeRuntime.PeerCount() || telemetry.AI.HashrateAvailable {
		t.Fatal("runtime telemetry missing or fabricated")
	}
}
