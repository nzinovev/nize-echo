// Package staticassets defines the manifest embedded with the public assets.
package staticassets

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Manifest maps source paths to the fingerprints and transformed CSS produced
// before compilation.
type Manifest map[string]Asset

// Asset contains the hash and any rewritten CSS for one source file.
type Asset struct {
	Hash string `json:"hash"`
	CSS  string `json:"css,omitempty"`
}

// Filename inserts the asset's fingerprint before the extension.
func (a Asset) Filename(name string) string {
	ext := path.Ext(name)
	return strings.TrimSuffix(name, ext) + "." + a.Hash + ext
}

// Build fingerprints the supplied tree and resolves local CSS references.
func Build(fsys fs.FS) (Manifest, error) {
	var names []string
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Match go:embed's rules for files inside an embedded directory.
		if name != "." && (strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_")) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
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

	manifest := make(Manifest)
	for _, name := range names {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read static file %s: %w", name, err)
		}
		var asset Asset
		if path.Ext(name) == ".css" {
			rewritten, err := manifest.rewriteCSS(name, data)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(data, rewritten) {
				asset.CSS = string(rewritten)
				data = rewritten
			}
		}
		sum := sha256.Sum256(data)
		asset.Hash = hex.EncodeToString(sum[:6])
		manifest[name] = asset
	}
	return manifest, nil
}

// cssURL matches url(...) in a stylesheet, with or without quotes.
var cssURL = regexp.MustCompile(`url\(\s*(['"]?)([^'")]+)(['"]?)\s*\)`)

// rewriteCSS points relative url() references in the stylesheet name at the
// hashed names of the files they refer to.
func (m Manifest) rewriteCSS(name string, data []byte) ([]byte, error) {
	dir := path.Dir(name)
	var missing error
	out := cssURL.ReplaceAllFunc(data, func(match []byte) []byte {
		sub := cssURL.FindSubmatch(match)
		ref := string(sub[2])
		if isExternalRef(ref) {
			return match
		}
		target := path.Join(dir, ref)
		asset, ok := m[target]
		if !ok {
			if missing == nil {
				missing = fmt.Errorf("static file %s refers to %s, which does not exist", name, ref)
			}
			return match
		}
		// Hashing changes only the base name, so the reference keeps its
		// directory part.
		rel := path.Join(path.Dir(ref), path.Base(asset.Filename(target)))
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
