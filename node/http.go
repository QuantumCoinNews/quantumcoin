package node

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const HTTPRuntimeVersionV1 = "QC_NODE_HTTP_RUNTIME_V1"

const defaultHTTPMaxHeaderBytesV1 = 1 << 20

type HTTPServerOptionsV1 struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

func DefaultHTTPServerOptionsV1() HTTPServerOptionsV1 {
	return HTTPServerOptionsV1{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    defaultHTTPMaxHeaderBytesV1,
	}
}

func effectiveHTTPMaxHeaderBytesV1(
	value int,
) int {
	if value <= 0 {
		return defaultHTTPMaxHeaderBytesV1
	}

	return value
}

func newHTTPServerV1(
	addr string,
	handler http.Handler,
	options HTTPServerOptionsV1,
) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: options.ReadHeaderTimeout,
		ReadTimeout:       options.ReadTimeout,
		WriteTimeout:      options.WriteTimeout,
		IdleTimeout:       options.IdleTimeout,
		MaxHeaderBytes:    effectiveHTTPMaxHeaderBytesV1(options.MaxHeaderBytes),
	}
}

// ServeHTTP owns creation and execution of the node HTTP server.
// Route registration and middleware composition remain outside this layer.
func (r *Runtime) ServeHTTP(
	addr string,
	handler http.Handler,
	options HTTPServerOptionsV1,
) error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			HTTPRuntimeVersionV1,
		)
	}

	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf(
			"%s: listen address is empty",
			HTTPRuntimeVersionV1,
		)
	}

	if handler == nil {
		return fmt.Errorf(
			"%s: handler is unavailable",
			HTTPRuntimeVersionV1,
		)
	}

	if options.ReadHeaderTimeout <= 0 ||
		options.ReadTimeout <= 0 ||
		options.WriteTimeout <= 0 ||
		options.IdleTimeout <= 0 {
		return fmt.Errorf(
			"%s: HTTP timeouts must be positive",
			HTTPRuntimeVersionV1,
		)
	}

	server := newHTTPServerV1(addr, handler, options)

	r.SetHTTPServer(server)

	r.EmitEvent(
		NodeEvent{
			Type:     "HTTP_SERVER_STARTING",
			Severity: NodeEventSeverityInfo,
			Message:  "node HTTP server starting",
			Data: map[string]any{
				"addr": addr,
			},
		},
	)

	err := server.ListenAndServe()
	if err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		r.EmitEvent(
			NodeEvent{
				Type:     "HTTP_SERVER_FAILED",
				Severity: NodeEventSeverityError,
				Message:  err.Error(),
				Data: map[string]any{
					"addr": addr,
				},
			},
		)

		return fmt.Errorf(
			"%s: listen %q: %w",
			HTTPRuntimeVersionV1,
			addr,
			err,
		)
	}

	r.EmitEvent(
		NodeEvent{
			Type:     "HTTP_SERVER_STOPPED",
			Severity: NodeEventSeverityInfo,
			Message:  "node HTTP server stopped",
			Data: map[string]any{
				"addr": addr,
			},
		},
	)

	return nil
}
