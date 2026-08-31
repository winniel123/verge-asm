# ADR-0130: The console scroll-restore is hardened by a same-URL PRG plus full-URL-key contract

- **Status:** Accepted
- **Date:** 2026-08-31
- **Ticket:** [#945 Spec the scroll-restore hardening (classes A/B/C/E)](https://github.com/winniel123/verge-asm/issues/945)
- **Map:** [#937 Wayfinder: actions must not throw the operator to the top of the page](https://github.com/winniel123/verge-asm/issues/937)

## Context

Every mutating action in the console is a full-page POST → 303 → GET (ADR-0001). On a
long screen this reloads at the top, which throws the operator away from the row they acted
on. A scroll-restore already exists in `shell.tmpl:233-259`: on `submit` it stashes
`window.scrollY` in `sessionStorage` keyed by `location.pathname`, and on the next load of
that same pathname it restores the value, but only within a 5000 ms window.

[#938](https://github.com/winniel123/verge-asm/issues/938) root-caused why the restore
still drops the operator. Five failure classes were found. Four are fixable inside the
restore contract and are this decision:

- **A — error path.** A validation failure re-renders inline at the POST handler
  (`annotations.go` `fail()` → `renderSignals` under the `/signals` POST, `seeds.go`
  refusal), so no 303 fires and the stash key never matches. The operator both loses scroll
  and, with JavaScript off, sees the same full reset.
- **B — non-form actions.** `<a>` links and `type=button` controls never reach the `submit`
  listener, so a same-list re-filter through a link never stashes.
- **C — freshness window.** The 5000 ms `FRESH_MS` expires on slow handlers (scan trigger,
  report run, restore-from-backup), so the restore that should fire is discarded as stale.
- **E — key scheme.** The pathname-only key collides across `?tab=` / `?filter=` / `?sev=`
  variants. Worse, a mutating handler that redirects to a **bare path** drops the operator's
  filter, so a `scrollY` measured against filtered content is applied to unfiltered content.

The fifth class (D — inner scroll containers and modal state) is out of this contract. It
routes through the narrowed mechanism decision
[#942](https://github.com/winniel123/verge-asm/issues/942).

[#939](https://github.com/winniel123/verge-asm/issues/939) chose the hybrid, `a′`-first fix:
harden the existing restore for A/B/C/E without an architecture change, and hold the hard
constraint that every hardened action still works with JavaScript off (progressive
enhancement). This ADR fixes the contract that hardening implements.

## Decision

Every mutating action lands the operator back at the **exact URL it was submitted from**, and
the restore keys on that full URL and fires on any normal navigation. The contract has three
parts. The four failure classes fall out of it.

### 1. Same-URL PRG — errors redirect too (fixes A)

An error path is a POST-Redirect-GET like a success path. On a validation failure the handler
does **not** re-render inline at the POST URL. It stashes the field errors and the operator's
typed values in a **server-side session flash**, then answers `303` to the GET URL the form
was submitted from (§3 preserves the query). The GET handler reads the flash once, renders the
same inline error callouts, and clears it.

- The flash is server-side and single-consume, keyed by the session. It is **not** a query
  parameter. The typed reason text is bulky and can be sensitive, so it must not enter a URL,
  a log, or the browser history. This extends the existing PRG flash vocabulary (the `?toast=`
  query decoded by `decodeToasts` in `chrome.go`), but the query carrier stays for toasts
  only.
- With JavaScript off the operator still sees the error inline on the GET page, so the bar
  holds. With JavaScript on the existing restore now fires, because the load is an ordinary
  303→GET to the same URL, indistinguishable from a success.

### 2. A full-URL key and a navigation-type gate (fixes C and E)

The stash key is the **full path plus query** (`location.pathname + location.search`), not the
pathname alone. Variant lists no longer collide, and a restored `scrollY` is always measured
against the same URL, so against the same content.

The freshness gate is a **navigation-type check**, not a fixed time budget. The restore fires
only when `PerformanceNavigationTiming.type` is `navigate` (an ordinary navigation), never on a
reload or a back/forward. The strict 5000 ms window is dropped: the stash is already
single-consume (`removeItem` runs before the restore), so a slow handler no longer loses its
restore. A **wide fallback TTL** (about 5 minutes) is kept only to bound a stash the operator
abandoned, so a fresh visit minutes later does not inherit a stale `scrollY`.

### 3. Redirects preserve the submitting URL (fixes E, completes A and B)

A mutating handler redirects to the **same URL the form was submitted from**, query included,
not to a bare path. The submitting URL is carried explicitly (a hidden form field is
preferred over `Referer`, which is not reliable). Some handlers already do this with
`q.Encode()` (`scantrigger.go`, `onboarding.go`); this generalises that to every mutating
handler whose page carries an operative query. This both keeps the operator's filter and makes
the key in §2 match on both ends.

With §3 in place, a link or button that re-filters the **same list** (class B) resolves to the
same key. The `submit`-only listener is broadened to also stash on a click of an `<a>` or
`type=button` control **whose target resolves to the current restore key**. A link that
navigates to a genuinely different page keeps the current no-stash behaviour, because a scroll
reset there is correct. No class-B control is converted to a form.

## Consequences

- Error handling gets a reusable server-side session-flash carrier for field errors and typed
  values. A build session adds it once; `annotations.go` and `seeds.go` are the first callers.
  Every mutating handler that re-renders inline on error migrates to it over time.
- Every mutating handler must know the URL it was submitted from and redirect back to it. The
  build session audits the ~80 redirects (`cmd/web/*.go`) and adds the hidden submitting-URL
  field to the forms whose page carries an operative query.
- The restore script in `shell.tmpl` is rewritten: full-URL key, navigation-type gate, wide
  fallback TTL, and the broadened click-stash for same-key links and buttons.
- Progressive enhancement holds throughout. Every hardened path is a plain PRG that works with
  JavaScript off; the restore is a pure enhancement layered on top.
- Class D (inner scroll containers, modal state) is untouched here. If residue still hurts
  after this hardening ships, it routes through [#942](https://github.com/winniel123/verge-asm/issues/942).

## Rejected alternatives

- **Restore on the POST-response document (class A).** Keep the inline render at the POST URL
  and make the JavaScript restore fire on a POST response, keyed by the form's eventual GET
  target. Rejected: it gives the JavaScript-off operator nothing, and the cross-URL keying
  (stash under `/signals`, render at the `/annotations` POST) is fragile. The PRG path needs
  no such special case.
- **Query-carried errors (class A).** Carry field errors and typed values in the redirect
  query, like `?toast=`. Rejected: the typed reason text is bulky and can be sensitive, and a
  URL is logged and kept in history. The session flash keeps it off the URL.
- **Just widen `FRESH_MS` (class C).** Bump the window to 15-30 s. Rejected: it is an
  arbitrary budget that a slow scan or report run can still exceed. The navigation-type gate
  removes the budget.
- **Drop the time gate entirely (class C).** Rely only on single-consume with no TTL.
  Rejected: an abandoned stash could restore a stale `scrollY` on a fresh visit to the same
  URL minutes later. The wide fallback TTL bounds that without re-introducing the slow-handler
  failure.
- **Full-URL key without preserving the redirect query (class E).** Key on path+query but
  leave bare-path redirects as they are. Rejected: the keys still miss (submit from
  `/signals?filter=x`, land at `/signals`), and the operator still loses the filter. §3 is
  required for the key to do its job.
- **Convert class-B links and buttons to forms.** Rejected: a heavy markup change that makes a
  plain GET link worse under progressive enhancement. Broadening the click-stash to same-key
  targets is enough.
