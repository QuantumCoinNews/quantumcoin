package webui

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Handler: SPA (single-page app) servis eder. dist yoksa hata döndürür.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return nil, fmt.Errorf("webui: dist not found; build frontend into webui/dist")
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if fp == "" {
			fp = "index.html"
		}
		if f, err := sub.Open(fp); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback → index.html
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	}), nil
}
