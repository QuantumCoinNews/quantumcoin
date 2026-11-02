package node

import (
	"strings"
	"testing"
)

func TestRuntimeObserverReceivesEventV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	received := make(
		chan NodeEvent,
		1,
	)

	err := runtime.AddObserver(
		ObserverFunc(
			func(event NodeEvent) {
				received <- event
			},
		),
	)
	if err != nil {
		t.Fatalf(
			"register observer: %v",
			err,
		)
	}

	runtime.EmitEvent(
		NodeEvent{
			Type:    "NODE_STARTED",
			Message: "node runtime started",
		},
	)

	event := <-received

	if event.Version != NodeEventVersionV1 {
		t.Fatalf(
			"version = %q, want %q",
			event.Version,
			NodeEventVersionV1,
		)
	}

	if event.Type != "NODE_STARTED" {
		t.Fatalf(
			"type = %q, want NODE_STARTED",
			event.Type,
		)
	}

	if event.Time.IsZero() {
		t.Fatal(
			"event time was not populated",
		)
	}

	if event.Severity !=
		NodeEventSeverityInfo {
		t.Fatalf(
			"severity = %q, want INFO",
			event.Severity,
		)
	}
}

func TestRuntimeRejectsNilObserverV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.AddObserver(nil)
	if err == nil {
		t.Fatal(
			"nil observer was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"observer is unavailable",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRuntimeObserverPanicDoesNotStopOtherObserversV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.AddObserver(
		ObserverFunc(
			func(NodeEvent) {
				panic("observer failure")
			},
		),
	)
	if err != nil {
		t.Fatalf(
			"register panic observer: %v",
			err,
		)
	}

	called := false

	err = runtime.AddObserver(
		ObserverFunc(
			func(NodeEvent) {
				called = true
			},
		),
	)
	if err != nil {
		t.Fatalf(
			"register healthy observer: %v",
			err,
		)
	}

	runtime.EmitEvent(
		NodeEvent{
			Type: "OBSERVER_TEST",
		},
	)

	if !called {
		t.Fatal(
			"healthy observer was blocked by panic observer",
		)
	}
}

func TestNilRuntimeObserverMethodsAreSafeV1(
	t *testing.T,
) {
	var runtime *Runtime

	if runtime.ObserverCount() != 0 {
		t.Fatal(
			"nil runtime returned observers",
		)
	}

	runtime.EmitEvent(
		NodeEvent{
			Type: "IGNORED",
		},
	)

	err := runtime.AddObserver(
		ObserverFunc(
			func(NodeEvent) {},
		),
	)
	if err == nil {
		t.Fatal(
			"nil runtime accepted observer",
		)
	}
}
