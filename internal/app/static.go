package app

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	almanachweb "github.com/go-go-golems/almanach/internal/web"
)

// registerStaticRoutes serves the Almanach Studio SPA files.
//
// If webDir points at an existing directory, files are served from disk for
// development. Otherwise the prebuilt assets copied by cmd/build-web into
// internal/web/embed/public are served, and builds with -tags embed bundle them
// into the Go binary.
func registerStaticRoutes(mux *http.ServeMux, webDir string) {
	staticFS := almanachweb.PublicFS
	serveFromDisk := false
	if webDir != "" {
		if st, err := os.Stat(webDir); err == nil && st.IsDir() {
			serveFromDisk = true
		} else {
			log.Printf("WARNING: web dir %s does not exist — falling back to bundled Almanach Studio assets", webDir)
		}
	}

	mux.HandleFunc("/almanach", func(w http.ResponseWriter, r *http.Request) {
		if serveFromDisk {
			serveDiskFile(w, filepath.Join(webDir, "index.html"), "text/html; charset=utf-8")
			return
		}
		serveFSFile(w, staticFS, "index.html", "text/html; charset=utf-8")
	})

	mux.HandleFunc("/almanach/bundle.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		if serveFromDisk {
			serveDiskFile(w, filepath.Join(webDir, "almanach-bundle.js"), "application/javascript; charset=utf-8")
			return
		}
		serveFSFile(w, staticFS, "almanach-bundle.js", "application/javascript; charset=utf-8")
	})

	mux.HandleFunc("/setup", func(w http.ResponseWriter, r *http.Request) {
		if serveFromDisk {
			serveDiskFile(w, filepath.Join(webDir, "setup.html"), "text/html; charset=utf-8")
			return
		}
		serveFSFile(w, staticFS, "setup.html", "text/html; charset=utf-8")
	})

	mux.HandleFunc("/setup/bundle.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		if serveFromDisk {
			serveDiskFile(w, filepath.Join(webDir, "setup-bundle.js"), "application/javascript; charset=utf-8")
			return
		}
		serveFSFile(w, staticFS, "setup-bundle.js", "application/javascript; charset=utf-8")
	})
}

func serveDiskFile(w http.ResponseWriter, path, contentType string) {
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeStaticBytes(w, data, contentType)
}

func serveFSFile(w http.ResponseWriter, source fs.FS, name, contentType string) {
	data, err := fs.ReadFile(source, name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeStaticBytes(w, data, contentType)
}

func writeStaticBytes(w http.ResponseWriter, data []byte, contentType string) {
	data = bytesTrimTrailingNUL(data)
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}

func bytesTrimTrailingNUL(data []byte) []byte {
	for len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	return data
}

// sanitizePath prevents directory traversal.
func sanitizePath(p string) string {
	p = filepath.Clean(p)
	for strings.HasPrefix(p, "../") {
		p = strings.TrimPrefix(p, "../")
	}
	return p
}
