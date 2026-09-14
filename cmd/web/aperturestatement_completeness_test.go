package main

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
)

const (
	v1SpecPath       = "../../docs/spec/v1-spec.md"
	apertureSpecPath = "../../docs/spec/aperture-statement.md"
	apiGuidePath     = "../../docs/guides/api.md"
)

// A typed copy of the seven names is the defect a typed `38` is, so each list is read (SPEC §6.1).

func apertureInputNames(rows []apertureRowView) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Input)
	}
	return out
}

func readDoc(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func cleanInputName(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "*`"))
}

var (
	v1ApertureSentenceRe = regexp.MustCompile(`\*\*([A-Za-z]+) aperture inputs\*\*[^:]*:([^.]*)\.`)
	parentheticalRe      = regexp.MustCompile(`\s*\([^)]*\)`)
)

// The list is prose across four lines, so the anchor is its own sentence and never a line number.

func v1SpecApertureInputs(t *testing.T) (count string, inputs []string) {
	t.Helper()
	text := strings.Join(strings.Fields(readDoc(t, v1SpecPath)), " ")
	m := v1ApertureSentenceRe.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("%s states no aperture-input sentence, so the ledger has nothing to compare against", v1SpecPath)
	}
	for _, part := range strings.Split(parentheticalRe.ReplaceAllString(m[2], ""), ",") {
		name := cleanInputName(strings.TrimPrefix(strings.TrimSpace(part), "and "))
		if name != "" {
			inputs = append(inputs, name)
		}
	}
	return m[1], inputs
}

// §2.5's table is the ledger's own order, so the gate reads it rather than restating it.

func apertureSpecTableInputs(t *testing.T) []string {
	t.Helper()
	lines := strings.Split(readDoc(t, apertureSpecPath), "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "### 2.5 ") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s holds no §2.5, so the ledger has no stated order", apertureSpecPath)
	}

	var out []string
	seen := false
	for _, line := range lines[start:] {
		if !strings.HasPrefix(line, "|") {
			if seen {
				break
			}
			continue
		}
		seen = true
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) < 2 {
			continue
		}
		if strings.TrimSpace(cells[0]) == "#" || strings.Contains(cells[0], "---") {
			continue
		}
		out = append(out, cleanInputName(cells[1]))
	}
	return out
}

func apiGuideDocumentedInputs(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, m := range regexp.MustCompile(`"input":\s*"([^"]*)"`).FindAllStringSubmatch(readDoc(t, apiGuidePath), -1) {
		out = append(out, m[1])
	}
	return out
}

func assertSameInputs(t *testing.T, wantLabel string, want []string, gotLabel string, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s states %d inputs and %s carries %d: %v against %v", wantLabel, len(want), gotLabel, len(got), want, got)
		return
	}
	for i := range want {
		if !strings.EqualFold(want[i], got[i]) {
			t.Errorf("input %d: %s states %q and %s carries %q", i+1, wantLabel, want[i], gotLabel, got[i])
		}
	}
}

func numberWord(t *testing.T, n int) string {
	t.Helper()
	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	if n < 0 || n >= len(words) {
		t.Fatalf("the ledger renders %d rows, which no spec sentence names", n)
	}
	return words[n]
}

func nameOnlyStatement(t *testing.T) []apertureRowView {
	t.Helper()
	return apertureStatement(nil, true, nameOnlySeeds(), testDNSCadenceSeconds, true, nil, true)
}

// SPEC docs/spec/aperture-statement.md §8 criterion 1 — the ledger states what both specs state.

func TestTheLedgerRendersTheSevenInputsTheSpecsState(t *testing.T) {
	got := apertureInputNames(nameOnlyStatement(t))

	word, v1 := v1SpecApertureInputs(t)
	assertSameInputs(t, v1SpecPath, v1, "the ledger", got)
	assertSameInputs(t, apertureSpecPath+" §2.5", apertureSpecTableInputs(t), "the ledger", got)

	if want := numberWord(t, len(got)); !strings.EqualFold(word, want) {
		t.Errorf("%s names %s aperture inputs and the ledger renders %s rows", v1SpecPath, word, want)
	}
}

// The address scope is a lever on the custody gate and never a peer of it (ADR-0079, #1906).

func TestTheLedgerRendersNoEighthRowOnAnyConfiguration(t *testing.T) {
	_, v1 := v1SpecApertureInputs(t)
	for _, tc := range apertureConfigurations(t) {
		if len(tc.rows) != len(v1) {
			t.Errorf("%s: the ledger renders %d rows, and the spec states %d", tc.name, len(tc.rows), len(v1))
			continue
		}
		assertSameInputs(t, v1SpecPath, v1, tc.name, apertureInputNames(tc.rows))
	}
}

type apertureConfig struct {
	name string
	rows []apertureRowView
}

func apertureConfigurations(t *testing.T) []apertureConfig {
	t.Helper()
	bothLegs := []custody.VantageClass{custody.ClassInternet, custody.ClassInternal}
	extended := []db.ListSeedsRow{{Kind: "name", CustodyExtension: true}}
	return []apertureConfig{
		{"a fresh install", apertureStatement(nil, true, nil, testDNSCadenceSeconds, true, nil, true)},
		{"a name-only estate", nameOnlyStatement(t)},
		{"an estate with one address scope", apertureStatement(nil, true, addressScopeSeeds(t), testDNSCadenceSeconds, true, bothLegs, true)},
		{"an extended name scope", apertureStatement(nil, true, extended, testDNSCadenceSeconds, true, bothLegs, true)},
		{"every read withheld", apertureStatement(nil, false, nameOnlySeeds(), 0, false, nil, false)},
	}
}

// SPEC docs/spec/aperture-statement.md §2.3 — an empty cell renders `none` and its reason.

func TestNoLedgerCellRendersBlankOnAnyConfiguration(t *testing.T) {
	for _, tc := range apertureConfigurations(t) {
		for _, row := range tc.rows {
			assertNoBlankCell(t, tc.name+": "+row.Input, row)
			for i, fig := range row.Figures {
				if strings.TrimSpace(fig.Text) == "" {
					t.Errorf("%s: %s: figure %d renders blank, and SPEC §2.3 bars that", tc.name, row.Input, i+1)
				}
			}
		}
	}
}

// Two rows fold over an empty Seed list, which is a declared state and not a pending read (§2.3).

func TestAFreshInstallStatesNoneRatherThanABlankCell(t *testing.T) {
	rows := apertureStatement(nil, true, nil, testDNSCadenceSeconds, true, nil, true)
	for _, input := range []string{vantageClassInput, controlProbeInput} {
		row := statementRow(t, rows, input)
		if row.State != apertureNone {
			t.Errorf("%s: a fresh install reads %q, and SPEC §2.3 asks for %q", input, row.State, apertureNone)
		}
		if strings.TrimSpace(row.StateDetail) == "" {
			t.Errorf("%s: %q carries no reason, and SPEC §2.3 asks for one", input, apertureNone)
		}
	}
}

// A withheld read lands in the cell it reaches, so the reader never meets a value we lack (#989).

func TestAWithheldReadNamesItselfInTheCellItReaches(t *testing.T) {
	rows := apertureStatement(nil, false, nameOnlySeeds(), 0, false, nil, false)
	const notRead = "not read"

	for _, input := range []string{enabledSourcesInput, vantageClassInput} {
		if row := statementRow(t, rows, input); row.State != notRead {
			t.Errorf("%s: a withheld read reads %q in the State cell, want %q", input, row.State, notRead)
		}
	}
	for _, input := range []string{queriedQtypeInput, controlProbeInput} {
		if row := statementRow(t, rows, input); row.Cadence != notRead {
			t.Errorf("%s: a withheld read reads %q in the Cadence cell, want %q", input, row.Cadence, notRead)
		}
	}
}

var (
	ledgerRowRe  = regexp.MustCompile(`(?s)<tr>(.*?)</tr>`)
	ledgerCellRe = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	htmlTagRe    = regexp.MustCompile(`<[^>]*>`)
)

func renderedLedgerBody(t *testing.T, page string) string {
	t.Helper()
	card := strings.Index(page, `class="cv-ledger"`)
	if card < 0 {
		t.Fatalf("the page renders no ledger; body: %s", page)
	}
	rest := page[card:]
	first, last := strings.Index(rest, "<tbody>"), strings.Index(rest, "</tbody>")
	if first < 0 || last < first {
		t.Fatalf("the ledger renders no row body; body: %s", page)
	}
	return rest[first:last]
}

func renderedLedgerRows(t *testing.T, body string) [][]string {
	t.Helper()
	var out [][]string
	for _, row := range ledgerRowRe.FindAllStringSubmatch(body, -1) {
		var cells []string
		for _, cell := range ledgerCellRe.FindAllStringSubmatch(row[1], -1) {
			cells = append(cells, strings.TrimSpace(htmlTagRe.ReplaceAllString(cell[1], " ")))
		}
		out = append(out, cells)
	}
	return out
}

func assertRenderedLedgerIsComplete(t *testing.T, where, page string, want []string) {
	t.Helper()
	body := renderedLedgerBody(t, page)
	// A tooltip is unreachable by touch and by keyboard, so a reason rides its cell (ADR-1875).
	if strings.Contains(body, "title=") {
		t.Errorf("%s: a ledger cell carries a tooltip, and SPEC §2.2 asks for an inline reason", where)
	}
	rows := renderedLedgerRows(t, body)
	if len(rows) != len(want) {
		t.Fatalf("%s: the ledger renders %d rows, and the spec states %d; body: %s", where, len(rows), len(want), page)
	}
	for i, cells := range rows {
		if len(cells) != 4 {
			t.Errorf("%s: row %d renders %d cells, and SPEC §2.2 asks every row to answer four", where, i+1, len(cells))
			continue
		}
		if !strings.EqualFold(cells[0], want[i]) {
			t.Errorf("%s: row %d renders %q, and the spec states %q", where, i+1, cells[0], want[i])
		}
		for j, cell := range cells {
			if cell == "" {
				t.Errorf("%s: %s: column %d renders blank, and SPEC §2.3 bars that", where, want[i], j+1)
			}
		}
	}
}

// SPEC docs/spec/aperture-statement.md §8 criteria 1 and 7, against the rendered markup.

func TestCoverageRendersSevenCompleteLedgerRows(t *testing.T) {
	_, want := v1SpecApertureInputs(t)

	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	assertRenderedLedgerIsComplete(t, "a fresh install", coverageBody(t, ac, base), want)

	declare(t, ac, base, "name", "example.com").Body.Close()
	assertRenderedLedgerIsComplete(t, "a name-only estate", coverageBody(t, ac, base), want)

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()
	assertRenderedLedgerIsComplete(t, "an estate with one address scope", coverageBody(t, ac, base), want)
}

// Every Remedy points at an admin-only control, so a viewer reads the same ledger (#1922).

func TestCoverageRendersSevenCompleteLedgerRowsForAViewer(t *testing.T) {
	_, want := v1SpecApertureInputs(t)

	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	declare(t, login(t, base, "admin", "hunter2hunter2"), base, "name", "example.com").Body.Close()

	vc := login(t, base, "viewer", "hunter2hunter2")
	assertRenderedLedgerIsComplete(t, "a viewer session", coverageBody(t, vc, base), want)
}

// SPEC docs/spec/aperture-statement.md §8 criterion 4 — one computation reaches both renderers.

func TestAPIv1CoverageReturnsTheSevenSpecInputsInOrder(t *testing.T) {
	_, want := v1SpecApertureInputs(t)

	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)
	rec := serveAPI(t, f, http.MethodGet, "/api/v1/coverage", "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var got struct {
		Statement []apiApertureRow `json:"statement"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v (%q)", err, rec.Body.String())
	}
	inputs := make([]string, 0, len(got.Statement))
	for _, row := range got.Statement {
		inputs = append(inputs, row.Input)
	}
	assertSameInputs(t, v1SpecPath, want, "/api/v1/coverage", inputs)
}

// An example short of a row teaches a client a ledger the API does not return (SPEC §5.2).

func TestTheAPIGuideDocumentsTheSevenRowsInOrder(t *testing.T) {
	_, want := v1SpecApertureInputs(t)
	assertSameInputs(t, v1SpecPath, want, apiGuidePath, apiGuideDocumentedInputs(t))
}
