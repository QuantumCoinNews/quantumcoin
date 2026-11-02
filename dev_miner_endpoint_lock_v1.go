package main

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Dev/miner control endpoints are intentionally available only to direct
// loopback clients while explicit development mode is enabled.
//
// Loopback is necessary but not sufficient for mutation endpoints: a local
// reverse proxy also appears as loopback to the backend. The stricter request
// gate below therefore additionally requires a local Host, rejects forwarding
// headers, validates browser origin metadata and requires an explicitly
// configured API token supplied only through an HTTP header.
func devMinerEndpointAllowedV1(r *http.Request) bool {
	if r == nil {
		return false
	}

	if isPublicTestnetAPITokenRequired() {
		return false
	}

	return isDevModeEnabled() && isLocalRequest(r)
}

func minerControlRequestHostIsLocal(r *http.Request) bool {
	if r == nil {
		return false
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		return false
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	host = strings.Trim(
		strings.TrimSpace(host),
		"[]",
	)

	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

func minerControlHasForwardingHeaders(r *http.Request) bool {
	if r == nil {
		return true
	}

	for _, name := range []string{
		"Forwarded",
		"X-Forwarded-For",
		"X-Real-IP",
	} {
		if strings.TrimSpace(r.Header.Get(name)) != "" {
			return true
		}
	}

	return false
}

func minerControlBrowserOriginAllowed(r *http.Request) bool {
	if r == nil {
		return false
	}

	origin := strings.TrimSpace(
		r.Header.Get("Origin"),
	)

	if origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil ||
			parsed.Scheme == "" ||
			parsed.Host == "" {
			return false
		}

		if parsed.Scheme != "http" &&
			parsed.Scheme != "https" {
			return false
		}

		if !strings.EqualFold(
			parsed.Host,
			strings.TrimSpace(r.Host),
		) {
			return false
		}
	}

	fetchSite := strings.ToLower(
		strings.TrimSpace(
			r.Header.Get("Sec-Fetch-Site"),
		),
	)

	if fetchSite != "" &&
		fetchSite != "same-origin" {
		return false
	}

	return true
}

func minerControlHeaderTokenValid(r *http.Request) bool {
	if r == nil {
		return false
	}

	expected := strings.TrimSpace(
		os.Getenv("QC_API_TOKEN"),
	)

	// Miner process mutation is never allowed with an implicit/empty token,
	// even in development mode.
	if expected == "" {
		return false
	}

	token := strings.TrimSpace(
		r.Header.Get("X-QC-API-KEY"),
	)

	if token == "" {
		auth := strings.TrimSpace(
			r.Header.Get("Authorization"),
		)

		if strings.HasPrefix(
			strings.ToLower(auth),
			"bearer ",
		) {
			token = strings.TrimSpace(
				auth[len("Bearer "):],
			)
		}
	}

	if token == "" {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(token),
		[]byte(expected),
	) == 1
}

func requireDevMinerEndpointAllowedV1(
	w http.ResponseWriter,
	r *http.Request,
	endpoint string,
) bool {
	if r == nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid miner control request",
		)
		return false
	}

	expectedMethod := http.MethodPost
	if endpoint == "/api/miner/status" {
		expectedMethod = http.MethodGet
	}

	if r.Method != expectedMethod {
		w.Header().Set(
			"Allow",
			expectedMethod,
		)

		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return false
	}

	if !devMinerEndpointAllowedV1(r) {
		writeError(
			w,
			http.StatusForbidden,
			endpoint+
				" disabled (dev-only, direct localhost, not public testnet)",
		)
		return false
	}

	if !minerControlRequestHostIsLocal(r) {
		writeError(
			w,
			http.StatusForbidden,
			endpoint+
				" requires a local request Host",
		)
		return false
	}

	// Never infer the original client from proxy headers for this local
	// process-control boundary. A proxied request is intentionally rejected.
	if minerControlHasForwardingHeaders(r) {
		writeError(
			w,
			http.StatusForbidden,
			endpoint+
				" does not accept proxied requests",
		)
		return false
	}

	if !minerControlBrowserOriginAllowed(r) {
		writeError(
			w,
			http.StatusForbidden,
			endpoint+
				" rejected browser origin",
		)
		return false
	}

	// Unlike generic protected endpoints, miner process mutation requires an
	// explicitly configured token and accepts it only through a request
	// header. Query-string and form token fallbacks are intentionally invalid.
	if !minerControlHeaderTokenValid(r) {
		w.Header().Set(
			"WWW-Authenticate",
			`Bearer realm="QuantumCoin Miner Control"`,
		)

		writeError(
			w,
			http.StatusUnauthorized,
			"invalid or missing miner control API token",
		)
		return false
	}

	return true
}
