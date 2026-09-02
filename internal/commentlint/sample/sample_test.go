package sample

import (
	"fmt"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/commentlint/rule"
)

const src = `package p

// Alpha does a thing.
func Alpha() {}

// Beta reads ADR-0101 before it decides.
func Beta() {}

// gamma does a thing.
func gamma() {}

func Delta() {
	// x := 1

	// short label here
	_ = 1 // trailing label
}
`

func TestAdmittedKeepsTheScreenedSet(t *testing.T) {
	got, err := Admitted("p.go", []byte(src))
	if err != nil {
		t.Fatalf("Admitted: %v", err)
	}
	want := []struct {
		line  int
		class rule.Class
	}{
		{3, rule.DocstringExportedConventional},
		{9, rule.DocstringUnexported},
		{13, rule.CommentedOutCode},
		{15, rule.ShortLabel},
	}
	if len(got) != len(want) {
		t.Fatalf("admitted %d blocks, want %d: %v", len(got), len(want), spans(got))
	}
	for i, w := range want {
		if got[i].StartLine != w.line || got[i].Class != w.class {
			t.Errorf("block %d = line %d %s, want line %d %s",
				i, got[i].StartLine, got[i].Class, w.line, w.class)
		}
	}
}

func TestAdmittedWithholdsACitedBlock(t *testing.T) {
	got, err := Admitted("p.go", []byte(src))
	if err != nil {
		t.Fatalf("Admitted: %v", err)
	}
	for _, i := range got {
		if i.StartLine == 6 {
			t.Fatalf("the ADR-0101 block reached the delete pass: %+v", i)
		}
	}
}

func TestAdmittedNeverReachesATrailingComment(t *testing.T) {
	got, err := Admitted("p.go", []byte(src))
	if err != nil {
		t.Fatalf("Admitted: %v", err)
	}
	for _, i := range got {
		if i.StartLine == 16 {
			t.Fatalf("a trailing comment reached the delete pass: %+v", i)
		}
	}
}

func TestExcerptCarriesTheDeclarationBelow(t *testing.T) {
	got, err := Admitted("p.go", []byte(src))
	if err != nil {
		t.Fatalf("Admitted: %v", err)
	}
	if got[0].ExcerptStart != 1 {
		t.Errorf("ExcerptStart = %d, want 1", got[0].ExcerptStart)
	}
	if !strings.Contains(got[0].Excerpt, "func Alpha() {}") {
		t.Errorf("excerpt misses the declaration:\n%s", got[0].Excerpt)
	}
}

func TestPopulationOf(t *testing.T) {
	cases := map[string]Population{
		"internal/a/b.go":            Production,
		"internal/a/b_test.go":       Test,
		`internal\a\b_test.go`:       Test,
		"internal/a/testing_util.go": Production,
	}
	for name, want := range cases {
		if got := PopulationOf(name); got != want {
			t.Errorf("PopulationOf(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestDrawIsDeterministicAndSized(t *testing.T) {
	items := fakeItems(500)
	seed := Seed(1, Production)
	first := Draw(items, Size, seed)
	if len(first) != Size {
		t.Fatalf("drew %d, want %d", len(first), Size)
	}
	second := Draw(items, Size, seed)
	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Error("two draws on one seed differ")
	}
	if fmt.Sprint(first) == fmt.Sprint(Draw(items, Size, Seed(2, Production))) {
		t.Error("round 2 re-drew round 1's sample")
	}
}

func TestDrawReturnsEveryItemWhenThePopulationIsSmall(t *testing.T) {
	items := fakeItems(7)
	if got := Draw(items, Size, Seed(1, Test)); len(got) != 7 {
		t.Fatalf("drew %d, want 7", len(got))
	}
}

func TestDrawReadsInSourceOrder(t *testing.T) {
	got := Draw(fakeItems(200), Size, Seed(1, Production))
	for i := 1; i < len(got); i++ {
		if !less(got[i-1], got[i]) {
			t.Fatalf("item %d sorts before item %d", i, i-1)
		}
	}
}

func TestSupplementCoversAMissedClassOnly(t *testing.T) {
	items := fakeItems(300)
	rare := Item{File: "z.go", StartLine: 1, EndLine: 1, Class: rule.CommentedOutCode}
	items = append(items, rare)
	drawn := Draw(items, Size, Seed(1, Production))
	extra := Supplement(items, drawn, SupplementCap, Seed(1, Production))

	if Counts(drawn)[rule.CommentedOutCode] != 0 {
		t.Skip("the draw already reached the rare class")
	}
	if len(extra) != 1 || extra[0].File != "z.go" {
		t.Fatalf("supplement = %v, want the one commented-out-code block", spans(extra))
	}
}

func TestSupplementIsEmptyWhenTheDrawCoversEveryClass(t *testing.T) {
	items := fakeItems(120)
	drawn := Draw(items, Size, Seed(1, Production))
	if got := Supplement(items, drawn, SupplementCap, Seed(1, Production)); len(got) != 0 {
		t.Fatalf("supplement = %v, want none", spans(got))
	}
}

func TestSupplementCapsEachClass(t *testing.T) {
	var items []Item
	for n := 1; n <= 40; n++ {
		items = append(items, Item{
			File: "a.go", StartLine: n, EndLine: n, Class: rule.SectionDivider,
		})
	}
	got := Supplement(items, nil, SupplementCap, Seed(1, Production))
	if len(got) != SupplementCap {
		t.Fatalf("supplement = %d blocks, want %d", len(got), SupplementCap)
	}
}

func TestRenderCarriesTheGateFacts(t *testing.T) {
	items, err := Admitted("p.go", []byte(src))
	if err != nil {
		t.Fatalf("Admitted: %v", err)
	}
	var b strings.Builder
	sheet := Sheet{Population: Test, Round: 2, Files: 1, Admitted: items, Drawn: items}
	if err := Render(&b, sheet); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		"test Go, round 2",
		"--population test --round 2",
		"Blocks the §3.2 screen admits for deletion: 4",
		"| `docstring-exported-conventional` | 1 | 1 | 0 |",
		"| `section-divider` | 0 | 0 | 0 |",
		"| `section-divider` | 0 | n/a | n/a |",
		"### 1. `p.go:3` — `docstring-exported-conventional`",
		"Load-bearing: [ ]",
		"3 | // Alpha does a thing.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sheet misses %q:\n%s", want, out)
		}
	}
}

func fakeItems(n int) []Item {
	classes := []rule.Class{
		rule.DocstringExportedConventional,
		rule.DocstringUnexported,
		rule.ShortLabel,
		rule.SectionDivider,
	}
	out := make([]Item, 0, n)
	for i := range n {
		out = append(out, Item{
			File:      fmt.Sprintf("pkg%02d/a.go", i%20),
			StartLine: i + 1,
			EndLine:   i + 1,
			Class:     classes[i%len(classes)],
		})
	}
	return out
}

func spans(items []Item) []string {
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, i.Span())
	}
	return out
}
