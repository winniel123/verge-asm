package corpus

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
)

// update regenerates the golden files and the lock. It is the deliberate bless
// action: run `go test ./... -run Corpus -update` after an intended output or
// parameter change, having bumped blanketdiscrim.Version, and commit the result.
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
	existing, _ := LoadLock(".")
	return WriteLock(".", Lock{
		LeafVersion:    blanketdiscrim.Version,
		CorpusDigest:   CorpusDigest(rendered),
		ParamsDigest:   ParamsDigest(),
		UncoveredMoves: existing.UncoveredMoves,
	})
}

// A1: self-identity. Rendering twice in one process is byte-identical — Go
// randomises map iteration per iterator, so an unstable corpus would make every
// other assertion uninterpretable.
func TestCorpusSelfIdentity(t *testing.T) {
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

// A2: expectation. Each row's rendered output equals its checked-in golden file.
// Output moved and the version did not -> fail (ADR-0021's gate, first direction).
func TestCorpusExpectation(t *testing.T) {
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

// A5: coverage. Every cell of the block holds at least one row, and no row cites a
// cell outside the enumeration. A missing cell fails the build, naming it.
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
}

// The lock gate: recomputed digests must equal the checked-in lock, and the lock's
// leaf version must equal the code's. Any output or parameter change forces a lock
// edit; binding that edit to a version bump is the CI gate's job.
func TestCorpusLock(t *testing.T) {
	lock, err := LoadLock(".")
	if err != nil {
		t.Fatalf("load lock (run with -update to create it): %v", err)
	}
	if lock.LeafVersion != blanketdiscrim.Version {
		t.Errorf("lock leaf version %q != code version %q: bump one to match, and re-bless with -update",
			lock.LeafVersion, blanketdiscrim.Version)
	}
	rendered, err := RenderAll()
	if err != nil {
		t.Fatal(err)
	}
	if got := CorpusDigest(rendered); got != lock.CorpusDigest {
		t.Errorf("corpus digest moved: a row's expected output changed.\n  locked: %s\n  now:    %s\nBump blanketdiscrim.Version and re-bless with -update (or the change is unintended).",
			lock.CorpusDigest, got)
	}
	if got := ParamsDigest(); got != lock.ParamsDigest {
		t.Errorf("params digest moved: a declared parameter changed.\n  locked: %s\n  now:    %s\nBump blanketdiscrim.Version and re-bless with -update.",
			lock.ParamsDigest, got)
	}
}
