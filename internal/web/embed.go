package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var content embed.FS

func Mount(mux *http.ServeMux) error {
	dist, err := fs.Sub(content, "dist")
	if err != nil {
		return fmt.Errorf("web embed: %w", err)
	}
	files := http.FileServerFS(dist)
	mux.HandleFunc("GET /{spaPath...}", func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("spaPath")
		if path != "" {
			if info, statErr := fs.Stat(dist, path); statErr == nil && !info.IsDir() {
				setCacheHeader(w, path)
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		files.ServeHTTP(w, r)
	})
	return nil
}

func setCacheHeader(w http.ResponseWriter, path string) {
	if strings.HasPrefix(path, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
}
