package node

import (
	"fmt"
	"strings"
	"time"
)

const NodeEventVersionV1 = "QC_NODE_EVENT_V1"

const (
	NodeEventSeverityInfo    = "INFO"
	NodeEventSeverityWarning = "WARNING"
	NodeEventSeverityError   = "ERROR"
)

// NodeEvent is a read-only notification emitted by the node runtime.
//
// Observers may inspect or record events, but must not mutate consensus,
// balances, total supply, rewards, difficulty or chain validity.
type NodeEvent struct {
	Version  string         `json:"version"`
	Type     string         `json:"type"`
	Time     time.Time      `json:"time"`
	Severity string         `json:"severity"`
	Message  string         `json:"message,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

// Observer receives read-only node lifecycle and operational events.
type Observer interface {
	ObserveNodeEvent(NodeEvent)
}

// ObserverFunc adapts a function to the Observer interface.
type ObserverFunc func(NodeEvent)

// ObserveNodeEvent implements Observer.
func (f ObserverFunc) ObserveNodeEvent(event NodeEvent) {
	if f != nil {
		f(event)
	}
}

// AddObserver registers a read-only runtime observer.
func (r *Runtime) AddObserver(
	observer Observer,
) error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			NodeEventVersionV1,
		)
	}

	if observer == nil {
		return fmt.Errorf(
			"%s: observer is unavailable",
			NodeEventVersionV1,
		)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.observers = append(
		r.observers,
		observer,
	)

	return nil
}

// ObserverCount returns the number of registered observers.
func (r *Runtime) ObserverCount() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.observers)
}

// EmitEvent delivers a read-only event to a snapshot of registered observers.
// A broken observer cannot stop the node or prevent other observers running.
func (r *Runtime) EmitEvent(
	event NodeEvent,
) {
	if r == nil {
		return
	}

	event.Version = NodeEventVersionV1

	event.Type = strings.TrimSpace(
		event.Type,
	)
	if event.Type == "" {
		event.Type = "NODE_EVENT"
	}

	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}

	event.Severity = strings.ToUpper(
		strings.TrimSpace(
			event.Severity,
		),
	)
	if event.Severity == "" {
		event.Severity =
			NodeEventSeverityInfo
	}

	r.mu.RLock()

	observers := append(
		[]Observer(nil),
		r.observers...,
	)

	r.mu.RUnlock()

	for _, observer := range observers {
		safelyNotifyObserverV1(
			observer,
			event,
		)
	}
}

func safelyNotifyObserverV1(
	observer Observer,
	event NodeEvent,
) {
	if observer == nil {
		return
	}

	defer func() {
		_ = recover()
	}()

	observer.ObserveNodeEvent(
		event,
	)
}
