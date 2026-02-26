package axon

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// SPAHandler serves embedded static files with SPA fallback.
// Files are served from the given subdirectory of the embed.FS.
// Unknown routes get index.html (for client-side routing).
// Paths under /_app/ return 404 if not found (prevents serving HTML for missing JS/CSS).
// index.html is served with Cache-Control: no-cache.
func SPAHandler(files embed.FS, subdir string) http.Handler {
	staticSub, err := fs.Sub(files, subdir)
	if err != nil {
		panic("axon: failed to create static sub filesystem: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(staticSub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Try to serve the file directly
		if _, err := fs.Stat(staticSub, path[1:]); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Don't serve index.html fallback for static asset paths --
		// returning HTML for a missing JS/CSS file breaks the browser.
		if strings.HasPrefix(path, "/_app/") {
			http.NotFound(w, r)
			return
		}

		// Fallback to index.html for SPA routing.
		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
