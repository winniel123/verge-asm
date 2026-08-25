package designfs

import (
	"io/fs"
	"testing"
)

// TestFSExposesDesignArtifacts pins the read-only surface every screen
// conversion and the verify harness depend on: the inventory template, the
// full token set, the fixture corpus, and the verify configs must all be
// reachable through FS by their package-relative paths.
func TestFSExposesDesignArtifacts(t *testing.T) {
	want := []string{
		"templates/inventory.tmpl",
		"tokens/base.css",
		"tokens/colors.css",
		"fixtures/fixtures.json",
		"verify/config.json",
		"verify/states.json",
	}
	for _, name := range want {
		b, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Errorf("FS is missing %q: %v", name, err)
			continue
		}
		if len(b) == 0 {
			t.Errorf("FS entry %q is empty", name)
		}
	}
}

// TestTokensGlobbed asserts the tokens/*.css glob captured the whole set, not
// just the two files spot-checked above, so a token file added by a future
// package version is served without touching this package.
func TestTokensGlobbed(t *testing.T) {
	entries, err := fs.Glob(FS, "tokens/*.css")
	if err != nil {
		t.Fatalf("glob tokens/*.css: %v", err)
	}
	if len(entries) < 7 {
		t.Errorf("expected the full token set embedded, got %d: %v", len(entries), entries)
	}
}
