package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReportsWritableReadOnlyAndUnknownMode(t *testing.T) {
	for _, mode := range []string{"false", "true", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				extra := ""
				if mode != "unknown" {
					extra = `,"apiReadOnly":` + mode
				}
				fmt.Fprintf(w, `{"ok":true,"service":%q,"healthVersion":%q%s}`, HealthServiceV1, HealthProtocolV1, extra)
			}))
			defer server.Close()
			health, err := ProbeHealth(context.Background(), server.URL+CanonicalHealthPathV1, "")
			if err != nil || !health.Authenticated || health.ModeKnown != (mode != "unknown") || health.ReadOnly != (mode == "true") {
				t.Fatalf("unexpected mode %+v %v", health, err)
			}
		})
	}
}
