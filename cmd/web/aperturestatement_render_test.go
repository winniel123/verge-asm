package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/tlsoffer"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
	"github.com/winniel123/verge-asm/internal/scan"
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

// The fake covers 10.0.0.0/8, so the fixture's two vantages derive one class each.

func TestCoverageRendersTheVantageClassRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedClassFixtureVantages(t, f)
	base := start(t, f, "")
	page := coverageBody(t, login(t, base, "admin", "hunter2hunter2"), base)

	for _, want := range []string{
		"Vantage class",
		"1 internet · 1 internal",
		"A vantage reads from each side of your boundary, so no class is missing.",
		"A class is derived from the addresses a vantage presents and your declared address scopes, never from a stored field.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the class row is missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "Provision a prober") {
		t.Error("both legs have a reader, so the row must offer no act")
	}
}

func TestCoverageNamesTheMissingInternalLeg(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	txt := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
	// One vantage, presenting an address no declared scope covers, so only the internet leg reads.
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "outside", Class: "unverified", Resolver: "9.9.9.9:53",
		Host: txt("outside.example.net"), Port: pgtype.Int4{Int32: 22, Valid: true},
		Username: txt("scanner"), Availability: txt("available"),
		DialledAddr: txt("203.0.113.9"), Egress: txt("203.0.113.9"),
	})
	f.vantageNextID = 2

	base := start(t, f, "")
	page := coverageBody(t, login(t, base, "admin", "hunter2hunter2"), base)

	for _, want := range []string{
		"1 internet",
		"Provision a prober inside your estate",
		`href="/settings?tab=vantages"`,
		"Run a prober inside your estate, then declare its egress as an address scope.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the class row is missing %q; body: %s", want, page)
		}
	}
	// No control makes a vantage internal, so the withdrawn label must not return (SPEC §4.2).
	if strings.Contains(page, "Add an internal vantage") {
		t.Error("the row offers a control that does not exist")
	}
}

// The gate row sits in slot 3, between the port tiers and the class (SPEC §2.5).

func TestCoverageRendersTheCustodyGateRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	page := coverageBody(t, ac, base)

	ports := strings.Index(page, "Port and transport tiers")
	gate := strings.Index(page, "The custody gate")
	class := strings.Index(page, "Vantage class")
	if ports < 0 || gate < 0 || class < 0 || ports > gate || gate > class {
		t.Errorf("the gate row must sit in slot 3; ports at %d, gate at %d, class at %d", ports, gate, class)
	}
	for _, want := range []string{
		"every dispatch · daily",
		"the extension&#39;s fan-out test rides the daily edge-fanout Scan",
		"total · extension off",
		"0 of 1 name scope extended",
		"A declared address scope admits an address directly.",
		"Extend custody to a name scope",
		"A switch on the Scope screen extends custody to the addresses a name scope resolves into.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the gate row is missing %q; body: %s", want, page)
		}
	}
}

// The TLS row sits in slot 5, between the qtypes and the class (SPEC §2.5).

func TestCoverageRendersTheTLSCandidateSetRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	page := coverageBody(t, ac, base)

	qtypes := strings.Index(page, "The queried qtype set")
	tlsSet := strings.Index(page, "The TLS candidate set")
	class := strings.Index(page, "Vantage class")
	if qtypes < 0 || tlsSet < 0 || class < 0 || qtypes > tlsSet || tlsSet > class {
		t.Errorf("the TLS row must sit in slot 5; qtypes at %d, TLS at %d, class at %d", qtypes, tlsSet, class)
	}
	for _, want := range []string{
		// A `fixed` chip drops the status dot, because this input has no on and no off.
		fmt.Sprintf(`<span class="cv-chip">TLS 1.0 · 1.1 · 1.2 · 1.3 · %d cipher suites</span>`, len(tlsoffer.Ciphers())),
		"weekly · daily · monthly",
		"the certificate handshake carries the same list on whichever port tier makes the connect: hot daily, and cold monthly where the cold tier runs.",
		"Release-coupled: a cadence dial ships for the dns and zone Scans alone.",
		"The floor is TLS 1.0 on purpose",
		"widening it would cost a Break on every acceptance and certificate timeline at once.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the TLS row is missing %q; body: %s", want, page)
		}
	}
	// No tls-acceptance setter ships, so a pointer here would be #1854's silence in a new costume.
	if strings.Contains(page, "Narrow the TLS") {
		t.Error("the row offers a control that does not exist")
	}
}

// The qtype row sits in slot 4, between the gate and the class (SPEC §2.5).

func TestCoverageRendersTheQueriedQtypeSetRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.dnsCadence = 7 * 86400
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	page := coverageBody(t, ac, base)

	gate := strings.Index(page, "The custody gate")
	qtypes := strings.Index(page, "The queried qtype set")
	class := strings.Index(page, "Vantage class")
	if gate < 0 || qtypes < 0 || class < 0 || gate > qtypes || qtypes > class {
		t.Errorf("the qtype row must sit in slot 4; gate at %d, qtypes at %d, class at %d", gate, qtypes, class)
	}
	for _, want := range []string{
		// A `fixed` chip drops the status dot, because this input has no on and no off.
		`<span class="cv-chip">A · AAAA · CNAME · NS · SOA · MX · TXT</span>`,
		// The dial moved, so a typed `daily` would render a cadence the operator withdrew.
		"every 7 days",
		"the DNS scan interval on the Scope screen moves it.",
		"An offer the operator can narrow is a finding the operator can silence",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the qtype row is missing %q; body: %s", want, page)
		}
	}
}

// The control-probe row sits in slot 7, the last of the ledger (SPEC §2.5).

func TestCoverageRendersTheControlProbePopulationRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A fresh install declares no Seed, so the population is empty before the scope lands.
	empty := coverageBody(t, ac, base)
	if !strings.Contains(empty, "No name scope is declared, so no name resolves under a parent inside one") {
		t.Errorf("the empty population states no reason on a fresh install; body: %s", empty)
	}

	declare(t, ac, base, "name", "example.com").Body.Close()
	page := coverageBody(t, ac, base)

	class := strings.Index(page, "Vantage class")
	controls := strings.Index(page, "The control-probe population")
	if class < 0 || controls < 0 || class > controls {
		t.Errorf("the control-probe row must sit last; class at %d, controls at %d", class, controls)
	}
	for _, want := range []string{
		// A `fixed` chip drops the status dot, because this input has no on and no off.
		fmt.Sprintf(`<span class="cv-chip">derived per batch · %d control labels per parent</span>`, wildcarddiscrim.LabelCount),
		"The dns Scan rebuilds the population from its own resolution scope on every run.",
		"The label count is per surviving parent, never per name.",
		"No switch suppresses a control probe, so this population carries no toggle of its own.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the control-probe row is missing %q; body: %s", want, page)
		}
	}
	// The gate stops the population at the Seed, so no screen holds a control over it.
	if strings.Contains(page, "Narrow the control") || strings.Contains(page, "Add a control probe") {
		t.Error("the row offers a control that does not exist")
	}
}

// One computation feeds two renderers, so a cell a later row fills must reach both (SPEC §5).

func TestAPIApertureRowCarriesEveryCellOfTheComputation(t *testing.T) {
	view := reflect.TypeOf(apertureRowView{})
	api := reflect.TypeOf(apiApertureRow{})

	carried := make(map[string]struct{}, api.NumField())
	for i := 0; i < api.NumField(); i++ {
		carried[api.Field(i).Name] = struct{}{}
	}
	for i := 0; i < view.NumField(); i++ {
		name := view.Field(i).Name
		if name == "Figures" {
			continue
		}
		if _, ok := carried[name]; !ok {
			t.Errorf("apiApertureRow drops %s, so the JSON renderer cannot reproduce the row", name)
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
	// The fake declares no vantage, so the class row reads its own no-leg case.
	wantRows := apertureStatement(readInputs(), nil)
	if len(got.Statement) != len(wantRows) {
		t.Fatalf("statement rows = %d, want %d (body %q)", len(got.Statement), len(wantRows), rec.Body.String())
	}

	for i, row := range got.Statement {
		want := wantRows[i]
		if row.StateKind != want.StateKind {
			t.Errorf("%s: state_kind: got %q, want %q", want.Input, row.StateKind, want.StateKind)
		}
		if row.Input != want.Input || row.State != want.State || row.Remedy != want.Remedy || row.RemedyHref != want.RemedyHref {
			t.Errorf("the API row diverges from the computation: got %+v, want %+v", row, want)
		}
		if len(row.Figures) != len(want.Figures) {
			t.Fatalf("%s: figures = %d, want %d", want.Input, len(row.Figures), len(want.Figures))
		}
		for j, f := range row.Figures {
			if f.Text != want.Figures[j].Text {
				t.Errorf("%s: figure %d: got %q, want %q", want.Input, j+1, f.Text, want.Figures[j].Text)
			}
		}
		for name, cell := range map[string]string{
			"input": row.Input, "cadence": row.Cadence, "cadence_why": row.CadenceWhy,
			"state": row.State, "state_detail": row.StateDetail,
			"remedy": row.Remedy, "remedy_why": row.RemedyWhy,
		} {
			if strings.TrimSpace(cell) == "" {
				t.Errorf("%s: %s is blank, and SPEC §2.3 bars a blank cell on either renderer", want.Input, name)
			}
		}
	}
}

// The fake holds no override, so the card draws the shipped default over a real catalogue read.

func TestCoverageRendersTheEnabledSourcesRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := coverageBody(t, ac, base)

	// The row leads the ledger, so #1854's operator meets the source set first (SPEC §2.5).
	sources, ports := strings.Index(page, "Enabled sources"), strings.Index(page, "Port and transport tiers")
	if sources < 0 || ports < 0 || sources > ports {
		t.Errorf("the sources row must lead the port-tier row; sources at %d, ports at %d", sources, ports)
	}
	for _, want := range []string{
		"daily · every 5 minutes",
		"The ct Scan asks daily and the ct-tail Scan every 5 minutes.",
		">crt.sh<",
		fmt.Sprintf("1 of %d sources enabled", len(apertureToggleableSources())),
		"Enable a source",
		`href="/settings?tab=sources"`,
		"A toggle on the Sources tab reaches each source still off: CT drift tail (logs-direct) · Cert Spotter (operator key). Cert Spotter also needs its key on the worker.",
		"so this row names what is declared, never what ran",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the sources row is missing %q; body: %s", want, page)
		}
	}

	f.sourceStates[scan.CTTailSource] = db.SourceState{Slug: scan.CTTailSource, Enabled: true}
	page = coverageBody(t, ac, base)

	if !strings.Contains(page, "crt.sh · CT drift tail (logs-direct)") {
		t.Errorf("a declared override must reach the state cell; body: %s", page)
	}
	if strings.Contains(page, "still off: CT drift tail") {
		t.Errorf("an enabled source must drop out of the remedy; body: %s", page)
	}
}
