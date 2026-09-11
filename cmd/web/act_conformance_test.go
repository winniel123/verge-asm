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

// Every auditable route this map names has no Record call yet, and the wiring ticket that owes
// it is the value. POST defaults auditable and fails closed (spec §7.5), so this gate lands
// before the bulk wiring and would otherwise fail on all 62. Decided by map #1826, not by the
// SPEC. Each wiring ticket deletes its own rows. #1835 is the last one, so #1835 leaves this
// map empty and asserts that it is.

var actPending = map[string]string{
	"POST /seeds/custody":              "#1831 seed.custody.moved",
	"POST /seeds/zone":                 "#1831 zone.declared",
	"POST /exclusions":                 "#1831 exclusion.declared",
	"POST /exclusions/delete":          "#1831 exclusion.lifted",
	"POST /settings/cold":              "#1831 cold.moved",
	"POST /settings/probers":           "#1831 vantage.declared",
	"POST /settings/vantages/resolver": "#1831 vantage.resolver.set",
	"POST /reports/schedule/new":       "#1831 schedule.declared",
	"POST /reports/schedule/{id}/edit": "#1831 schedule.edited",
	"POST /reports/schedule/delete":    "#1831 schedule.withdrawn",
	"POST /annotations":                "#1831 annotation.declared",
	"POST /annotations/withdraw":       "#1831 annotation.withdrawn",
	"POST /proposals/confirm":          "#1831 proposal.confirmed",
	"POST /proposals/decline":          "#1831 proposal.declined",
	"POST /proposals/undo-decline":     "#1831 proposal.decline.undone",
	"POST /verge-core/frequency":       "#1831 frequency.moved",
	"POST /sources/toggle":             "#1831 source.moved",
	"POST /settings/sources":           "#1831 source.moved",

	"POST /seeds/zone/interval":              "#1832 zone.cadence.set",
	"POST /seeds/dns/interval":               "#1832 dns.cadence.set",
	"POST /settings/retention":               "#1832 transcript.currency.set",
	"POST /coverage/retention":               "#1832 observation.currency.set and dispatch.cadence.set, two rows (§2.2)",
	"POST /settings/address-cap":             "#1832 address.cap.set",
	"POST /settings/updates/check":           "#1832 update.check.moved",
	"POST /settings/channels":                "#1832 channel.declared",
	"POST /settings/channels/update":         "#1832 channel.updated",
	"POST /settings/channels/delete":         "#1832 channel.withdrawn",
	"POST /settings/integrations/install":    "#1832 integration.installed",
	"POST /settings/integrations/remove":     "#1832 integration.removed",
	"POST /settings/integrations/disconnect": "#1832 integration.removed",
	"POST /settings/integrations/channel":    "#1832 integration.channel.bound",

	"POST /setup":                           "#1833 setup.completed",
	"POST /reset":                           "#1833 password.reset",
	"POST /invite":                          "#1833 invite.accepted",
	"POST /profile/password":                "#1833 password.changed",
	"POST /profile/tokens":                  "#1833 token.minted",
	"POST /profile/tokens/revoke":           "#1833 token.revoked",
	"POST /profile/sso/unlink":              "#1833 sso.unlinked",
	"POST /accounts":                        "#1833 account.created",
	"POST /account/totp/confirm":            "#1833 totp.enrolled",
	"POST /settings/accounts":               "#1833 invite.minted",
	"POST /settings/accounts/role":          "#1833 account.role.moved",
	"POST /settings/accounts/reenroll":      "#1833 totp.stripped",
	"POST /settings/accounts/remove":        "#1833 account.removed",
	"POST /settings/api":                    "#1833 api.access.moved",
	"POST /settings/sso":                    "#1833 sso.provider.declared",
	"POST /settings/sso/update":             "#1833 sso.provider.updated",
	"POST /settings/sso/secret":             "#1833 sso.provider.secret.set",
	"POST /settings/sso/delete":             "#1833 sso.provider.withdrawn",
	"POST /settings/sso/identity/remove":    "#1833 sso.binding.removed",
	"GET /profile/sso/{slug}/link/callback": "#1833 sso.binding.created",

	"POST /settings/restore": "#1834 restore.applied, the one tx-bound recorder (§7.6 ruling 4)",

	"POST /onboarding/finish":          "#1835 onboarding.finished",
	"POST /proposals":                  "#1835 proposal.queried",
	"POST /proposals/search":           "#1835 proposal.queried",
	"POST /scans/trigger":              "#1835 scan.triggered",
	"POST /scans/stop":                 "#1835 scan.stopped",
	"POST /scans/terminate":            "#1835 scan.terminated",
	"POST /settings/channels/test":     "#1835 channel.tested",
	"POST /settings/integrations/test": "#1835 integration.tested",
	"GET /run/{id}/raw":                "#1835 transcript.disclosed",
	"GET /runs/{id}/raw":               "#1835 transcript.disclosed",
}

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

func TestTheHarvestReadsEveryMethodAndNamesTheThreeNonPOSTActs(t *testing.T) {
	c := parseWebPackage(t)
	// 144 registrations less the four that carry no s.<handler> name (spec §7.2).
	if len(c.routes) != 140 {
		t.Errorf("the harvest read %d routes, want 140; a POST-only harvest sees none of the three non-POST acts", len(c.routes))
	}
	if len(c.postHandlers) != 81 {
		t.Errorf("the harvest read %d POST routes, want 81", len(c.postHandlers))
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
