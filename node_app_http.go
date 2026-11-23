package main

import (
	"crypto/subtle"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"quantumcoin/internal/httpresponse"
	"strings"
)

const apiMaxBodyBytes int64 = 1 << 20

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	httpresponse.WriteJSON(w, status, v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	httpresponse.WriteError(w, status, msg)
}

// --- DEV / LOCAL KONTROLLERİ ---

// QC_DEV_MODE ortam değişkeni "1", "true" veya "yes" ise dev modu açık sayıyoruz.
func isDevModeEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("QC_DEV_MODE")))
	return v == "1" || v == "true" || v == "yes"
}

// İstek sadece localhost'tan mı geldi?
func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Beklenmeyen formatta ise, olduğu gibi deneyelim
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// protected, handler'ı zorunlu header token kontrolü ve body limitiyle sarar.
// QC_API_TOKEN her çalışma modunda zorunludur.
func protected(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1) API token güvenliği
		if !checkAPIToken(w, r) {
			// checkAPIToken false döndürdüyse zaten 401/403 yazdı.
			return
		}

		// 2) Body boyut limiti (POST / PUT için)
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			r.Body = http.MaxBytesReader(w, r.Body, apiMaxBodyBytes)
		}

		// 3) Asıl handler
		h(w, r)
	}
}

/* ---------- HTTP API ---------- */

// main.go içinde, HTTP API bölümünde
func withCORS(next http.Handler) http.Handler {
	return withCORSForListenV1(next, getAPIListenAddr())
}

func withCORSForListenV1(next http.Handler, listenAddr string) http.Handler {
	restricted := apiRestrictedBindV1(listenAddr)
	allowedOrigins := corsAllowedOriginsFromEnvV1()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corsOK, originPresent := applyCORSHeadersV1(w, r, restricted, allowedOrigins)

		if r.Method == http.MethodOptions {
			if !corsOK {
				writeError(w, http.StatusForbidden, "cors origin not allowed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Public browser-originated requests must fail closed before any
		// application handler can perform reads or side effects.
		// Requests without Origin remain available to CLI/server clients.
		if originPresent && !corsOK {
			writeError(w, http.StatusForbidden, "cors origin not allowed")
			return
		}

		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

/* ---------- Handler yardımcıları ---------- */

// Eğer yukarıda zaten writeOK tanımlıysa, bunu SİLEBİLİRSİN (tek tanım kalsın).
func writeOK(w http.ResponseWriter, v interface{}) {
	writeJSON(w, http.StatusOK, v)
}

func apiEnvBool(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// apiRestrictedBindV1 takes the selected listener address, not the network
// label. Only a numeric loopback address is eligible for local CORS.
func apiRestrictedBindV1(listen string) bool {
	if apiEnvBool("QC_PUBLIC_TESTNET") ||
		apiEnvBool("QC_API_PUBLIC_TESTNET") ||
		apiEnvBool("QC_TESTNET_PUBLIC_API") {
		return true
	}

	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return true
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

func validateAPIBindAuthV1(listen string) error {
	if apiRestrictedBindV1(listen) && strings.TrimSpace(os.Getenv("QC_API_TOKEN")) == "" {
		return errors.New("QC_API_TOKEN is required before binding an exposed API listener")
	}
	return nil
}

// checkAPIToken requires a configured credential on protected routes in every mode.
func checkAPIToken(w http.ResponseWriter, r *http.Request) bool {
	// Custom API-key headers do not imply private proxy/browser caching.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Add("Vary", "Authorization")
	w.Header().Add("Vary", "X-QC-API-KEY")
	expected := strings.TrimSpace(os.Getenv("QC_API_TOKEN"))
	if expected == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="QuantumCoin API"`)
		writeError(w, http.StatusUnauthorized, "missing QC_API_TOKEN configuration")
		return false
	}

	// API tokens are accepted only through request headers in every mode.
	// URL query and form/body transport can expose credentials through logs,
	// browser history, reverse proxies, analytics, and referrer propagation.
	token, valid := apiTokenFromHeadersV1(r)
	if !valid || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		w.Header().Set("WWW-Authenticate", `Bearer realm="QuantumCoin API"`)
		writeError(w, http.StatusUnauthorized, "invalid or missing API token")
		return false
	}
	return true
}
