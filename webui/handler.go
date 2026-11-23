// Package webui contains QuantumCoin project code for this module.
package webui

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Handler serves the embedded UI and limits index fallback to navigation paths.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return nil, fmt.Errorf("webui: dist not found; build frontend into webui/dist")
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean(r.URL.Path)
		if isAPIPath(r.URL.Path) || isAPIPath(cleanPath) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"message": "api endpoint not found",
			})
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'unsafe-inline'; connect-src 'self'; worker-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")

		fp := strings.TrimPrefix(cleanPath, "/")
		if fp == "" {
			fp = "index.html"
		}
		if entry, err := fs.Stat(sub, fp); err == nil && !entry.IsDir() {
			if path.Ext(fp) == ".mjs" {
				// Module scripts require a JavaScript MIME type even if the
				// host's extension registry has no .mjs association.
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		if path.Ext(cleanPath) != "" || cleanPath == "/assets" || strings.HasPrefix(cleanPath, "/assets/") {
			http.NotFound(w, r)
			return
		}
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = "/"
		r2.URL.RawPath = ""
		fileServer.ServeHTTP(w, r2)
	}), nil
}

func isAPIPath(p string) bool {
	return p == "/api" || strings.HasPrefix(p, "/api/")
}
