package node

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"quantumcoin/ai"
	"quantumcoin/monitoring"
	"quantumcoin/security"
	"sync"
	"time"
)

// Fixed category slots coalesce HTTP ingress writes across IP churn. Each
// category has its own reservation so warning floods cannot mute API failures.
type httpObservationBudgetV1 struct {
	mu     sync.Mutex
	nextAt time.Time
	busy   bool
}

func (b *httpObservationBudgetV1) begin(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.busy || (!b.nextAt.IsZero() && now.Before(b.nextAt)) {
		return false
	}
	b.busy = true
	b.nextAt = now.Add(time.Second)
	return true
}

func (b *httpObservationBudgetV1) end() {
	b.mu.Lock()
	b.busy = false
	b.mu.Unlock()
}

func (r *Runtime) Monitoring() *monitoring.Engine {
	if r == nil {
		e, _ := monitoring.Open("")
		return e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.monitor == nil {
		r.monitor, _ = monitoring.Open("")
		r.monitorEpoch = monitoring.NewID()
	}
	return r.monitor
}
func (r *Runtime) initializeMonitoring() {
	_, cfg := r.Snapshot()
	if cfg == nil {
		return
	}
	path, err := filepath.Abs(filepath.Join(cfg.ChainFile+".monitoring", "state.json"))
	if err != nil {
		ai.RecordPersistenceFailure("monitoring path", err)
		return
	}
	engine, err := monitoring.Open(path)
	r.mu.Lock()
	r.monitor = engine
	r.monitorEpoch = monitoring.NewID()
	r.mu.Unlock()
	if err != nil {
		ai.RecordPersistenceFailure("monitoring load", err)
	}
	_ = engine.Record(monitoring.NewEvent(monitoring.NewID(), "node", "startup", monitoring.SeverityInfo, "canonical monitoring started"), false)
}
func (r *Runtime) monitoringRecord(ev monitoring.Event) {
	if r == nil {
		return
	}
	if err := r.Monitoring().Record(ev, !r.IsAPIReadOnly()); err != nil {
		ai.RecordPersistenceFailure("monitoring record", err)
	}
}
func (r *Runtime) observeMonitoringEvent(event NodeEvent) {
	security.LiveEvidence.ObserveMiner(event.Type)
	severity := monitoring.SeverityInfo
	if event.Severity == NodeEventSeverityWarning {
		severity = monitoring.SeverityWarning
	}
	if event.Severity == NodeEventSeverityError {
		severity = monitoring.SeverityCritical
	}
	if event.Severity != NodeEventSeverityInfo && event.Severity != NodeEventSeverityWarning && event.Severity != NodeEventSeverityError {
		severity = monitoring.Severity(event.Severity)
	}
	r.monitoringRecord(monitoring.NewEvent(monitoring.NewID(), "node", event.Type, severity, event.MessageOrType()))
}
func (event NodeEvent) MessageOrType() string {
	if event.Message != "" {
		if len(event.Message) > 2048 {
			return event.Message[:2048]
		}
		return event.Message
	}
	return event.Type
}
func (r *Runtime) RecordHTTPObservation(ip, path, method string, status int) {
	security.LiveEvidence.ObserveHTTP(ip, path, method, status)
	category := ""
	slot := 0
	severity := monitoring.SeverityWarning
	switch {
	case status == 401:
		category = "auth_failure"
	case status == 429:
		slot = 1
		category = "action_rate_limit"
	case status >= 500:
		slot = 2
		category = "api_failure"
		severity = monitoring.SeverityCritical
	}
	if category == "" {
		return
	}
	if r == nil || !r.httpMonitorBudgets[slot].begin(time.Now()) {
		return
	}
	defer r.httpMonitorBudgets[slot].end()
	if len(ip) > 128 {
		ip = "unknown"
	}
	// Stable per-minute identity deduplicates ingress floods without storing bodies or secrets.
	digest := sha256.Sum256(fmt.Appendf(nil, "%s/%s/%d", ip, category, time.Now().Unix()/60))
	ev := monitoring.NewEvent(fmt.Sprintf("%x", digest), "http", category, severity, "HTTP security boundary observed "+category)
	if len(ip) > 128 {
		ip = "unknown"
	}
	ev.Metadata["subject"] = ip
	r.monitoringRecord(ev)
}
func (r *Runtime) PollMonitoring() {
	if r == nil {
		return
	}
	reports := r.SecurityRuntimeView()
	r.Monitoring()
	r.mu.RLock()
	epoch := r.monitorEpoch
	r.mu.RUnlock()
	for _, name := range []string{"actionBurstGuard", "minerSafety", "doubleSpendConflict", "ipApiReputation"} {
		risk, _ := reports[name]["overallRisk"].(string)
		if risk == "HIGH" || risk == "MEDIUM" || risk == "WATCH" {
			sev := monitoring.SeverityWarning
			if risk == "HIGH" {
				sev = monitoring.SeverityCritical
			}
			id := fmt.Sprintf("%s-review-%s-%v", epoch, name, reports[name]["evidenceSequence"])
			r.monitoringRecord(monitoring.NewEvent(id, "runtime-security", name, sev, "Observed "+name+" risk: "+risk))
		}
	}
	health := reports["nodeHealth"]
	if status, _ := health["syncStatus"].(string); status == "NODE_BEHIND_NETWORK" || status == "NODE_CRITICALLY_BEHIND" {
		r.monitoringRecord(monitoring.NewEvent(fmt.Sprintf("%s-sync-%v-%v-%s", epoch, health["localHeight"], health["networkHeight"], status), "runtime-security", "node_sync_drift", monitoring.SeverityWarning, "Node is behind validated peer observations"))
	}
	if !r.IsAPIReadOnly() {
		if err := r.Monitoring().Flush(); err != nil {
			ai.RecordPersistenceFailure("monitoring flush", err)
		}
	}
}
