package main

import (
	"net/http"
	"os"
	"strings"
)

// CORS_PUBLIC_MODE_SECURITY_V1
//
// Exposed listeners use an exact allowlist, including on local-release and mainnet.

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

func corsAllowedOriginV1(origin string, restricted bool, allowedOrigins []string) (string, bool) {
	origin = strings.TrimSpace(origin)

	if !restricted {
		return "*", true
	}

	if origin == "" {
		return "", false
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return origin, true
		}
	}

	return "", false
}

// corsRequestOriginV1 distinguishes omission (native clients) from an explicit
// empty or ambiguous Origin. Header.Get alone hides repeated header values.
func corsRequestOriginV1(r *http.Request) (origin string, present bool, valid bool) {
	for name, values := range r.Header {
		if !strings.EqualFold(name, "Origin") {
			continue
		}
		if present || len(values) != 1 {
			return "", true, false
		}
		present = true
		origin = strings.TrimSpace(values[0])
		if origin == "" {
			return "", true, false
		}
	}
	return origin, present, true
}

func applyCORSHeadersV1(w http.ResponseWriter, r *http.Request, restricted bool, allowedOrigins []string) (bool, bool) {
	origin, present, valid := corsRequestOriginV1(r)
	allowedOrigin, ok := corsAllowedOriginV1(origin, restricted, allowedOrigins)
	ok = ok && valid

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-QC-API-KEY")
	w.Header().Add("Vary", "Origin")
	w.Header().Del("Access-Control-Allow-Origin")

	if ok {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	}

	return ok, present
}
