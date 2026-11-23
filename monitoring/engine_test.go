package monitoring

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestStrictSeverityIdentityAndTimestamp(t *testing.T) {
	for _, raw := range []Severity{"CRITCAL", "EROR", "", "DEBUG"} {
		e := NewEvent("id", "node", "fault", raw, "critical event")
		if ValidateEvent(e) == nil {
			t.Fatal("unknown severity accepted", raw)
		}
		if alert := ClassifyAlert(e); !alert.Open || alert.Severity != SeverityCritical {
			t.Fatal("invalid severity downgraded", alert)
		}
	}
	e := NewEvent("id", "node", "fault", Severity("warn"), "warning")
	if err := ValidateEvent(e); err != nil || e.Severity != SeverityWarning {
		t.Fatal("documented alias rejected", err)
	}
	for _, mutate := range []func(*Event){func(e *Event) { e.ID = "" }, func(e *Event) { e.At = "" }, func(e *Event) { e.At = "not-a-date" }} {
		bad := e
		mutate(&bad)
		if ValidateEvent(bad) == nil {
			t.Fatal("invalid event identity/time accepted")
		}
	}
}

func TestPersistentIncidentLifecycleAndLocalNotificationOutbox(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	engine, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	e := NewEvent("one", "node", "api_down", SeverityWarning, "offline")
	e.Metadata["subject"] = "api"
	if err = engine.Record(e, true); err != nil {
		t.Fatal(err)
	}
	e.Metadata["subject"] = "mutated"
	first := engine.Snapshot()
	if len(first.Incidents) != 1 || len(first.Notifications) != 1 || first.Events[0].Metadata["subject"] != "api" {
		t.Fatal("input aliased or incident missing")
	}
	e.Metadata["subject"] = "api"
	if err = engine.Record(e, true); err != nil {
		t.Fatal(err)
	}
	if len(engine.Snapshot().Events) != 1 {
		t.Fatal("replay not deduplicated")
	}
	e.ID = "two"
	if err = engine.Record(e, true); err != nil {
		t.Fatal(err)
	}
	if got := engine.Snapshot(); got.Incidents[0].Occurrences != 2 || len(got.Notifications) != 1 {
		t.Fatal("notification storm")
	}
	id := first.Incidents[0].ID
	if err = engine.Transition(id, "ack", "operator", "investigating"); err != nil {
		t.Fatal(err)
	}
	if engine.Snapshot().Incidents[0].AcknowledgedAt == "" {
		t.Fatal("ack audit missing")
	}
	e.ID = "three"
	e.Severity = SeverityCritical
	if err = engine.Record(e, true); err != nil {
		t.Fatal(err)
	}
	if engine.Snapshot().Incidents[0].Status != "open" {
		t.Fatal("escalation did not reopen")
	}
	if err = engine.Transition(id, "resolve", "operator", "fixed"); err != nil {
		t.Fatal(err)
	}
	if engine.Snapshot().Incidents[0].ResolvedAt == "" {
		t.Fatal("resolution audit missing")
	}
	e.ID = "four"
	if err = engine.Record(e, true); err != nil {
		t.Fatal(err)
	}
	if engine.Snapshot().Incidents[0].Status != "open" {
		t.Fatal("recurrence not reopened")
	}
	if err = engine.Transition(first.Notifications[0].ID, "consume", "operator", ""); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(engine.Snapshot().State, loaded.Snapshot().State) {
		t.Fatal("state/outbox restart drift")
	}
	if loaded.Snapshot().Notifications[0].ConsumedBy != "operator" {
		t.Fatal("consumption receipt lost")
	}
}

func TestConcurrentSinkRetentionAndFailureRollback(t *testing.T) {
	engine, _ := Open("")
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				if err := engine.Record(NewEvent(fmt.Sprintf("%d-%d", worker, i), "node", "same", SeverityWarning, "signal"), false); err != nil {
					t.Error(err)
				}
			}
		}(worker)
	}
	wg.Wait()
	snapshot := engine.Snapshot()
	if len(snapshot.Events) != 120 || snapshot.Incidents[0].Occurrences != 120 || len(snapshot.Notifications) != 1 {
		t.Fatal("concurrent event loss")
	}
	path := filepath.Join(t.TempDir(), "state.json")
	disk, _ := Open(path)
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := disk.Record(NewEvent("failed", "node", "x", SeverityCritical, "test"), true); err == nil {
		t.Fatal("write failure hidden")
	}
	if len(disk.Snapshot().Events) != 0 || disk.Snapshot().Failures == 0 {
		t.Fatal("failed mutation not rolled back")
	}
	corruptPath := filepath.Join(t.TempDir(), "state.json")
	raw := []byte("corrupt evidence")
	os.WriteFile(corruptPath, raw, 0600)
	broken, err := Open(corruptPath)
	if err == nil {
		t.Fatal("corrupt state accepted")
	}
	if err = broken.Record(NewEvent("new", "node", "x", SeverityInfo, "test"), true); err == nil {
		t.Fatal("corrupt state overwritten")
	}
	got, _ := os.ReadFile(corruptPath)
	if string(got) != string(raw) {
		t.Fatal("lost corruption evidence")
	}
}

func TestOutboxCapacityDoesNotDropUnconsumedNotifications(t *testing.T) {
	engine, _ := Open("")
	for i := 0; i < MaxNotifications; i++ {
		engine.state.Notifications = append(engine.state.Notifications, Notification{ID: fmt.Sprint(i)})
	}
	if err := engine.Record(NewEvent("full", "node", "critical", SeverityCritical, "signal"), false); err == nil {
		t.Fatal("outbox overflow hidden")
	}
	if len(engine.Snapshot().Incidents) != 0 || len(engine.Snapshot().Notifications) != MaxNotifications {
		t.Fatal("partial lifecycle commit")
	}
	engine.state.Notifications[0].ConsumedBy = "operator"
	if err := engine.Record(NewEvent("retry", "node", "critical", SeverityCritical, "signal"), false); err != nil {
		t.Fatal(err)
	}
	if len(engine.Snapshot().Notifications) != MaxNotifications {
		t.Fatal("outbox retention")
	}
	data, err := json.Marshal(engine.Snapshot())
	if err != nil || len(data) == 0 {
		t.Fatal(err)
	}
}
