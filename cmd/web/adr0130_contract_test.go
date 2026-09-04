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

type contractPkg struct {
	fset         *token.FileSet
	methods      map[string]*ast.FuncDecl
	funcs        map[string]*ast.FuncDecl
	postHandlers map[string]string
	consts       map[string]string
}

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
		funcs:        map[string]*ast.FuncDecl{},
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
			if !ok {
				continue
			}
			if fn.Recv == nil {
				c.funcs[fn.Name.Name] = fn
				continue
			}
			if len(fn.Recv.List) != 1 {
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

var gateMethods = map[string]bool{
	"requireLogin": true, "requireAdmin": true, "requireSettingsAdmin": true,
	"requireAPIAuth": true, "redirectTo": true,
}

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

func inspectLive(fn *ast.FuncDecl, f func(ast.Node) bool) {
	skip := devModeBodies(fn)
	ast.Inspect(fn, func(n ast.Node) bool {
		if n != nil && skip[n] {
			return false
		}
		return f(n)
	})
}

func (c *contractPkg) calleesOf(fn *ast.FuncDecl) []string {
	var out []string
	inspectLive(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch f := call.Fun.(type) {
		case *ast.SelectorExpr:
			id, ok := f.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch id.Name {
			case "s":
				out = append(out, "s."+f.Sel.Name)
			case "http":
				if f.Sel.Name == "Error" || f.Sel.Name == "Redirect" {
					out = append(out, "http."+f.Sel.Name)
				}
			}
		case *ast.Ident:
			if _, ok := c.funcs[f.Name]; ok {
				out = append(out, f.Name)
			}
		}
		return true
	})
	return out
}

func (c *contractPkg) decl(name string) (*ast.FuncDecl, bool) {
	if m, ok := strings.CutPrefix(name, "s."); ok {
		fn, found := c.methods[m]
		return fn, found
	}
	fn, found := c.funcs[name]
	return fn, found
}

func (c *contractPkg) reach(start string, stop map[string]bool, visit func(name string, fn *ast.FuncDecl)) {
	seen := map[string]bool{}
	var walk func(string)
	walk = func(name string) {
		if seen[name] || stop[name] {
			return
		}
		seen[name] = true
		fn, ok := c.decl(name)
		if !ok {
			return
		}
		visit(shortName(name), fn)
		for _, callee := range c.calleesOf(fn) {
			walk(callee)
		}
	}
	walk(start)
}

func shortName(name string) string {
	if m, ok := strings.CutPrefix(name, "s."); ok {
		return m
	}
	return name
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

var bodyAnswers = map[string]bool{
	"s.render": true, "s.renderStatus": true, "s.renderSettings": true, "s.renderSeeds": true,
	"s.renderSignals": true, "s.renderProfile": true, "s.renderOnboard": true,
	"s.renderScheduleWizard": true,
	"http.Error":             true,
}

var errorPageMethods = map[string]bool{
	"s.serverError": true, "s.renderError": true, "s.notFound": true, "s.forbidden": true,
	"s.renderMissingSubject": true, "s.renderMissingRun": true, "s.settingsForbidden": true,
	"s.requireLogin": true, "s.requireAdmin": true, "s.requireSettingsAdmin": true,
}

var classAExemptRoutes = map[string]string{
	"POST /setup":      "first-run credential form; no session exists yet to key a flash on",
	"POST /login":      "sign-in form; pre-authentication, no session and no list to return to",
	"POST /login/totp": "second-factor prompt; pre-authentication, same reason as /login",
	"POST /forgot":     "password-reset request; pre-authentication",
	"POST /reset":      "password-reset form; pre-authentication, and the token rides the page",
	"POST /invite":     "invite acceptance; pre-authentication, the invitee holds no account yet",
	"POST /onboarding": "the checklist advances by rendering its next step; it refuses no form",

	// A redirect would have to stash the plaintext secret this response mints and shows once.
	"POST /account/totp/enable":  "renders the enrolment screen (QR + secret), a step and not a refusal",
	"POST /account/totp/confirm": "the same enrolment screen, re-rendered with its own prompt",
}

// A route-level entry blesses every later answer in the handler, including a fresh regression.

var classAExemptAnswers = map[string]string{
	"POST /profile/tokens · revealMintedToken → s.renderProfile": "the minted plaintext is revealed once in the body and never stored",

	"POST /settings/integrations/install · installIntegration → http.Error":     "an unknown catalogue slug is a hand-crafted request, not an operator refusal",
	"POST /settings/integrations/remove · removeIntegration → http.Error":       "an unknown catalogue slug is a hand-crafted request, not an operator refusal",
	"POST /settings/integrations/disconnect · removeIntegration → http.Error":   "an unknown catalogue slug is a hand-crafted request, not an operator refusal",
	"POST /settings/integrations/test · testIntegration → http.Error":           "an unknown catalogue slug is a hand-crafted request, not an operator refusal",
	"POST /settings/integrations/channel · bindIntegrationChannel → http.Error": "an unknown catalogue slug, or an unparseable channel id, is a hand-crafted request",
	"POST /proposals/confirm · confirmProposal → http.Error":                    "an unparseable proposal id; the row's own form always carries a valid one (#976)",
	"POST /proposals/decline · declineLookup → http.Error":                      "an unparseable form body; there is no form state to echo back (#976)",

	"POST /settings/backup · backupDownload → http.Error": "answers with the archive; the unavailable-mode line is its only other answer",

	"POST /settings/restore/preflight · restorePreflight → http.Error": "states that this build mode has no database to restore into",
	"POST /settings/restore · restoreApply → http.Error":               "states that this build mode has no database to restore into",
}

func classAStop() map[string]bool {
	stop := map[string]bool{}
	for k := range errorPageMethods {
		stop[k] = true
	}
	for k := range bodyAnswers {
		stop[k] = true
	}
	return stop
}

func TestNoMutatingHandlerAnswersAValidationFailureWithABody(t *testing.T) {
	c := parseWebPackage(t)
	var bad []string
	for pattern, handler := range c.postHandlers {
		if _, ok := classAExemptRoutes[pattern]; ok {
			continue
		}
		var hits []string
		c.reach("s."+handler, classAStop(), func(name string, fn *ast.FuncDecl) {
			for _, callee := range c.calleesOf(fn) {
				if !bodyAnswers[callee] {
					continue
				}
				hit := pattern + " · " + name + " → " + callee
				if _, ok := classAExemptAnswers[hit]; !ok {
					hits = append(hits, hit)
				}
			}
		})
		bad = append(bad, uniqSorted(hits)...)
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("these POST handlers answer with a body instead of a 303 (ADR-0130 §1):\n  %s\n"+
			"Stash the refusal on the session form flash and redirect back, or add a justified entry to\n"+
			"classAExemptAnswers (one answer) or classAExemptRoutes (a surface §1 does not govern).",
			strings.Join(bad, "\n  "))
	}
}

func TestContractExemptionsAreLive(t *testing.T) {
	c := parseWebPackage(t)
	for _, m := range []map[string]string{classAExemptRoutes, classEExempt} {
		for pattern := range m {
			if _, ok := c.postHandlers[pattern]; !ok {
				t.Errorf("an exemption names %q, which is no longer a POST route; delete the entry", pattern)
			}
		}
	}
	live := map[string]bool{}
	for pattern, handler := range c.postHandlers {
		c.reach("s."+handler, classAStop(), func(name string, fn *ast.FuncDecl) {
			for _, callee := range c.calleesOf(fn) {
				if bodyAnswers[callee] {
					live[pattern+" · "+name+" → "+callee] = true
				}
			}
		})
	}
	for hit := range classAExemptAnswers {
		if !live[hit] {
			t.Errorf("classAExemptAnswers names %q, which no handler answers any more; delete the entry", hit)
		}
	}
}

// Each helper's bare-path fallback answers a forged or absent field, not a chosen destination.

var backHelpers = map[string]bool{
	"s.redirectBack": true, "s.toastRedirectBack": true,
	"s.backToSection": true, "s.toastBackToSection": true, "s.failSettings": true, "s.flashSettings": true,
	"s.backToScope": true, "s.flashScopeBack": true, "s.flashScopeToastBack": true,
}

var classEExempt = map[string]string{
	"POST /setup":      "first-run setup completes into the console at a fixed destination",
	"POST /login":      "sign-in navigates to the console or back to the form; neither is a list",
	"POST /login/totp": "second-factor prompt; same fixed navigation as /login",
	"POST /logout":     "sign-out navigates to the sign-in screen by design",
	"POST /signout":    "the shell's alias onto the same /logout handler",
	"POST /forgot":     "password-reset request; pre-authentication navigation",
	"POST /reset":      "password-reset completion; pre-authentication navigation",
	"POST /invite":     "invite acceptance lands the new account in the console",

	"POST /onboarding":        "the checklist advances to its next step; the move is the point",
	"POST /onboarding/finish": "the finish leaves onboarding for the scan monitor",

	"POST /profile/password":               "/profile carries no operative query; see failProfile",
	"POST /profile/tokens":                 "/profile carries no operative query; see failProfile",
	"POST /profile/tokens/revoke":          "/profile carries no operative query; see failProfile",
	"POST /profile/session/revoke":         "ends the caller's own session; the destination is the sign-in screen",
	"POST /profile/sessions/revoke":        "/profile carries no operative query; see failProfile",
	"POST /profile/sessions/revoke-others": "/profile carries no operative query; see failProfile",
	"POST /profile/sso/unlink":             "/profile carries no operative query; see failProfile",
	"POST /account/totp/enable":            "two-factor enrolment is a screen sequence, not a list act",
	"POST /account/totp/confirm":           "two-factor enrolment is a screen sequence, not a list act",

	"POST /reports/schedule/new":       "the wizard's back/next/invalid-step redirects are moves between steps",
	"POST /reports/schedule/{id}/edit": "the edit wizard's step moves, same as the new-schedule wizard",

	"POST /settings/restore": "a completed restore ends every session; the caller can only land on sign-in",
}

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
		c.reach("s."+handler, stop, func(name string, fn *ast.FuncDecl) {
			params := paramNames(fn)
			for _, dest := range redirectDestinations(fn) {
				if lit, ok := literalPath(dest, consts, params); ok {
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
	// The real take needs a session read, so takeFormFlashIf itself is never exercised here.
	delete(store.m, 7)

	if _, ok := store.m[7]; ok {
		t.Errorf("the flash survived its take; a reload would re-show a spent callout")
	}
	if store.pending(now) {
		t.Errorf("the store still reports a pending refusal after the only one was taken")
	}

	store.set(1, signalsForms{annoError: "one"}, now)
	store.set(2, signalsForms{annoError: "two"}, now)
	delete(store.m, 1)
	if _, ok := store.m[2]; !ok {
		t.Errorf("one session's take consumed another session's refusal")
	}

	store.set(3, signalsForms{annoError: "stranded"}, now)
	if store.pending(now.Add(formFlashTTL + time.Second)) {
		t.Errorf("an expired refusal is still collectable")
	}
	if _, ok := store.m[3]; ok {
		t.Errorf("pending answered without pruning the expired entry")
	}
}

func paramNames(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if fn.Type == nil || fn.Type.Params == nil {
		return out
	}
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			out[name.Name] = true
		}
	}
	return out
}

func literalPath(e ast.Expr, consts map[string]string, params map[string]bool) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		s, ok := stringLit(v)
		return s, ok && strings.HasPrefix(s, "/")
	case *ast.Ident:
		if s, ok := consts[v.Name]; ok {
			return s, strings.HasPrefix(s, "/")
		}
		// A caller's parameter spells the destination one frame up, never the form's own URL.
		if params[v.Name] {
			return v.Name + " (supplied by the caller)", true
		}
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return "", false
		}
		if s, ok := literalPath(v.X, consts, params); ok {
			return strings.TrimSuffix(s, "…") + "…", true
		}
	}
	return "", false
}
