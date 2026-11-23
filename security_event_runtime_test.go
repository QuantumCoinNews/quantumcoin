package main

import (
	"net/http/httptest"
	qnode "quantumcoin/node"
	"quantumcoin/p2p"
	"quantumcoin/security"
	"strings"
	"testing"
)

func TestSecurityEventOperatorRouteAndLifecycle(t *testing.T) {
	externalMiningMux(t)
	t.Setenv("QC_DIFFICULTY_BITS", "")
	runtime, err := qnode.BootstrapWithConfig(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	nodeRuntime = runtime
	h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	event := security.NewEvent("isolated-operator-test", security.SeverityCritical, "test", "test-subject", "evidence", nil)
	if err := security.AppendEvent(".", event); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QC_MONITORING_ADMIN_TOKEN", "isolated-security-operator")
	body := `{"id":"` + event.ID + `","action":"resolve","note":"recovered"}`
	request := func(operator string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/security/events/action", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:32100"
		req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
		req.Header.Set("X-QC-MONITORING-ADMIN", operator)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	if got := request(""); got.Code != 403 {
		t.Fatal("operator boundary bypass", got.Code)
	}
	if got := request("isolated-security-operator"); got.Code != 200 {
		t.Fatal(got.Code, got.Body.String())
	}
	found := false
	for _, ev := range security.ReadEvents(".", 1000).Events {
		if ev.ID == event.ID {
			found = true
			if ev.Status != "resolved" || ev.Actor != "node-operator" {
				t.Fatal(ev)
			}
		}
	}
	if !found {
		t.Fatal("event missing")
	}
	runtime.SetAPIReadOnly(true)
	if got := request("isolated-security-operator"); got.Code != 403 {
		t.Fatal("read-only mutation allowed", got.Code)
	}
}

func TestSecurityAdminPeerRiskUsesProducerSchema(t *testing.T) {
	externalMiningMux(t)
	previous := p2p.DefaultPeerSecurity
	defer func() { p2p.DefaultPeerSecurity = previous }()
	p2p.DefaultPeerSecurity = p2p.NewPeerSecurity()
	p2p.DefaultPeerSecurity.AdjustReputation("192.0.2.10", -100)
	snapshot := livePeerReputation()
	if snapshot["overallRisk"] != "HIGH" {
		t.Fatal(snapshot)
	}
	view := buildSecurityView(httptest.NewRecorder())
	if view.dashboard["peerReputation"].(map[string]any)["overallRisk"] != "HIGH" {
		t.Fatal(view.dashboard)
	}
	p2p.DefaultPeerSecurity = nil
	if livePeerReputation()["overallRisk"] != "UNKNOWN" {
		t.Fatal("unavailable producer reported healthy")
	}
}
