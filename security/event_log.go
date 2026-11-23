// Package security contains QuantumCoin project code for this module.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// SECURITY_EVENT_LOG_V1
// Read/write security event log for local node monitoring.
// Safe boundary: does not change chain, balances, wallets, mining rewards, or consensus.

const SecurityEventLogVersion = "SECURITY_EVENT_LOG_V1"
const MaxSecurityEventLogBytes = 8 << 20

type EventSeverity string

const (
	SeverityInfo     EventSeverity = "INFO"
	SeverityWarning  EventSeverity = "WARNING"
	SeverityHigh     EventSeverity = "HIGH"
	SeverityCritical EventSeverity = "CRITICAL"
)

type SecurityEvent struct {
	ID        string         `json:"id,omitempty"`
	Status    string         `json:"status,omitempty"`
	Actor     string         `json:"actor,omitempty"`
	Note      string         `json:"note,omitempty"`
	UpdatedAt string         `json:"updatedAt,omitempty"`
	Version   string         `json:"version"`
	TimeUnix  int64          `json:"timeUnix"`
	TimeUTC   string         `json:"timeUtc"`
	Type      string         `json:"type"`
	Severity  EventSeverity  `json:"severity"`
	Source    string         `json:"source"`
	Subject   string         `json:"subject,omitempty"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type EventLogFile struct {
	Version    string          `json:"version"`
	Events     []SecurityEvent `json:"events"`
	Error      string          `json:"error,omitempty"`
	WriteError string          `json:"writeError,omitempty"`
	Truncated  bool            `json:"truncated,omitempty"`
}

func readEventLogBounded(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, MaxSecurityEventLogBytes+1))
	if len(raw) > MaxSecurityEventLogBytes {
		return nil, fmt.Errorf("security event file exceeds read limit")
	}
	return raw, err
}

var eventMu sync.Mutex

func NewEvent(eventType string, severity EventSeverity, source, subject, message string, meta map[string]any) SecurityEvent {
	now := time.Now().UTC()
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		digest := sha256.Sum256(fmt.Appendf(nil, "%d:%d:%d", os.Getpid(), now.UnixNano(), eventSequence.Add(1)))
		copy(id[:], digest[:16])
	}
	return SecurityEvent{
		ID:       hex.EncodeToString(id[:]),
		Status:   "open",
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

func AppendEvent(dir string, ev SecurityEvent) (err error) {
	eventMu.Lock()
	defer eventMu.Unlock()
	defer func() {
		if err != nil {
			writeErrors[resolveEventDirectory(dir)] = err.Error()
		} else {
			delete(writeErrors, resolveEventDirectory(dir))
		}
	}()

	dir = resolveEventDirectory(dir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	unlock, err := lockRecordDirectory(dir)
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(dir, "security_events.json")

	payload := EventLogFile{
		Version: SecurityEventLogVersion,
		Events:  []SecurityEvent{},
	}

	if raw, err := readEventLogBounded(path); err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("damaged security log preserved: %w", err)
		}
		if payload.Version != SecurityEventLogVersion {
			return fmt.Errorf("unsupported security log version")
		}
		if err := validateEventLog(payload); err != nil {
			return err
		}
	} else if err == nil {
		return fmt.Errorf("empty security log preserved")
	} else if !os.IsNotExist(err) {
		return err
	}

	if payload.Version == "" {
		payload.Version = SecurityEventLogVersion
	}

	if ev.ID == "" {
		assignLegacyID(&ev)
	}
	ev.Status = "open"
	ev.Actor = ""
	ev.Note = ""
	ev.UpdatedAt = ""
	if err := validateEventLog(EventLogFile{Events: []SecurityEvent{ev}}); err != nil {
		return err
	}
	assignStoredEventIDs(payload.Events)
	for _, old := range payload.Events {
		if old.ID == ev.ID {
			return fmt.Errorf("duplicate security event identity")
		}
	}
	payload.Events = append(payload.Events, ev)

	// Keep last 1000 events for local node safety.
	if len(payload.Events) > 1000 {
		payload.Events = payload.Events[len(payload.Events)-1000:]
		payload.Truncated = true
	}

	return writeEventLogAtomic(path, payload)
}

func ReadEvents(dir string, limit int) EventLogFile {
	eventMu.Lock()
	defer eventMu.Unlock()

	dir = resolveEventDirectory(dir)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	path := filepath.Join(dir, "security_events.json")

	payload := EventLogFile{
		Version:    SecurityEventLogVersion,
		Events:     []SecurityEvent{},
		WriteError: writeErrors[dir],
	}

	if _, statErr := os.Stat(dir); statErr == nil {
		unlock, lockErr := lockRecordDirectory(dir)
		if lockErr != nil {
			payload.Error = lockErr.Error()
			return payload
		}
		defer unlock()
	}
	raw, err := readEventLogBounded(path)
	if err != nil || len(raw) == 0 {
		payload.Truncated = err != nil && !os.IsNotExist(err)
		if payload.Truncated {
			payload.Error = err.Error()
		} else if err == nil {
			payload.Error = "empty security log"
		}
		return payload
	}

	if err := json.Unmarshal(raw, &payload); err != nil {
		payload.Events = nil
		payload.Error = "damaged security log"
		return payload
	}
	if payload.Version != SecurityEventLogVersion {
		payload.Events = nil
		payload.Error = "unsupported security log version"
		return payload
	}
	if err := validateEventLog(payload); err != nil {
		payload.Events = nil
		payload.Error = err.Error()
		return payload
	}
	payload.WriteError = writeErrors[dir]
	assignStoredEventIDs(payload.Events)

	if len(payload.Events) > limit {
		payload.Events = payload.Events[len(payload.Events)-limit:]
	}

	return payload
}

func Summary(dir string) map[string]any {
	log := ReadEvents(dir, 1000)
	events := ActiveSecurityEvents(log.Events, time.Now())

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

	if log.Error != "" || (log.WriteError != "" && networkHealth == "GOOD") {
		networkHealth = "UNKNOWN"
	}
	return map[string]any{
		"version": SecurityEventLogVersion,
		"error":   log.Error, "writeError": log.WriteError, "windowSeconds": int(SecurityRiskWindow / time.Second),
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
	log := ReadEvents(dir, 1000)
	events := ActiveSecurityEvents(log.Events, time.Now())

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
			// Informational observations do not lower the security score.
		}

		if ev.Severity == SeverityInfo {
			continue
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

	if log.Error != "" || (log.WriteError != "" && risk == "LOW") {
		risk = "UNKNOWN"
	}
	return map[string]any{
		"error": log.Error, "writeError": log.WriteError, "windowSeconds": int(SecurityRiskWindow / time.Second), "acknowledgementClearsRisk": false,
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

var writeErrors = map[string]string{}
var eventSequence atomic.Uint64
var directoryMu sync.RWMutex
var runtimeEventDirectory string

const SecurityRiskWindow = 24 * time.Hour

// ConfigureEventDirectory binds compatibility callers to a validated runtime profile.
// Explicit directories remain available to isolated tools/tests. Legacy logs are never auto-imported.
func ConfigureEventDirectory(chainPath, profile string) error {
	switch profile {
	case "local-release", "testnet-local", "testnet-public", "mainnet":
	default:
		return fmt.Errorf("unknown security network profile")
	}
	absolute, err := filepath.Abs(chainPath)
	if err != nil {
		return err
	}
	directoryMu.Lock()
	runtimeEventDirectory = absolute + ".security/" + profile
	directoryMu.Unlock()
	return nil
}
func resolveEventDirectory(dir string) string {
	if dir != "" && dir != "." {
		return dir
	}
	directoryMu.RLock()
	selected := runtimeEventDirectory
	directoryMu.RUnlock()
	if selected != "" {
		return selected
	}
	return "."
}
func EventDirectory() string { return resolveEventDirectory(".") }
func assignLegacyID(ev *SecurityEvent) {
	if ev.ID != "" {
		return
	}
	raw, _ := json.Marshal(ev)
	digest := sha256.Sum256(raw)
	ev.ID = hex.EncodeToString(digest[:])
}

func assignStoredEventIDs(events []SecurityEvent) {
	for i := range events {
		if events[i].ID != "" {
			continue
		}
		raw, _ := json.Marshal(struct {
			Index int
			Event SecurityEvent
		}{i, events[i]})
		digest := sha256.Sum256(raw)
		events[i].ID = hex.EncodeToString(digest[:])
	}
}

func validateEventLog(log EventLogFile) error {
	if log.Events == nil {
		return fmt.Errorf("security log has no event array")
	}
	seen := map[string]bool{}
	for _, ev := range log.Events {
		if ev.ID != "" {
			if seen[ev.ID] {
				return fmt.Errorf("duplicate stored security identity")
			}
			seen[ev.ID] = true
		}
		if ev.TimeUnix <= 0 {
			return fmt.Errorf("security event timestamp missing")
		}
		switch ev.Severity {
		case SeverityInfo, SeverityWarning, SeverityHigh, SeverityCritical:
		default:
			return fmt.Errorf("unknown security severity")
		}
		switch ev.Status {
		case "", "open", "acknowledged", "resolved":
		default:
			return fmt.Errorf("unknown security event status")
		}
	}
	return nil
}
func writeEventLogAtomic(path string, payload EventLogFile) error {
	payload.WriteError = ""
	payload.Error = ""
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if len(raw) > MaxSecurityEventLogBytes {
		return fmt.Errorf("security event file exceeds write limit")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".security-event-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(raw)
	}
	if err == nil {
		err = f.Sync()
	}
	closed := f.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	return replaceRecord(name, path)
}

// ActiveSecurityEvents Risk groups repeated category/source/subject evidence, retains the highest severity,
// and expires observations after 24h. Acknowledgement alone does not resolve risk.
func ActiveSecurityEvents(events []SecurityEvent, now time.Time) []SecurityEvent {
	groups := map[string]SecurityEvent{}
	rank := map[EventSeverity]int{SeverityInfo: 0, SeverityWarning: 1, SeverityHigh: 2, SeverityCritical: 3}
	for _, ev := range events {
		if ev.Status == "resolved" || ev.TimeUnix <= 0 || ev.TimeUnix < now.Add(-SecurityRiskWindow).Unix() || ev.TimeUnix > now.Add(5*time.Minute).Unix() {
			continue
		}
		key := ev.Source + "\x00" + ev.Type + "\x00" + ev.Subject
		old, ok := groups[key]
		if !ok || rank[ev.Severity] > rank[old.Severity] || (rank[ev.Severity] == rank[old.Severity] && ev.TimeUnix > old.TimeUnix) {
			groups[key] = ev
		}
	}
	out := make([]SecurityEvent, 0, len(groups))
	for _, ev := range groups {
		out = append(out, ev)
	}
	return out
}

// TransitionEvent acknowledges/resolves the evidence group through the selected ID.
// A later event remains open. Changes and actor notes are committed atomically.
func TransitionEvent(dir, id, action, actor, note string) (err error) {
	if actor == "" || len(actor) > 128 || len(note) > 1024 || (action != "acknowledge" && action != "resolve") {
		return fmt.Errorf("invalid security action")
	}
	eventMu.Lock()
	defer eventMu.Unlock()
	dir = resolveEventDirectory(dir)
	unlock, err := lockRecordDirectory(dir)
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(dir, "security_events.json")
	raw, err := readEventLogBounded(path)
	if err != nil {
		return err
	}
	var log EventLogFile
	if err = json.Unmarshal(raw, &log); err != nil {
		return err
	}
	if log.Version != SecurityEventLogVersion {
		return fmt.Errorf("unsupported security log version")
	}
	if err := validateEventLog(log); err != nil {
		return err
	}
	index := -1
	assignStoredEventIDs(log.Events)
	for i := range log.Events {
		if log.Events[i].ID == id {
			index = i
		}
	}
	if index < 0 {
		return fmt.Errorf("security event not found")
	}
	target := log.Events[index]
	if target.Status == "resolved" && action == "acknowledge" {
		return fmt.Errorf("resolved evidence cannot be acknowledged")
	}
	for i := 0; i <= index; i++ {
		ev := &log.Events[i]
		if ev.Source == target.Source && ev.Type == target.Type && ev.Subject == target.Subject {
			if ev.Status == "resolved" {
				continue
			}
			ev.Status = "acknowledged"
			if action == "resolve" {
				ev.Status = "resolved"
			}
			ev.Actor = actor
			ev.Note = note
			ev.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
	}
	err = writeEventLogAtomic(path, log)
	if err != nil {
		writeErrors[dir] = err.Error()
	} else {
		delete(writeErrors, dir)
	}
	return err
}
