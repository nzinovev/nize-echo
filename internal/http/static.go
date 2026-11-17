package http

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/nzinovev/nize-echo/internal/staticassets"
)

// staticPrefix is the URL path the static files are served under.
const staticPrefix = "/static/"

const (
	// cacheImmutable is sent for content-hashed names: the bytes behind such a
	// name never change.
	cacheImmutable = "public, max-age=31536000, immutable"
	// cacheRevalidate is sent for plain names, whose content changes with
	// each build.
	cacheRevalidate = "public, no-cache"
)

// staticTypes lists the media types Go's built-in table lacks; the runtime
// image has no /etc/mime.types to fill the gap.
var staticTypes = map[string]string{
	".woff2": "font/woff2",
	".txt":   "text/plain; charset=utf-8",
}

// staticAsset is one file ready to serve.
type staticAsset struct {
	data []byte
	etag string
	// immutable is true when the name carries the content hash.
	immutable bool
}

// staticAssets serves a file tree under /static/, each file both by its own
// name and by a content-hashed one (bundle.css → bundle.3f9a0c1d2e4b.css).
// Pages link the hashed names through URL, so browsers may cache them
// for the lifetime advertised by cacheImmutable.
type staticAssets struct {
	files  map[string]*staticAsset // by served name, plain and hashed
	hashed map[string]string       // plain name → hashed name
}

// newStaticAssets loads the files and applies the prebuilt manifest.
func newStaticAssets(fsys fs.FS, manifest staticassets.Manifest) (*staticAssets, error) {
	s := &staticAssets{files: make(map[string]*staticAsset), hashed: make(map[string]string)}
	for name, asset := range manifest {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read static file %s: %w", name, err)
		}
		if asset.CSS != "" {
			data = []byte(asset.CSS)
		}
		hashedName := asset.Filename(name)
		etag := `"` + asset.Hash + `"`
		s.files[name] = &staticAsset{data: data, etag: etag}
		s.files[hashedName] = &staticAsset{data: data, etag: etag, immutable: true}
		s.hashed[name] = hashedName
	}
	return s, nil
}

// URL returns the path a page should link to for the static file name: its
// content-hashed name under /static/.
func (s *staticAssets) URL(name string) (string, error) {
	hashedName, ok := s.hashed[name]
	if !ok {
		return "", fmt.Errorf("unknown static file %q", name)
	}
	return staticPrefix + hashedName, nil
}

// ServeHTTP serves the file named by the path after /static/.
func (s *staticAssets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, staticPrefix)
	f, ok := s.files[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	if f.immutable {
		h.Set("Cache-Control", cacheImmutable)
	} else {
		h.Set("Cache-Control", cacheRevalidate)
	}
	h.Set("ETag", f.etag)
	if ct, ok := staticTypes[path.Ext(name)]; ok {
		h.Set("Content-Type", ct)
	}
	// ServeContent sets any other Content-Type from the extension and answers
	// conditional and range requests.
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(f.data))
}
