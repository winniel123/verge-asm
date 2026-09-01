package corpus

import (
	"bytes"
	"flag"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
)

// update regenerates the golden files and the lock. It is the deliberate bless
// action: run `go test ./internal/custody/... -run Corpus -update` after an
// intended output or parameter change, having bumped custody.Version, and commit
// the result.
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
	// The register is append-only and maintained by hand, so a lock that exists
	// and fails to decode must STOP the bless rather than yield a zero Lock. A
	// JSON typo added while appending an uncovered move would otherwise be
	// blessed away as `"uncovered_moves": null`, erasing the justification a
	// later bump depends on (golden-corpus.md §9.2).
	existing, err := LoadLock(".")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("corpus: refusing to re-bless over an unreadable lock: %w", err)
	}
	return WriteLock(".", Lock{
		LeafVersion:    custody.Version,
		CorpusDigest:   CorpusDigest(rendered),
		ParamsDigest:   ParamsDigest(),
		UncoveredMoves: existing.UncoveredMoves,
	})
}

// A1: self-identity. Rendering twice in one process is byte-identical — Go
// randomises map iteration per iterator, and this corpus builds two maps per row
// (the Shared verdicts and the candidate set), so an unstable corpus would make
// every other assertion uninterpretable.
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
			t.Errorf("cell(s) %v: output moved without a blessed golden update.\n  claim: %s\n--- want (%s) ---\n%s\n--- got ---\n%s",
				r.Cells, r.Claim, r.Golden, want, rendered[r.Golden])
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

// TestRowsAreWellFormed closes the two ways a row can cite a cell while pinning
// nothing. Both survive A2 and A5, so neither is caught anywhere else, and both
// are the silent move this block exists to catch.
func TestRowsAreWellFormed(t *testing.T) {
	// One golden per row. RenderAll keys by filename, so two rows sharing one
	// would collapse to a single render: A5 would still count the second row's
	// cells as covered, A2 would compare both rows against the one surviving
	// file, and the digest would hash one entry fewer.
	byGolden := make(map[string]string, len(Rows))
	for _, r := range Rows {
		if r.Golden == "" {
			t.Errorf("row citing %v declares no golden filename", r.Cells)
			continue
		}
		if first, dup := byGolden[r.Golden]; dup {
			t.Errorf("golden %s is claimed by two rows (%v and %v): the second render replaces the first and its cells are pinned by nothing",
				r.Golden, first, r.Cells)
			continue
		}
		byGolden[r.Golden] = fmt.Sprint(r.Cells)
	}

	// Every observed address is an address the row renders or resolves. A
	// mistyped literal in Observed parses fine, attaches to no rendered address,
	// and silently degrades a boundary row into a measurement-pending one — a row
	// citing C1 that pins C3's claim, blessed on the next -update.
	for _, r := range Rows {
		known := make(map[netip.Addr]struct{}, len(r.Step.Under)+len(r.Step.Resolutions))
		for _, s := range r.Step.Under {
			known[netipMust(t, s)] = struct{}{}
		}
		for _, res := range r.Step.Resolutions {
			known[netipMust(t, res.Address)] = struct{}{}
		}
		for spelling := range r.Step.Observed {
			if _, ok := known[netipMust(t, spelling)]; !ok {
				t.Errorf("row %s observes %s, which it neither renders nor resolves: the measurement attaches to no address and the row reads as measurement-pending",
					r.Golden, spelling)
			}
		}
		// An address under test that no resolution cites renders a line the
		// estate has no basis for, so the row would claim a reach nothing
		// resolved to.
		for _, s := range r.Step.Under {
			addr := netipMust(t, s)
			cited := false
			for _, res := range r.Step.Resolutions {
				if netipMust(t, res.Address) == addr {
					cited = true
				}
			}
			if !cited {
				t.Errorf("row %s renders %s, which no resolution in its estate cites", r.Golden, s)
			}
		}
	}
}

// The lock gate: recomputed digests must equal the checked-in lock, and the lock's
// derivation version must equal the code's. Any output or parameter change forces
// a lock edit; binding that edit to a version bump is the CI gate's job.
func TestCorpusLock(t *testing.T) {
	lock, err := LoadLock(".")
	if err != nil {
		t.Fatalf("load lock (run with -update to create it): %v", err)
	}
	if lock.LeafVersion != custody.Version {
		t.Errorf("lock derivation version %q != code version %q: bump one to match, and re-bless with -update",
			lock.LeafVersion, custody.Version)
	}
	rendered, err := RenderAll()
	if err != nil {
		t.Fatal(err)
	}
	if got := CorpusDigest(rendered); got != lock.CorpusDigest {
		t.Errorf("corpus digest moved: a row's expected output changed.\n  locked: %s\n  now:    %s\nBump custody.Version and re-bless with -update (or the change is unintended).",
			lock.CorpusDigest, got)
	}
	if got := ParamsDigest(); got != lock.ParamsDigest {
		t.Errorf("params digest moved: a declared parameter changed.\n  locked: %s\n  now:    %s\nBump custody.Version and re-bless with -update.",
			lock.ParamsDigest, got)
	}
}

// TestFixtureStraddlesTheThreshold pins the two boundary fixtures against the
// SHIPPED constant. It is the first thing to go red on a threshold move, and its
// message is what tells the session what it owes.
//
// It also guards the fixture itself. The SAN sets reduce through the Public
// Suffix List, so a PSL revision that ever listed `invalid` would silently
// collapse both counts to zero and quietly turn both rows into `not-shared`. That
// failure arrives here, named, rather than as an unexplained digest move.
func TestFixtureStraddlesTheThreshold(t *testing.T) {
	if atThreshold != custody.SharedEdgeThreshold {
		t.Errorf("the corpus boundary is authored at %d and custody.SharedEdgeThreshold is %d.\n"+
			"The threshold moved. That is a Break (ADR-0129 §3): bump custody.Version, move\n"+
			"belowThreshold/atThreshold to straddle the new value, and re-bless with -update.",
			atThreshold, custody.SharedEdgeThreshold)
	}
	if belowThreshold != atThreshold-1 {
		t.Errorf("the boundary pair is not adjacent: below=%d at=%d. ADR-0085 wants one row on each side of the boundary",
			belowThreshold, atThreshold)
	}

	below, at := SANsBelowThreshold(), SANsAtThreshold()
	if got := custody.FanOut(below); got != belowThreshold {
		t.Errorf("the below-threshold fixture reduces to %d registrable domains, not %d.\n"+
			"Either the reduction moved or the PSL now lists `invalid`; the boundary rows measure nothing until this is repaired.", got, belowThreshold)
	}
	if got := custody.FanOut(at); got != atThreshold {
		t.Errorf("the at-threshold fixture reduces to %d registrable domains, not %d.\n"+
			"Either the reduction moved or the PSL now lists `invalid`; the boundary rows measure nothing until this is repaired.", got, atThreshold)
	}
	if custody.SharedEdge(below) {
		t.Errorf("%d distinct registrable domains reads as shared; the boundary collapsed toward the veto", belowThreshold)
	}
	if !custody.SharedEdge(at) {
		t.Errorf("%d distinct registrable domains does not read as shared; the comparison is at-least, not greater-than", atThreshold)
	}
}

// TestThresholdMoveFailsTheGate is #986's own acceptance criterion, discharged as
// a test rather than as prose: moving the threshold without a version bump fails
// the gate, and here is the proof of each way it fails.
//
// The threshold is a `const`, so the move cannot be performed at run time. What
// the test proves instead is that the two gate inputs a move would touch BOTH
// already differ from the checked-in lock at the moved value, which is exactly
// what TestCorpusLock and TestCorpusExpectation compare.
func TestThresholdMoveFailsTheGate(t *testing.T) {
	lock, err := LoadLock(".")
	if err != nil {
		t.Fatalf("load lock: %v", err)
	}

	// Limb 1 — the declared parameter set. A moved threshold renders a different
	// Params, so the params digest no longer matches the lock and TestCorpusLock
	// fails. This limb catches a move even where no row's output crossed the
	// boundary.
	moved := custody.DefaultParams()
	moved.SharedEdgeThreshold--
	if moved.Digest() == lock.ParamsDigest {
		t.Errorf("a threshold of %d digests the same as the locked %d: the params digest does not carry the threshold, so a move would pass the lock gate",
			moved.SharedEdgeThreshold, custody.SharedEdgeThreshold)
	}

	// Limb 2 — the Public Suffix List revision. ADR-0129's #954 amendment makes a
	// list update the same kind of change as a threshold move, so the same digest
	// carries it.
	relisted := custody.DefaultParams()
	relisted.PublicSuffixList += " (a later revision)"
	if relisted.Digest() == lock.ParamsDigest {
		t.Error("a different Public Suffix List revision digests the same as the locked one: a list update would pass the lock gate")
	}

	// Limb 3 — the rows' own output. The below-threshold fixture sits exactly one
	// domain under the constant, so a move of one flips its verdict, its rendered
	// line and its golden. This is what makes the boundary POSITION pinned rather
	// than merely its shape.
	below := SANsBelowThreshold()
	if got := custody.FanOut(below); got != custody.SharedEdgeThreshold-1 {
		t.Fatalf("the below-threshold row measures %d and the threshold is %d: the row no longer sits one domain under the boundary, so a move of one would leave its golden untouched",
			got, custody.SharedEdgeThreshold)
	}
	if custody.SharedEdge(below) {
		t.Fatal("the below-threshold row already reads as shared; it pins nothing")
	}
	// The count equals the moved threshold, and the comparison is at-least, so at
	// a threshold one lower this same set reads shared. The row's rendered
	// `shared_edge`, `custody` and `may_probe_internet` all move with it, and
	// TestCorpusExpectation fails on the golden.
}

// netipMust parses an address literal for a test, failing the test rather than
// panicking, so a bad literal reads as this test's failure and not as a crash
// inside the corpus package.
func netipMust(t *testing.T, s string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return addr.Unmap()
}

// TestSeedLimbSurvivesAGlobalVeto states, as a test beside the golden, the guard
// ADR-0129's #956 amendment asks the corpus to leave behind. A session that
// "repairs" the apparent inconsistency by making the veto global would move the
// Seed-covered address off `operator`, and this names what broke before A2's byte
// diff has to be read.
func TestSeedLimbSurvivesAGlobalVeto(t *testing.T) {
	var row Row
	for _, r := range Rows {
		if r.Golden == "seed_covered_and_veto.ndjson" {
			row = r
		}
	}
	if row.Golden == "" {
		t.Fatal("the disjointness row is gone; ADR-0129's strongest guard went with it")
	}
	estate := row.Step.Estate()
	declared := netipMust(t, "104.16.132.10")
	if got := estate.Derive(declared); got != custody.Operator {
		t.Errorf("a Seed-covered address measured shared derives %q, not %q.\n"+
			"The veto reached the address-scope limb. A measurement may narrow a Derived reach; it may never overrule a Declared act (ADR-0129's #956 amendment).",
			got, custody.Operator)
	}
	if !estate.MayProbe(declared, custody.ClassInternet) {
		t.Error("a Seed-covered address measured shared is not probed: the veto reached the declaration")
	}
	undeclared := netipMust(t, "23.20.0.10")
	if got := estate.Derive(undeclared); got != custody.ThirdParty {
		t.Errorf("the undeclared sibling of the same measurement derives %q, not %q: the veto stopped working", got, custody.ThirdParty)
	}
}
