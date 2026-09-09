package main

// Throwaway prototype for #1791. Lands nothing. It measures whether an AST conformance test can
// gate "every auditable act reaches the recorder", using cmd/web/adr0130_contract_test.go as the
// model Settled #11 names.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 1 · The widened harvest
// ---------------------------------------------------------------------------

// adr0130's gateMethods misses apiBearer, which only matters once the harvest widens past POST.
var actGateMethods = map[string]bool{
	"requireLogin": true, "requireAdmin": true, "requireSettingsAdmin": true,
	"requireAPIAuth": true, "redirectTo": true, "apiBearer": true,
}

func actHandlerName(e ast.Expr) string {
	var found string
	ast.Inspect(e, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "s" || actGateMethods[sel.Sel.Name] || found != "" {
			return true
		}
		found = sel.Sel.Name
		return true
	})
	return found
}

// allRoutes harvests every mux.HandleFunc, not only the POST ones.
func allRoutes(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read cmd/web: %v", err)
	}
	fset := token.NewFileSet()
	out := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 || !isSelector(call.Fun, "mux", "HandleFunc") {
				return true
			}
			pattern, ok := stringLit(call.Args[0])
			if !ok {
				return true
			}
			if h := actHandlerName(call.Args[1]); h != "" {
				out[pattern] = h
			}
			return true
		})
	}
	if len(out) == 0 {
		t.Fatalf("found no routes; the route reader is broken, not the tree")
	}
	return out
}

// ---------------------------------------------------------------------------
// 2 · #1789's ruling, transcribed
// ---------------------------------------------------------------------------

// The 20 exempt POST routes. #1789 counts 23 exemptions: 20 POST plus 3 non-route.
var actExemptPOST = map[string]string{
	"POST /login":                            "login family",
	"POST /login/totp":                       "login family; a recovery-code spend is a use, not a grant",
	"POST /logout":                           "login family",
	"POST /signout":                          "login family; same handler as /logout",
	"POST /forgot":                           "unauthenticated, no principal, attacker-supplied username",
	"POST /profile/session/revoke":           "login family",
	"POST /profile/sessions/revoke":          "login family",
	"POST /profile/sessions/revoke-others":   "login family",
	"POST /settings/sessions/revoke":         "login family; cross-principal but the grant is untouched",
	"POST /settings/sessions/revoke-account": "login family; same",
	"POST /onboarding":                       "wizard step navigation; writes nothing",
	"POST /seeds/preview":                    "read-only withdrawal receipt",
	"POST /exclusions/preview":               "read-only preview",
	"POST /reports/schedule/run":             "inserts a Delivery at generated and pushes nothing (tripwire)",
	"POST /messages/read":                    "per-account read state",
	"POST /messages/read-all":                "per-account read state",
	"POST /messages/unread":                  "per-account read state",
	"POST /account/totp/enable":              "stages a secret with totp_enabled false; the confirm is the act",
	"POST /settings/backup":                  "transcript is on backupExcluded and secrets are redacted",
	"POST /settings/restore/preflight":       "stages the archive in process memory only",
}

// The auditable non-POST routes. #1789 finds exactly these.
var actAuditableGET = map[string]string{
	"GET /profile/sso/{slug}/link/callback": "limb 2; InsertSSOIdentity binds an identity (ADR-0113)",
	"GET /run/{id}/raw":                     "limb 4; a Transcript disclosure (#1794)",
	"GET /runs/{id}/raw":                    "limb 4; the same handler",
}

// ---------------------------------------------------------------------------
// 3 · Does the harvest reproduce #1789's count?
// ---------------------------------------------------------------------------

func auditableRoutes(t *testing.T) map[string]string {
	routes := allRoutes(t)
	out := map[string]string{}
	for pattern, handler := range routes {
		switch {
		case strings.HasPrefix(pattern, "POST "):
			if _, exempt := actExemptPOST[pattern]; exempt {
				continue
			}
			out[pattern] = handler
		default:
			if _, audited := actAuditableGET[pattern]; audited {
				out[pattern] = handler
			}
		}
	}
	return out
}

func TestProtoHarvestMatches1789(t *testing.T) {
	routes := allRoutes(t)
	var posts, gets int
	for pattern := range routes {
		if strings.HasPrefix(pattern, "POST ") {
			posts++
		} else {
			gets++
		}
	}
	t.Logf("harvest: %d routes total — %d POST, %d non-POST", len(routes), posts, gets)

	// Every exemption must name a live route, or it is stale.
	for pattern := range actExemptPOST {
		if _, ok := routes[pattern]; !ok {
			t.Errorf("actExemptPOST names %q, which is no longer a route", pattern)
		}
	}
	for pattern := range actAuditableGET {
		if _, ok := routes[pattern]; !ok {
			t.Errorf("actAuditableGET names %q, which is no longer a route", pattern)
		}
	}

	// #1789's headline is 63 acts / 59 classes. By route pattern the harvest measures 64, behind
	// 61 distinct handlers. The gap is arithmetic: #1789 folds the two raw routes into one act to
	// reach 63, then subtracts four route pairs to reach 59 — subtracting the raw pair twice.
	aud := auditableRoutes(t)
	if len(aud) != 64 {
		t.Errorf("harvest yields %d auditable route patterns, measured 64", len(aud))
	}
	handlers := map[string]bool{}
	for _, h := range aud {
		handlers[h] = true
	}
	if len(handlers) != 61 {
		t.Errorf("harvest yields %d distinct handlers, measured 61", len(handlers))
	}
	t.Logf("auditable: %d route patterns, %d distinct handlers", len(aud), len(handlers))
	t.Logf("#1789 reports 63 acts / 59 classes; by pattern it is 64, by class 60")
}

// TestProtoPOSTOnlyHarvestDropsTheGETActs measures what the model test loses as-is.
func TestProtoPOSTOnlyHarvestDropsTheGETActs(t *testing.T) {
	c := parseWebPackage(t)
	for pattern := range actAuditableGET {
		if _, ok := c.postHandlers[pattern]; ok {
			t.Errorf("unexpected: the POST-only harvest sees %q", pattern)
		}
	}
	t.Logf("the adr0130 harvest sees %d routes and drops all %d auditable non-POST acts",
		len(c.postHandlers), len(actAuditableGET))
}

// ---------------------------------------------------------------------------
// 4 · The walk — can it see a recorder call?
// ---------------------------------------------------------------------------

// recorderCallsIn finds every `<anything>.Record(...)` in a live (non-devMode) body. It is a leaf
// detector, deliberately not a callee: the receiver shape is irrelevant, which is what lets one
// rule accept both #1788's pool-bound and tx-bound receivers.
func recorderCallsIn(fn *ast.FuncDecl) []string {
	var out []string
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Record" {
			return true
		}
		out = append(out, recvString(sel.X)+".Record")
		return true
	})
	return out
}

func recvString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return recvString(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		return recvString(v.Fun) + "()"
	}
	return "?"
}

// actStop halts the walk at the error and render leaves, so a shared helper does not bless a caller.
func actStop() map[string]bool {
	stop := map[string]bool{}
	for k := range errorPageMethods {
		stop[k] = true
	}
	if actStopAtAnswerHelpers {
		for k := range backHelpers {
			stop[k] = true
		}
		for k := range bodyAnswers {
			stop[k] = true
		}
	}
	return stop
}

// An answer helper is a redirect or a render. A recorder call inside one records a refusal too, so
// the walk halting there is the mitigation, not a blind spot.
var actStopAtAnswerHelpers = true

func (c *contractPkg) reachesRecorder(handler string) (bool, []string) {
	var hits []string
	c.reach("s."+handler, actStop(), func(name string, fn *ast.FuncDecl) {
		for _, r := range recorderCallsIn(fn) {
			hits = append(hits, name+" → "+r)
		}
	})
	return len(hits) > 0, uniqSorted(hits)
}

func TestProtoEveryAuditableHandlerReachesTheRecorder(t *testing.T) {
	c := parseWebPackage(t)
	aud := auditableRoutes(t)

	var missing, found []string
	for pattern, handler := range aud {
		ok, hits := c.reachesRecorder(handler)
		if ok {
			found = append(found, pattern+" ("+handler+"): "+strings.Join(hits, ", "))
			continue
		}
		missing = append(missing, pattern+" ("+handler+")")
	}
	sort.Strings(missing)
	sort.Strings(found)
	t.Logf("REACHES the recorder: %d", len(found))
	for _, f := range found {
		t.Logf("  %s", f)
	}
	t.Logf("MISSING a recorder: %d", len(missing))
	for _, m := range missing {
		t.Logf("  %s", m)
	}
}

// TestProtoNoExemptHandlerReachesTheRecorder is the other half of the gate: an exempt route that
// grows a recorder call is a ruling silently reversed.
func TestProtoNoExemptHandlerReachesTheRecorder(t *testing.T) {
	c := parseWebPackage(t)
	routes := allRoutes(t)
	var bad []string
	for pattern := range actExemptPOST {
		handler, ok := routes[pattern]
		if !ok {
			continue
		}
		if reached, hits := c.reachesRecorder(handler); reached {
			bad = append(bad, pattern+" ("+handler+"): "+strings.Join(hits, ", "))
		}
	}
	sort.Strings(bad)
	for _, b := range bad {
		t.Logf("exempt but records: %s", b)
	}
	t.Logf("exempt routes that reach a recorder: %d", len(bad))
}

// ---------------------------------------------------------------------------
// 5 · The sink-keyed alternative — key the gate on the store write, not on the route
// ---------------------------------------------------------------------------

var queryNameRE = regexp.MustCompile(`^--\s*name:\s*(\w+)\s*:(\w+)`)

// mutatingQueries reads db/queries/*.sql and returns the sqlc method names whose SQL mutates.
func mutatingQueries(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join("..", "..", "db", "queries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	out := map[string]string{}
	total := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			t.Fatalf("read %s: %v", e.Name(), rerr)
		}
		lines := strings.Split(string(b), "\n")
		for i, line := range lines {
			m := queryNameRE.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil {
				continue
			}
			total++
			// Scan the body to the next name line, so a mutation inside a CTE still counts.
			var body []string
			for j := i + 1; j < len(lines); j++ {
				if queryNameRE.MatchString(strings.TrimSpace(lines[j])) {
					break
				}
				body = append(body, lines[j])
			}
			if verb := mutatingVerb(strings.Join(body, "\n")); verb != "" {
				out[m[1]] = verb
			}
		}
	}
	t.Logf("db/queries: %d named queries, %d mutating", total, len(out))
	return out
}

var verbRE = regexp.MustCompile(`(?im)^\s*\(?\s*(INSERT|UPDATE|DELETE|TRUNCATE)\b`)

func mutatingVerb(body string) string {
	// A line comment can carry the word INSERT; strip comments before matching.
	var clean []string
	for _, line := range strings.Split(body, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		clean = append(clean, line)
	}
	if m := verbRE.FindStringSubmatch(strings.Join(clean, "\n")); m != nil {
		return strings.ToUpper(m[1])
	}
	return ""
}

// storeCallsIn finds every `s.<field>.<Method>(...)` in a live body.
func storeCallsIn(fn *ast.FuncDecl) []string {
	var out []string
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		inner, ok := sel.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := inner.X.(*ast.Ident); !ok || id.Name != "s" {
			return true
		}
		out = append(out, sel.Sel.Name)
		return true
	})
	return out
}

func TestProtoSinkKeyedGate(t *testing.T) {
	c := parseWebPackage(t)
	mut := mutatingQueries(t)
	routes := allRoutes(t)

	writers := map[string]bool{} // routes that reach a mutating query
	perRoute := map[string][]string{}
	for pattern, handler := range routes {
		var hits []string
		c.reach("s."+handler, actStop(), func(name string, fn *ast.FuncDecl) {
			for _, m := range storeCallsIn(fn) {
				if v, ok := mut[m]; ok {
					hits = append(hits, m+" ("+v+")")
				}
			}
		})
		if len(hits) > 0 {
			writers[pattern] = true
			perRoute[pattern] = uniqSorted(hits)
		}
	}

	aud := auditableRoutes(t)

	var writerNotAuditable, auditableNotWriter []string
	for pattern := range writers {
		if _, ok := aud[pattern]; !ok {
			writerNotAuditable = append(writerNotAuditable, pattern+": "+strings.Join(perRoute[pattern], ", "))
		}
	}
	for pattern := range aud {
		if !writers[pattern] {
			auditableNotWriter = append(auditableNotWriter, pattern)
		}
	}
	sort.Strings(writerNotAuditable)
	sort.Strings(auditableNotWriter)

	t.Logf("routes reaching a mutating query: %d", len(writers))
	t.Logf("auditable per #1789: %d", len(aud))
	t.Logf("")
	t.Logf("MUTATES BUT EXEMPT (%d) — a sink-keyed gate would demand a row here:", len(writerNotAuditable))
	for _, s := range writerNotAuditable {
		t.Logf("  %s", s)
	}
	t.Logf("")
	t.Logf("AUDITABLE BUT NO LOCAL MUTATION (%d) — a sink-keyed gate cannot see these:", len(auditableNotWriter))
	for _, s := range auditableNotWriter {
		t.Logf("  %s", s)
	}
}

// TestProtoTranscriptOpenIsASink measures limb 4's sink specifically: transcript.Open.
func TestProtoTranscriptOpenIsASink(t *testing.T) {
	c := parseWebPackage(t)
	routes := allRoutes(t)
	var reach []string
	for pattern, handler := range routes {
		var hits []string
		c.reach("s."+handler, actStop(), func(name string, fn *ast.FuncDecl) {
			inspectLive(fn, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || !isSelector(call.Fun, "transcript", "Open") {
					return true
				}
				hits = append(hits, name)
				return true
			})
		})
		if len(hits) > 0 {
			reach = append(reach, pattern+" ("+handler+") → "+strings.Join(uniqSorted(hits), ", "))
		}
	}
	sort.Strings(reach)
	t.Logf("routes reaching transcript.Open: %d", len(reach))
	for _, r := range reach {
		t.Logf("  %s", r)
	}
}
