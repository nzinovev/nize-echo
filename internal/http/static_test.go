package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/nzinovev/nize-echo/web"
)

func embeddedStatic(t *testing.T) *staticAssets {
	t.Helper()
	manifest, err := web.AssetManifest()
	if err != nil {
		t.Fatal(err)
	}
	s, err := newStaticAssets(web.Static(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStaticServing(t *testing.T) {
	h := newHandler(t, Options{Logger: slog.New(&logRecorder{})})
	s := embeddedStatic(t)

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
	s := embeddedStatic(t)
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
