package main

import (
	"net/http"
	"os"
	"strings"
)

// CORS_PUBLIC_MODE_SECURITY_V1
//
// Public testnet mode must not use wildcard CORS.
// Local/dev mode keeps the old wildcard behavior to avoid breaking local tools.

func corsAllowedOriginsFromEnvV1() []string {
	raw := strings.TrimSpace(os.Getenv("QC_CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("QC_CORS_ALLOWED_ORIGIN"))
	}
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" && v != "*" {
			out = append(out, v)
		}
	}
	return out
}

func corsAllowedOriginV1(origin string) (string, bool) {
	origin = strings.TrimSpace(origin)

	if !isPublicTestnetAPITokenRequired() {
		return "*", true
	}

	if origin == "" {
		return "", false
	}

	for _, allowed := range corsAllowedOriginsFromEnvV1() {
		if origin == allowed {
			return origin, true
		}
	}

	return "", false
}

func applyCORSHeadersV1(w http.ResponseWriter, r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	allowedOrigin, ok := corsAllowedOriginV1(origin)

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-QC-API-KEY")
	w.Header().Set("Vary", "Origin")

	if ok {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	}

	return ok
}
