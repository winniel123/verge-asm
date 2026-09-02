package sample

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/winniel123/verge-asm/internal/commentlint/rule"
	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

const Size = 100

const SupplementCap = 10

const ContextLines = 3

type Population string

const (
	Production Population = "production"
	Test       Population = "test"
)

func PopulationOf(name string) Population {
	if strings.HasSuffix(strings.ReplaceAll(name, `\`, "/"), "_test.go") {
		return Test
	}
	return Production
}

type Item struct {
	File         string
	StartLine    int
	EndLine      int
	Class        rule.Class
	Excerpt      string
	ExcerptStart int
}

func (i Item) Span() string {
	if i.StartLine == i.EndLine {
		return fmt.Sprintf("%s:%d", i.File, i.StartLine)
	}
	return fmt.Sprintf("%s:%d-%d", i.File, i.StartLine, i.EndLine)
}

// Admitted reads surface.Result.Blocks alone, because §3.4 holds every
// trailing comment back from the delete pass.
func Admitted(name string, src []byte) ([]Item, error) {
	res, err := surface.Go{}.Lex(src)
	if err != nil {
		return nil, err
	}
	lines := sourceLines(src)
	var out []Item
	for _, b := range res.Blocks {
		class, _, deletable := rule.Deletable(b)
		if !deletable {
			continue
		}
		start, text := excerpt(lines, b.StartLine, b.EndLine)
		out = append(out, Item{
			File:         name,
			StartLine:    b.StartLine,
			EndLine:      b.EndLine,
			Class:        class,
			Excerpt:      text,
			ExcerptStart: start,
		})
	}
	return out, nil
}

func Seed(round int, p Population) string {
	return fmt.Sprintf("comment-policy-3.9/round-%d/%s", round, p)
}

// Draw orders the population by a seeded digest and keeps the first size
// items. A pseudo-random generator draws the same sample, and a digest
// re-derives it from the seed alone on any machine.
func Draw(items []Item, size int, seed string) []Item {
	ranked := rank(items, seed)
	if size < len(ranked) {
		ranked = ranked[:size]
	}
	return inReadingOrder(ranked)
}

// Supplement covers a delete-set class the draw missed outright, because a
// class holding a few blocks tree-wide loses its §3.9 verdict otherwise. A
// reviewer adjudicates it apart, so it never dilutes the 100-block gate.
func Supplement(items, drawn []Item, limit int, seed string) []Item {
	seen := map[rule.Class]bool{}
	for _, i := range drawn {
		seen[i.Class] = true
	}
	byClass := map[rule.Class][]Item{}
	for _, i := range items {
		if !seen[i.Class] {
			byClass[i.Class] = append(byClass[i.Class], i)
		}
	}
	var out []Item
	for _, c := range sortedClasses(byClass) {
		ranked := rank(byClass[c], seed)
		if limit < len(ranked) {
			ranked = ranked[:limit]
		}
		out = append(out, ranked...)
	}
	return inReadingOrder(out)
}

func Counts(items []Item) map[rule.Class]int {
	out := map[rule.Class]int{}
	for _, i := range items {
		out[i.Class]++
	}
	return out
}

type keyed struct {
	Item
	key [sha256.Size]byte
}

func rank(items []Item, seed string) []Item {
	keys := make([]keyed, 0, len(items))
	for _, i := range items {
		keys = append(keys, keyed{
			Item: i,
			key:  sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", seed, i.File, i.StartLine))),
		})
	}
	sort.Slice(keys, func(a, b int) bool {
		if c := bytes.Compare(keys[a].key[:], keys[b].key[:]); c != 0 {
			return c < 0
		}
		return less(keys[a].Item, keys[b].Item)
	})
	out := make([]Item, 0, len(keys))
	for _, k := range keys {
		out = append(out, k.Item)
	}
	return out
}

func inReadingOrder(items []Item) []Item {
	out := append([]Item(nil), items...)
	sort.Slice(out, func(a, b int) bool { return less(out[a], out[b]) })
	return out
}

func less(a, b Item) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	return a.StartLine < b.StartLine
}

func sortedClasses[T any](m map[rule.Class]T) []rule.Class {
	out := make([]rule.Class, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

func sourceLines(src []byte) []string {
	raw := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	for i, line := range raw {
		raw[i] = strings.TrimSuffix(line, "\r")
	}
	return raw
}

func excerpt(lines []string, startLine, endLine int) (int, string) {
	from := max(startLine-ContextLines, 1)
	to := min(endLine+ContextLines, len(lines))
	return from, strings.Join(lines[from-1:to], "\n")
}

type Sheet struct {
	Population Population
	Round      int
	Files      int
	Admitted   []Item
	Drawn      []Item
	Extra      []Item
}

func Render(w io.Writer, s Sheet) error {
	b := &strings.Builder{}
	fmt.Fprintf(b, "# Comment policy validation gate — %s Go, round %d\n\n", s.Population, s.Round)
	fmt.Fprint(b, "SPEC `docs/spec/comment-policy.md` §3.9. Regenerate this sheet with:\n\n")
	fmt.Fprintf(b, "```sh\ngo run ./cmd/commentlint sample --population %s --round %d\n```\n\n",
		s.Population, s.Round)
	fmt.Fprintf(b, "- In-scope Go files read: %d\n", s.Files)
	fmt.Fprintf(b, "- Blocks the §3.2 screen admits for deletion: %d\n", len(s.Admitted))
	fmt.Fprintf(b, "- Blocks drawn into the gate sample: %d\n", len(s.Drawn))
	fmt.Fprintf(b, "- Blocks drawn into the coverage supplement: %d\n\n", len(s.Extra))
	fmt.Fprint(b, "Accept a class at 2 or fewer load-bearing blocks. A class that fails three "+
		"rounds leaves the v1 delete set and stays in the flag set.\n\n")

	writeCounts(b, s)
	writeVerdicts(b, s)
	writeBlocks(b, "Gate sample", s.Drawn, 1)
	if len(s.Extra) > 0 {
		fmt.Fprint(b, "The draw reached none of the classes below. A reviewer adjudicates each "+
			"one apart from the 100-block gate.\n\n")
		writeBlocks(b, "Coverage supplement", s.Extra, len(s.Drawn)+1)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeCounts(b *strings.Builder, s Sheet) {
	admitted := Counts(s.Admitted)
	drawn := Counts(s.Drawn)
	extra := Counts(s.Extra)
	fmt.Fprint(b, "## Class coverage\n\n")
	fmt.Fprint(b, "| Class | Admitted | Gate sample | Supplement |\n| --- | --- | --- | --- |\n")
	for _, c := range rule.DeleteSet() {
		fmt.Fprintf(b, "| `%s` | %d | %d | %d |\n", c, admitted[c], drawn[c], extra[c])
	}
	fmt.Fprint(b, "\n")
}

func writeVerdicts(b *strings.Builder, s Sheet) {
	admitted := Counts(s.Admitted)
	drawn := Counts(s.Drawn)
	extra := Counts(s.Extra)
	fmt.Fprint(b, "## Verdicts\n\n")
	fmt.Fprint(b, "A reviewer fills the last two columns. A class that admits no block on this "+
		"population reads `n/a`.\n\n")
	fmt.Fprint(b, "| Class | Read | Load-bearing | Verdict |\n| --- | --- | --- | --- |\n")
	for _, c := range rule.DeleteSet() {
		if admitted[c] == 0 {
			fmt.Fprintf(b, "| `%s` | 0 | n/a | n/a |\n", c)
			continue
		}
		fmt.Fprintf(b, "| `%s` | %d | | |\n", c, drawn[c]+extra[c])
	}
	fmt.Fprint(b, "\n")
}

func writeBlocks(b *strings.Builder, heading string, items []Item, first int) {
	fmt.Fprintf(b, "## %s\n\n", heading)
	for n, i := range items {
		fmt.Fprintf(b, "### %d. `%s` — `%s`\n\n", first+n, i.Span(), i.Class)
		fmt.Fprint(b, "Load-bearing: [ ]\n\n")
		fmt.Fprintf(b, "```go\n%s\n```\n\n", numbered(i))
	}
}

func numbered(i Item) string {
	lines := strings.Split(i.Excerpt, "\n")
	width := len(fmt.Sprint(i.ExcerptStart + len(lines) - 1))
	out := make([]string, 0, len(lines))
	for n, line := range lines {
		out = append(out, fmt.Sprintf("%*d | %s", width, i.ExcerptStart+n, line))
	}
	return strings.Join(out, "\n")
}
