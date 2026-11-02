package node

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDefaultHTTPServerOptionsV1AreSafeV1(
	t *testing.T,
) {
	options := DefaultHTTPServerOptionsV1()

	if options.ReadHeaderTimeout <= 0 ||
		options.ReadTimeout <= 0 ||
		options.WriteTimeout <= 0 ||
		options.IdleTimeout <= 0 {
		t.Fatal(
			"default HTTP timeouts must be positive",
		)
	}
}

func TestRuntimeServeHTTPRejectsEmptyAddressV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.ServeHTTP(
		"",
		http.NewServeMux(),
		DefaultHTTPServerOptionsV1(),
	)
	if err == nil {
		t.Fatal(
			"empty HTTP address was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"listen address is empty",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRuntimeServeHTTPRejectsNilHandlerV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.ServeHTTP(
		"127.0.0.1:0",
		nil,
		DefaultHTTPServerOptionsV1(),
	)
	if err == nil {
		t.Fatal(
			"nil HTTP handler was accepted",
		)
	}
}

func TestRuntimeServeHTTPRejectsUnsafeTimeoutsV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	options :=
		DefaultHTTPServerOptionsV1()

	options.ReadHeaderTimeout =
		0 * time.Second

	err := runtime.ServeHTTP(
		"127.0.0.1:0",
		http.NewServeMux(),
		options,
	)
	if err == nil {
		t.Fatal(
			"zero HTTP timeout was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"HTTP timeouts must be positive",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}
