// Package httpresponse owns JSON serialization for node HTTP adapters.
package httpresponse

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type Error struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// WriteJSON encodes before committing headers. Successful endpoint DTOs keep
// their existing shape. Failures are observable without logging response data
// or custom marshaler/writer error strings, which may contain secrets.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("http: JSON response encode failed (%T)", err)
		status = http.StatusInternalServerError
		data = []byte(`{"success":false,"message":"failed to encode response"}`)
	}
	data = append(data, '\n')
	w.Header().Del("Content-Length")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		log.Printf("http: JSON response write failed (%T)", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, Error{Success: false, Message: message})
}
