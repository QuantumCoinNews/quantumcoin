package main

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"quantumcoin/monitoring"
	"quantumcoin/security"
	"strings"
)

func handlePublisherMonitoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, 405, "method not allowed")
		return
	}
	remote, _, _ := net.SplitHostPort(r.RemoteAddr)
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = strings.Trim(r.Host, "[]")
	}
	expected := strings.TrimSpace(os.Getenv("QC_PUBLISHER_MONITOR_TOKEN"))
	if !net.ParseIP(remote).IsLoopback() || !net.ParseIP(host).IsLoopback() ||
		r.Header.Get("Forwarded") != "" || r.Header.Get("X-Forwarded-For") != "" ||
		expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(r.Header.Get("X-QC-PUBLISHER-MONITOR"))) != 1 {
		writeError(w, 403, "publisher authorization required")
		return
	}
	if nodeRuntime == nil || nodeRuntime.IsAPIReadOnly() {
		writeError(w, 403, "runtime is read-only")
		return
	}
	var report monitoring.PublisherReport
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		if !writeRequestBodyTooLargeV1(w, err) {
			writeError(w, 400, "invalid publisher report")
		}
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if writeRequestBodyTooLargeV1(w, err) {
			return
		}
		writeError(w, 400, "invalid publisher report")
		return
	}
	events, err := report.Events()
	if err != nil {
		writeError(w, 400, "invalid publisher report")
		return
	}
	for _, event := range events {
		if nodeRuntime.Monitoring().Record(event, true) != nil {
			writeError(w, 503, "monitoring storage unavailable")
			return
		}
	}
	writeOK(w, map[string]bool{"success": true})
}

type observationWriter struct {
	http.ResponseWriter
	status int
}

func (w *observationWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *observationWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(p)
}
func (w *observationWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func observeRuntimeHTTP(next http.Handler) http.Handler {
	runtime := nodeRuntime
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &observationWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			return
		}
		status := recorder.status
		if status == 0 {
			status = 200
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = "unknown"
		}
		if net.ParseIP(ip) == nil {
			ip = "unknown"
		}
		runtime.RecordHTTPObservation(ip, r.URL.Path, r.Method, status)
	})
}
func handleMonitoring(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	writeOK(w, nodeRuntime.Monitoring().Snapshot())
}
func handleMonitoringAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}
	if nodeRuntime == nil || nodeRuntime.IsAPIReadOnly() {
		writeError(w, 403, "runtime is read-only")
		return
	}
	expected := strings.TrimSpace(os.Getenv("QC_MONITORING_ADMIN_TOKEN"))
	supplied := r.Header.Get("X-QC-MONITORING-ADMIN")
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(supplied)) != 1 {
		writeError(w, 403, "monitoring operator authorization required")
		return
	}
	var body struct {
		ID     string `json:"id"`
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		if writeRequestBodyTooLargeV1(w, err) {
			return
		}
		writeError(w, 400, "invalid monitoring action")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if writeRequestBodyTooLargeV1(w, err) {
			return
		}
		writeError(w, 400, "multiple actions rejected")
		return
	}
	var actionErr error
	if r.URL.Path == "/api/security/events/action" {
		actionErr = security.TransitionEvent(".", body.ID, body.Action, "node-operator", body.Note)
	} else {
		actionErr = nodeRuntime.Monitoring().Transition(body.ID, body.Action, "node-operator", body.Note)
	}
	if err := actionErr; err != nil {
		writeError(w, 409, err.Error())
		return
	}
	writeOK(w, map[string]any{"success": true, "action": body.Action, "id": body.ID})
}
