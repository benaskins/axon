package axon

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

type spaConfig struct {
	staticPrefix string
}

// SPAOption configures SPAHandler behavior.
type SPAOption func(*spaConfig)

// WithStaticPrefix sets a URL prefix for static assets that should 404
// on miss instead of falling back to index.html. For example,
// WithStaticPrefix("/_app/") prevents serving HTML for missing JS/CSS.
// When not set, all unknown paths fall back to index.html.
func WithStaticPrefix(prefix string) SPAOption {
	return func(c *spaConfig) {
		c.staticPrefix = prefix
	}
}

// SPAHandler serves embedded static files with SPA fallback.
// Files are served from the given subdirectory of the embed.FS.
// Unknown routes get index.html (for client-side routing).
// Use WithStaticPrefix to make paths under a given prefix 404 on miss
// instead of falling back to index.html.
// index.html is served with Cache-Control: no-cache.
func SPAHandler(files embed.FS, subdir string, opts ...SPAOption) http.Handler {
	cfg := &spaConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

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
		if cfg.staticPrefix != "" && strings.HasPrefix(path, cfg.staticPrefix) {
			http.NotFound(w, r)
			return
		}

		// Fallback to index.html for SPA routing.
		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
