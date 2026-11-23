package monitoring

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const MaxEvents = 1000
const MaxIncidents = 256
const MaxNotifications = 512

var fallbackSequence atomic.Uint64

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%d-%d-%d", os.Getpid(), time.Now().UnixNano(), fallbackSequence.Add(1))
}

type Incident struct {
	Action         string   `json:"action"`
	SafeBoundary   string   `json:"safe_boundary"`
	LastEventID    string   `json:"last_event_id"`
	AcknowledgedAt string   `json:"acknowledged_at,omitempty"`
	ResolvedAt     string   `json:"resolved_at,omitempty"`
	ID             string   `json:"id"`
	Key            string   `json:"key"`
	Source         string   `json:"source"`
	Category       string   `json:"category"`
	Severity       Severity `json:"severity"`
	Status         string   `json:"status"`
	Occurrences    int      `json:"occurrences"`
	FirstSeen      string   `json:"first_seen"`
	LastSeen       string   `json:"last_seen"`
	Actor          string   `json:"actor,omitempty"`
	Note           string   `json:"note,omitempty"`
}
type Notification struct {
	Category   string   `json:"category"`
	Severity   Severity `json:"severity"`
	Actor      string   `json:"actor,omitempty"`
	Note       string   `json:"note,omitempty"`
	ConsumedAt string   `json:"consumed_at,omitempty"`
	ID         string   `json:"id"`
	IncidentID string   `json:"incident_id"`
	Transition string   `json:"transition"`
	At         string   `json:"at"`
	ConsumedBy string   `json:"consumed_by,omitempty"`
}
type State struct {
	DroppedEvents uint64         `json:"dropped_events"`
	Events        []Event        `json:"events"`
	Incidents     []Incident     `json:"incidents"`
	Notifications []Notification `json:"notifications"`
}
type Engine struct {
	mu        sync.Mutex
	path      string
	state     State
	lastError string
	failures  uint64
	blocked   bool
}
type Snapshot struct {
	State
	LastError string `json:"last_error,omitempty"`
	Failures  uint64 `json:"failures"`
}

func Open(path string) (engine *Engine, openErr error) {
	e := &Engine{path: path}
	defer func() {
		if openErr != nil {
			e.blocked = true
			e.lastError = openErr.Error()
			e.failures++
		}
	}()
	if path == "" {
		return e, nil
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return e, nil
	}
	if err != nil {
		return e, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (8<<20)+1))
	if err != nil {
		return e, err
	}
	if len(b) > 8<<20 {
		return e, errors.New("monitoring state exceeds size limit")
	}
	var state State
	if err = json.Unmarshal(b, &state); err != nil {
		return e, err
	}
	if len(state.Events) > MaxEvents || len(state.Incidents) > MaxIncidents || len(state.Notifications) > MaxNotifications {
		return e, errors.New("monitoring state exceeds capacity")
	}
	for _, ev := range state.Events {
		if err = ValidateEvent(ev); err != nil {
			return e, err
		}
	}
	for _, incident := range state.Incidents {
		if incident.ID == "" || incident.Key == "" || (incident.Status != "open" && incident.Status != "acknowledged" && incident.Status != "resolved") {
			return e, errors.New("invalid monitoring incident")
		}
	}
	e.state = state
	return e, nil
}
func clone(s State) State {
	b, _ := json.Marshal(s)
	var out State
	_ = json.Unmarshal(b, &out)
	return out
}
func (e *Engine) Snapshot() Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	return Snapshot{State: clone(e.state), LastError: e.lastError, Failures: e.failures}
}
func (e *Engine) update(persist bool, fn func(*State) error) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	state := clone(e.state)
	if err := fn(&state); err != nil {
		return err
	}
	if persist && e.path != "" {
		if err := e.write(state); err != nil {
			e.lastError = err.Error()
			e.failures++
			return err
		}
	}
	e.state = state
	return nil
}
func (e *Engine) Flush() error { return e.update(true, func(*State) error { return nil }) }
func (e *Engine) write(state State) error {
	if e.blocked {
		return errors.New("monitoring store load failed; preserve file and repair before restart")
	}
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(b) > 8<<20 {
		return errors.New("monitoring state exceeds size limit")
	}
	if err = os.MkdirAll(filepath.Dir(e.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(e.path), ".monitoring-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
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
	return os.Rename(name, e.path)
}
func key(ev Event) string { return ev.Source + "\x00" + ev.Category + "\x00" + ev.Metadata["subject"] }
func notification(s *State, incidentID, transition string) error {
	// Discard consumed records first. Unconsumed notifications are never silently lost.
	for len(s.Notifications) >= MaxNotifications {
		removed := false
		for i, n := range s.Notifications {
			if n.ConsumedBy != "" {
				s.Notifications = append(s.Notifications[:i], s.Notifications[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			return errors.New("monitoring notification outbox is full")
		}
	}
	s.Notifications = append(s.Notifications, Notification{ID: NewID(), IncidentID: incidentID, Transition: transition, At: time.Now().UTC().Format(time.RFC3339Nano)})
	for _, in := range s.Incidents {
		if in.ID == incidentID {
			n := &s.Notifications[len(s.Notifications)-1]
			n.Category = in.Category
			n.Severity = in.Severity
			n.Actor = in.Actor
			n.Note = in.Note
			break
		}
	}
	return nil
}
func (e *Engine) Record(ev Event, persist bool) error {
	if err := ValidateEvent(ev); err != nil {
		return err
	}
	metadata := make(map[string]string, len(ev.Metadata))
	for k, v := range ev.Metadata {
		metadata[k] = v
	}
	ev.Metadata = metadata
	e.mu.Lock()
	for _, in := range e.state.Incidents {
		if in.LastEventID == ev.ID {
			e.mu.Unlock()
			return nil
		}
	}
	for _, old := range e.state.Events {
		if old.ID == ev.ID {
			e.mu.Unlock()
			return nil
		}
	}
	e.mu.Unlock()
	return e.update(persist, func(s *State) error {
		for _, old := range s.Events {
			if old.ID == ev.ID {
				return nil
			}
		}
		s.Events = append(s.Events, ev)
		if len(s.Events) > MaxEvents {
			s.DroppedEvents += uint64(len(s.Events) - MaxEvents)
			s.Events = s.Events[len(s.Events)-MaxEvents:]
		}
		if ev.Severity == SeverityInfo {
			return nil
		}
		for i := range s.Incidents {
			in := &s.Incidents[i]
			if in.Key != key(ev) {
				continue
			}
			transition := ""
			if in.Status == "resolved" {
				in.AcknowledgedAt = ""
				in.ResolvedAt = ""
				in.Status = "open"
				in.Actor = ""
				in.Note = ""
				in.Severity = ev.Severity
				transition = "reopened"
			}
			if ev.Severity == SeverityCritical && in.Severity != SeverityCritical {
				in.AcknowledgedAt = ""
				in.Severity = ev.Severity
				in.Status = "open"
				in.Actor = ""
				in.Note = ""
				transition = "escalated"
			}
			in.Occurrences++
			in.LastEventID = ev.ID
			in.LastSeen = ev.At
			if transition != "" {
				return notification(s, in.ID, transition)
			}
			return nil
		}
		if len(s.Incidents) >= MaxIncidents {
			removed := false
			for i, in := range s.Incidents {
				if in.Status == "resolved" {
					s.Incidents = append(s.Incidents[:i], s.Incidents[i+1:]...)
					removed = true
					break
				}
			}
			if !removed {
				return errors.New("monitoring incident capacity reached")
			}
		}
		in := Incident{LastEventID: ev.ID, ID: NewID(), Key: key(ev), Source: ev.Source, Category: ev.Category, Severity: ev.Severity, Status: "open", Occurrences: 1, FirstSeen: ev.At, LastSeen: ev.At}
		classified := ClassifyAlert(ev)
		in.Action = classified.Action
		in.SafeBoundary = classified.SafeBoundary
		s.Incidents = append(s.Incidents, in)
		return notification(s, in.ID, "opened")
	})
}
func (e *Engine) Transition(id, action, actor, note string) error {
	if id == "" || actor == "" || len(actor) > 128 || len(note) > 1024 {
		return errors.New("invalid monitoring action")
	}
	return e.update(true, func(s *State) error {
		if action == "consume" {
			for i := range s.Notifications {
				if s.Notifications[i].ID == id {
					s.Notifications[i].ConsumedBy = actor
					s.Notifications[i].ConsumedAt = time.Now().UTC().Format(time.RFC3339Nano)
					return nil
				}
			}
			return errors.New("notification not found")
		}
		for i := range s.Incidents {
			in := &s.Incidents[i]
			if in.ID != id {
				continue
			}
			switch action {
			case "ack":
				if in.Status == "resolved" {
					return errors.New("resolved incident cannot be acknowledged")
				}
				if in.Status == "acknowledged" {
					return nil
				}
				in.Status = "acknowledged"
				in.AcknowledgedAt = time.Now().UTC().Format(time.RFC3339Nano)
			case "resolve":
				if in.Status == "resolved" {
					return nil
				}
				in.Status = "resolved"
				in.ResolvedAt = time.Now().UTC().Format(time.RFC3339Nano)
			default:
				return errors.New("unknown monitoring action")
			}
			in.Actor = actor
			in.Note = note
			return notification(s, in.ID, in.Status)
		}
		return errors.New("incident not found")
	})
}
