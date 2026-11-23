package httpresponse

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type secretMarshaler struct{}

func (secretMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("private-response-detail")
}

func TestHTTPResponseEncodingFailsBeforeCommit(t *testing.T) {
	var logged bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logged)
	defer log.SetOutput(old)
	for _, value := range []any{make(chan int), math.NaN(), math.Inf(1), secretMarshaler{}} {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Length", "1")
		w.Header().Set("Allow", "GET")
		WriteJSON(w, 200, value)
		var failure Error
		if w.Code != 500 || json.Unmarshal(w.Body.Bytes(), &failure) != nil || failure.Success || failure.Message != "failed to encode response" {
			t.Fatalf("bad fallback: %d %s", w.Code, w.Body.String())
		}
		if w.Header().Get("Content-Length") != "" || w.Header().Get("Allow") != "GET" {
			t.Fatal("stale length or lost method header")
		}
	}
	if !strings.Contains(logged.String(), "encode failed") || strings.Contains(logged.String(), "private-response-detail") {
		t.Fatal("encoding error was hidden or leaked payload", logged.String())
	}
}

func TestHTTPResponseTypedSuccessShapesStayCompatible(t *testing.T) {
	for _, value := range []any{map[string]any{"address": "public-address"}, []int{1, 2}, []int(nil), map[string]any{"success": true, "txid": "public-id"}} {
		w := httptest.NewRecorder()
		WriteJSON(w, 201, value)
		want, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != 201 || !bytes.Equal(w.Body.Bytes(), append(want, '\n')) || w.Header().Get("Content-Type") != "application/json" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("successful DTO changed", w.Body.String())
		}
	}
}

type failedWriter struct {
	header         http.Header
	status, writes int
	short          bool
}

func (w *failedWriter) Header() http.Header    { return w.header }
func (w *failedWriter) WriteHeader(status int) { w.status = status }
func (w *failedWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.short {
		return len(data) - 1, nil
	}
	return 0, errors.New("private-transport-detail")
}

func TestHTTPResponseTransportFailureIsLoggedWithoutSecondResponse(t *testing.T) {
	var logged bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logged)
	defer log.SetOutput(old)
	for _, short := range []bool{false, true} {
		w := &failedWriter{header: make(http.Header), short: short}
		WriteJSON(w, 200, map[string]bool{"ok": true})
		if w.writes != 1 || w.status != 200 {
			t.Fatal("tried to replace committed response")
		}
	}
	if strings.Count(logged.String(), "write failed") != 2 || strings.Contains(logged.String(), "private-transport-detail") {
		t.Fatal("write errors not safely observable", logged.String())
	}
}

func TestHTTPResponseErrorEnvelopeAndSingleDocument(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 405, 409, 410, 413, 429, 500, 501, 503} {
		w := httptest.NewRecorder()
		WriteError(w, status, "request rejected")
		decoder := json.NewDecoder(w.Body)
		var body map[string]any
		if w.Code != status || decoder.Decode(&body) != nil || body["success"] != false || body["message"] != "request rejected" || decoder.Decode(new(any)) != io.EOF {
			t.Fatal("invalid common error", status, body)
		}
	}
}
