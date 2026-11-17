// Package web embeds the public site's templates and static files into the
// binary.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"

	"github.com/nzinovev/nize-echo/internal/staticassets"
)

//go:generate go run ./cmd/assetgen

//go:embed static
var static embed.FS

//go:embed static-assets.json
var assetManifest []byte

//go:embed templates
var templates embed.FS

// Static returns the files under web/static, rooted at that directory.
func Static() fs.FS { return sub(static, "static") }

// AssetManifest returns the fingerprints produced by the build.
func AssetManifest() (staticassets.Manifest, error) {
	var manifest staticassets.Manifest
	err := json.Unmarshal(assetManifest, &manifest)
	return manifest, err
}

// Templates returns the files under web/templates, rooted at that directory.
func Templates() fs.FS { return sub(templates, "templates") }

func sub(fsys embed.FS, dir string) fs.FS {
	s, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err) // dir is a constant embedded above
	}
	return s
}
