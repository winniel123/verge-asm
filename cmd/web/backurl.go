package main

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

// The submitting-URL carrier (ADR-0130 §3, map #969 ticket #971). A mutating form
// tells the server the exact URL it was submitted from, and the handler redirects
// back to that URL rather than to a bare path. This closes the redirect half of
// failure class E: an operator who acts from `/signals?tab=open&sev=High&page=3`
// lands back on that same list, so the filter survives the act and the scroll key
// the shell stashes on submit (shell.tmpl, ticket #970) hits on both ends.
//
// The URL rides a hidden form field, never the `Referer` header. `Referer` is
// stripped by proxies and privacy settings, and is absent under several referrer
// policies, so a handler that trusted it would silently lose the filter for some
// operators and not others. Nothing in this file reads `Referer`.
//
// Progressive enhancement is untouched. The field is server-rendered markup and the
// answer is a plain 303, so every path here works with JavaScript off. The scroll
// restore stays a pure enhancement layered on top.
//
// A form field is operator-controlled input, so resolveBack is a real open-redirect
// guard — the first in this repo. It admits only a same-origin relative path that
// this server actually serves a GET at, and falls back to a caller-supplied path on
// anything else. A rejected value never reaches the Location header.

// backField is the hidden form field the "backfield" template partial emits and
// resolveBack reads. It keeps the name the Inbox acts already use for the same job
// (`return`, inbox.tmpl / settings.tmpl), so the console carries one name for this
// datum rather than two. The Inbox acts kept their own inline check until ticket #977
// deleted messageReturn; every handler in the tree now reads this field through
// resolveBack and nothing else parses it.
const backField = "return"

// backURL is the submitting URL to stamp into a page's forms: the request's own path
// plus its query, and nothing else. It is deliberately relative — a host or a scheme
// would make the field an open-redirect vector for no gain, since every destination
// is on this server.
//
// The query is preserved BYTE FOR BYTE, in the order it arrived. Do not rebuild it
// through url.Values.Encode(): Encode sorts the parameters alphabetically, and the
// scroll key ticket #970 set is a raw string compare on `location.pathname +
// location.search` (shell.tmpl). A re-ordered query is a different string, so the
// stash written on submit would miss on the landing — the exact class-C/E failure
// this map exists to close. The orders in play are not alphabetical: the /signals
// filter form submits `tab, sev, sort, dir, q` in DOM order, and its severity links
// emit `tab, q, sev, sort, dir`.
//
// Only the `toast` parameter is dropped. It is a single-consume PRG receipt (shell.go
// toastRedirect): carrying it back would re-fire a spent toast on the next landing,
// and would let a stale receipt win over the fresh one, because decodeToasts reads
// only the first `toast` value. Every other parameter is kept, because the filter
// state is the whole point of the carrier.
func backURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	p := r.URL.EscapedPath()
	if p == "" {
		p = "/"
	}
	q := stripToastParam(r.URL.RawQuery)
	if q == "" {
		return p
	}
	return p + "?" + q
}

// stripToastParam removes every `toast` pair from a raw query. It is stripParams at
// the one name every carrier in this file drops.
func stripToastParam(raw string) string { return stripParams(raw, "toast") }

// stripParams removes every pair whose key is one of names from a raw query and
// returns what is left, with the order and the encoding of every other pair
// untouched. It walks the raw string rather than parsing it, because a
// parse-and-re-encode round trip is what re-orders the query and breaks the scroll
// key (see backURL).
//
// The key is unescaped before it is compared, so a percent-encoded spelling of the
// name is dropped too. A malformed key is compared raw, which is the safe direction:
// an unrecognised pair is kept rather than silently discarded.
func stripParams(raw string, names ...string) string {
	if raw == "" || len(names) == 0 {
		return raw
	}
	drop := func(key string) bool {
		for _, n := range names {
			if key == n {
				return true
			}
		}
		return false
	}
	var kept []string
	for _, pair := range strings.Split(raw, "&") {
		if pair == "" {
			continue
		}
		key := pair
		if i := strings.Index(pair, "="); i >= 0 {
			key = pair[:i]
		}
		if name, err := url.QueryUnescape(key); err == nil {
			key = name
		}
		if drop(key) {
			continue
		}
		kept = append(kept, pair)
	}
	return strings.Join(kept, "&")
}

// stripDestParams removes names from the query of a resolved destination URL, keeping
// the path and every surviving pair exactly as they were. It is the redirect-time twin
// of stripParams: a caller uses it when a parameter is part of the page's transient UI
// state rather than of the list the operator wants back (see settings.go dialogParams).
func stripDestParams(dest string, names ...string) string {
	i := strings.IndexByte(dest, '?')
	if i < 0 || len(names) == 0 {
		return dest
	}
	q := stripParams(dest[i+1:], names...)
	if q == "" {
		return dest[:i]
	}
	return dest[:i+1] + q
}

// resolveBack reads the submitting URL off the posted form and returns it when it
// passes the guard, or fallback when it does not. The caller always supplies a
// fallback, so a handler is never left without a destination.
//
// The guard admits a value only when ALL of these hold. Each rejection is a real
// attack shape, not a formality:
//
//   - It is not empty. A form that carries no field states no opinion.
//   - It holds no backslash. A browser folds `\` to `/` in a URL, so `/\evil.example`
//     and `\\evil.example` reach an origin this server does not own.
//   - It starts with a single `/`. This rejects `https://evil.example/x` (absolute)
//     and `//evil.example/x` (scheme-relative, which inherits the page's scheme).
//   - It parses, and carries no scheme, no host, no userinfo, no opaque part and no
//     fragment. A parse that yields any of those is not the relative path it claimed
//     to be.
//   - Its path is already clean. `/signals/../admin` is rejected rather than folded,
//     so the routing check below reads the same path the browser would request.
//   - This server serves a GET at that path. A path the router does not serve is a
//     303 into a 404, which loses the operator's place as surely as a bare path does.
//
// The query rides through once the path passes, in its own order — it is the filter
// state the carrier exists to preserve, and re-ordering it would break the scroll key
// (see backURL). One parameter is stripped: `toast`. backURL never emits one, so a
// legitimate value is unchanged, but a hand-crafted value must not be allowed to plant
// a receipt. decodeToasts (chrome.go) reads only the FIRST `toast`, so a planted one
// would beat the real toast toastRedirectBack appends and put an operator-chosen
// system message on the landing page.
func (s *server) resolveBack(r *http.Request, fallback string) string {
	if r == nil {
		return fallback
	}
	raw := strings.TrimSpace(r.FormValue(backField))
	if raw == "" {
		return fallback
	}
	if strings.Contains(raw, `\`) {
		return fallback
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return fallback
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fallback
	}
	if u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return fallback
	}
	if u.Path == "" || path.Clean(u.Path) != u.Path {
		return fallback
	}
	if !s.routeServesGET(u.Path) {
		return fallback
	}
	// Strip a planted `toast` receipt, keeping every other byte of the path and of the
	// surviving query in place. Splitting the raw string rather than re-encoding the
	// URL is deliberate: a parse-and-re-encode round trip re-orders the query.
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		q := stripToastParam(raw[i+1:])
		if q == "" {
			return raw[:i]
		}
		return raw[:i+1] + q
	}
	return raw
}

// routeServesGET reports whether this server serves a GET at p. It asks the very mux
// handler() built (s.routes), so the guard tracks the route table rather than a second
// list that drifts from it.
//
// Two answers from ServeMux need care. An unmatched path yields an empty pattern, and
// a path the mux would first redirect (a trailing-slash or a cleaning hop) yields a
// bare path rather than a pattern — neither starts with the method, so requiring the
// "GET " prefix rejects both. The catch-all `GET /` is the second: it matches every
// path, but its handler (auth.go home) answers 404 for anything but "/", so this
// returns true for that pattern only at the root.
//
// A server whose handler() never ran has no route table, and the honest answer is then
// no: the caller falls back rather than trusting a path nothing checked.
func (s *server) routeServesGET(p string) bool {
	if s == nil || s.routes == nil {
		return false
	}
	probe := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: p},
		Host:   "localhost",
	}
	_, pattern := s.routes.Handler(probe)
	if !strings.HasPrefix(pattern, "GET ") {
		return false
	}
	if pattern == "GET /" {
		return p == "/"
	}
	return true
}

// redirectBack answers the post-redirect-get of a mutating act with a 303 to the URL
// the form was submitted from, or to fallback when the form carried none that passed
// the guard. It is the plain form of the carrier: no toast, just the operator's own
// list again.
//
// A request-derived value reaches http.Redirect here, and that is admitted deliberately
// rather than overlooked. The stronger pattern is to enumerate the destinations and
// redirect to a string LITERAL at each branch, which is what the Inbox acts did until
// ticket #977 (markMessageUnread). It cannot express this contract: ADR-0130 §3 requires
// a redirect back to an arbitrary filtered list URL, and an arbitrary URL has no literal
// form — which is why the Inbox pattern had to go, since its two literals dropped the
// filter and closed the open message. The guard above is the substitute the ADR asks
// for. Do not read this as licence to skip the literal pattern where a handler CAN
// enumerate its destinations.
//
// This line carries no `#nosec`. Measured with CI's own flags (-exclude-generated
// -severity high -confidence high), gosec raises nothing here, so an annotation would
// suppress no finding and would only imply one had been silenced.
func (s *server) redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	http.Redirect(w, r, s.resolveBack(r, fallback), http.StatusSeeOther)
}

// toastRedirectBack is redirectBack composed with the toast carrier (shell.go
// toastRedirect): it 303s to the submitting URL AND fires one toast there.
//
// toastRedirect appends `toast=` with the right separator, so a submitting URL that
// already carries its own query keeps every parameter of it — the `?`/`&` choice is
// made against the resolved destination, not against the fallback. The two carriers
// compose rather than compete: this one owns the path and the filter state, that one
// owns the receipt.
func (s *server) toastRedirectBack(w http.ResponseWriter, r *http.Request, fallback, tone, title, description string) {
	s.toastRedirect(w, r, s.resolveBack(r, fallback), tone, title, description)
}
