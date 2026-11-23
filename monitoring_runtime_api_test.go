package main

import (
	"encoding/gob"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/monitoring"
	qnode "quantumcoin/node"
	"quantumcoin/p2p"
	"quantumcoin/security"
	"quantumcoin/wallet"
)

func TestCanonicalSpendRejectionReachesSecurityReport(t *testing.T) {
	chain, owner, to, c := cliSendFixture(t)
	previous := nodeRuntime
	nodeRuntime = qnode.NewRuntime(chain, c)
	defer func() { nodeRuntime = previous }()
	for _, amount := range []string{"2", "3"} {
		tx, err := blockchain.NewTransactionFromQCAmountV1(owner.GetAddress(), to.GetAddress(), amount, chain)
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Sign(owner.PrivateKey); err != nil {
			t.Fatal(err)
		}
		err = chain.AddTransaction(tx)
		if amount == "2" && err != nil {
			t.Fatal(err)
		}
		if amount == "3" && (err == nil || !strings.Contains(err.Error(), "pending")) {
			t.Fatal("fixture did not create canonical pending conflict", err)
		}
	}
	w := httptest.NewRecorder()
	view := buildSecurityView(w)
	report := view.dashboard["doubleSpendConflict"].(map[string]any)
	found := false
	for _, finding := range report["findings"].([]security.DoubleSpendConflictFinding) {
		if finding.Subject == owner.GetAddress() && finding.Risk != "LOW" {
			found = true
		}
	}
	if !found {
		t.Fatal("canonical rejection did not reach admin analysis", report)
	}
	if chain.PendingTransactionCountSnapshot() != 1 {
		t.Fatal("analysis changed canonical admission")
	}
}

func TestHTTPMonitoringIgnoresForwardedClientIdentity(t *testing.T) {
	h := readOnlyMuxFixtureV1(t)
	req := httptest.NewRequest("GET", "/api/health", nil)
	req.RemoteAddr = "127.0.0.1:45000"
	req.Host = "127.0.0.1:8082"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("unauthenticated request was not rejected", w.Code)
	}
	found := false
	for _, event := range nodeRuntime.Monitoring().Snapshot().Events {
		if event.Category == "auth_failure" {
			found = true
			if event.Metadata["subject"] != "127.0.0.1" {
				t.Fatal("spoofed subject", event)
			}
		}
	}
	if !found {
		t.Fatal("auth rejection not monitored")
	}
}

func TestMonitoringOperatorActionsUseSeparateCredentialAndRejectReadOnly(t *testing.T) {
	externalMiningMux(t)
	t.Setenv("QC_DIFFICULTY_BITS", "")
	r, err := qnode.BootstrapWithConfig(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	nodeRuntime = r
	h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	r.EmitEvent(qnode.NodeEvent{Type: "TEST_FAULT", Severity: qnode.NodeEventSeverityError, Message: "test fault"})
	snapshot := r.Monitoring().Snapshot()
	if len(snapshot.Incidents) == 0 {
		t.Fatal("runtime event sink disconnected")
	}
	id := snapshot.Incidents[0].ID
	t.Setenv("QC_MONITORING_ADMIN_TOKEN", "isolated-monitoring-test-token")
	request := func(token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/monitoring/action", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:34567"
		req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
		req.Header.Set("X-QC-MONITORING-ADMIN", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	body := `{"id":"` + id + `","action":"ack","note":"checked"}`
	if got := request("", body); got.Code != 403 {
		t.Fatal("API credential alone authorized monitoring mutation", got.Code)
	}
	if got := request("isolated-monitoring-test-token", body+"garbage"); got.Code != 400 {
		t.Fatal("malformed suffix accepted", got.Code)
	}
	if got := request("isolated-monitoring-test-token", body); got.Code != 200 {
		t.Fatal(got.Code, got.Body.String())
	}
	found := false
	for _, in := range r.Monitoring().Snapshot().Incidents {
		if in.ID == id && in.Status == "acknowledged" && in.Actor == "node-operator" {
			found = true
		}
	}
	if !found {
		t.Fatal("operator acknowledgement not persisted")
	}
	r.SetAPIReadOnly(true)
	if got := request("isolated-monitoring-test-token", strings.Replace(body, "ack", "resolve", 1)); got.Code != 403 {
		t.Fatal("read-only mutation accepted", got.Code)
	}
	view := webRequest(h, "GET", "/api/monitoring", "")
	if view.Code != 200 {
		t.Fatal("read-only monitoring unavailable", view.Code)
	}
	var state monitoring.Snapshot
	if err = json.Unmarshal(view.Body.Bytes(), &state); err != nil || len(state.Notifications) == 0 {
		t.Fatal("local notification outbox unavailable", err)
	}
}

func TestPublisherMonitoringUsesDedicatedLoopbackAuthAndDedupes(t *testing.T) {
	externalMiningMux(t)
	t.Setenv("QC_DIFFICULTY_BITS", "")
	r, err := qnode.BootstrapWithConfig(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	nodeRuntime = r
	h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	t.Setenv("QC_PUBLISHER_MONITOR_TOKEN", "isolated-publisher-monitor-token")
	body := `{"version":"QC_PUBLISHER_REPORT_V1","jobId":"2026-11-10.slot1","editorial":"passed","deliveries":[{"platform":"x","status":"uncertain"}]}`
	send := func(remote, token, forwarded, body string) int {
		req := httptest.NewRequest("POST", "http://127.0.0.1/api/monitoring/publisher", strings.NewReader(body))
		req.RemoteAddr = remote
		req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
		req.Header.Set("X-QC-PUBLISHER-MONITOR", token)
		req.Header.Set("X-Forwarded-For", forwarded)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	const local = "127.0.0.1:35678"
	const token = "isolated-publisher-monitor-token"
	if send(local, "", "", body) != 403 ||
		send("192.0.2.2:45678", token, "", body) != 403 ||
		send(local, token, "127.0.0.1", body) != 403 {
		t.Fatal("unauthorized publisher report was accepted")
	}
	if code := send(local, token, "", strings.Replace(body, `"platform"`, `"url"`, 1)); code != 400 {
		t.Fatal("unknown report field accepted", code)
	}
	for i := 0; i < 2; i++ {
		if code := send(local, token, "", body); code != 200 {
			t.Fatal("valid publisher report rejected", code)
		}
	}
	state := r.Monitoring().Snapshot()
	events, incidents := 0, 0
	for _, event := range state.Events {
		if event.Source == "autopublisher" {
			events++
		}
	}
	for _, incident := range state.Incidents {
		if incident.Category == "publisher_delivery" && incident.Severity == monitoring.SeverityCritical {
			incidents++
		}
	}
	if events != 3 || incidents != 1 {
		t.Fatalf("unexpected monitoring state: %d events, %d critical incidents", events, incidents)
	}
	r.SetAPIReadOnly(true)
	if code := send(local, token, "", body); code != 403 {
		t.Fatal("observer accepted publisher reports", code)
	}
}

func TestSecurityReportsUseLiveP2PConnectionAndValidatedHeight(t *testing.T) {
	h := externalMiningMux(t)
	r := nodeRuntime
	done := make(chan error, 1)
	go func() { done <- r.ServeP2P("127.0.0.1:0") }()
	deadline := time.Now().Add(4 * time.Second)
	for r.P2PListenAddress() == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if r.P2PListenAddress() == "" {
		t.Fatal("listener not ready")
	}
	t.Cleanup(func() {
		r.StopP2P()
		select {
		case <-done:
		case <-time.After(4 * time.Second):
			t.Error("P2P listener did not stop")
		}
	})
	conn, err := net.DialTimeout("tcp", r.P2PListenAddress(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(4 * time.Second))
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	var hello p2p.Message
	if err = decoder.Decode(&hello); err != nil {
		t.Fatal(err)
	}
	if err = encoder.Encode(p2p.HandshakeMessage(r.Blockchain)); err != nil {
		t.Fatal(err)
	}
	for p2p.NetworkViewSnapshot().HandshakenPeers == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if p2p.NetworkViewSnapshot().HandshakenPeers == 0 {
		t.Fatal("handshake not accepted")
	}
	chain, _ := r.Snapshot()
	if err = encoder.Encode(p2p.ChainMessage(chain)); err != nil {
		t.Fatal(err)
	}
	go io.Copy(io.Discard, conn)
	for !p2p.NetworkViewSnapshot().HeightKnown && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !p2p.NetworkViewSnapshot().HeightKnown {
		t.Fatal("validated peer height not observed")
	}
	conn.SetDeadline(time.Time{})
	// Subsequent accepted blocks must refresh the network view too; otherwise
	// an active connection would keep an old height until another full chain.
	remoteChain := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(chain))
	difficulty, err := remoteChain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	block, err := remoteChain.MineBlock(wallet.NewWallet().GetAddress(), difficulty)
	if err != nil {
		t.Fatal(err)
	}
	if err = encoder.Encode(p2p.BlockMessage(block)); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(4 * time.Second)
	for p2p.NetworkViewSnapshot().NetworkHeight != block.Index && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if p2p.NetworkViewSnapshot().NetworkHeight != block.Index || chain.HeightSnapshot() != block.Index {
		t.Fatal("accepted block did not refresh observed peer height")
	}
	for _, path := range []string{"/api/security/dashboard", "/api/security/admin-report", "/api/ai/security-dashboard", "/api/security/finality-checkpoint"} {
		w := webRequest(h, http.MethodGet, path, "")
		if w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), `"health":"ISOLATED"`) || strings.Contains(w.Body.String(), "LOCAL_ONLY_FINALITY_VIEW") {
			t.Fatal("connected peer reported as isolated", path, w.Body.String())
		}
	}
}
