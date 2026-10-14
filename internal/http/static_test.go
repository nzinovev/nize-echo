package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/nzinovev/nize-echo/web"
)

func TestStaticAssetsHashAndRewrite(t *testing.T) {
	fsys := fstest.MapFS{
		"fonts/a.woff2": {Data: []byte("font A")},
		"fonts.css":     {Data: []byte(`@font-face{src:url(fonts/a.woff2) format('woff2')}`)},
		"nested/x.css":  {Data: []byte(`a{background:url("../fonts/a.woff2")}b{background:url(data:image/png;base64,AA==)}c{background:url(https://example.com/i.png)}`)},
	}
	s, err := newStaticAssets(fsys)
	if err != nil {
		t.Fatalf("newStaticAssets: %v", err)
	}

	fontURL, err := s.URL("fonts/a.woff2")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^/static/fonts/a\.[0-9a-f]{12}\.woff2$`).MatchString(fontURL) {
		t.Fatalf("URL(fonts/a.woff2) = %q, want a hashed name", fontURL)
	}
	fontHashed := strings.TrimPrefix(fontURL, "/static/")

	cssURL, _ := s.URL("fonts.css")
	css := s.files[strings.TrimPrefix(cssURL, "/static/")].data
	if want := "url(" + fontHashed + ")"; !bytes.Contains(css, []byte(want)) {
		t.Errorf("fonts.css = %s, want it to contain %s", css, want)
	}

	nestedURL, _ := s.URL("nested/x.css")
	nested := string(s.files[strings.TrimPrefix(nestedURL, "/static/")].data)
	for _, want := range []string{
		`url("../fonts/` + strings.TrimPrefix(fontHashed, "fonts/") + `")`,
		`url(data:image/png;base64,AA==)`,
		`url(https://example.com/i.png)`,
	} {
		if !strings.Contains(nested, want) {
			t.Errorf("nested/x.css = %s, want it to contain %s", nested, want)
		}
	}

	if _, err := s.URL("missing.css"); err == nil {
		t.Error("URL(missing.css) returned no error")
	}
}

// A font change must change the hash of the stylesheet that refers to it,
// or browsers would keep the old stylesheet forever.
func TestStaticAssetsHashFollowsReferences(t *testing.T) {
	css := &fstest.MapFile{Data: []byte(`@font-face{src:url(a.woff2)}`)}
	s1, err := newStaticAssets(fstest.MapFS{"a.woff2": {Data: []byte("v1")}, "f.css": css})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := newStaticAssets(fstest.MapFS{"a.woff2": {Data: []byte("v2")}, "f.css": css})
	if err != nil {
		t.Fatal(err)
	}
	u1, _ := s1.URL("f.css")
	u2, _ := s2.URL("f.css")
	if u1 == u2 {
		t.Errorf("f.css kept the name %s after the font it refers to changed", u1)
	}
}

func TestStaticAssetsMissingReference(t *testing.T) {
	_, err := newStaticAssets(fstest.MapFS{"f.css": {Data: []byte(`a{background:url(gone.png)}`)}})
	if err == nil || !strings.Contains(err.Error(), "gone.png") {
		t.Fatalf("newStaticAssets error = %v, want one naming gone.png", err)
	}
}

func TestStaticServing(t *testing.T) {
	h := newHandler(t, Options{Logger: slog.New(&logRecorder{})})
	s, err := newStaticAssets(web.Static())
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, contentType string
	}{
		{"bundle.css", "text/css; charset=utf-8"},
		{"tokens.css", "text/css; charset=utf-8"},
		{"fonts.css", "text/css; charset=utf-8"},
		{"fonts/spectral-400-cyrillic.woff2", "font/woff2"},
		{"fonts/onest-variable-cyrillic.woff2", "font/woff2"},
		{"fonts/ibm-plex-mono-400-cyrillic.woff2", "font/woff2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := s.URL(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			res := serve(t, h, http.MethodGet, u, nil)
			if res.Code != http.StatusOK {
				t.Fatalf("GET %s: status %d", u, res.Code)
			}
			if got := res.Header().Get("Cache-Control"); got != cacheImmutable {
				t.Errorf("GET %s: Cache-Control %q, want %q", u, got, cacheImmutable)
			}
			if got := res.Header().Get("Content-Type"); got != tc.contentType {
				t.Errorf("GET %s: Content-Type %q, want %q", u, got, tc.contentType)
			}

			plain := serve(t, h, http.MethodGet, "/static/"+tc.name, nil)
			if plain.Code != http.StatusOK {
				t.Fatalf("GET plain %s: status %d", tc.name, plain.Code)
			}
			if got := plain.Header().Get("Cache-Control"); got != cacheRevalidate {
				t.Errorf("GET plain %s: Cache-Control %q, want %q", tc.name, got, cacheRevalidate)
			}
			etag := plain.Header().Get("ETag")
			again := serve(t, h, http.MethodGet, "/static/"+tc.name, http.Header{"If-None-Match": {etag}})
			if again.Code != http.StatusNotModified {
				t.Errorf("GET plain %s with If-None-Match: status %d, want 304", tc.name, again.Code)
			}
		})
	}

	if res := serve(t, h, http.MethodGet, "/static/nope.css", nil); res.Code != http.StatusNotFound {
		t.Errorf("GET /static/nope.css: status %d, want 404", res.Code)
	}
}

// The public CSP allows only same-origin resources, so the served CSS must
// not reach out to Google Fonts or any other host.
func TestStaticCSSIsSelfContained(t *testing.T) {
	s, err := newStaticAssets(web.Static())
	if err != nil {
		t.Fatal(err)
	}
	for name, f := range s.files {
		if !strings.HasSuffix(name, ".css") {
			continue
		}
		if bytes.Contains(f.data, []byte("@import")) || bytes.Contains(f.data, []byte("http")) {
			t.Errorf("%s imports or refers to another origin", name)
		}
	}
}

func TestStyleguidePage(t *testing.T) {
	h := newHandler(t, Options{Logger: slog.New(&logRecorder{})})
	res := serve(t, h, http.MethodGet, "/styleguide", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("GET /styleguide: status %d", res.Code)
	}
	if got := res.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag %q, want noindex", got)
	}
	body := res.Body.String()
	links := regexp.MustCompile(`<link rel="stylesheet" href="([^"]+)">`).FindAllStringSubmatch(body, -1)
	if len(links) != 4 {
		t.Fatalf("found %d stylesheets, want 4", len(links))
	}
	for _, l := range links {
		if !regexp.MustCompile(`^/static/[a-z]+\.[0-9a-f]{12}\.css$`).MatchString(l[1]) {
			t.Errorf("stylesheet %q is not a hashed /static/ name", l[1])
		}
		if res := serve(t, h, http.MethodGet, l[1], nil); res.Code != http.StatusOK {
			t.Errorf("GET %s: status %d", l[1], res.Code)
		}
	}
	for _, class := range []string{`class="nz-root`, `class="nz-header"`, `class="nz-prose nz-prose--dropcap nz-prose--numbered"`, `class="nz-cta`, `class="nz-btn nz-btn--primary"`} {
		if !strings.Contains(body, class) {
			t.Errorf("page lacks %s", class)
		}
	}
}

// The copies in web/static must match the design system they come from.
// design/ is not in git, so this runs only where it is present.
func TestStaticMatchesDesignSystem(t *testing.T) {
	designDir := filepath.Join("..", "..", "design", "system")
	if _, err := os.Stat(designDir); err != nil {
		t.Skip("design/system is not present")
	}
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	staticDir := filepath.Join("..", "..", "web", "static")

	if !bytes.Equal(read(filepath.Join(staticDir, "tokens.css")), read(filepath.Join(designDir, "tokens.css"))) {
		t.Error("web/static/tokens.css differs from design/system/tokens.css; copy it again")
	}
	// bundle.css is copied without its first line, the Google Fonts @import,
	// and the blank line after it; fonts.css replaces the import.
	want := read(filepath.Join(designDir, "bundle.css"))
	first, rest, _ := bytes.Cut(want, []byte("\n"))
	if !bytes.HasPrefix(first, []byte("@import url('https://fonts.googleapis.com/")) {
		t.Fatalf("design/system/bundle.css no longer starts with the Google Fonts @import: %s", first)
	}
	want = bytes.TrimPrefix(rest, []byte("\n"))
	if !bytes.Equal(read(filepath.Join(staticDir, "bundle.css")), want) {
		t.Error("web/static/bundle.css differs from design/system/bundle.css; copy it again without the @import line")
	}
}
