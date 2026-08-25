package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// A blanket responder's internet reach is a Gap (ADR-0104), so a sensitive port on
// it reads as ABSENT — HasInternetReach=false — and sensitive-port-reached-from-
// internet returns not-evaluable (never not-fired, and never fired). An ordinary
// reached sensitive port on another address still fires. The damping is at the
// measurement: no rule is narrowed.
func TestBlanketResponderDampsSensitivePortSignal(t *testing.T) {
	f := newFakeStore()
	// A blanket responder: a sensitive internet pair whose reach is a Gap.
	f.addClassReachability(t, "198.51.100.50:3389/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
	// An ordinary origin: the same sensitive port reached from the internet -> fires.
	f.addClassReachability(t, "198.51.100.51:3389/tcp", "internet", obsClock, `{"outcome":"reached"}`)

	srv := newServer(f, testKey, "", fixedClock())
	req := httptest.NewRequest(http.MethodGet, "/signals", nil)
	facts, _, err := srv.buildServiceFacts(req)
	if err != nil {
		t.Fatalf("buildServiceFacts: %v", err)
	}

	byKey := map[string]signal.ServiceFacts{}
	for _, sf := range facts {
		byKey[sf.Subject] = sf
	}
	blanket, ok := byKey["198.51.100.50:3389/tcp"]
	if !ok {
		t.Fatal("blanket responder service missing from facts — a blanketed Service is still a subject")
	}
	if !blanket.OnSensitiveList {
		t.Error("the blanketed pair is 3389/tcp — it must stay in the rule's domain")
	}
	if blanket.HasInternetReach {
		t.Errorf("a blanketed internet reach must read as absent, got InternetReach=%q", blanket.InternetReach)
	}
	if origin := byKey["198.51.100.51:3389/tcp"]; !origin.HasInternetReach || origin.InternetReach != signal.Reached {
		t.Errorf("an ordinary reached origin must keep its value, got %+v", origin)
	}

	// The rule buckets the blanket responder as not-evaluable (its evidence is a
	// Gap), and the origin as fired — never the reverse.
	var rule signal.ServiceRule
	for _, r := range signal.AllServiceRules() {
		if r.Name() == "sensitive-port-reached-from-internet" {
			rule = r
		}
	}
	if rule == nil {
		t.Fatal("sensitive-port-reached-from-internet rule not found")
	}
	if got := rule.Eval(blanket); got != signal.NotEvaluable {
		t.Errorf("blanket responder verdict = %v, want not-evaluable (not not-fired)", got)
	}
	if got := rule.Eval(byKey["198.51.100.51:3389/tcp"]); got != signal.Fired {
		t.Errorf("ordinary origin verdict = %v, want fired", got)
	}
}

// seedZone declares a name-scope Seed and attaches a zone file to it, returning
// nothing — the Signals reads pick it up through ListZoneDeclarations.
func seedZone(t *testing.T, f *fakeStore, admin db.Account, domain, content string) {
	t.Helper()
	s, err := f.CreateNameSeed(t.Context(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.CreateZoneFile(t.Context(), db.CreateZoneFileParams{
		SeedID:     s.ID,
		SuppliedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
		Content:    content,
		UploadedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
}

// The Open tab renders the flat per-instance table (Signals.jsx + SignalData.jsx,
// P2.2): one row per currently-fired (rule, subject) pair, with a SeverityBadge, a
// SIG-#### id, the asset, and the filter / sort affordances — no per-rule census.
func TestSignalsOpenTabRendersFlatInstanceTable(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	zone := "$ORIGIN example.com.\n" +
		"@ IN SOA ns1 admin 1 2 3 4 5\n" +
		"www IN A 203.0.113.20\n" +
		"api IN A 203.0.113.21\n" +
		"missing IN A 203.0.113.22\n"
	seedZone(t, f, admin, "example.com", zone)

	// lame-delegation: an all-refusing delegation → composed Lame → FIRED.
	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)

	// An internal address in a public answer → non-global FIRED.
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)

	// A dangling CNAME → cname-target-name-error FIRED.
	f.addClassResolution(t, "alias.example.com", "internet", obsClock, `{"outcome":"NoData"}`)
	f.addDNSRecord(t, "alias.example.com", "CNAME", obsClock, `{"rrs":[{"name":"alias.example.com","type":"CNAME","data":"gone.example.com"}]}`)
	f.addClassResolution(t, "gone.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	// A declared name our resolver NXDOMAINs → zone-declared-…-name-error FIRED.
	f.addClassResolution(t, "missing.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	// A resolving name inside the zone but not declared → absent-from-zone FIRED.
	f.addClassResolution(t, "orphan.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.30"]}`)
	// A declared name that resolves → both zone rules NOT-FIRED (no row here).
	f.addClassResolution(t, "www.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.20"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/signals", http.StatusOK)

	// The flat table's filter / sort affordances render: the text filter, the
	// severity filter and the "X of Y shown" count.
	for _, want := range []string{`name="q"`, `name="sev"`, "All severities", "shown"} {
		if !strings.Contains(page, want) {
			t.Errorf("Signals flat table missing affordance %q; body: %s", want, page)
		}
	}

	// Severity is a real datum now — a SeverityBadge and a minted SIG id render on
	// the fired rows.
	if !strings.Contains(page, `var(--sev-`) {
		t.Errorf("Signals table renders no SeverityBadge; body: %s", page)
	}
	if !strings.Contains(page, "SIG-") {
		t.Errorf("Signals table renders no SIG-#### id; body: %s", page)
	}

	// Every fired (rule, subject) pair is a row, keyed on the asset.
	for _, asset := range []string{"lame.example.com", "leak.example.com", "orphan.example.com", "missing.example.com", "alias.example.com"} {
		if !strings.Contains(page, asset) {
			t.Errorf("fired signal on %q not rendered as an instance row; body: %s", asset, page)
		}
	}

	// The Open / Annotated / Withdrawn tabs frame the screen and default to Open.
	if !strings.Contains(page, `class="sg-tabs"`) {
		t.Errorf("Signals page renders no tabs; body: %s", page)
	}
	for _, href := range []string{`href="/signals?tab=open"`, `href="/signals?tab=annotated"`, `href="/signals?tab=withdrawn"`} {
		if !strings.Contains(page, href) {
			t.Errorf("Signals tabs missing %q", href)
		}
	}
	if !strings.Contains(page, "sg-tab on") {
		t.Errorf("no active tab on the default Signals view")
	}

	// The per-rule census grouping has left the screen (P2.2, ADR-0116).
	for _, gone := range []string{"Did not fire", "Not-evaluable", "No population"} {
		if strings.Contains(page, gone) {
			t.Errorf("Signals page still renders the census markup %q; the flat table replaces it", gone)
		}
	}

	// 'finding' is a rejected word (the domain says signal, never finding).
	if strings.Contains(strings.ToLower(page), "finding") {
		t.Errorf("Signals page uses the rejected word 'finding'")
	}
}

// The Withdrawn tab surfaces an accepted risk whose subject has left its rule's
// population — an orphan on read, withdrawn by the world (never resolved by an
// operator, ADR-0092), carrying the WithdrawnMark. It does not appear on the
// Annotated tab, which is accepted risks still in population, and the row's reason
// reads in its Drawer.
func TestSignalsWithdrawnTabSurfacesOrphanAnnotations(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A pair whose subject the rule never censuses → orphan → withdrawn.
	annotate(t, ac, base, "ghost.example.com", "lame-delegation", "kept for when it returns").Body.Close()

	withdrawnPage := getBody(t, ac, base+"/signals?tab=withdrawn", http.StatusOK)
	for _, want := range []string{"ghost.example.com", "withdrawn"} {
		if !strings.Contains(withdrawnPage, want) {
			t.Errorf("Withdrawn tab missing %q; body: %s", want, withdrawnPage)
		}
	}

	// The reason reads in the row Drawer, not the table.
	annos, _ := f.ListAnnotations(t.Context())
	if len(annos) != 1 {
		t.Fatalf("precondition: annotations = %d, want 1", len(annos))
	}
	drawer := getBody(t, ac, base+"/signals?tab=withdrawn&view="+strconv.FormatInt(annos[0].ID, 10), http.StatusOK)
	if !strings.Contains(drawer, "kept for when it returns") {
		t.Errorf("Withdrawn drawer missing the reason; body: %s", drawer)
	}

	// The same orphan does not appear on the Annotated tab (accepted risks still in
	// population). With only the orphan declared, Annotated is the empty state.
	annotatedPage := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if strings.Contains(annotatedPage, "ghost.example.com") {
		t.Errorf("orphan annotation wrongly listed on the Annotated tab")
	}
	if !strings.Contains(annotatedPage, "No annotation is declared") {
		t.Errorf("Annotated tab should show the empty state when only an orphan exists; body: %s", annotatedPage)
	}
}

// The Annotated tab lists accepted risks still in population as instance rows, and
// the row Drawer carries the operator dial (the AnnotationControl — the accepted
// risk, its reason, and the remove control).
func TestSignalsAnnotatedTabListsAcceptedRisk(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "accepted, being retired").Body.Close()

	page := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if !strings.Contains(page, "lame.example.com") {
		t.Errorf("Annotated tab missing the accepted subject as a row; body: %s", page)
	}

	annos, _ := f.ListAnnotations(t.Context())
	if len(annos) != 1 {
		t.Fatalf("precondition: annotations = %d, want 1", len(annos))
	}
	drawer := getBody(t, ac, base+"/signals?tab=annotated&view="+strconv.FormatInt(annos[0].ID, 10), http.StatusOK)
	for _, want := range []string{"accepted, being retired", "accepted risk", "Remove annotation", `action="/annotations/withdraw"`} {
		if !strings.Contains(drawer, want) {
			t.Errorf("Annotated drawer missing %q; body: %s", want, drawer)
		}
	}
}

// The row-detail Drawer opens server-side via ?view=, reading one signal's detail
// with its AnnotationControl remove control.
func TestSignalsDetailDrawerOpens(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "reviewed and accepted").Body.Close()
	annos, _ := f.ListAnnotations(t.Context())
	if len(annos) != 1 {
		t.Fatalf("precondition: annotations = %d, want 1", len(annos))
	}

	page := getBody(t, ac, base+"/signals?tab=annotated&view="+strconv.FormatInt(annos[0].ID, 10), http.StatusOK)
	for _, want := range []string{`class="sg-drawer"`, `role="dialog"`, "lame.example.com", "reviewed and accepted", "Remove annotation"} {
		if !strings.Contains(page, want) {
			t.Errorf("detail drawer missing %q; body: %s", want, page)
		}
	}
}

// The typed-name descope ConfirmDialog is admin-only, uses the shared dialog/scrim
// surface, and is wired to the real POST /exclusions act (kind=subtree) — the
// "remove a name and its subjects from scope, recorded on Scope" act — so descoping
// from Signals narrows measurement without adding a route here.
func TestSignalsDescopeConfirmDialogWiredToExclusions(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	// A firing signal so a row exists to descope; ?descope=<ViewKey> resolves to its asset.
	lameName(t, f, "lame.example.com")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	// Render once to mint the instance identity, then read its SIG id (the row's ViewKey).
	getBody(t, ac, base+"/signals", http.StatusOK)
	insts, _ := f.ListSignalInstances(t.Context())
	if len(insts) == 0 {
		t.Fatal("precondition: no signal instance minted for the firing rule")
	}
	key := formatSigID(insts[0].ID)

	// The typed-confirm dialog resolves the row's asset — the exact string the operator must retype
	// — and posts the typed value to the real POST /exclusions (kind=subtree).
	dialog := getBody(t, ac, base+"/signals?descope="+key, http.StatusOK)
	for _, want := range []string{`class="sg-dialog"`, `class="sg-scrim dlg"`, "Descope seed", `action="/exclusions"`, `name="value"`, `value="subtree"`, "to confirm", "lame.example.com"} {
		if !strings.Contains(dialog, want) {
			t.Errorf("descope confirm dialog missing %q; body: %s", want, dialog)
		}
	}

	// The dialog posts to the real exclusion act and records a name exclusion.
	resp := postForm(t, ac, base+"/exclusions", url.Values{"kind": {"subtree"}, "value": {"old.example.com"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("descope POST /exclusions: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if excl, _ := f.ListExclusions(t.Context()); len(excl) != 1 {
		t.Fatalf("descope did not record an exclusion; exclusions = %d, want 1", len(excl))
	}

	// A viewer can never OPEN the typed-confirm dialog: the descope route renders no dialog for a
	// non-admin (the menu item is design-owned client markup, but the act + dialog are admin-gated).
	vc := login(t, base, "viewer", "hunter2hunter2")
	vp := getBody(t, vc, base+"/signals?descope="+key, http.StatusOK)
	if strings.Contains(vp, `class="sg-dialog"`) {
		t.Errorf("a viewer must not open the descope confirm dialog; body: %s", vp)
	}
}

// Fired Service and Endpoint signals render as flat per-instance rows keyed on
// their Service / Endpoint subject key, with a SeverityBadge and a SIG id. A rule
// that does not fire raises no row (the census's not-fired / not-evaluable members
// have left the screen).
func TestSignalsRendersServiceAndEndpointRules(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// A name in the estate, so the redirect-to-host rule has an estate to test against.
	f.addClassResolution(t, "good.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	// sensitive-port-reached-from-internet: a sensitive pair (3389/tcp) reached from
	// the internet → FIRED. A non-sensitive pair (443/tcp) is outside the domain.
	f.addClassReachability(t, "198.51.100.1:3389/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.9:443/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	// A sensitive pair seen only internally → no internet leg → NOT-EVALUABLE.
	f.addClassReachability(t, "198.51.100.2:445/tcp", "internal", obsClock, `{"outcome":"reached"}`)

	// A presented certificate → the cert-detail rules render it NOT-EVALUABLE (no
	// fired row), and it is inside hostname-san-mismatch's domain.
	f.addCertificate(t, "secure.example.com@198.51.100.1:443/tcp", obsClock, `{"outcome":"presented","chain":["sha256:abc"]}`)

	// A plaintext endpoint: HTTP responded (200) and the certificate is no-tls →
	// plaintext-http-no-https FIRED, and unauthenticated-request-answered FIRED.
	f.addHTTPIdentity(t, "plain.example.com@198.51.100.5:80/tcp", obsClock, `{"outcome":"responded","status":200}`)
	f.addCertificate(t, "plain.example.com@198.51.100.5:80/tcp", obsClock, `{"outcome":"no-tls"}`)

	// A redirect that does not upgrade and points outside the estate → both redirect
	// rules FIRED.
	f.addHTTPIdentity(t, "redir.example.com@198.51.100.6:80/tcp", obsClock, `{"outcome":"responded","status":301,"redirect_location":"http://outside.test/x"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/signals", http.StatusOK)

	// The fired Service and Endpoint pairs render as rows, keyed on their subject.
	for _, asset := range []string{
		"198.51.100.1:3389/tcp",                 // sensitive-port fired
		"plain.example.com@198.51.100.5:80/tcp", // plaintext-http + unauthenticated fired
		"redir.example.com@198.51.100.6:80/tcp", // both redirect rules fired
	} {
		if !strings.Contains(page, asset) {
			t.Errorf("fired signal on %q not rendered as an instance row; body: %s", asset, page)
		}
	}
	// Severity and the minted id render on the rows.
	if !strings.Contains(page, `var(--sev-`) || !strings.Contains(page, "SIG-") {
		t.Errorf("Service/Endpoint rows missing a SeverityBadge or SIG id; body: %s", page)
	}
	// A not-evaluable certificate member raises no row: the census markup is gone.
	if strings.Contains(page, "No population") || strings.Contains(page, "Not-evaluable") {
		t.Errorf("Signals page still renders census markup; the flat table shows only fired instances")
	}
}

// An empty estate raises no signals: the Open tab degrades via the spec's empty
// pattern (fact + next action), never a census of zeroes.
func TestSignalsEmptyEstateRendersEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(page, "No open signals") {
		t.Errorf("empty estate did not render the open empty state; body: %s", page)
	}
	// The screen still frames itself — the tabs render.
	if !strings.Contains(page, `class="sg-tabs"`) {
		t.Errorf("empty estate dropped the tabs; body: %s", page)
	}
}

// GET /signals/export streams a text/csv attachment of the current tab's filtered
// per-instance rows (Signals.jsx header; PARITY collision #6): a header row plus one
// row per signal instance, carrying its severity, id, signal, asset, port and
// instants. A lame delegation fires lame-delegation, so its instance exports.
func TestSignalsExportCSV(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A lame delegation → composed Lame → lame-delegation FIRES on this subject.
	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/signals/export")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signals export status = %d, want 200 (body: %s)", resp.StatusCode, got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("signals export Content-Type = %q, want text/csv", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=") || !strings.Contains(cd, ".csv") {
		t.Errorf("signals export Content-Disposition = %q, want an attachment .csv filename", cd)
	}
	for _, want := range []string{
		"severity,id,signal,asset,port,first_seen,last_seen", // header row
		"medium",           // lame-delegation's severity
		"lame-delegation",  // the fired rule
		"lame.example.com", // the fired subject
	} {
		if !strings.Contains(got, want) {
			t.Errorf("signals export CSV missing %q; body:\n%s", want, got)
		}
	}
}

// The Export CSV button is gated on the current tab having rows to export: disabled
// on an estate with no signals, a live link once a rule fires.
func TestSignalsExportButtonGated(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Empty estate: no signals fire, so the button is disabled (no export link).
	empty := getBody(t, ac, base+"/signals", http.StatusOK)
	if strings.Contains(empty, `href="/signals/export`) {
		t.Errorf("empty estate should not offer a signals export link; body: %s", empty)
	}

	// A lame delegation fires a signal, enabling the export.
	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)
	full := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(full, `href="/signals/export`) {
		t.Errorf("populated Signals did not enable the export link; body: %s", full)
	}
}

// TestDeriveSignalInstancesDatum is the P0.1 datum end-to-end: the handler folds
// the live censuses into the flat per-instance table SignalData.jsx renders — each
// currently-fired (rule, subject) pair carrying its rule's severity, a stable minted
// SIG-#### id, and its first/last-seen instants — ordered by the severity ramp.
func TestDeriveSignalInstancesDatum(t *testing.T) {
	f := newFakeStore()
	srv := newServer(f, testKey, "", fixedClock())

	// A critical service firing and a medium name firing, given out of ramp order to
	// prove the derivation sorts by severity.
	censuses := []signal.Census{
		{Rule: "lame-delegation", Fired: []signal.Member{{Subject: "edge.example.com"}}},
		{Rule: "sensitive-port-reached-from-internet", Fired: []signal.Member{{Subject: "198.51.100.7:5900/tcp"}}},
	}

	insts, err := srv.deriveSignalInstances(t.Context(), censuses)
	if err != nil {
		t.Fatalf("deriveSignalInstances: %v", err)
	}
	if len(insts) != 2 {
		t.Fatalf("want 2 instances (one per fired pair), got %d", len(insts))
	}

	// Severity ramp order: critical before medium.
	crit := insts[0]
	if crit.Signal != "sensitive-port-reached-from-internet" {
		t.Fatalf("critical instance should sort first, got %q", crit.Signal)
	}
	if crit.Severity != "critical" || crit.SevRank != 0 {
		t.Errorf("severity = %q rank %d, want critical/0", crit.Severity, crit.SevRank)
	}
	if crit.IP != "198.51.100.7" || crit.Port != ":5900" {
		t.Errorf("addr/port = %q/%q, want 198.51.100.7/:5900", crit.IP, crit.Port)
	}
	if crit.Title != "Sensitive port reached from internet" {
		t.Errorf("title = %q", crit.Title)
	}
	if !strings.HasPrefix(crit.SigID, "SIG-") {
		t.Errorf("SigID = %q, want a SIG-#### id", crit.SigID)
	}
	if crit.First == "" || crit.Last == "" {
		t.Errorf("instants: first=%q last=%q, both must be present", crit.First, crit.Last)
	}
	if med := insts[1]; med.Severity != "medium" || med.IP != "" || med.Port != "" {
		t.Errorf("name instance = %+v, want medium severity and no ip/port", med)
	}

	// Identity is stable: re-deriving the same fired set keeps each SIG id and its
	// first-seen instant (the mint is idempotent on the pair).
	again, err := srv.deriveSignalInstances(t.Context(), censuses)
	if err != nil {
		t.Fatalf("re-derive: %v", err)
	}
	if again[0].SigID != crit.SigID {
		t.Errorf("SIG id not stable: %q then %q", crit.SigID, again[0].SigID)
	}
	if again[0].First != crit.First {
		t.Errorf("first-seen not stable: %q then %q", crit.First, again[0].First)
	}
}

func TestSignalsRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/signals")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /signals: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
