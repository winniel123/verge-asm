package corpus

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	he "github.com/winniel123/verge-asm/internal/measure/httpexchange"
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
	existing, _ := LoadLock(".")
	return WriteLock(".", Lock{
		LeafVersion:    he.Version,
		CorpusDigest:   CorpusDigest(rendered),
		ParamsDigest:   ParamsDigest(),
		UncoveredMoves: existing.UncoveredMoves,
	})
}

func TestCorpusSelfIdentity(t *testing.T) {
	// Go randomises map iteration, so an unstable render would make every other assertion unreadable.
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
	if len(AllCells) != 6 {
		t.Errorf("http-exchange block is 6 cells; AllCells has %d", len(AllCells))
	}
}

func TestRedirectIssuesExactlyOneRequest(t *testing.T) {
	var row Row
	for _, r := range Rows {
		if r.Golden == "http_redirect_not_followed.ndjson" {
			row = r
		}
	}
	// The exchanger is a package value reused across renders, so only the delta is stable.
	before := make(map[string]int, len(row.Step.Exchange.calls))
	for k, n := range row.Step.Exchange.calls {
		before[k] = n
	}
	if _, err := RenderRow(row); err != nil {
		t.Fatal(err)
	}
	// A per-render delta of one is what "not followed" means structurally, not only in bytes.
	for key, n := range row.Step.Exchange.calls {
		if n-before[key] != 1 {
			t.Errorf("redirect followed: %s requested %d times in one render, want exactly 1", key, n-before[key])
		}
	}
}

func TestCorpusLock(t *testing.T) {
	lock, err := LoadLock(".")
	if err != nil {
		t.Fatalf("load lock (run with -update to create it): %v", err)
	}
	if lock.LeafVersion != he.Version {
		t.Errorf("lock leaf version %q != code version %q: bump one to match, and re-bless with -update",
			lock.LeafVersion, he.Version)
	}
	rendered, err := RenderAll()
	if err != nil {
		t.Fatal(err)
	}
	if got := CorpusDigest(rendered); got != lock.CorpusDigest {
		t.Errorf("corpus digest moved: a row's expected output changed.\n  locked: %s\n  now:    %s\nBump httpexchange.Version and re-bless with -update.",
			lock.CorpusDigest, got)
	}
	// A declared-parameter change is the second thing that may justify a version bump.
	if got := ParamsDigest(); got != lock.ParamsDigest {
		t.Errorf("params digest moved: a declared parameter changed.\n  locked: %s\n  now:    %s\nBump httpexchange.Version and re-bless with -update.",
			lock.ParamsDigest, got)
	}
}
