package security

import (
	"net"
	"strings"
	"sync"
	"time"
)

// HTTPObservation Runtime observations are bounded advisory evidence, never authorization.
type HTTPObservation struct {
	At time.Time
	IPAPIReputationObservation
}
type RuntimeEvidence struct {
	mu                sync.Mutex
	http              []HTTPObservation
	minerErrors       int
	consecutiveErrors int
	spends            []DoubleSpendConflictObservation
	httpRevision      uint64
	spendRevision     uint64
	minerRevision     uint64
}

func (r *RuntimeEvidence) ObserveSpend(txid, subject, reason string) {
	if len(txid) > 128 {
		txid = txid[:128]
	}
	if len(subject) > 128 {
		subject = subject[:128]
	}
	if len(reason) > 1024 {
		reason = reason[:1024]
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.spends = append(r.spends, DoubleSpendConflictObservation{TxID: txid, Subject: subject, Reason: reason, Source: "canonical-blockchain", BlockchainCorePath: true})
	r.spendRevision++
	if len(r.spends) > 256 {
		r.spends = r.spends[len(r.spends)-256:]
	}
}

var LiveEvidence = &RuntimeEvidence{}

func (r *RuntimeEvidence) ObserveHTTP(ip, endpoint, method string, status int) {
	if len(ip) > 128 || len(endpoint) > 256 || len(method) > 16 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-time.Minute)
	start := 0
	for start < len(r.http) && r.http[start].At.Before(cutoff) {
		start++
	}
	r.http = append(r.http[:0], r.http[start:]...)
	r.http = append(r.http, HTTPObservation{At: time.Now(), IPAPIReputationObservation: IPAPIReputationObservation{ClientIP: ip, Endpoint: endpoint, Method: method, StatusCode: status, RateLimited: status == 429, AuthFailed: status == 401, Blocked: status == 403}})
	r.httpRevision++
	if len(r.http) > 2000 {
		r.http = r.http[len(r.http)-2000:]
	}
}
func (r *RuntimeEvidence) ObserveMiner(eventType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if eventType == "MINER_FAILED" || eventType == "MINER_STARTED" || eventType == "BLOCK_MINED" {
		r.minerRevision++
	}
	if eventType == "MINER_FAILED" {
		if r.minerErrors < 1000 {
			r.minerErrors++
		}
		if r.consecutiveErrors < 1000 {
			r.consecutiveErrors++
		}
	}
	if eventType == "BLOCK_MINED" || eventType == "MINER_STARTED" {
		r.consecutiveErrors = 0
	}
}
func (r *RuntimeEvidence) Reports(mining, stopPresent bool) map[string]map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	observations := make([]IPAPIReputationObservation, 0, len(r.http))
	guards := make([]GUICLIAPIGuardrailObservation, 0, len(r.http))
	bursts := map[string]*ActionBurstObservation{}
	for _, obs := range r.http {
		if time.Since(obs.At) > time.Minute {
			continue
		}
		observations = append(observations, obs.IPAPIReputationObservation)
		b := bursts[obs.ClientIP]
		if b == nil {
			b = &ActionBurstObservation{Subject: obs.ClientIP, ClientIP: obs.ClientIP, WindowSeconds: 60}
			bursts[obs.ClientIP] = b
		}
		b.RequestCount++
		if obs.RateLimited {
			b.RateLimitedCount++
			b.Status429Count++
		}
		if obs.Blocked {
			b.BlockedCount++
		}
		b.SensitiveEndpoint = b.SensitiveEndpoint || isSensitiveAPIEndpointV1(obs.Endpoint)
		ip := net.ParseIP(obs.ClientIP)
		guards = append(guards, GUICLIAPIGuardrailObservation{Subject: obs.ClientIP, Surface: "API", Endpoint: obs.Endpoint, UsesLocalhost: ip != nil && ip.IsLoopback(), UsesAPIBalance: strings.HasPrefix(obs.Endpoint, "/api/wallet/balance/"), UsesTxSendEndpoint: obs.Endpoint == "/api/tx/send", TransferEndpoint: obs.Endpoint == "/api/tx/send", CriticalEndpoint: isSensitiveAPIEndpointV1(obs.Endpoint), TimeoutOrAPIErrorCount: boolCount(obs.StatusCode >= 500)})
	}
	burstList := make([]ActionBurstObservation, 0, len(bursts))
	for _, b := range bursts {
		burstList = append(burstList, *b)
	}
	miner := AnalyzeMinerSafety([]MinerSafetyObservation{{Subject: "canonical-runtime", Mode: "node", LoopActive: mining, StopFlagPresent: stopPresent, DelayMs: 50, MineErrorCount: r.minerErrors, ConsecutiveErrorCount: r.consecutiveErrors}})
	sections := map[string]map[string]any{"actionBurstGuard": AnalyzeActionBurst(burstList), "ipApiReputation": AnalyzeIPAPIReputation(observations), "guiCliApiGuardrails": AnalyzeGUICLIAPIGuardrails(guards), "minerSafety": miner, "doubleSpendConflict": AnalyzeDoubleSpendConflicts(r.spends)}
	for _, name := range []string{"actionBurstGuard", "ipApiReputation", "guiCliApiGuardrails"} {
		sections[name]["evidenceSequence"] = r.httpRevision
	}
	sections["minerSafety"]["evidenceSequence"] = r.minerRevision
	sections["doubleSpendConflict"]["evidenceSequence"] = r.spendRevision
	return sections
}
func boolCount(v bool) int {
	if v {
		return 1
	}
	return 0
}
