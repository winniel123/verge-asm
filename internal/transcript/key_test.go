package transcript

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestLoadOrCreateKeyCreates(t *testing.T) {
	dir := t.TempDir()

	key, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateKey: %v", err)
	}
	if len(key) != keyLen {
		t.Fatalf("key length = %d, want %d", len(key), keyLen)
	}

	info, err := os.Stat(filepath.Join(dir, keyFile))
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	// Unix permission bits are not meaningful on Windows.
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("key file mode = %o, want 600", perm)
		}
	}
}

func TestLoadOrCreateKeyStable(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateKey first: %v", err)
	}
	second, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateKey second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("a second LoadOrCreateKey returned a different key")
	}
}

// TestLoadOrCreateKeyConcurrent proves the create is race-safe: many callers on a
// fresh shared dir all converge on one key, and exactly one key file remains (no
// split-brain, no leftover temp files). This is the web-and-worker co-boot case.
func TestLoadOrCreateKeyConcurrent(t *testing.T) {
	dir := t.TempDir()

	const n = 24
	keys := make([][]byte, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			keys[i], errs[i] = LoadOrCreateKey(dir)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	for i := 1; i < n; i++ {
		if !bytes.Equal(keys[i], keys[0]) {
			t.Fatalf("goroutine %d got a different key: split-brain", i)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read key dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != keyFile {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("key dir holds %v, want only %q (leftover temp or split-brain)", names, keyFile)
	}
}

func TestLoadOrCreateKeyRejectsCorruptLength(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, keyFile), []byte("too short"), 0o600); err != nil {
		t.Fatalf("write corrupt key: %v", err)
	}
	if _, err := LoadOrCreateKey(dir); err == nil {
		t.Fatal("LoadOrCreateKey accepted a wrong-length key, want corruption error")
	}
}

func TestEnsureKey(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureKey(dir); err != nil {
		t.Fatalf("EnsureKey: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, keyFile)); err != nil {
		t.Fatalf("EnsureKey did not create the key file: %v", err)
	}
}
