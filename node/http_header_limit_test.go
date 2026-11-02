package node

import (
	"net/http"
	"testing"
)

func TestDefaultHTTPServerOptionsV1HasExplicitHeaderLimit(
	t *testing.T,
) {
	options := DefaultHTTPServerOptionsV1()

	if options.MaxHeaderBytes !=
		defaultHTTPMaxHeaderBytesV1 {
		t.Fatalf(
			"MaxHeaderBytes=%d want=%d",
			options.MaxHeaderBytes,
			defaultHTTPMaxHeaderBytesV1,
		)
	}

	if options.MaxHeaderBytes <= 0 {
		t.Fatal(
			"default MaxHeaderBytes must be positive",
		)
	}
}

func TestNewHTTPServerV1AppliesHeaderLimit(
	t *testing.T,
) {
	options := DefaultHTTPServerOptionsV1()
	options.MaxHeaderBytes = 64 << 10

	server := newHTTPServerV1(
		"127.0.0.1:8082",
		http.HandlerFunc(
			func(
				http.ResponseWriter,
				*http.Request,
			) {
			},
		),
		options,
	)

	if server.MaxHeaderBytes != options.MaxHeaderBytes {
		t.Fatalf(
			"server MaxHeaderBytes=%d want=%d",
			server.MaxHeaderBytes,
			options.MaxHeaderBytes,
		)
	}
}

func TestNewHTTPServerV1FallsBackForNonPositiveHeaderLimit(
	t *testing.T,
) {
	options := DefaultHTTPServerOptionsV1()
	options.MaxHeaderBytes = 0

	server := newHTTPServerV1(
		"127.0.0.1:8082",
		http.HandlerFunc(
			func(
				http.ResponseWriter,
				*http.Request,
			) {
			},
		),
		options,
	)

	if server.MaxHeaderBytes !=
		defaultHTTPMaxHeaderBytesV1 {
		t.Fatalf(
			"fallback MaxHeaderBytes=%d want=%d",
			server.MaxHeaderBytes,
			defaultHTTPMaxHeaderBytesV1,
		)
	}
}
