package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPGuardRejectsKnownOversizedBodyBeforeHandler(
	t *testing.T,
) {
	guard := NewHTTPGuard()
	handlerCalled := false

	handler := guard.Wrap(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				handlerCalled = true
				w.WriteHeader(http.StatusNoContent)
			},
		),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/body-limit",
		strings.NewReader(
			strings.Repeat(
				"x",
				int(httpMaxBodyBytes)+1,
			),
		),
	)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if handlerCalled {
		t.Fatal(
			"known oversized body reached application handler",
		)
	}

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"status=%d want=%d",
			rr.Code,
			http.StatusRequestEntityTooLarge,
		)
	}
}

func TestHTTPGuardTranslatesUnknownLengthOversizedBodyTo413(
	t *testing.T,
) {
	guard := NewHTTPGuard()

	handler := guard.Wrap(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				_, err := io.ReadAll(r.Body)
				if err != nil {
					if writeRequestBodyTooLargeV1(
						w,
						err,
					) {
						return
					}

					writeError(
						w,
						http.StatusBadRequest,
						"read body: "+err.Error(),
					)

					return
				}

				w.WriteHeader(http.StatusNoContent)
			},
		),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/body-limit",
		strings.NewReader(
			strings.Repeat(
				"x",
				int(httpMaxBodyBytes)+1,
			),
		),
	)

	// Simulate chunked/unknown-length input so the early Content-Length
	// rejection cannot be the mechanism under test.
	req.ContentLength = -1

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"status=%d want=%d",
			rr.Code,
			http.StatusRequestEntityTooLarge,
		)
	}
}

func TestHTTPGuardAllowsBodyWithinLimit(
	t *testing.T,
) {
	guard := NewHTTPGuard()
	handlerCalled := false

	handler := guard.Wrap(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				handlerCalled = true

				if _, err := io.ReadAll(r.Body); err != nil {
					t.Fatalf(
						"read body: %v",
						err,
					)
				}

				w.WriteHeader(http.StatusNoContent)
			},
		),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/body-limit",
		strings.NewReader(`{"ok":true}`),
	)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal(
			"body within limit did not reach handler",
		)
	}

	if rr.Code != http.StatusNoContent {
		t.Fatalf(
			"status=%d want=%d",
			rr.Code,
			http.StatusNoContent,
		)
	}
}

func TestRequestBodyTooLargeClassifierIgnoresOtherErrors(
	t *testing.T,
) {
	rr := httptest.NewRecorder()

	if writeRequestBodyTooLargeV1(
		rr,
		errors.New("ordinary read failure"),
	) {
		t.Fatal(
			"ordinary read error was classified as body-too-large",
		)
	}

	if rr.Code != http.StatusOK {
		t.Fatalf(
			"unexpected response status=%d",
			rr.Code,
		)
	}
}
