package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"
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
// forever. Relative url() references inside CSS files are rewritten to the
// hashed names first, so a font change also changes the stylesheet's hash.
type staticAssets struct {
	files  map[string]*staticAsset // by served name, plain and hashed
	hashed map[string]string       // plain name → hashed name
}

// newStaticAssets reads every file in fsys. It fails if a stylesheet refers
// to a file that is not there.
func newStaticAssets(fsys fs.FS) (*staticAssets, error) {
	var names []string
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			names = append(names, name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list static files: %w", err)
	}
	// Stylesheets go last, so the files they refer to are already hashed.
	slices.SortStableFunc(names, func(a, b string) int {
		ac, bc := path.Ext(a) == ".css", path.Ext(b) == ".css"
		switch {
		case ac == bc:
			return strings.Compare(a, b)
		case ac:
			return 1
		}
		return -1
	})

	s := &staticAssets{files: make(map[string]*staticAsset), hashed: make(map[string]string)}
	for _, name := range names {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read static file %s: %w", name, err)
		}
		if path.Ext(name) == ".css" {
			if data, err = s.rewriteCSS(name, data); err != nil {
				return nil, err
			}
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:6])
		ext := path.Ext(name)
		hashedName := strings.TrimSuffix(name, ext) + "." + hash + ext
		etag := `"` + hash + `"`
		s.files[name] = &staticAsset{data: data, etag: etag}
		s.files[hashedName] = &staticAsset{data: data, etag: etag, immutable: true}
		s.hashed[name] = hashedName
	}
	return s, nil
}

// cssURL matches url(...) in a stylesheet, with or without quotes.
var cssURL = regexp.MustCompile(`url\(\s*(['"]?)([^'")]+)(['"]?)\s*\)`)

// rewriteCSS points relative url() references in the stylesheet name at the
// hashed names of the files they refer to.
func (s *staticAssets) rewriteCSS(name string, data []byte) ([]byte, error) {
	dir := path.Dir(name)
	var missing error
	out := cssURL.ReplaceAllFunc(data, func(m []byte) []byte {
		sub := cssURL.FindSubmatch(m)
		ref := string(sub[2])
		if isExternalRef(ref) {
			return m
		}
		target := path.Join(dir, ref)
		hashedName, ok := s.hashed[target]
		if !ok {
			if missing == nil {
				missing = fmt.Errorf("static file %s refers to %s, which does not exist", name, ref)
			}
			return m
		}
		// Hashing changes only the base name, so the reference keeps its
		// directory part.
		rel := path.Join(path.Dir(ref), path.Base(hashedName))
		return []byte("url(" + string(sub[1]) + rel + string(sub[3]) + ")")
	})
	if missing != nil {
		return nil, missing
	}
	return out, nil
}

// isExternalRef reports whether a url() reference points outside the
// static tree: another origin, an absolute path, a data URI or a fragment.
func isExternalRef(ref string) bool {
	return strings.Contains(ref, ":") || strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "#")
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
