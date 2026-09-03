package corpus

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

var update = flag.Bool("update", false, "regenerate golden NDJSON files and corpus.lock.json")

const testdataDir = "testdata"

func TestMain(m *testing.M) {
	flag.Parse()
	if *update {
		if err := regenerate(); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}

func regenerate() error {
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		return err
	}
	rendered, err := RenderAll()
	if err != nil {
		return err
	}
	for name, b := range rendered {
		if err := os.WriteFile(filepath.Join(testdataDir, name), b, 0o644); err != nil {
			return err
		}
	}
	// golden-corpus.md §9's register is append-only, so a regenerate carries the existing rows.
	existing, _ := LoadLock(".")
	return WriteLock(".", Lock{
		LeafVersion:    wd.Version,
		CorpusDigest:   CorpusDigest(rendered),
		ParamsDigest:   ParamsDigest(),
		UncoveredMoves: existing.UncoveredMoves,
	})
}

func TestCorpusSelfIdentity(t *testing.T) {
	// A1: Go randomises map iteration, so an unstable render makes every later assertion moot.
	first, err := RenderAll()
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderAll()
	if err != nil {
		t.Fatal(err)
	}
	for name, b := range first {
		if !bytes.Equal(b, second[name]) {
			t.Errorf("row %s is not stable across two renders in one process", name)
		}
	}
}

func TestCorpusExpectation(t *testing.T) {
	// A2 is ADR-0021's gate in its first direction: output moved and the version did not.
	rendered, err := RenderAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range Rows {
		want, err := os.ReadFile(filepath.Join(testdataDir, r.Golden))
		if err != nil {
			t.Errorf("cell(s) %v: missing golden %s (run with -update): %v", r.Cells, r.Golden, err)
			continue
		}
		if !bytes.Equal(rendered[r.Golden], want) {
			t.Errorf("cell(s) %v: output moved without a blessed golden update.\n--- want (%s) ---\n%s\n--- got ---\n%s",
				r.Cells, r.Golden, want, rendered[r.Golden])
		}
	}
}

func TestCorpusCoverage(t *testing.T) {
	inEnum := make(map[string]bool, len(AllCells))
	for _, c := range AllCells {
		inEnum[c] = true
	}
	covered := make(map[string]bool, len(AllCells))
	for _, r := range Rows {
		for _, c := range r.Cells {
			if !inEnum[c] {
				t.Errorf("row %s cites cell %q, which is not in AllCells", r.Golden, c)
			}
			covered[c] = true
		}
	}
	for _, c := range AllCells {
		if !covered[c] {
			t.Errorf("cell %q holds no corpus row", c)
		}
	}
	if len(AllCells) != 19 {
		t.Errorf("wildcard-discrimination block is 19 cells (golden-corpus.md §8.4); AllCells has %d", len(AllCells))
	}
}

func TestCorpusLock(t *testing.T) {
	// Binding a lock edit to a version bump is A6's job in CI, not this test's.
	lock, err := LoadLock(".")
	if err != nil {
		t.Fatalf("load lock (run with -update to create it): %v", err)
	}
	if lock.LeafVersion != wd.Version {
		t.Errorf("lock leaf version %q != code version %q: bump one to match, and re-bless with -update",
			lock.LeafVersion, wd.Version)
	}
	rendered, err := RenderAll()
	if err != nil {
		t.Fatal(err)
	}
	if got := CorpusDigest(rendered); got != lock.CorpusDigest {
		t.Errorf("corpus digest moved: a row's expected output changed.\n  locked: %s\n  now:    %s\nBump wildcarddiscrim.Version and re-bless with -update (or the change is unintended).",
			lock.CorpusDigest, got)
	}
	if got := ParamsDigest(); got != lock.ParamsDigest {
		t.Errorf("params digest moved: a declared parameter changed.\n  locked: %s\n  now:    %s\nBump wildcarddiscrim.Version and re-bless with -update.",
			lock.ParamsDigest, got)
	}
}
