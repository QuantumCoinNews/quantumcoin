package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Inspect every case-insensitive header occurrence. Header.Get alone hides
// duplicate values and in-process callers can also supply casing aliases.
func apiCredentialHeaderV1(r *http.Request, name string) (value string, present, valid bool) {
	if r == nil {
		return "", false, false
	}
	for key, values := range r.Header {
		if !strings.EqualFold(key, name) {
			continue
		}
		if present || len(values) != 1 {
			return "", true, false
		}
		present = true
		value = strings.TrimSpace(values[0])
		if value == "" {
			return "", true, false
		}
	}
	return value, present, true
}

// Credentials come only from headers. A supplied malformed/empty credential
// never falls back to another source. Two distinct supported headers are
// compatible only when both contain the same token.
func apiTokenFromHeadersV1(r *http.Request) (string, bool) {
	key, keyPresent, keyValid := apiCredentialHeaderV1(r, "X-QC-API-KEY")
	auth, authPresent, authValid := apiCredentialHeaderV1(r, "Authorization")
	if !keyValid || !authValid || (!keyPresent && !authPresent) {
		return "", false
	}
	var bearer string
	if authPresent {
		scheme, value, ok := strings.Cut(auth, " ")
		bearer = strings.TrimSpace(value)
		if !ok || !strings.EqualFold(scheme, "Bearer") || bearer == "" {
			return "", false
		}
	}
	if keyPresent && authPresent && subtle.ConstantTimeCompare([]byte(key), []byte(bearer)) != 1 {
		return "", false
	}
	if keyPresent {
		return key, true
	}
	return bearer, true
}
