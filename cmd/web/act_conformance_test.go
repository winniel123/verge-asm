package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// What this gate proves, stated honestly (spec §7.5).
//
// It proves that every auditable act's handler holds a Record call somewhere on its live call
// graph, and that no exempt route does. It does not prove the call runs, runs once, runs after
// the mutation, runs only on success, or carries the right Actor or Subject. A handler that
// records the wrong subject passes. A handler that records inside a branch never taken passes.
// Ordering, cardinality and content are held by review and by unit tests, never by this gate.
//
// This does not make the violation inexpressible the way ADR-0123's bearer path does. It makes
// it fail CI — for POST. For a non-POST act outside limb 4, it does not even do that.
//
// Three coverage gaps, named rather than smoothed (spec §7.5).
//
//  1. A new auditable GET is not caught. POST defaults auditable and fails closed. Non-POST
//     defaults exempt behind a three-route allowlist and fails open. Only limb 4's
//     transcript.Open sink fails closed. A future limb-2 act on a GET, shaped like the SSO
//     link callback, would ship unrecorded and silent.
//  2. Non-route acts sit outside the mux walk. The bootstrap and the restore are both routes,
//     so both are covered. goose.Up is not, and §1.6 rules it writes no Act.
//  3. The five integrations routes register inside `if integrationsEnabled`, which is a const
//     true at cmd/web/integrations.go:33, so they harvest today. If the flag flipped, the
//     harvest would read their absence as compliance rather than as a gap.

// A POST route is auditable unless this map names it. §2.3 rules each of these, with its reason.

var actExemptRoutes = map[string]string{
	"POST /login":                            "login family: which credential is presented, never who may act (§1.2)",
	"POST /login/totp":                       "login family: the second factor moves no grant (§1.2)",
	"POST /logout":                           "login family: ending a session leaves every grant standing (§1.2)",
	"POST /signout":                          "login family: the shell's alias onto the same /logout handler (§1.2)",
	"POST /forgot":                           "login family: unauthenticated, and it takes an attacker-supplied username (§1.2)",
	"POST /profile/session/revoke":           "login family: the caller ends its own session (§1.2)",
	"POST /profile/sessions/revoke":          "login family: the caller ends one of its own sessions (§1.2)",
	"POST /profile/sessions/revoke-others":   "login family: the caller ends its own other sessions (§1.2)",
	"POST /settings/sessions/revoke":         "login family: cross-principal, and still the target's grant is untouched (§2.3)",
	"POST /settings/sessions/revoke-account": "login family: typed-name-confirmed, and still no grant moves (§2.3)",

	"POST /account/totp/enable":        "stages a secret with totp_enabled still false; the confirm is the act (§2.3)",
	"POST /settings/restore/preflight": "stages the archive in process memory only; the apply is the act (§2.3)",

	"POST /onboarding":         "wizard step navigation; it writes nothing (§2.3)",
	"POST /seeds/preview":      "renders a withdrawal receipt into a flash and writes nothing (§2.3)",
	"POST /exclusions/preview": "read-only preview (§2.3)",
	"POST /messages/read":      "per-account read state, already legible to its own reader (§2.3)",
	"POST /messages/read-all":  "per-account read state, already legible to its own reader (§2.3)",
	"POST /messages/unread":    "per-account read state, already legible to its own reader (§2.3)",

	"POST /settings/backup":      "declares nothing, grants nothing, directs nothing; transcript is backupExcluded (§2.3)",
	"POST /reports/schedule/run": "inserts one report_delivery and pushes to no channel; the day it delivers it is limb 3 (§2.3)",
}

// Non-POST defaults exempt, so the three acts that are not POST ride an allowlist (spec §7.5).

var actAuditableNonPOST = map[string]string{
	"GET /profile/sso/{slug}/link/callback": "sso.binding.created, one of ADR-0113's three named audited acts (§2.1)",
	"GET /run/{id}/raw":                     "transcript.disclosed, limb 4 (§1.4)",
	"GET /runs/{id}/raw":                    "transcript.disclosed, limb 4 (§1.4)",
}

// GET /profile/sso/{slug}/link and GET /login/sso/{slug}/callback are the two non-POST
// exemptions §2.3 names. They need no entry: non-POST is exempt by default, and neither
// reaches the limb-4 sink. They are held instead by the no-exempt-route-records rule below.

// It carried every auditable route that had no Record call yet, keyed to the wiring ticket that
// owed it, because POST defaults auditable and fails closed (spec §7.5) and the gate landed
// before the bulk wiring. #1835 wired the last ten, so the map is empty and stays empty: a new
// auditable route is wired in its own PR, never parked here. Decided by map #1826, not the SPEC.

var actPending = map[string]string{}

type actRules struct {
	exempt           map[string]string
	auditableNonPOST map[string]string
}

func liveActRules() actRules {
	return actRules{exempt: actExemptRoutes, auditableNonPOST: actAuditableNonPOST}
}

type actReport struct {
	auditable map[string]bool
	records   map[string]bool
	sink      map[string]bool
	handler   map[string]string
}

// One line inside s.backToScope bought eleven false passes, and backToScope is a refusal path,
// so the stop set is part of the rule and not an optimisation (spec §7.3).

func actStop() map[string]bool {
	stop := map[string]bool{}
	for k := range backHelpers {
		stop[k] = true
	}
	for k := range bodyAnswers {
		stop[k] = true
	}
	return stop
}

func (c *contractPkg) actAudit(rules actRules) actReport {
	rep := actReport{
		auditable: map[string]bool{},
		records:   map[string]bool{},
		sink:      map[string]bool{},
		handler:   map[string]string{},
	}
	for pattern, handler := range c.routes {
		rep.handler[pattern] = handler
		var records, sink bool
		c.reach("s."+handler, actStop(), func(_ string, fn *ast.FuncDecl) {
			if callsRecord(fn) {
				records = true
			}
			if opensTranscript(fn) {
				sink = true
			}
		})
		rep.records[pattern] = records
		rep.sink[pattern] = sink
		rep.auditable[pattern] = auditableRoute(pattern, sink, rules)
	}
	return rep
}

func auditableRoute(pattern string, sink bool, rules actRules) bool {
	// transcript.Open is reached by two routes tree-wide, so limb 4's sink rule is exact (§7.4).
	if sink {
		return true
	}
	if _, ok := rules.auditableNonPOST[pattern]; ok {
		return true
	}
	if !strings.HasPrefix(pattern, "POST ") {
		return false
	}
	_, exempt := rules.exempt[pattern]
	return !exempt
}

func (r actReport) missing() []string {
	var out []string
	for pattern := range r.auditable {
		if r.auditable[pattern] && !r.records[pattern] {
			out = append(out, pattern)
		}
	}
	sort.Strings(out)
	return out
}

func (r actReport) unexpected() []string {
	var out []string
	for pattern := range r.auditable {
		if !r.auditable[pattern] && r.records[pattern] {
			out = append(out, pattern)
		}
	}
	sort.Strings(out)
	return out
}

func callsRecord(fn *ast.FuncDecl) bool {
	found := false
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		// s.recorder().Record and txRecorder(tx).Record share the one method name (spec §7.1).
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Record" {
			found = true
		}
		return true
	})
	return found
}

func opensTranscript(fn *ast.FuncDecl) bool {
	found := false
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isSelector(call.Fun, "transcript", "Open") {
			found = true
		}
		return true
	})
	return found
}

func TestEveryAuditableActReachesARecordCall(t *testing.T) {
	c := parseWebPackage(t)
	rep := c.actAudit(liveActRules())

	var unwired []string
	for _, pattern := range rep.missing() {
		if _, pending := actPending[pattern]; !pending {
			unwired = append(unwired, pattern+" ("+rep.handler[pattern]+")")
		}
	}
	if len(unwired) > 0 {
		t.Errorf("these auditable routes reach no Record call (spec §2.1, §7.5):\n  %s\n"+
			"Call the recorder after the act, or add a justified entry to actExemptRoutes.",
			strings.Join(unwired, "\n  "))
	}
}

// The map was the gate's own scaffolding, and an empty one is what retires it (#1835).

func TestTheActPendingSetIsEmpty(t *testing.T) {
	if len(actPending) == 0 {
		return
	}
	var left []string
	for pattern, owner := range actPending {
		left = append(left, pattern+" ("+owner+")")
	}
	t.Errorf("actPending still names %d route(s):\n  %s\n"+
		"Every auditable route is wired. Wire a new one in its own PR rather than parking it here.",
		len(left), strings.Join(uniqSorted(left), "\n  "))
}

func TestNoExemptRouteReachesARecordCall(t *testing.T) {
	c := parseWebPackage(t)
	rep := c.actAudit(liveActRules())
	if bad := rep.unexpected(); len(bad) > 0 {
		t.Errorf("these exempt routes reach a Record call (spec §2.3):\n  %s\n"+
			"Either the act is auditable and its exemption must go, or the call is in the wrong place.",
			strings.Join(bad, "\n  "))
	}
}

func TestActExemptionsAndPendingEntriesAreLive(t *testing.T) {
	c := parseWebPackage(t)
	rep := c.actAudit(liveActRules())

	for pattern := range actExemptRoutes {
		if _, ok := c.postHandlers[pattern]; !ok {
			t.Errorf("actExemptRoutes names %q, which is no longer a POST route; delete the entry", pattern)
		}
	}
	for pattern := range actAuditableNonPOST {
		if _, ok := c.routes[pattern]; !ok {
			t.Errorf("actAuditableNonPOST names %q, which is no longer a route; delete the entry", pattern)
		}
	}
	for pattern, owner := range actPending {
		if _, ok := c.routes[pattern]; !ok {
			t.Errorf("actPending names %q, which is no longer a route; delete the entry", pattern)
			continue
		}
		if !rep.auditable[pattern] {
			t.Errorf("actPending names %q, which the gate reads as exempt; delete the entry", pattern)
			continue
		}
		if rep.records[pattern] {
			t.Errorf("actPending still names %q, which now reaches a Record call; %s delete the entry",
				pattern, owner)
		}
	}
}

// The stop set hides a call from both directions, so a Record on a refusal path would be
// invisible to the exempt-route rule too. Once the route's own Record lands, nothing else can
// see the stray one, and every refused form writes a false Act (spec §7.3).

func TestNoAnswerHelperReachesARecordCall(t *testing.T) {
	c := parseWebPackage(t)
	var bad []string
	for name := range actStop() {
		if _, ok := c.decl(name); !ok {
			continue
		}
		c.reach(name, map[string]bool{}, func(short string, fn *ast.FuncDecl) {
			if callsRecord(fn) {
				bad = append(bad, name+" → "+short)
			}
		})
	}
	if len(bad) > 0 {
		t.Errorf("these answer helpers reach a Record call (spec §7.3):\n  %s\n"+
			"A body answer and a back helper are refusal paths. Record the act in the handler.",
			strings.Join(uniqSorted(bad), "\n  "))
	}
}

// The gate reads one selector name, so §7.1's free-token premise is what makes it sound. Nothing
// else keeps that token free, and a foreign Record would grant a silent pass to every route
// reaching it while failing the exempt-route rule with a message naming the wrong cause.

func TestTheRecordTokenStaysFreeInThisPackage(t *testing.T) {
	c := parseWebPackage(t)
	var foreign []string
	visit := func(owner string, fn *ast.FuncDecl) {
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Record" || fromRecorder(sel.X) {
				return true
			}
			foreign = append(foreign, owner)
			return true
		})
	}
	for name, fn := range c.methods {
		visit("s."+name, fn)
	}
	for name, fn := range c.funcs {
		visit(name, fn)
	}
	if len(foreign) > 0 {
		t.Errorf("these hold a Record call on no recorder (spec §7.1):\n  %s\n"+
			"The gate keys on the bare method name. Rename the other Record, or widen fromRecorder\n"+
			"deliberately and say in the PR what the gate now reads.",
			strings.Join(uniqSorted(foreign), "\n  "))
	}
}

func fromRecorder(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	if isSelector(call.Fun, "s", "recorder") {
		return true
	}
	id, ok := call.Fun.(*ast.Ident)
	return ok && id.Name == "txRecorder"
}

func TestTheHarvestReadsEveryMethodAndNamesTheThreeNonPOSTActs(t *testing.T) {
	c := parseWebPackage(t)
	// A count would go red on any PR that adds a route, which is no part of this property.
	if len(c.routes) <= len(c.postHandlers) {
		t.Errorf("the harvest read %d routes and %d POST routes; it still stops at POST (spec §7.2)",
			len(c.routes), len(c.postHandlers))
	}
	for _, pattern := range []string{
		"GET /profile/sso/{slug}/link/callback", "GET /run/{id}/raw", "GET /runs/{id}/raw",
	} {
		if _, ok := c.routes[pattern]; !ok {
			t.Errorf("the harvest missed %q, one of the three auditable non-POST acts", pattern)
		}
	}
	// Without apiBearer in gateMethods every /api/v1 route resolves to the wrapper (spec §7.2).
	if got := c.routes["GET /api/v1/inventory"]; got != "apiInventory" {
		t.Errorf("GET /api/v1/inventory resolved to %q, want apiInventory", got)
	}
}

func TestTheLimbFourSinkKeysOnTranscriptOpen(t *testing.T) {
	c := parseWebPackage(t)
	rep := c.actAudit(liveActRules())
	var reached []string
	for pattern, sink := range rep.sink {
		if sink {
			reached = append(reached, pattern)
		}
	}
	sort.Strings(reached)
	want := []string{"GET /run/{id}/raw", "GET /runs/{id}/raw"}
	if strings.Join(reached, ", ") != strings.Join(want, ", ") {
		t.Errorf("transcript.Open is reached by %v, want %v; §7.4's sink rule is exact only while that holds",
			reached, want)
	}
}

const actRecordCall = "s.recorder().Record(r.Context(), actingAccount(acct), act.Thing{})"

func actFixture(wired, exempt string) string {
	return `package main

func (s *server) mount(mux *http.ServeMux) {
	mux.HandleFunc("POST /wired", s.requireAdmin(s.wired))
	mux.HandleFunc("POST /exempt", s.requireAdmin(s.exempt))
	mux.HandleFunc("GET /raw", s.requireAdmin(s.raw))
	mux.HandleFunc("GET /plain", s.requireLogin(s.plain))
	mux.HandleFunc("POST /refused", s.requireAdmin(s.refused))
}

func (s *server) wired(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.mutate()
	` + wired + `
}

func (s *server) exempt(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.mutate()
	` + exempt + `
}

func (s *server) refused(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.mutate()
	s.backToScope(w, r)
}

func (s *server) backToScope(w http.ResponseWriter, r *http.Request) {
	s.recorder().Record(r.Context(), actingAccount(db.Account{}), act.Thing{})
}

func (s *server) raw(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.fill()
}

func (s *server) fill() error {
	_, err := transcript.Open(s.transcriptKey, nil)
	return err
}

func (s *server) plain(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.render(w, r, "plain", nil)
}

func (s *server) mutate() {}
`
}

func fixturePkg(t *testing.T, src string) *contractPkg {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse the fixture: %v", err)
	}
	return newContractPkg(fset, []*ast.File{f})
}

func TestTheGateCatchesARemovedAndAnAddedRecordCall(t *testing.T) {
	rules := actRules{
		exempt:           map[string]string{"POST /exempt": "the fixture's exempt route"},
		auditableNonPOST: map[string]string{},
	}

	base := fixturePkg(t, actFixture(actRecordCall, "")).actAudit(rules)
	if got := base.missing(); strings.Join(got, ", ") != "GET /raw, POST /refused" {
		t.Errorf("the wired fixture reports missing %v, want GET /raw and POST /refused", got)
	}
	if got := base.unexpected(); len(got) != 0 {
		t.Errorf("the wired fixture reports unexpected %v, want none", got)
	}

	removed := fixturePkg(t, actFixture("", "")).actAudit(rules)
	if !contains(removed.missing(), "POST /wired") {
		t.Errorf("removing the Record call left POST /wired passing; the gate proves nothing")
	}

	added := fixturePkg(t, actFixture(actRecordCall, actRecordCall)).actAudit(rules)
	if !contains(added.unexpected(), "POST /exempt") {
		t.Errorf("adding a Record call to an exempt route left the gate passing")
	}

	// The sink fails closed: GET /raw is on no allowlist and still reads as auditable (§7.4).
	if !base.auditable["GET /raw"] {
		t.Errorf("a GET reaching transcript.Open read as exempt; limb 4's sink rule fails open")
	}
	if base.auditable["GET /plain"] {
		t.Errorf("a plain GET read as auditable; non-POST must default exempt (spec §7.5)")
	}
}

func TestTheStopSetIsPartOfTheRule(t *testing.T) {
	c := fixturePkg(t, actFixture(actRecordCall, ""))
	rules := actRules{
		exempt:           map[string]string{"POST /exempt": "the fixture's exempt route"},
		auditableNonPOST: map[string]string{},
	}

	// s.backToScope is a refusal path, so a Record inside it also breaks §7.5 while CI says PASS.
	if !contains(c.actAudit(rules).missing(), "POST /refused") {
		t.Errorf("a Record reached only through s.backToScope passed POST /refused (spec §7.3)")
	}

	reached := func(handler string, stop map[string]bool) bool {
		hit := false
		c.reach("s."+handler, stop, func(_ string, fn *ast.FuncDecl) {
			if callsRecord(fn) {
				hit = true
			}
		})
		return hit
	}
	if !reached("refused", map[string]bool{}) {
		t.Fatalf("the fixture's refusal path reaches no Record at all; it proves nothing")
	}
	if reached("refused", actStop()) {
		t.Errorf("actStop lets the walk into a back helper; every handler sharing one passes falsely")
	}
	if !reached("wired", actStop()) {
		t.Errorf("actStop swallowed a real call shape; the stop set is too wide")
	}
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
