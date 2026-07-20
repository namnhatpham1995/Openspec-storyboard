package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"storyboard/frontend"
)

func spaHandler() http.Handler {
	assets := http.FileServer(http.FS(frontend.Dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "." {
			if info, err := fs.Stat(frontend.Dist, name); err == nil && !info.IsDir() {
				assets.ServeHTTP(w, r)
				return
			}
		}

		fallback := r.Clone(r.Context())
		fallback.URL.Path = "/"
		assets.ServeHTTP(w, fallback)
	})
}
