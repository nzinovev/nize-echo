package staticassets

import (
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticAssetsHashAndRewrite(t *testing.T) {
	fsys := fstest.MapFS{
		".DS_Store":     {Data: []byte("local file")},
		"fonts/a.woff2": {Data: []byte("font A")},
		"fonts.css":     {Data: []byte(`@font-face{src:url(fonts/a.woff2) format('woff2')}`)},
		"nested/x.css":  {Data: []byte(`a{background:url("../fonts/a.woff2")}b{background:url(data:image/png;base64,AA==)}c{background:url(https://example.com/i.png)}`)},
	}
	s, err := Build(fsys)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, ok := s[".DS_Store"]; ok {
		t.Fatal("Build included a file skipped by go:embed")
	}

	fontHashed := s["fonts/a.woff2"].Filename("fonts/a.woff2")
	if !regexp.MustCompile(`^fonts/a\.[0-9a-f]{12}\.woff2$`).MatchString(fontHashed) {
		t.Fatalf("URL(fonts/a.woff2) = %q, want a hashed name", fontHashed)
	}

	css := s["fonts.css"].CSS
	if want := "url(" + fontHashed + ")"; !strings.Contains(css, want) {
		t.Errorf("fonts.css = %s, want it to contain %s", css, want)
	}

	nested := s["nested/x.css"].CSS
	for _, want := range []string{
		`url("../fonts/` + strings.TrimPrefix(fontHashed, "fonts/") + `")`,
		`url(data:image/png;base64,AA==)`,
		`url(https://example.com/i.png)`,
	} {
		if !strings.Contains(nested, want) {
			t.Errorf("nested/x.css = %s, want it to contain %s", nested, want)
		}
	}
}

// A font change must change the hash of the stylesheet that refers to it,
// or browsers would keep the old stylesheet forever.
func TestStaticAssetsHashFollowsReferences(t *testing.T) {
	css := &fstest.MapFile{Data: []byte(`@font-face{src:url(a.woff2)}`)}
	s1, err := Build(fstest.MapFS{"a.woff2": {Data: []byte("v1")}, "f.css": css})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := Build(fstest.MapFS{"a.woff2": {Data: []byte("v2")}, "f.css": css})
	if err != nil {
		t.Fatal(err)
	}
	u1 := s1["f.css"].Filename("f.css")
	u2 := s2["f.css"].Filename("f.css")
	if u1 == u2 {
		t.Errorf("f.css kept the name %s after the font it refers to changed", u1)
	}
}

func TestStaticAssetsMissingReference(t *testing.T) {
	_, err := Build(fstest.MapFS{"f.css": {Data: []byte(`a{background:url(gone.png)}`)}})
	if err == nil || !strings.Contains(err.Error(), "gone.png") {
		t.Fatalf("Build error = %v, want one naming gone.png", err)
	}
}
