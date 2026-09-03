package designfs

import (
	"io/fs"
	"testing"
)

func TestFSExposesDesignArtifacts(t *testing.T) {
	want := []string{
		"templates/inventory.tmpl",
		"tokens/base.css",
		"tokens/colors.css",
		"fixtures/fixtures.json",
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

func TestTokensGlobbed(t *testing.T) {
	entries, err := fs.Glob(FS, "tokens/*.css")
	if err != nil {
		t.Fatalf("glob tokens/*.css: %v", err)
	}
	if len(entries) < 7 {
		t.Errorf("expected the full token set embedded, got %d: %v", len(entries), entries)
	}
}
