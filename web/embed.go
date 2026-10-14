// Package web embeds the public site's templates and static files into the
// binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var static embed.FS

//go:embed templates
var templates embed.FS

// Static returns the files under web/static, rooted at that directory.
func Static() fs.FS { return sub(static, "static") }

// Templates returns the files under web/templates, rooted at that directory.
func Templates() fs.FS { return sub(templates, "templates") }

func sub(fsys embed.FS, dir string) fs.FS {
	s, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err) // dir is a constant embedded above
	}
	return s
}
