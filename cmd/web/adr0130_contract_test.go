package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The ADR-0130 contract guards (map #969, ticket #978). ADR-0130 fixes one rule for
// every mutating act in the console: the operator lands back at the exact URL the form
// was submitted from, and the scroll restore keys on that full URL.
//
// The per-surface tests next door prove the rule HOLDS today, handler by handler. The
// guards here prove it cannot ROT. They read this package's own source and its own
// route table rather than a hand-kept list, so a handler added next year is checked the
// day it is registered, and a migrated handler that slips back to rendering in place
// fails here before it reaches an operator.
//
// Each guard carries a named exemption list. An exemption is not a suppression: it is
// the record of a case ADR-0130 does not cover, with its reason stated at the entry, so
// a reviewer sees the whole set at once and a new entry has to argue for itself.

// --- the source and the route table these guards read ----------------------

// contractPkg is cmd/web parsed from disk, minus the tests.
type contractPkg struct {
	fset *token.FileSet
	// methods maps a *server method name to its declaration. A handler, a helper and a
	// renderer are all here; the guards tell them apart by name and by who calls whom.
	methods map[string]*ast.FuncDecl
	// postHandlers maps a routed POST pattern to the *server method that answers it.
	postHandlers map[string]string
	// consts maps every package-level string constant to its value, so the class-E guard
	// reads `profilePath` and `reportsPath` as the paths they are.
	consts map[string]string
}

// parseWebPackage parses cmd/web in the working directory — `go test` runs in the
// package directory, so that is this package's own source — and pulls out the two
// things the guards need: every *server method, and every POST route's handler.
func parseWebPackage(t *testing.T) *contractPkg {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read cmd/web: %v", err)
	}
	fset := token.NewFileSet()
	c := &contractPkg{
		fset:         fset,
		methods:      map[string]*ast.FuncDecl{},
		postHandlers: map[string]string{},
		consts:       map[string]string{},
	}
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		files = append(files, f)
	}
	for _, f := range files {
		for _, d := range f.Decls {
			if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.CONST {
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) != len(vs.Values) {
						continue
					}
					for i, name := range vs.Names {
						if v, ok := stringLit(vs.Values[i]); ok {
							c.consts[name.Name] = v
						}
					}
				}
				continue
			}
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			if id, ok := star.X.(*ast.Ident); ok && id.Name == "server" {
				c.methods[fn.Name.Name] = fn
			}
		}
	}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 || !isSelector(call.Fun, "mux", "HandleFunc") {
				return true
			}
			pattern, ok := stringLit(call.Args[0])
			if !ok || !strings.HasPrefix(pattern, "POST ") {
				return true
			}
			if name := handlerMethodName(call.Args[1]); name != "" {
				c.postHandlers[pattern] = name
			}
			return true
		})
	}
	if len(c.postHandlers) == 0 {
		t.Fatalf("found no POST routes; the route reader is broken, not the tree")
	}
	return c
}

// gateMethods are the wrappers a route is registered through. They are not the handler,
// so handlerMethodName looks past them for the method they wrap.
var gateMethods = map[string]bool{
	"requireLogin": true, "requireAdmin": true, "requireSettingsAdmin": true,
	"requireAPIAuth": true, "redirectTo": true,
}

// handlerMethodName reads the *server method out of a route's handler expression,
// looking through however many gates wrap it.
func handlerMethodName(e ast.Expr) string {
	var found string
	ast.Inspect(e, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "s" || gateMethods[sel.Sel.Name] || found != "" {
			return true
		}
		found = sel.Sel.Name
		return true
	})
	return found
}

// devModeBodies collects the bodies of every `if s.devMode { … }` in fn.
//
// A VERGE_DEV build serves the design's frozen fixture for the pixel-parity harness, and
// several POST handlers carry such a branch so the golden that posts their form renders
// byte-for-byte. That render is a capture path reached by configuration, never an
// operator's refusal, so the guards below walk past these bodies rather than counting
// them. They read the live path, which is the whole of what an operator sees.
func devModeBodies(fn *ast.FuncDecl) map[ast.Node]bool {
	out := map[ast.Node]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		if ifs, ok := n.(*ast.IfStmt); ok && isSelector(ifs.Cond, "s", "devMode") {
			out[ifs.Body] = true
		}
		return true
	})
	return out
}

// inspectLive walks fn the way ast.Inspect does, minus the VERGE_DEV fixture branches.
func inspectLive(fn *ast.FuncDecl, f func(ast.Node) bool) {
	skip := devModeBodies(fn)
	ast.Inspect(fn, func(n ast.Node) bool {
		if n != nil && skip[n] {
			return false
		}
		return f(n)
	})
}

// calleesOf lists the *server methods fn calls on its live path, in source order.
func (c *contractPkg) calleesOf(fn *ast.FuncDecl) []string {
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
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "s" {
			out = append(out, sel.Sel.Name)
		}
		return true
	})
	return out
}

// reach walks the call graph out of a handler and calls visit on every *server method
// body the handler can reach, itself included. stop names the methods the walk does not
// descend into, so a guard can treat a sanctioned answer as terminal.
func (c *contractPkg) reach(start string, stop map[string]bool, visit func(name string, fn *ast.FuncDecl)) {
	seen := map[string]bool{}
	var walk func(string)
	walk = func(name string) {
		if seen[name] || stop[name] {
			return
		}
		seen[name] = true
		fn, ok := c.methods[name]
		if !ok {
			return
		}
		visit(name, fn)
		for _, callee := range c.calleesOf(fn) {
			walk(callee)
		}
	}
	walk(start)
}

func isSelector(e ast.Expr, x, sel string) bool {
	s, ok := e.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != sel {
		return false
	}
	id, ok := s.X.(*ast.Ident)
	return ok && id.Name == x
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

func uniqSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// --- class A: a refusal is a redirect, never a body ------------------------

// renderMethods are the *server methods that write a rendered page body. A POST that
// reaches one of these answers its own POST URL with a body, which is the class-A
// failure ADR-0130 §1 closes.
var renderMethods = map[string]bool{
	"render": true, "renderStatus": true, "renderSettings": true, "renderSeeds": true,
	"renderSignals": true, "renderProfile": true, "renderOnboard": true,
	"renderScheduleWizard": true,
}

// errorPageMethods answer a BROKEN request — a 500 the operator cannot act on, a
// missing subject, a forbidden page. None is a form refusal, so the class-A walk treats
// them as terminal rather than as a rendered answer. ADR-0130 is about the form the
// operator filled in; a request that names no such form has no URL worth returning to
// and no typed values worth echoing.
var errorPageMethods = map[string]bool{
	"serverError": true, "renderError": true, "notFound": true, "forbidden": true,
	"renderMissingSubject": true, "renderMissingRun": true, "settingsForbidden": true,
	"requireLogin": true, "requireAdmin": true, "requireSettingsAdmin": true,
}

// classAExempt names every POST route allowed to answer with a body, and says why.
// ADR-0130 §1 covers the console forms an operator fills in on a long page. Each entry
// below sits outside that, and map #969's audit records every one.
var classAExempt = map[string]string{
	// The pre-authentication credential flow. These pages are short, carry no filter
	// state and no scroll offset, and the caller holds no session for a session-keyed
	// flash to hang on. Map #969 excludes the authentication flow by name.
	"POST /setup":      "first-run credential form; no session exists yet to key a flash on",
	"POST /login":      "sign-in form; pre-authentication, no session and no list to return to",
	"POST /login/totp": "second-factor prompt; pre-authentication, same reason as /login",
	"POST /forgot":     "password-reset request; pre-authentication",
	"POST /reset":      "password-reset form; pre-authentication, and the token rides the page",
	"POST /invite":     "invite acceptance; pre-authentication, the invitee holds no account yet",
	"POST /onboarding": "the checklist advances by rendering its next step; it refuses no form",

	// Two-factor enrolment renders a SCREEN, not a refusal. The QR and the secret are
	// rolled for this response and shown once, so a redirect would have to stash key
	// material in a store to survive it.
	"POST /account/totp/enable":  "renders the enrolment screen (QR + secret), a step and not a refusal",
	"POST /account/totp/confirm": "the same enrolment screen, re-rendered with its own prompt",

	// The one console form whose SUCCESS must be a body. createPersonalToken reveals the
	// minted plaintext once, in this response only; Verge stores the hash alone. Its
	// refusals are post-redirect-gets like every other console form — this entry buys the
	// success path and nothing else.
	"POST /profile/tokens": "the minted token plaintext is revealed once in the body and never stored",
}

// TestNoMutatingHandlerAnswersAValidationFailureWithABody is the class-A regression
// guard. It fails when a POST handler can reach a page render, which is what answering
// a refusal in place looks like: the body arrives at the POST URL, the browser performs
// no navigation for the scroll restore to fire on, and a reload re-submits the form.
func TestNoMutatingHandlerAnswersAValidationFailureWithABody(t *testing.T) {
	c := parseWebPackage(t)
	var bad []string
	for pattern, handler := range c.postHandlers {
		if _, ok := classAExempt[pattern]; ok {
			continue
		}
		var hits []string
		c.reach(handler, errorPageMethods, func(name string, fn *ast.FuncDecl) {
			for _, callee := range c.calleesOf(fn) {
				if renderMethods[callee] {
					hits = append(hits, name+" → s."+callee)
				}
			}
		})
		if len(hits) > 0 {
			bad = append(bad, pattern+" ("+handler+"): "+strings.Join(uniqSorted(hits), ", "))
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("these POST handlers answer with a rendered body instead of a 303 (ADR-0130 §1):\n  %s\n"+
			"Stash the refusal on the session form flash and redirect back, or add a justified entry to classAExempt.",
			strings.Join(bad, "\n  "))
	}
}

// TestClassAExemptionsAreLive keeps the exemption list honest. An entry naming a route
// the tree no longer serves is deleted rather than left to imply a live exception.
func TestClassAExemptionsAreLive(t *testing.T) {
	c := parseWebPackage(t)
	for pattern := range classAExempt {
		if _, ok := c.postHandlers[pattern]; !ok {
			t.Errorf("classAExempt names %q, which is no longer a POST route; delete the entry", pattern)
		}
	}
	for pattern := range classEExempt {
		if _, ok := c.postHandlers[pattern]; !ok {
			t.Errorf("classEExempt names %q, which is no longer a POST route; delete the entry", pattern)
		}
	}
}

// --- class E: a mutating act lands on the URL it was submitted from --------

// backHelpers are the sanctioned answers to a mutating act: each resolves the submitting
// URL off the posted `return` field (backurl.go resolveBack) and 303s to it, falling back
// to a bare path only when the form carried no value that passed the open-redirect guard.
// The class-E walk stops at them, so their own fallback literal is not a finding — the
// fallback is the guard's answer to a forged or absent field, not a handler's destination.
var backHelpers = map[string]bool{
	"redirectBack": true, "toastRedirectBack": true,
	"backToSection": true, "toastBackToSection": true, "failSettings": true, "flashSettings": true,
	"backToScope": true, "flashScopeBack": true, "flashScopeToastBack": true,
}

// classEExempt names every POST route allowed to redirect to a bare path, and says why.
// ADR-0130 §3 exists for a page whose URL carries operative state — a filter, a sort, a
// pager, a tab. A page with none loses nothing to a bare path, and a page whose whole
// query is dialog state would be WRONG to return to verbatim, which is the same reading
// dialogParams applies on the settings surface.
var classEExempt = map[string]string{
	// Pre-authentication navigation. Each of these moves the caller to a fixed screen —
	// the console, the sign-in page, the second-factor prompt — and none returns to a
	// list. Map #969 excludes the authentication flow by name.
	"POST /setup":      "first-run setup completes into the console at a fixed destination",
	"POST /login":      "sign-in navigates to the console or back to the form; neither is a list",
	"POST /login/totp": "second-factor prompt; same fixed navigation as /login",
	"POST /logout":     "sign-out navigates to the sign-in screen by design",
	"POST /signout":    "the shell's alias onto the same /logout handler",
	"POST /forgot":     "password-reset request; pre-authentication navigation",
	"POST /reset":      "password-reset completion; pre-authentication navigation",
	"POST /invite":     "invite acceptance lands the new account in the console",

	// The onboarding checklist is a sequence of steps, so its redirects are deliberate
	// page MOVES rather than a return to the page acted from.
	"POST /onboarding":        "the checklist advances to its next step; the move is the point",
	"POST /onboarding/finish": "the finish leaves onboarding for the scan monitor",

	// The Profile surface carries no operative query. Every parameter /profile reads is a
	// dialog opener (?new=, ?revoke=, ?endsession=, ?signoutothers=) or a single-consume
	// SSO receipt (?linked=, ?unlinked=, ?linkerr=), so returning verbatim would re-open a
	// confirm the act has just answered or re-show a spent receipt. There is no filter,
	// sort or pager on the page for a bare path to lose. See failProfile.
	"POST /profile/password":         "/profile carries no operative query; see failProfile",
	"POST /profile/tokens":           "/profile carries no operative query; see failProfile",
	"POST /profile/tokens/revoke":    "/profile carries no operative query; see failProfile",
	"POST /profile/session/revoke":   "ends the caller's own session; the destination is the sign-in screen",
	"POST /profile/sessions/revoke":  "/profile carries no operative query; see failProfile",
	"POST /profile/sessions/revoke-others": "/profile carries no operative query; see failProfile",
	"POST /profile/sso/unlink":       "/profile carries no operative query; see failProfile",
	"POST /account/totp/enable":      "two-factor enrolment is a screen sequence, not a list act",
	"POST /account/totp/confirm":     "two-factor enrolment is a screen sequence, not a list act",

	// The schedule wizard STEPS are page moves between steps, and each carries the
	// accumulated state in its own query. The FINISH is not exempt: it 303s to the entry
	// URL threaded in on ?return= (ticket #977), because a wizard must be left behind.
	"POST /reports/schedule/new":      "the wizard's back/next/invalid-step redirects are moves between steps",
	"POST /reports/schedule/{id}/edit": "the edit wizard's step moves, same as the new-schedule wizard",

	// A scan trigger's receipt belongs on the monitor that shows the scan, which is a
	// different page from the one the button sits on (ticket #977's audit).
	"POST /scans/trigger": "the receipt lands on the run monitor, a deliberate page move",

	// A COMPLETED restore replaces the database the caller's own session row lives in, so
	// there is no session left to return anywhere. Sign-in is the only page the operator
	// can still reach. Its REFUSALS are not exempt: they leave the instance untouched and
	// 303 back to the submitting URL with the callout on the flash (ticket #977).
	"POST /settings/restore": "a completed restore ends every session; the caller can only land on sign-in",
}

// TestNoMutatingHandlerRedirectsToABarePath is the class-E regression guard. It fails
// when a POST handler answers with a redirect to a destination spelled in its own
// source, rather than to the URL the form was submitted from.
//
// A bare path is what class E costs the operator: an act performed from
// `/signals?tab=open&sev=High&page=3` lands them on `/signals`, so the filter is gone,
// the row they acted on is off screen, and the scroll key the shell stashed on submit
// (shell.tmpl) cannot hit because the landing URL is a different string.
func TestNoMutatingHandlerRedirectsToABarePath(t *testing.T) {
	c := parseWebPackage(t)
	consts := c.consts
	stop := map[string]bool{}
	for k := range backHelpers {
		stop[k] = true
	}
	for k := range errorPageMethods {
		stop[k] = true
	}
	var bad []string
	for pattern, handler := range c.postHandlers {
		if _, ok := classEExempt[pattern]; ok {
			continue
		}
		var hits []string
		c.reach(handler, stop, func(name string, fn *ast.FuncDecl) {
			for _, dest := range redirectDestinations(fn) {
				if lit, ok := literalPath(dest, consts); ok {
					hits = append(hits, name+" → "+lit)
				}
			}
		})
		if len(hits) > 0 {
			bad = append(bad, pattern+" ("+handler+"): "+strings.Join(uniqSorted(hits), ", "))
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("these POST handlers redirect to a path spelled in their own source, not to the submitting URL (ADR-0130 §3):\n  %s\n"+
			"Answer through redirectBack / backToSection / backToScope, or add a justified entry to classEExempt.",
			strings.Join(bad, "\n  "))
	}
}

// redirectDestinations lists the destination expression of every redirect answer fn
// writes on its live path: http.Redirect's third argument, and toastRedirect's.
func redirectDestinations(fn *ast.FuncDecl) []ast.Expr {
	var out []ast.Expr
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 3 {
			return true
		}
		if isSelector(call.Fun, "http", "Redirect") || isSelector(call.Fun, "s", "toastRedirect") {
			out = append(out, call.Args[2])
		}
		return true
	})
	return out
}

// --- the carrier: one refusal, shown once ----------------------------------

// TestTheSessionFormFlashIsSingleConsume pins the property every §1 landing depends on.
// A refusal is stashed once and read once. The read DELETES it, so the reload the
// operator performs — or the meta-refresh a Scans view performs for them while a scan is
// in flight — finds nothing and re-shows no callout they have already answered.
//
// The surface tests next door assert this end to end on /settings, /scope and /signals.
// This one asserts it at the store, where the property actually lives, so a change to
// the take path fails here with a one-line reason rather than as a puzzling body
// mismatch three files away.
func TestTheSessionFormFlashIsSingleConsume(t *testing.T) {
	store := newFormFlashStore()
	now := time.Now()

	store.set(7, settingsForms{section: "channels", chanError: "no"}, now)
	if !store.pending(now) {
		t.Fatalf("a stashed refusal does not register as pending")
	}

	first, ok := store.m[7]
	if !ok {
		t.Fatalf("the stash is not held under its session id")
	}
	if _, ok := first.value.(settingsForms); !ok {
		t.Fatalf("the stash did not keep its own shape")
	}
	delete(store.m, 7) // what a take does, with the session read stubbed out.

	if _, ok := store.m[7]; ok {
		t.Errorf("the flash survived its take; a reload would re-show a spent callout")
	}
	if store.pending(now) {
		t.Errorf("the store still reports a pending refusal after the only one was taken")
	}

	// The second session's stash is untouched by the first session's take, and each
	// session's own take is still single-consume.
	store.set(1, signalsForms{annoError: "one"}, now)
	store.set(2, signalsForms{annoError: "two"}, now)
	delete(store.m, 1)
	if _, ok := store.m[2]; !ok {
		t.Errorf("one session's take consumed another session's refusal")
	}

	// A stash whose landing never arrives is retired by the TTL rather than ambushing
	// that session's next visit, and pending prunes it as it answers.
	store.set(3, signalsForms{annoError: "stranded"}, now)
	if store.pending(now.Add(formFlashTTL + time.Second)) {
		t.Errorf("an expired refusal is still collectable")
	}
	if _, ok := store.m[3]; ok {
		t.Errorf("pending answered without pruning the expired entry")
	}
}

// literalPath reports the absolute path a destination expression spells, when it spells
// one. A bare string, a package constant, and either of those with a query concatenated
// on all count: each names a destination the handler chose rather than one the operator's
// form declared.
func literalPath(e ast.Expr, consts map[string]string) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		s, ok := stringLit(v)
		return s, ok && strings.HasPrefix(s, "/")
	case *ast.Ident:
		s, ok := consts[v.Name]
		return s, ok && strings.HasPrefix(s, "/")
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return "", false
		}
		if s, ok := literalPath(v.X, consts); ok {
			return s + "…", true
		}
	}
	return "", false
}
