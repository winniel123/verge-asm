package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

func portTierCopy(t *testing.T) (nameOnly, healthy []string) {
	t.Helper()
	core := vergecore.Default()
	sensitive := core.Count().Sensitive
	udp := sensitiveUDPPairs(core)
	shared := []string{
		"Port and transport tiers",
		"daily · monthly",
		"hot daily, cold monthly. Release-coupled: no operator dial exists.",
		"hot on · cold off · udp no flag",
		"The cold tier&#39;s state is the shadow of an empty scope list, not a switch. UDP has no flag at all.",
		fmt.Sprintf("0 of %d sensitive pairs the instrument cannot report as reached", sensitive),
		fmt.Sprintf("0 of %d rules unevaluable", len(signal.AllRuleNames())),
	}
	nameOnly = append(append([]string(nil), shared...),
		fmt.Sprintf("%d of %d sensitive pairs unread", sensitive, sensitive),
		"Declare an address scope",
		"/scope",
		"No declared scope reads a sensitive pair. An address scope, or a custody extension on a name scope, moves this figure.",
	)
	healthy = append(append([]string(nil), shared...),
		fmt.Sprintf("%d of %d sensitive pairs unread", udp, sensitive),
		fmt.Sprintf("The %d pairs still unread are UDP. No tier reads them, and no setting opens one.", udp),
	)
	return nameOnly, healthy
}

func TestCoverageRendersThePortTierLedgerRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	nameOnly, healthy := portTierCopy(t)

	declare(t, ac, base, "name", "example.com").Body.Close()
	page := coverageBody(t, ac, base)

	// The ledger sits above the grid, so #1854's estate meets it before the meters (SPEC §2.1).
	ledger, grid := strings.Index(page, `class="cv-ledger"`), strings.Index(page, `class="cv-grid"`)
	if ledger < 0 || grid < 0 || ledger > grid {
		t.Errorf("the statement must render above the two-column grid; ledger at %d, grid at %d", ledger, grid)
	}
	for _, col := range []string{">Input<", ">Cadence<", ">State<", ">Remedy<"} {
		if !strings.Contains(page, col) {
			t.Errorf("the ledger is missing the %s column; body: %s", col, page)
		}
	}
	for _, want := range nameOnly {
		if !strings.Contains(page, want) {
			t.Errorf("a name-only estate is missing %q; body: %s", want, page)
		}
	}

	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()
	page = coverageBody(t, ac, base)

	for _, want := range healthy {
		if !strings.Contains(page, want) {
			t.Errorf("an estate with one address scope is missing %q; body: %s", want, page)
		}
	}
	for _, gone := range []string{"Declare an address scope", "No declared scope reads a sensitive pair"} {
		if strings.Contains(page, gone) {
			t.Errorf("a declared address scope must drop %q from the remedy; body: %s", gone, page)
		}
	}
}

func TestAPIv1CoverageCarriesTheStatementBesideTheMeters(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	rec := serveAPI(t, f, http.MethodGet, "/api/v1/coverage", "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var got struct {
		Meters    []apiCoverageMeter `json:"meters"`
		Statement []apiApertureRow   `json:"statement"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v (%q)", err, rec.Body.String())
	}
	if got.Meters == nil {
		t.Error("the statement must ride beside meters, never replace it")
	}
	if len(got.Statement) != 1 {
		t.Fatalf("statement rows = %d, want 1 (body %q)", len(got.Statement), rec.Body.String())
	}

	row := got.Statement[0]
	want := apertureStatement(nil)[0]
	if row.Input != want.Input || row.State != want.State || row.Remedy != want.Remedy {
		t.Errorf("the API row diverges from the computation: got %+v, want %+v", row, want)
	}
	if len(row.Figures) != len(want.Figures) {
		t.Fatalf("figures = %d, want %d", len(row.Figures), len(want.Figures))
	}
	for i, f := range row.Figures {
		if f.Text != want.Figures[i].Text {
			t.Errorf("figure %d: got %q, want %q", i+1, f.Text, want.Figures[i].Text)
		}
	}
	for name, cell := range map[string]string{
		"input": row.Input, "cadence": row.Cadence, "cadence_why": row.CadenceWhy,
		"state": row.State, "state_detail": row.StateDetail,
		"remedy": row.Remedy, "remedy_why": row.RemedyWhy,
	} {
		if strings.TrimSpace(cell) == "" {
			t.Errorf("%s is blank, and SPEC §2.3 bars a blank cell on either renderer", name)
		}
	}
}
