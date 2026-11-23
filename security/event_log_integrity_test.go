package security

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSecurityLogDamagedEvidencePreserved(t *testing.T) {
	for _, data := range []string{"", "{broken", `{"version":"unknown","events":[]}`} {
		dir := t.TempDir()
		path := filepath.Join(dir, "security_events.json")
		os.WriteFile(path, []byte(data), 0600)
		if err := AppendEvent(dir, NewEvent("test", SeverityCritical, "test", "subject", "evidence", nil)); err == nil {
			t.Fatal("damaged log overwritten")
		}
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, []byte(data)) {
			t.Fatal("evidence changed")
		}
		if ReadEvents(dir, 100).Error == "" || Score(dir)["risk"] != "UNKNOWN" {
			t.Fatal("damage silently reported healthy")
		}
	}
}
func TestSecurityEventLifecycleScoreAndReopen(t *testing.T) {
	dir := t.TempDir()
	old := NewEvent("old", SeverityCritical, "test", "subject", "old", nil)
	old.TimeUnix = time.Now().Add(-25 * time.Hour).Unix()
	if err := AppendEvent(dir, old); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		if err := AppendEvent(dir, NewEvent("repeat", SeverityWarning, "test", "subject", "repeat", nil)); err != nil {
			t.Fatal(err)
		}
	}
	if score := Score(dir); score["eventsAnalyzed"] != 1 || score["warnings"] != 1 || score["risk"] != "WATCH" {
		t.Fatal(score)
	}
	log := ReadEvents(dir, 1000)
	id := log.Events[len(log.Events)-1].ID
	if err := TransitionEvent(dir, id, "acknowledge", "test-operator", "investigated"); err != nil {
		t.Fatal(err)
	}
	if Score(dir)["risk"] != "WATCH" {
		t.Fatal("ack cleared unresolved risk")
	}
	if err := TransitionEvent(dir, id, "resolve", "test-operator", "recovered"); err != nil {
		t.Fatal(err)
	}
	if Score(dir)["risk"] != "LOW" || Summary(dir)["networkHealth"] != "GOOD" || SummarizeEvents(ReadEvents(dir, 1000).Events)["overallRisk"] != "LOW" {
		t.Fatal("resolved risk retained")
	}
	reopened := NewEvent("repeat", SeverityHigh, "test", "subject", "new evidence", nil)
	if err := AppendEvent(dir, reopened); err != nil {
		t.Fatal(err)
	}
	if Score(dir)["risk"] != "HIGH" {
		t.Fatal("new evidence did not reopen risk")
	}
	if err := TransitionEvent(dir, id, "resolve", "test-operator", "old request"); err != nil {
		t.Fatal(err)
	}
	if Score(dir)["risk"] != "HIGH" {
		t.Fatal("old resolution hid newer evidence")
	}
	persisted := ReadEvents(dir, 1000)
	if persisted.Events[1].Status != "resolved" || persisted.Events[1].Actor != "test-operator" {
		t.Fatal("lifecycle not persisted")
	}
}
func TestSecurityLogProcessWorker(t *testing.T) {
	dir := os.Getenv("QC_SECURITY_LOG_TEST_DIR")
	if dir == "" {
		return
	}
	for i := 0; i < 20; i++ {
		if err := AppendEvent(dir, NewEvent("worker", SeverityWarning, "test", fmt.Sprint(os.Getpid()), "event", nil)); err != nil {
			t.Fatal(err)
		}
	}
}
func TestSecurityLogConcurrentProcesses(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	fail := make(chan string, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(os.Args[0], "-test.run=^TestSecurityLogProcessWorker$", "-test.timeout=30s")
			cmd.Env = append(os.Environ(), "QC_SECURITY_LOG_TEST_DIR="+dir)
			if out, err := cmd.CombinedOutput(); err != nil {
				fail <- fmt.Sprintf("%v %s", err, out)
			}
		}()
	}
	wg.Wait()
	close(fail)
	for err := range fail {
		t.Error(err)
	}
	log := ReadEvents(dir, 1000)
	if log.Error != "" || len(log.Events) != 80 {
		t.Fatalf("lost process writes: %+v", log)
	}
	ids := map[string]bool{}
	for _, ev := range log.Events {
		if ids[ev.ID] {
			t.Fatal("duplicate ID")
		}
		ids[ev.ID] = true
	}
}
func TestSecurityNetworkProfileIsolation(t *testing.T) {
	directoryMu.RLock()
	previous := runtimeEventDirectory
	directoryMu.RUnlock()
	defer func() { directoryMu.Lock(); runtimeEventDirectory = previous; directoryMu.Unlock() }()
	chain := filepath.Join(t.TempDir(), "chain.dat")
	for _, profile := range []string{"local-release", "testnet-local", "testnet-public", "mainnet"} {
		if err := ConfigureEventDirectory(chain, profile); err != nil {
			t.Fatal(err)
		}
		if len(ReadEvents(".", 100).Events) != 0 {
			t.Fatal("network evidence leaked")
		}
		if err := AppendEvent(".", NewEvent(profile, SeverityWarning, "test", "", "network", nil)); err != nil {
			t.Fatal(err)
		}
		if got := ReadEvents(".", 100); len(got.Events) != 1 || got.Events[0].Type != profile {
			t.Fatal("wrong network projection")
		}
	}
	if err := ConfigureEventDirectory(chain, "../mainnet"); err == nil {
		t.Fatal("profile traversal allowed")
	}
	for _, profile := range []string{"local-release", "testnet-local", "testnet-public", "mainnet"} {
		ConfigureEventDirectory(chain, profile)
		if len(ReadEvents(".", 100).Events) != 1 {
			t.Fatal("network log not persistent")
		}
	}
}
func TestSecurityLogConcurrentReadersAndWriters(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 15; j++ {
				if err := AppendEvent(dir, NewEvent("race", SeverityInfo, "test", "", "race", nil)); err != nil {
					t.Error(err)
				}
				if got := ReadEvents(dir, 1000); got.Error != "" {
					t.Error(got.Error)
				}
			}
		}()
	}
	wg.Wait()
	if len(ReadEvents(dir, 1000).Events) != 60 {
		t.Fatal("lost concurrent evidence")
	}
}

func TestSecurityLogRejectsUnknownSeverityAndFutureEvidence(t *testing.T) {
	dir := t.TempDir()
	ev := NewEvent("typed", EventSeverity("CRITCAL"), "test", "", "invalid", nil)
	if err := AppendEvent(dir, ev); err == nil {
		t.Fatal("misspelled severity accepted")
	}
	ev = NewEvent("future", SeverityCritical, "test", "", "future", nil)
	ev.TimeUnix = time.Now().Add(48 * time.Hour).Unix()
	if got := ActiveSecurityEvents([]SecurityEvent{ev}, time.Now()); len(got) != 0 {
		t.Fatal("future event kept risk active")
	}
}
func TestLegacySecurityEventIdentityMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "security_events.json")
	event := NewEvent("legacy", SeverityWarning, "test", "subject", "old", nil)
	event.ID = ""
	event.Status = ""
	log := EventLogFile{Version: SecurityEventLogVersion, Events: []SecurityEvent{event, event}}
	if err := writeEventLogAtomic(path, log); err != nil {
		t.Fatal(err)
	}
	read := ReadEvents(dir, 100)
	if len(read.Events) != 2 || read.Events[0].ID == read.Events[1].ID {
		t.Fatal("legacy identities collided")
	}
	target := read.Events[1].ID
	if err := TransitionEvent(dir, target, "resolve", "operator", "legacy recovered"); err != nil {
		t.Fatal(err)
	}
	if err := AppendEvent(dir, NewEvent("new", SeverityInfo, "test", "", "new", nil)); err != nil {
		t.Fatal(err)
	}
	read = ReadEvents(dir, 100)
	if len(read.Events) != 3 || read.Events[1].ID != target || read.Events[0].Status != "resolved" {
		t.Fatal("legacy lifecycle migration lost identity")
	}
}

func TestSecurityCriticalScoreCannotDisappearInAdminReport(t *testing.T) {
	report := BuildSecurityAdminReadonlyReport(SecurityAdminReportInput{Score: map[string]any{"risk": "CRITICAL", "score": 0}, NodeHealth: map[string]any{"health": "GOOD"}, PeerReputation: map[string]any{"overallRisk": "LOW"}})
	if report["overallRisk"] != "HIGH" || report["overallStatus"] != "ACTION_REQUIRED" {
		t.Fatal(report)
	}
	report = BuildSecurityAdminReadonlyReport(SecurityAdminReportInput{Score: map[string]any{"risk": "LOW", "score": 100}, NodeHealth: map[string]any{"health": "GOOD"}, PeerReputation: map[string]any{"overallRisk": "UNKNOWN"}})
	if report["overallRisk"] != "UNKNOWN" {
		t.Fatal("missing peer schema reported healthy", report)
	}
}

func TestSecurityLogCrashWorker(t *testing.T) {
	dir := os.Getenv("QC_SECURITY_CRASH_TEST_DIR")
	if dir == "" {
		return
	}
	for i := 0; i < 1000; i++ {
		if err := AppendEvent(dir, NewEvent("crash-worker", SeverityWarning, "test", "worker", "evidence", nil)); err != nil {
			t.Fatal(err)
		}
	}
}
func TestSecurityLogRecoversAfterWriterTermination(t *testing.T) {
	dir := t.TempDir()
	seed := NewEvent("seed", SeverityHigh, "test", "seed", "prior evidence", nil)
	if err := AppendEvent(dir, seed); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSecurityLogCrashWorker$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "QC_SECURITY_CRASH_TEST_DIR="+dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if len(ReadEvents(dir, 1000).Events) > 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child never committed evidence")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	got := ReadEvents(dir, 1000)
	if got.Error != "" || len(got.Events) == 0 || got.Events[0].ID != seed.ID {
		t.Fatal("writer termination lost committed evidence", got)
	}
	if err := AppendEvent(dir, NewEvent("recovery", SeverityInfo, "test", "", "resumed", nil)); err != nil {
		t.Fatal("crashed writer retained lock", err)
	}
}
