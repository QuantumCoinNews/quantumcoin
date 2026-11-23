package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestReviewAcrossProcesses(t *testing.T) {
	if dir := os.Getenv("QC_AI_RECORD_TEST_DIR"); dir != "" {
		for i := 0; i < 10; i++ {
			if err := WriteAISecurityReviewsTo(dir, []AISecurityReview{{TxID: fmt.Sprintf("%d-%d", os.Getpid(), i)}}); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	dir := t.TempDir()
	var wg sync.WaitGroup
	errs := make(chan string, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(os.Args[0], "-test.run=^TestReviewAcrossProcesses$")
			cmd.Env = append(os.Environ(), "QC_AI_RECORD_TEST_DIR="+dir)
			if out, err := cmd.CombinedOutput(); err != nil {
				errs <- fmt.Sprintf("%v: %s", err, out)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "ai_security_reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	var history AISecurityReviewFile
	if err = json.Unmarshal(data, &history); err != nil || len(history.Reviews) != 40 {
		t.Fatal("cross-process history lost", len(history.Reviews), err)
	}
	seen := map[string]bool{}
	for _, review := range history.Reviews {
		if seen[review.TxID] {
			t.Fatal("duplicate child record")
		}
		seen[review.TxID] = true
	}
}

func preserveObservations(t *testing.T) {
	t.Helper()
	alerts, telemetry, enabled := GetAlertsSnapshot(), GetTelemetrySnapshot(), Enabled()
	t.Cleanup(func() { setAlertsFromSnapshot(alerts); setTelemetryFromSnapshot(telemetry); SetEnabled(enabled) })
	SetEnabled(true)
}

func TestRetiredBonusCannotProduceStateOrFiles(t *testing.T) {
	preserveObservations(t)
	dir := t.TempDir()
	for _, enabled := range []bool{false, true} {
		SetEnabled(enabled)
		for _, amount := range []float64{1, 1000, -1, math.NaN(), math.Inf(1)} {
			SetBonus(amount, amount, "legacy")
			AddBonus(amount, "legacy")
			if GetBonusSnapshot() != (AIBonus{}) {
				t.Fatal("bonus state revived")
			}
		}
		for _, bonuses := range [][]BonusSuggestion{nil, {{WalletAddress: "owner", Amount: 100}}} {
			if !errors.Is(WriteAIBonusesTo(dir, bonuses), ErrAIBonusDisabled) {
				t.Fatal("export not retired")
			}
			if !errors.Is(CollectAndWrite(dir, bonuses), ErrAIBonusDisabled) {
				t.Fatal("collector bypass")
			}
		}
		if len(BuildAIBonuses(nil)) != 0 || len(OptimizeRewards(nil, nil)) != 0 || len(DistributeAIBonusesLite(nil)) != 0 {
			t.Fatal("bonus builder revived")
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("bonus wrote files", err)
	}
	SetEnabled(true)
}

func TestStateRoundTripIgnoresLegacyBonusAndPreservesCorruption(t *testing.T) {
	preserveObservations(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "ai_state.json")
	t.Setenv("QC_NODE_DIR", dir)
	legacy := []byte(`{"telemetry":{"timestamp":1,"peer_count":3,"hashrate":42},"bonus":{"total_bonus_qc":999999},"alerts":[{"id":"same","code":"A"},{"id":"same","code":"B"}]}`)
	if err := os.WriteFile(path, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadStateFrom(dir); err != nil {
		t.Fatal(err)
	}
	alerts := GetAlertsSnapshot()
	if len(alerts) != 2 || alerts[0].ID == alerts[1].ID {
		t.Fatal("legacy ID collision retained")
	}
	if GetBonusSnapshot() != (AIBonus{}) {
		t.Fatal("legacy bonus revived")
	}
	if err := SaveState(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if bytes.Contains(data, []byte(`"bonus"`)) {
		t.Fatal("bonus persisted")
	}
	setAlertsFromSnapshot(nil)
	UpdateTelemetry(0, 0)
	if err := LoadState(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(alerts, GetAlertsSnapshot()) || GetTelemetrySnapshot().PeerCount != 3 {
		t.Fatal("round trip lost observations")
	}
	before := GetAlertsSnapshot()
	for _, bad := range [][]byte{[]byte("broken"), []byte(`{"telemetry":{"peer_count":-1}}`), bytes.Repeat([]byte("x"), maxStoreBytes+1)} {
		if err := os.WriteFile(path, bad, 0600); err != nil {
			t.Fatal(err)
		}
		if err := LoadStateFrom(dir); err == nil {
			t.Fatal("bad state accepted")
		}
		if !reflect.DeepEqual(before, GetAlertsSnapshot()) {
			t.Fatal("failed load mutated alerts")
		}
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, bad) {
			t.Fatal("failed load replaced evidence")
		}
	}
}

func TestConcurrentAlertIDsAndBoundedSnapshots(t *testing.T) {
	preserveObservations(t)
	var wg sync.WaitGroup
	ids := make(chan string, 10000)
	for worker := 0; worker < 10; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				ids <- newAlertID()
				PushAlert(AlertInfo, "TEST", strings.Repeat("m", 2000), "test")
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatal("ID collision")
		}
		seen[id] = true
	}
	alerts := GetAlertsSnapshot()
	if len(alerts) != maxAlerts {
		t.Fatal("unbounded alerts")
	}
	for _, alert := range alerts {
		if len(alert.Message) > 1024 {
			t.Fatal("unbounded payload")
		}
	}
	alerts[0].Code = "mutated"
	if GetAlertsSnapshot()[0].Code == "mutated" {
		t.Fatal("snapshot aliases storage")
	}
}

func TestReviewHistoryConcurrentRetentionAndFailedWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai_security_reviews.json")
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- WriteAISecurityReviewsTo(dir, []AISecurityReview{{TxID: fmt.Sprint(i), Reason: "advisory", Score: 0.7}})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(path)
	var history AISecurityReviewFile
	if err := json.Unmarshal(data, &history); err != nil || len(history.Reviews) != 20 {
		t.Fatal("concurrent reviews lost", err)
	}
	batch := make([]AISecurityReview, 1100)
	for i := range batch {
		batch[i].TxID = fmt.Sprint(i)
	}
	if err := WriteAISecurityReviewsTo(dir, batch); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if err := json.Unmarshal(data, &history); err != nil || len(history.Reviews) != 1000 || history.Reviews[0].TxID != "100" || history.DroppedReviews != 120 {
		t.Fatal("retention mismatch", err)
	}
	if err := WriteAISecurityReviewsTo(dir, []AISecurityReview{{Score: math.NaN()}}); err == nil {
		t.Fatal("invalid JSON written")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(data, after) {
		t.Fatal("failed write damaged last good file")
	}
	corrupt := []byte("corrupt history")
	os.WriteFile(path, corrupt, 0600)
	if err := WriteAISecurityReviewsTo(dir, []AISecurityReview{{TxID: "later"}}); err == nil {
		t.Fatal("corrupt history silently replaced")
	}
	after, _ = os.ReadFile(path)
	if !bytes.Equal(after, corrupt) {
		t.Fatal("history evidence destroyed")
	}
}

func TestAtomicStateConcurrentWritersAndVisibleFailure(t *testing.T) {
	preserveObservations(t)
	dir := t.TempDir()
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				PushAlert(AlertInfo, "WRITE", "state", "test")
				UpdateTelemetry(j, math.NaN())
				errs <- SaveStateTo(dir)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := LoadStateFrom(dir); err != nil {
		t.Fatal(err)
	}
	if GetTelemetrySnapshot().HashrateAvailable {
		t.Fatal("unknown hashrate presented as measured")
	}
	badDir := filepath.Join(dir, "file")
	if err := os.WriteFile(badDir, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	err := SaveStateTo(badDir)
	if err == nil {
		t.Fatal("expected disk error")
	}
	before := GetPersistenceStatus().Failures
	RecordPersistenceFailure("test", err)
	if GetPersistenceStatus().Failures != before+1 || GetPersistenceStatus().LastError == "" {
		t.Fatal("storage failure invisible")
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ai-record-") {
			t.Fatal("temporary file leaked")
		}
	}
}
