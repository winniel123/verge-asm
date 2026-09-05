# ADR-0171: the finished route table is the authority the open-redirect guard asks, and an unbuilt table falls back

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1332 ADR gaps: cmd/web/handlers.go (#1203)](https://github.com/winniel123/verge-asm/issues/1332), gap 3
- **Sweep PR that deleted the comment:** [#1336](https://github.com/winniel123/verge-asm/pull/1336)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0130](./0130-scroll-restore-is-hardened-by-a-same-url-prg-plus-full-url-key-contract.md) §3, which rules that a mutating handler redirects to the URL the form was submitted from and that the URL rides an explicit hidden field rather than `Referer`. §3 creates an operator-supplied redirect target and states no bound on it. This ADR states the bound
- **Not bound by:** [ADR-0158](./0158-a-read-only-console-screen-may-scope-its-rendered-rows-in-the-client-and-a-screen-that-submits-a-form-carries-its-scope-in-the-query-string.md), which rules what a screen may put **into** its own URL. This ADR rules what the server will accept **back** from a submitted one. A value that satisfies ADR-0158 still faces this guard, and a value that fails this guard is refused whatever put it there

## Context

ADR-0130 §3 made every mutating console handler take its redirect destination from the request.
The carrier is a hidden field named `return` ([`cmd/web/backurl.go:12`](../../cmd/web/backurl.go)),
stamped by the `backfield` define at
[`design-system/templates/shell.tmpl:116`](../../design-system/templates/shell.tmpl) and invoked 55
times across five screen templates — `settings.tmpl` (37), `scope.tmpl` (11), `reports.tmpl` (3),
`inbox.tmpl` (2), `signals.tmpl` (2). Forty production call sites in nine files consume the resolved
value through `redirectBack`, `toastRedirectBack` or `resolveBack`: `reports_schedule.go` (11),
`integrations.go` (6), `messages.go` (5), `seeds.go` (5), `annotations.go` (4),
`channels_sendtest.go` (4), `settings.go` (3), `scantrigger.go` (1), `sources.go` (1).

**A hidden field is operator-controlled input.** Nothing about its provenance is checkable at the
handler. Unguarded, it turns every one of those forty sites into a `Location:` the caller chose —
the standard open redirect, on an authenticated console.

**ADR-0130 §3 does not rule this.** Read in full, §3 rules the destination (*"the same URL the form
was submitted from, query included, not to a bare path"*) and the carrier (*"a hidden form field is
preferred over `Referer`, which is not reliable"*). It says nothing about validating the value.
Neither does ADR-0130 §2 or its Consequences, and no file in `docs/` or `CONTEXT.md` states an
open-redirect rule. The deleted comment cited `ADR-0130 §3` for a rule §3 does not carry, which is
[`comment-policy.md`](../spec/comment-policy.md) §8.3 shape 1.

**One live ADR already reasons from this rule without stating it.**
[ADR-0157](./0157-a-receipt-on-a-self-refreshing-surface-rides-the-server-side-flash-never-the-toast-query.md)
§2 refuses a per-branch toast carrier because *"the branch would restate the route table, and
`resolveBack` reads the destination from the request."* That argument is only sound if the route
table is the authority. ADR-0157 borrows the rule; this ADR states it.

**What the tree holds, read on 2026-09-05.**

| Fact | Site |
| --- | --- |
| `server.routes` is a `*http.ServeMux` field | [`cmd/web/handlers.go:268`](../../cmd/web/handlers.go) |
| `handler()` assigns it after the last registration, including `mountAPIv1` | `handlers.go:323`, `:493`, `:495-496` |
| `handler()` returns `s.recoverPanics(mux)`, so the served handler and the stored table are one object | `handlers.go:498` |
| 134 registrations in `handler()` and 6 in `api_v1.go`; **every one carries an explicit method token**, none is method-less | `handlers.go:325-493`, [`cmd/web/api_v1.go:17-24`](../../cmd/web/api_v1.go) |
| `routeServesGET` builds a synthetic `GET` request and asks `(*http.ServeMux).Handler` for the matched pattern | [`cmd/web/backurl.go:115-134`](../../cmd/web/backurl.go), the call at `:124` |
| It fails closed on a nil receiver or a nil table | `backurl.go:116-118` |
| It narrows the `GET /` catch-all by hand, because `home` answers 404 for every path but the root | `backurl.go:129-132`, [`cmd/web/auth.go:450-454`](../../cmd/web/auth.go) |
| `resolveBack` refuses a backslash, a missing leading `/`, a `//` prefix, an unparseable value, any scheme, host, userinfo, opaque body or fragment, and any path `path.Clean` would change | `backurl.go:85-100` |
| The route-table question is asked last, at `:101-103`, and the fallback is the caller's own literal | `backurl.go:76`, `:101-103` |
| Tests: `backurl_test.go` holds 15 functions over 416 lines, 8 of which drive `resolveBack` or `routeServesGET` directly; `TestResolveBackRejectsAndFallsBack` alone pins 16 refused inputs; `scope_prg_test.go` adds a ninth | [`cmd/web/backurl_test.go:92-123`](../../cmd/web/backurl_test.go), [`cmd/web/scope_prg_test.go:281-294`](../../cmd/web/scope_prg_test.go) |

**The three hostile shapes the ruling names are all refused today**, each pinned by a test: an
absolute URL dies at `backurl.go:88` for want of a leading `/` (`backurl_test.go:99-100`); a
scheme-relative `//evil.example` dies at `:88` on the `//` prefix (`:101-102`); a traversal
`/signals/../login` dies at `:98` because `path.Clean` moves it (`:109`). The traversal is
**refused, not folded** — the guard never rewrites a value into acceptability.

## Decision

> **The submitting-URL carrier's guard refuses any path this server does not itself serve as a
> `GET`, and the finished `http.ServeMux` is the only authority on which paths those are. The guard
> asks the live table through `(*http.ServeMux).Handler`. It never matches a pattern and never
> consults a hand-maintained list. Where no table has been built, it refuses and falls back. A value
> is refused, never repaired.**

### 1. The authority is the finished mux, asked through `Handler`

`(*http.ServeMux).Handler(r)` returns the matched pattern string alongside the handler. Since Go
1.22 that pattern carries the method — `"GET /signals"` — so one call answers both halves of the
question: is this path routed, and is it routed for a `GET`. `routeServesGET` therefore tests
`strings.HasPrefix(pattern, "GET ")` (`backurl.go:126`) and nothing else. No second source of truth
is consulted, and none may be added.

The table must be the **finished** one. `handler()` assigns `s.routes` after `mountAPIv1`
(`handlers.go:493`, `:495-496`), so the field never holds a partial mux. A route added anywhere in
`handler()` is inside the guard's answer on the same commit, with no second edit. That property is
the whole reason the mux is the authority rather than a copy of it.

### 2. The table is asked last, and only about a value already known to be a local path

The route-table question is the final gate (`backurl.go:101-103`), after eight syntactic refusals.
The ordering is load-bearing. `ServeMux.Handler` cleans the path before matching, and on a redirect
hop it returns the pattern of the node it would redirect *to*, so a dirty path could be answered
about a **different** path than the one that lands in `Location:`. The `path.Clean` equality at
`:98` removes that class of input first, so the returned pattern is about the exact bytes the
redirect will carry.

### 3. No table, no trust

A `*server` built by `newServer` whose `handler()` has not run holds a nil `routes`
(`handlers.go:268` is zero-valued until `:496`). `routeServesGET` answers `false`
(`backurl.go:116-118`), so `resolveBack` returns the caller's literal fallback. That fallback is
safe by construction: all forty call sites pass a literal spelled in Go source — `"/signals"`,
`"/scope"`, `reportsPath`, `messagesFallback`, `"/settings?tab="+tab` — never a request value. An
absent authority costs the operator a landing, not an origin.
`TestResolveBackWithoutARouteTableFallsBack` (`backurl_test.go:152-160`) pins both halves.

### 4. A subtree pattern over-reports, and every one is narrowed by name

A pattern ending in `/` matches its whole subtree, so the table says "served" for paths the handler
behind it answers 404 for. `GET /` is such a pattern and is narrowed by hand at `backurl.go:129-132`,
because `home` 404s anything but the root (`auth.go:451-453`). That narrowing is not a special case
for the root: it is the general obligation. **A subtree pattern registered on this mux is refused by
the guard except for the paths its handler actually serves, and the narrowing is written beside the
`GET /` one in the same change that registers the pattern.**

## Consequences

- **`cmd/web/backurl.go` is the guard's only home.** No handler may bypass `resolveBack` to redirect
  to a request value. `reports_schedule.go:67` already states this for the wizard's held `Back`.
- **A new route is guarded on the commit that adds it**, if it is registered inside `handler()` and
  carries a method token. A registration after `handlers.go:496`, or on another mux, is a defect.
- **`GET /api/v1/` is an unnarrowed subtree, and limb 4 makes that a defect.** It is not an open
  redirect — the accepted paths are same-origin and answer 404 or 401 — but it is the guard
  over-answering, which limb 4 forbids.
- **`backurl.go:125`'s surviving comment is wrong about this mux.** It says an unmatched path yields
  an empty pattern. With `GET /` registered nothing is unmatched: `/nope` returns `"GET /"`, and the
  narrowing at `:131` is what refuses it. Go 1.26.8's `findHandler` (`net/http/server.go:2659-2699`)
  likewise returns the matched node's pattern on a trailing-slash redirect, not an empty string.
- **The rule does not come back as a comment.** It now has a document, so the deleted block stays
  deleted and the declaration position stays empty.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A prefix or regexp pattern** — accept anything matching `^/[a-z0-9/?=&.-]*$` | It answers a different question. A pattern tests the *shape* of a path; the guard must know whether *this server* serves it. Every unserved path passes — `/nope`, `/annotations` (`POST`-only), `/api/v1/inventory` — and the 303 lands the operator on a 404 or a JSON 401. It drifts the other way too: a route spelled with a character the class omits is refused with no error and no test failure |
| **A hand-maintained allowlist constant**, e.g. `var backDestinations = []string{"/signals", "/scope", …}` | It is a second copy of the route table, and nothing makes the two diverge loudly. A route added to `handler()` and not to the list silently drops every operator on that screen to a fallback; one deleted from `handler()` and left in the list keeps a dead destination alive. It also cannot express the wildcard routes — `GET /asset/{key}`, `GET /runs/{id}`, `GET /subjects/{key}` — without re-implementing the matching the mux already does |
| **Trust the field because it is same-origin by construction** — the console stamps it, so it is ours | Provenance is not checkable at the handler: the value arrives in a `POST` body the caller composes. This is the defining error of the open-redirect class, and `TestAnnotationActRefusesAnOffOriginReturn` (`backurl_test.go:287-315`) keeps it refused. The argument also fails inside the origin — a planted `?toast=` would forge a receipt on the landing page (`backurl.go:104`; `decodeToasts` reads one value, `chrome.go:130`) |
| **Fold a traversal to its cleaned form** instead of refusing it, so `/signals/../login` lands at `/login` | Repair guesses intent, and a rewritten value is no longer the URL the operator's page submitted, so ADR-0130 §2's scroll key misses at the far end anyway. Each fold is also a new place for the cleaned path and the emitted path to disagree |
| **Read `Referer` and validate that instead**, dropping the hidden field | ADR-0130 §3 already rejected `Referer` because proxies and referrer policies strip it (`backurl.go:10`). Validating it changes nothing about that, and it makes the destination depend on a header the browser may omit — a working redirect becomes a fallback for a whole class of deployment |
