// Package security contains QuantumCoin project code for this module.
package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SECURITY_EVENT_LOG_V1
// Read/write security event log for local node monitoring.
// Safe boundary: does not change chain, balances, wallets, mining rewards, or consensus.

const SecurityEventLogVersion = "SECURITY_EVENT_LOG_V1"

type EventSeverity string

const (
	SeverityInfo     EventSeverity = "INFO"
	SeverityWarning  EventSeverity = "WARNING"
	SeverityHigh     EventSeverity = "HIGH"
	SeverityCritical EventSeverity = "CRITICAL"
)

type SecurityEvent struct {
	Version  string         `json:"version"`
	TimeUnix int64          `json:"timeUnix"`
	TimeUTC  string         `json:"timeUtc"`
	Type     string         `json:"type"`
	Severity EventSeverity  `json:"severity"`
	Source   string         `json:"source"`
	Subject  string         `json:"subject,omitempty"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type EventLogFile struct {
	Version string          `json:"version"`
	Events  []SecurityEvent `json:"events"`
}

var eventMu sync.Mutex

func NewEvent(eventType string, severity EventSeverity, source, subject, message string, meta map[string]any) SecurityEvent {
	now := time.Now().UTC()
	return SecurityEvent{
		Version:  SecurityEventLogVersion,
		TimeUnix: now.Unix(),
		TimeUTC:  now.Format(time.RFC3339),
		Type:     eventType,
		Severity: severity,
		Source:   source,
		Subject:  subject,
		Message:  message,
		Metadata: meta,
	}
}

func AppendEvent(dir string, ev SecurityEvent) error {
	eventMu.Lock()
	defer eventMu.Unlock()

	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "security_events.json")

	payload := EventLogFile{
		Version: SecurityEventLogVersion,
		Events:  []SecurityEvent{},
	}

	if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}

	if payload.Version == "" {
		payload.Version = SecurityEventLogVersion
	}

	payload.Events = append(payload.Events, ev)

	// Keep last 1000 events for local node safety.
	if len(payload.Events) > 1000 {
		payload.Events = payload.Events[len(payload.Events)-1000:]
	}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

func ReadEvents(dir string, limit int) EventLogFile {
	eventMu.Lock()
	defer eventMu.Unlock()

	if dir == "" {
		dir = "."
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	path := filepath.Join(dir, "security_events.json")

	payload := EventLogFile{
		Version: SecurityEventLogVersion,
		Events:  []SecurityEvent{},
	}

	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		return payload
	}

	if err := json.Unmarshal(raw, &payload); err != nil {
		return EventLogFile{
			Version: SecurityEventLogVersion,
			Events:  []SecurityEvent{},
		}
	}

	if len(payload.Events) > limit {
		payload.Events = payload.Events[len(payload.Events)-limit:]
	}

	return payload
}

func Summary(dir string) map[string]any {
	events := ReadEvents(dir, 1000).Events

	counts := map[string]int{}
	severity := map[string]int{}

	for _, ev := range events {
		counts[ev.Type]++
		severity[string(ev.Severity)]++
	}

	networkHealth := "GOOD"
	if severity[string(SeverityCritical)] > 0 {
		networkHealth = "CRITICAL"
	} else if severity[string(SeverityHigh)] > 0 {
		networkHealth = "HIGH_RISK"
	} else if severity[string(SeverityWarning)] > 0 {
		networkHealth = "WATCH"
	}

	return map[string]any{
		"version":       SecurityEventLogVersion,
		"networkHealth": networkHealth,
		"totalEvents":   len(events),
		"byType":        counts,
		"bySeverity":    severity,
		"safeBoundary": map[string]bool{
			"changesChain":        false,
			"changesBalances":     false,
			"changesMiningReward": false,
			"changesConsensus":    false,
		},
	}
}

// Score documents the QuantumCoin component used by this module.
// SECURITY_SCORE_ENGINE_V1
// Produces a read-only security risk score from recent local security events.
// Safe boundary: no chain, balance, reward, supply, wallet, or consensus changes.
func Score(dir string) map[string]any {
	events := ReadEvents(dir, 1000).Events

	warnings := 0
	high := 0
	critical := 0
	points := 0

	recentTypes := map[string]int{}

	for _, ev := range events {
		recentTypes[ev.Type]++

		switch ev.Severity {
		case SeverityCritical:
			critical++
			points += 30
		case SeverityHigh:
			high++
			points += 12
		case SeverityWarning:
			warnings++
			points += 4
		default:
			points += 1
		}

		switch ev.Type {
		case "MEMPOOL_DOUBLE_SPEND_ATTEMPT",
			"TX_INTERNAL_DOUBLE_SPEND_ATTEMPT",
			"BLOCK_INVALID_POW",
			"BLOCK_PREVHASH_REJECTED",
			"CHAIN_REPLACED_REORG_WARNING",
			"P2P_INVALID_MAGIC_BANNED":
			points += 10
		case "HTTP_RATE_LIMIT_TRIGGERED",
			"ENDPOINT_RATE_LIMIT_TRIGGERED",
			"P2P_MESSAGE_DROPPED_BY_SECURITY":
			points += 5
		}
	}

	score := 100 - points
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	risk := "LOW"
	if critical > 0 || score < 40 {
		risk = "CRITICAL"
	} else if high > 0 || score < 70 {
		risk = "HIGH"
	} else if warnings > 0 || score < 90 {
		risk = "WATCH"
	}

	return map[string]any{
		"version":        "SECURITY_SCORE_ENGINE_V1",
		"score":          score,
		"risk":           risk,
		"eventsAnalyzed": len(events),
		"warnings":       warnings,
		"high":           high,
		"critical":       critical,
		"byType":         recentTypes,
		"safeBoundary": map[string]bool{
			"changesChain":        false,
			"changesBalances":     false,
			"changesMiningReward": false,
			"changesConsensus":    false,
			"changesTotalSupply":  false,
		},
	}
}
