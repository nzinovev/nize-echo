package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"regexp"
	"testing"

	"github.com/nzinovev/nize-echo/internal/staticassets"
)

func TestEmbeddedAssetManifestIsCurrent(t *testing.T) {
	got, err := AssetManifest()
	if err != nil {
		t.Fatal(err)
	}
	want, err := staticassets.Build(Static())
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(got, want) {
		t.Fatal("static-assets.json is stale; run make assets and commit the result")
	}
}

func TestStaticMatchesDesignRevision(t *testing.T) {
	data, err := os.ReadFile("design.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Revision string            `json:"studio_revision"`
		CSS      map[string]string `json:"css_sha256"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(lock.Revision) {
		t.Fatalf("invalid studio revision %q in design.lock.json", lock.Revision)
	}
	for _, name := range []string{"tokens.css", "bundle.css"} {
		source, err := fs.ReadFile(Static(), name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(source)
		if got := hex.EncodeToString(sum[:]); got != lock.CSS[name] {
			t.Errorf("%s does not match locked studio revision %s; see docs/design.md", name, lock.Revision)
		}
	}
}
