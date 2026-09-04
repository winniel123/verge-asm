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

func TestBlanketResponderDampsSensitivePortSignal(t *testing.T) {
	f := newFakeStore()
	f.addClassReachability(t, "198.51.100.50:3389/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
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

func TestTLS10AcceptedFiresFromPersistedAcceptance(t *testing.T) {
	f := newFakeStore()
	const (
		accepts10 = "198.51.100.7:443/tcp"
		modern    = "198.51.100.8:443/tcp"
		refused   = "198.51.100.9:443/tcp"
	)
	for _, svc := range []string{accepts10, modern, refused} {
		f.addClassReachability(t, svc, "internet", obsClock, `{"outcome":"reached"}`)
	}
	f.addTLSAcceptance(t, accepts10, obsClock,
		`{"outcome":"enumerated","versions":[{"version":"1.0","ciphers":["TLS_RSA_WITH_AES_128_CBC_SHA"]},{"version":"1.2","ciphers":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]}]}`)
	f.addTLSAcceptance(t, modern, obsClock,
		`{"outcome":"enumerated","versions":[{"version":"1.2"},{"version":"1.3"}]}`)
	f.addTLSAcceptance(t, refused, obsClock, `{"outcome":"tls-refused"}`)

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

	got := byKey[accepts10]
	if !got.TLSHandshakeCompleted || !got.TLSVersionsReadable || !got.TLS10Accepted {
		t.Errorf("accepting service facts = %+v; want handshake completed, versions readable, 1.0 accepted", got)
	}
	if m := byKey[modern]; !m.TLSHandshakeCompleted || !m.TLSVersionsReadable || m.TLS10Accepted {
		t.Errorf("modern service facts = %+v; want handshake completed, no 1.0", m)
	}
	if rf := byKey[refused]; rf.TLSHandshakeCompleted {
		t.Errorf("refused service must not read as handshake-completed, got %+v", rf)
	}

	var rule signal.ServiceRule
	for _, r := range signal.AllServiceRules() {
		if r.Name() == "tls-1.0-accepted" {
			rule = r
		}
	}
	if rule == nil {
		t.Fatal("tls-1.0-accepted rule not found")
	}
	if v := rule.Eval(byKey[accepts10]); v != signal.Fired {
		t.Errorf("tls-1.0-accepted on the 1.0-accepting service = %v, want fired", v)
	}
	if v := rule.Eval(byKey[modern]); v != signal.NotFired {
		t.Errorf("tls-1.0-accepted on the modern service = %v, want not-fired", v)
	}
	if v := rule.Eval(byKey[refused]); v != signal.OutsideDomain {
		t.Errorf("tls-1.0-accepted on the refusing service = %v, want outside-domain", v)
	}
}

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

func TestSignalsOpenTabRendersFlatInstanceTable(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	zone := "$ORIGIN example.com.\n" +
		"@ IN SOA ns1 admin 1 2 3 4 5\n" +
		"www IN A 203.0.113.20\n" +
		"api IN A 203.0.113.21\n" +
		"missing IN A 203.0.113.22\n"
	seedZone(t, f, admin, "example.com", zone)

	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)

	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)

	f.addClassResolution(t, "alias.example.com", "internet", obsClock, `{"outcome":"NoData"}`)
	f.addDNSRecord(t, "alias.example.com", "CNAME", obsClock, `{"rrs":[{"name":"alias.example.com","type":"CNAME","data":"gone.example.com"}]}`)
	f.addClassResolution(t, "gone.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	f.addClassResolution(t, "missing.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	f.addClassResolution(t, "orphan.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.30"]}`)
	f.addClassResolution(t, "www.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.20"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/signals", http.StatusOK)

	for _, want := range []string{`name="q"`, `name="sev"`, "All severities", "shown"} {
		if !strings.Contains(page, want) {
			t.Errorf("Signals flat table missing affordance %q; body: %s", want, page)
		}
	}

	if !strings.Contains(page, `var(--sev-`) {
		t.Errorf("Signals table renders no SeverityBadge; body: %s", page)
	}
	if !strings.Contains(page, "SIG-") {
		t.Errorf("Signals table renders no SIG-#### id; body: %s", page)
	}

	for _, asset := range []string{"lame.example.com", "leak.example.com", "orphan.example.com", "missing.example.com", "alias.example.com"} {
		if !strings.Contains(page, asset) {
			t.Errorf("fired signal on %q not rendered as an instance row; body: %s", asset, page)
		}
	}

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

	for _, gone := range []string{"Did not fire", "Not-evaluable", "No population"} {
		if strings.Contains(page, gone) {
			t.Errorf("Signals page still renders the census markup %q; the flat table replaces it", gone)
		}
	}

	if strings.Contains(strings.ToLower(page), "finding") {
		t.Errorf("Signals page uses the rejected word 'finding'")
	}
}

func TestSignalsWithdrawnTabSurfacesOrphanAnnotations(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "ghost.example.com", "lame-delegation", "kept for when it returns").Body.Close()

	withdrawnPage := getBody(t, ac, base+"/signals?tab=withdrawn", http.StatusOK)
	for _, want := range []string{"ghost.example.com", "withdrawn"} {
		if !strings.Contains(withdrawnPage, want) {
			t.Errorf("Withdrawn tab missing %q; body: %s", want, withdrawnPage)
		}
	}

	annos, _ := f.ListAnnotations(t.Context())
	if len(annos) != 1 {
		t.Fatalf("precondition: annotations = %d, want 1", len(annos))
	}
	drawer := getBody(t, ac, base+"/signals?tab=withdrawn&view="+strconv.FormatInt(annos[0].ID, 10), http.StatusOK)
	if !strings.Contains(drawer, "kept for when it returns") {
		t.Errorf("Withdrawn drawer missing the reason; body: %s", drawer)
	}

	annotatedPage := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if strings.Contains(annotatedPage, "ghost.example.com") {
		t.Errorf("orphan annotation wrongly listed on the Annotated tab")
	}
	if !strings.Contains(annotatedPage, "No annotation is declared") {
		t.Errorf("Annotated tab should show the empty state when only an orphan exists; body: %s", annotatedPage)
	}
}

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
	for _, want := range []string{
		`class="sg-drawer"`, `role="dialog"`, "lame.example.com", "reviewed and accepted", "Remove annotation",
		// The rendered version is itself "rule@v1", so the doubled @ is not a typo.
		"lame-delegation@rule@v1",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("detail drawer missing %q; body: %s", want, page)
		}
	}
}

func TestSignalsDescopeConfirmDialogWiredToExclusions(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	lameName(t, f, "lame.example.com")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	getBody(t, ac, base+"/signals", http.StatusOK)
	insts, _ := f.ListSignalInstances(t.Context())
	if len(insts) == 0 {
		t.Fatal("precondition: no signal instance minted for the firing rule")
	}
	key := formatSigID(insts[0].ID)

	dialog := getBody(t, ac, base+"/signals?descope="+key, http.StatusOK)
	for _, want := range []string{`class="sg-dialog"`, `class="sg-scrim dlg"`, "Descope seed", `action="/exclusions"`, `name="value"`, `value="subtree"`, "to confirm", "lame.example.com"} {
		if !strings.Contains(dialog, want) {
			t.Errorf("descope confirm dialog missing %q; body: %s", want, dialog)
		}
	}

	resp := postForm(t, ac, base+"/exclusions", url.Values{"kind": {"subtree"}, "value": {"old.example.com"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("descope POST /exclusions: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if excl, _ := f.ListExclusions(t.Context()); len(excl) != 1 {
		t.Fatalf("descope did not record an exclusion; exclusions = %d, want 1", len(excl))
	}

	vc := login(t, base, "viewer", "hunter2hunter2")
	vp := getBody(t, vc, base+"/signals?descope="+key, http.StatusOK)
	if strings.Contains(vp, `class="sg-dialog"`) {
		t.Errorf("a viewer must not open the descope confirm dialog; body: %s", vp)
	}
}

func TestSignalsRendersServiceAndEndpointRules(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	f.addClassResolution(t, "good.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	f.addClassReachability(t, "198.51.100.1:3389/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.9:443/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.2:445/tcp", "internal", obsClock, `{"outcome":"reached"}`)

	f.addCertificate(t, "secure.example.com@198.51.100.1:443/tcp", obsClock, `{"outcome":"presented","chain":["sha256:abc"]}`)

	f.addHTTPIdentity(t, "plain.example.com@198.51.100.5:80/tcp", obsClock, `{"outcome":"responded","status":200}`)
	f.addCertificate(t, "plain.example.com@198.51.100.5:80/tcp", obsClock, `{"outcome":"no-tls"}`)

	f.addHTTPIdentity(t, "redir.example.com@198.51.100.6:80/tcp", obsClock, `{"outcome":"responded","status":301,"redirect_location":"http://outside.test/x"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/signals", http.StatusOK)

	for _, asset := range []string{
		"198.51.100.1:3389/tcp",
		"plain.example.com@198.51.100.5:80/tcp",
		"redir.example.com@198.51.100.6:80/tcp",
	} {
		if !strings.Contains(page, asset) {
			t.Errorf("fired signal on %q not rendered as an instance row; body: %s", asset, page)
		}
	}
	if !strings.Contains(page, `var(--sev-`) || !strings.Contains(page, "SIG-") {
		t.Errorf("Service/Endpoint rows missing a SeverityBadge or SIG id; body: %s", page)
	}
	if strings.Contains(page, "No population") || strings.Contains(page, "Not-evaluable") {
		t.Errorf("Signals page still renders census markup; the flat table shows only fired instances")
	}
}

func TestSignalsEmptyEstateRendersEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(page, "No open signals") {
		t.Errorf("empty estate did not render the open empty state; body: %s", page)
	}
	if !strings.Contains(page, `class="sg-tabs"`) {
		t.Errorf("empty estate dropped the tabs; body: %s", page)
	}
}

func TestSignalsExportCSV(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
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
		"severity,id,signal,asset,port,first_seen,last_seen",
		"medium",
		"lame-delegation",
		"lame.example.com",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("signals export CSV missing %q; body:\n%s", want, got)
		}
	}
}

func TestSignalsExportButtonGated(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	empty := getBody(t, ac, base+"/signals", http.StatusOK)
	if strings.Contains(empty, `href="/signals/export`) {
		t.Errorf("empty estate should not offer a signals export link; body: %s", empty)
	}

	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)
	full := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(full, `href="/signals/export`) {
		t.Errorf("populated Signals did not enable the export link; body: %s", full)
	}
}

func TestDeriveSignalInstancesDatum(t *testing.T) {
	f := newFakeStore()
	srv := newServer(f, testKey, "", fixedClock())

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
