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

// The bless action: bump custody.Version first, then re-run with -update and commit the result.

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
	// A typo would otherwise bless the register away as null (golden-corpus.md §9.2).
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

func TestCorpusSelfIdentity(t *testing.T) {
	// Go randomises map iteration and a row builds two maps, so drift here voids every other test.
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
	// ADR-0021's gate, first direction: output moved and the version did not.
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

func TestRowsAreWellFormed(t *testing.T) {
	// A2 and A5 both pass on a row that pins nothing, so these two shapes are caught nowhere else.
	// RenderAll keys by filename, so two rows sharing a golden collapse and A5 still counts both.
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

	// A mistyped literal parses fine and silently degrades a boundary row into a pending one.
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

func TestCorpusLock(t *testing.T) {
	// Binding a lock edit to a version bump is CI's job, not this test's.
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

func TestFixtureStraddlesTheThreshold(t *testing.T) {
	// A PSL revision listing invalid would collapse both counts, and that failure arrives here named.
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

func TestThresholdMoveFailsTheGate(t *testing.T) {
	// #986's acceptance criterion, discharged as a test rather than as prose.
	// The threshold is a const, so a move cannot be performed at run time and is proved indirectly.
	lock, err := LoadLock(".")
	if err != nil {
		t.Fatalf("load lock: %v", err)
	}

	// This limb catches a move even where no row's output crossed the boundary.
	moved := custody.DefaultParams()
	moved.SharedEdgeThreshold--
	if moved.Digest() == lock.ParamsDigest {
		t.Errorf("a threshold of %d digests the same as the locked %d: the params digest does not carry the threshold, so a move would pass the lock gate",
			moved.SharedEdgeThreshold, custody.SharedEdgeThreshold)
	}

	// ADR-0129's #954 amendment makes a list update the same kind of change as a threshold move.
	relisted := custody.DefaultParams()
	relisted.PublicSuffixList += " (a later revision)"
	if relisted.Digest() == lock.ParamsDigest {
		t.Error("a different Public Suffix List revision digests the same as the locked one: a list update would pass the lock gate")
	}

	below := SANsBelowThreshold()
	if got := custody.FanOut(below); got != custody.SharedEdgeThreshold-1 {
		t.Fatalf("the below-threshold row measures %d and the threshold is %d: the row no longer sits one domain under the boundary, so a move of one would leave its golden untouched",
			got, custody.SharedEdgeThreshold)
	}
	if custody.SharedEdge(below) {
		t.Fatal("the below-threshold row already reads as shared; it pins nothing")
	}
	// One threshold lower this same set reads shared, so TestCorpusExpectation fails on its golden.
}

func TestTheExclusionCutsTheSeedLimbAlone(t *testing.T) {
	estate := rowByGolden(t, "excluded_but_extension_reached.ndjson").Step.Estate()

	reached := netipMust(t, "104.16.140.20")
	if got := estate.Derive(reached); got != custody.Operator {
		t.Errorf("an excluded address a custody extension ALSO reaches derives %q, not %q.\n"+
			"The exclusion cut the extension limb. It cuts the `Seed` limb alone: the set an exclusion\n"+
			"removes is never larger than the set the declaration added (ADR-0133 §1).",
			got, custody.Operator)
	}
	if !estate.MayProbe(reached, custody.ClassInternet) {
		t.Error("an excluded address the custody extension reaches is not probed: the exclusion reached the extension limb")
	}
	if estate.CoversAddressScope(reached) {
		t.Error("an excluded address still reads as address-scope covered: the exclusion did not narrow the Seed limb at all")
	}

	sibling := netipMust(t, "104.16.140.21")
	if got := estate.Derive(sibling); got != custody.ThirdParty {
		t.Errorf("the excluded sibling no extension reaches derives %q, not %q: the exclusion stopped working", got, custody.ThirdParty)
	}
	if estate.MayProbe(sibling, custody.ClassInternet) {
		t.Error("the excluded sibling is still probed: the gate does not read the narrowed limb")
	}
}

func TestRemovingTheExclusionRestoresOperator(t *testing.T) {
	addr := netipMust(t, "93.184.217.5")

	excluded := rowByGolden(t, "excluded_inside_a_scope.ndjson").Step.Estate()
	if got := excluded.Derive(addr); got != custody.ThirdParty {
		t.Errorf("an address inside a declared scope and inside an exclusion derives %q, not %q", got, custody.ThirdParty)
	}
	if excluded.MayProbe(addr, custody.ClassInternet) {
		t.Error("an excluded address is still probed from an internet-class Vantage")
	}

	restored := rowByGolden(t, "exclusion_removed.ndjson").Step.Estate()
	if got := restored.Derive(addr); got != custody.Operator {
		t.Errorf("the same address with the exclusion ABSENT derives %q, not %q.\n"+
			"The pair no longer straddles the exclusion, so the refused row above pins the fixture rather than the exclusion.",
			got, custody.Operator)
	}
	if !restored.MayProbe(addr, custody.ClassInternet) {
		t.Error("the same address with the exclusion absent is not probed: the pair pins nothing")
	}
}

func TestExcludedAddressLeavesTheFanOutPopulation(t *testing.T) {
	// ADR-0133 §3's cost half: an excluded address leaves the declaration-limb walk.
	estate := rowByGolden(t, "excluded_but_extension_reached.ndjson").Step.Estate()
	var got []netip.Addr
	for a := range estate.EdgeFanoutPopulation() {
		got = append(got, a)
	}
	want := []netip.Addr{netipMust(t, "104.16.140.20")}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("the fan-out population is %v, want %v: an excluded address is out of the DECLARATION limb, and an extension candidate stays in", got, want)
	}
}

func rowByGolden(t *testing.T, golden string) Row {
	t.Helper()
	// A zero Estate would quietly pass, so a deleted row must fail by name here.
	for _, r := range Rows {
		if r.Golden == golden {
			return r
		}
	}
	t.Fatalf("the row for golden %s is gone, and the guard it carries went with it", golden)
	return Row{}
}

func netipMust(t *testing.T, s string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return addr.Unmap()
}

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
