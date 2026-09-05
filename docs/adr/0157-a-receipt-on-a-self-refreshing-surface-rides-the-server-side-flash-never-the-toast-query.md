# ADR-0157: a receipt on a self-refreshing surface rides the server-side flash, never the `?toast=` query

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1343 ADR gaps: cmd/web/settings.go (#1205)](https://github.com/winniel123/verge-asm/issues/1343), gap 1
- **PR that deleted the comment:** [#1344](https://github.com/winniel123/verge-asm/pull/1344)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounds:** [ADR-0130](./0130-scroll-restore-is-hardened-by-a-same-url-prg-plus-full-url-key-contract.md) §1, at §1's own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [`scans-monitor-bounding.md`](../spec/scans-monitor-bounding.md) §1 and §6.5. That spec requires the in-flight monitor and its history to survive their own `<meta refresh>`. It rules what the page draws and says nothing about the receipt the page carries

## Context

[`cmd/web/settings.go`](../../cmd/web/settings.go) carried this at `:432-441`, until #1344 deleted
it:

```go
// toastBackToSection stashes a single-consume TOAST for the next render and answers
// backToSection's 303. It is flashRedirect (shell.go) over the submitting-URL carrier, and
// the dispatch stop and terminate acts are its callers (ticket #977).
//
// toastRedirectBack would be wrong for those two. The Scans surface re-renders itself with
// a meta-refresh while a dispatch is in flight, and a toast spelled on the URL fires again
// on every one of those reloads — the "Scan started" spam WORK-ORDER-DOGFOOD-R1 reported.
// The per-account flash store keeps the receipt single-consume; backToSection lands the
// operator on their own list, whichever of /settings?tab=scans or /scans they acted from,
// with the confirm dialog's ?stop= or ?terminate= dropped (dialogParams).
```

The sweep kept one uncited line of it, now at `cmd/web/settings.go:262`:

```go
// A toast spelled on the URL fires again on every meta-refresh the in-flight Scans page runs.
```

The comment asserts a rule someone chose. The rule binds five files outside `settings.go`. No
document on `main` states it. That is #1343's gap 1.

**The console has two receipt carriers, and both are live.** `s.toastRedirect`
([`cmd/web/shell.go:93`](../../cmd/web/shell.go)) marshals the tone, the title and the description to
JSON, encodes them base64, and appends the result to the destination as `?toast=`. `decodeToasts`
([`cmd/web/chrome.go:126`](../../cmd/web/chrome.go)) reads that parameter on every GET.
`s.flashRedirect` (`shell.go:107`) instead writes the same value to the process-local `flashStore`
([`cmd/web/flash.go`](../../cmd/web/flash.go)) under the account id, and redirects to a clean URL.
The chrome fill at [`cmd/web/auth.go:1868`](../../cmd/web/auth.go) takes the flash once per render
and deletes it.

**The Scans landings re-render themselves while a dispatch is in flight.**
`design-system/templates/shell.tmpl:5` emits `<meta http-equiv="refresh" content="6">` whenever the
page data carries a truthy `Refresh`. `fillScansSection`
([`cmd/web/scans.go:163`](../../cmd/web/scans.go)) sets `data["Refresh"] = len(active) > 0`, and both
`/scans` and `/settings?tab=scans` render through it — `scansPage` and `settingsPage` share
`renderSettings` ([`cmd/web/settings.go:604`](../../cmd/web/settings.go)). Run detail sets the same
flag from `runRefresh(view.Status)`, which answers `5` only while the status word is the literal
`running` ([`cmd/web/scans.go:630`](../../cmd/web/scans.go)).

**Run detail stops refreshing the moment a disposition is recorded, and that does not weaken the
rule.** A stop writes `stopped` over `fanned-out`, so `runStatusLabel` returns `stopped` and
`runRefresh` answers `0` while the running jobs are still committing
([ADR-0165](./0165-a-recorded-dispatch-disposition-overrides-the-live-status-derivation-and-the-run-pages-status-word-is-one-token-that-styles-and-labels-the-badge.md)
§4 rules that word and states the cost). The two Scans landings key on `len(active) > 0` instead, so
they keep refreshing while those jobs drain. A stop receipt that lands on one of them therefore
still meets a live `<meta refresh>`, which is the case this ADR rules.

**A query toast is part of the URL the refresh reloads.** The `<meta refresh>` re-requests the
current URL, `?toast=` included, so `decodeToasts` decodes the same receipt again every six seconds
for as long as the dispatch runs. The flash cannot repeat, because the take deletes it.

**The handler cannot pick the landing.** `backToSection` (`settings.go:241`) resolves the
destination from the submitted `return` field
([`cmd/web/backurl.go`](../../cmd/web/backurl.go) `resolveBack`), and falls back to
`/settings?tab=<tab>`. A stop or a terminate is offered on both Scans landings, so the handler knows
only that the operator lands on one of them.

**ADR-0130 §1 states the sentence this code departs from.** Its flash bullet ends: *"This extends
the existing PRG flash vocabulary (the `?toast=` query decoded by `decodeToasts` in `chrome.go`),
but the query carrier stays for toasts only."* Read alone and in the present tense, that sentence
sends a toast down the query. ADR-0130's own subject is the field-error carrier, and the clause is
there to keep a typed reason off the URL. It rules one direction and reads as though it ruled both.

**Two stores are called "flash", and they are not the same store.** `flashStore` holds one `toastVM`
per **account**, with no expiry. `formFlashStore` holds one form value per **session**, with a
one-minute TTL, and it is the store ADR-0130 §1 specifies for field errors. This ADR rules the
first. `cmd/web/adr0130_contract_test.go` guards the second, so no test covers the carrier choice.

**`WORK-ORDER-DOGFOOD-R1` names nothing that rules.** The token resolves to
`docs/research/sql-stripper-cross-check.md:78` and `docs/research/comment-gate-test.md:811`. Both are
comment-sweep artefacts that quote this same comment corpus, which is the false resolution
[`comment-policy.md`](../spec/comment-policy.md) §4.7 names. The reported symptom is recorded here
instead, and it is recorded as a report, not as a measurement this ADR repeated.

## Decision

> **A receipt that can land on a surface which re-renders itself carries on the server-side
> single-consume flash, and never on the `?toast=` query. The Scans landings are such surfaces, so
> every handler that can redirect an operator to `/scans`, `/settings?tab=scans` or `/runs/{id}`
> sets the flash. The `?toast=` query stays the carrier for a landing that does not refresh itself.
> The test is the landing, never the act.**

### 1. A self-refreshing surface is one whose page data carries a truthy `Refresh`

That flag is the whole test, because it is what emits the `<meta http-equiv="refresh">` in
`shell.tmpl`. Three surfaces set it today: `/scans` and `/settings?tab=scans` through
`fillScansSection`, and `/runs/{dispatch-id}` through `runRefresh`. A surface added to that set moves
every receipt that can land on it to the flash, in the same change.

### 2. The test is the landing, not the act

A handler chooses the carrier by where the operator arrives, not by what the operator did. A handler
whose landing set holds one refreshing page uses the flash for **all** of its landings. Mixing the
two carriers by branch is refused: the branch would restate the route table, and `resolveBack` reads
the destination from the request.

### 3. A handler sets one carrier, never both

The chrome fill replaces any decoded query toast with the flash when a flash is held, so a handler
that sets both drops one silently. Set the flash and redirect to a clean URL.

### 4. The flash is a courtesy, and it promises nothing more

It is process-local, account-keyed and unbounded in time. A restart drops it. Two tabs of one
account share one slot, so the second render to arrive wins. Put nothing on it that the operator
must be able to read back. A fact the operator must keep belongs in the page the redirect lands on,
or in a message.

### 5. This bounds ADR-0130 §1 in one direction only

§1 reserves the query carrier for toasts. Nothing but a toast may ride the query, and the field
errors and typed values §1 is about stay on the session flash. §1 does **not** oblige a toast to ride
the query. A toast may ride the server-side flash, and on a self-refreshing landing it must.

## Consequences

- **Three call sites already comply, and this ADR changes no Go behaviour.**
  `s.toastBackToSection` (`settings.go:261`) serves the stop, terminate and already-concluded
  receipts in `cmd/web/scans.go`. `triggerScan`
  ([`cmd/web/scantrigger.go:88`](../../cmd/web/scantrigger.go)) sets the flash and then redirects
  back. `finishOnboarding` (`scantrigger.go:104`) uses `flashRedirect` to `/scans`.
- **The `toastRedirectBack` callers stay on the query.** Their landings are `/scope`,
  `/settings?tab=channels`, `/settings?tab=integrations` and `/settings?tab=api`, and none of the
  four sets `Refresh`. `cmd/web/seeds.go`, `cmd/web/integrations.go`, `cmd/web/channels_sendtest.go`
  and `cmd/web/settings.go:1200` are unaffected.
- **One comment gains a citation.** `cmd/web/settings.go:262` keeps its reason and names this ADR.
  `cmd/web/flash.go`'s courtesy line names §4.
- **[ADR-0130](./0130-scroll-restore-is-hardened-by-a-same-url-prg-plus-full-url-key-contract.md) §1
  gains a bounding sentence at its own site**, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). A session
  reading §1 alone would otherwise spell a Scans receipt on the URL.
- **Nothing enforces this.** `cmd/web/adr0130_contract_test.go` guards the session form flash, the
  refusal-answers-no-body rule and the literal-destination rule. It does not read the carrier against
  the landing's `Refresh` flag. Review carries this rule. A guard is buildable — the package's route
  table and call graph are already walked by that file — and it is not written here.
- **`CONTEXT.md` gains nothing.** A receipt carrier is a console mechanism, not a domain term.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep `?toast=` everywhere and strip the parameter on the refresh** | The `<meta refresh>` re-requests the current URL and the server cannot rewrite it. Stripping would need the restore script to rewrite history on load, which is JavaScript, and ADR-0130's hard constraint is that every hardened path works with JavaScript off |
| **Keep `?toast=` and make the toast component dismiss itself after the first paint** | The reload builds a new document, so the second paint has no memory of the first. Dismissal would need a per-toast marker in `sessionStorage`, which is a second store and still JavaScript |
| **Stop the `<meta refresh>` while a receipt is on the URL** | It trades the receipt against the monitor. The operator stops a dispatch precisely to watch the counts fall, and `scans-monitor-bounding.md` §6.5 makes surviving the refresh an acceptance criterion for that page |
| **Move every receipt in the console to the flash** | It puts an account-keyed, process-local, un-expiring store on paths that have no reason to pay for it, and it loses the one property the query carrier has: the receipt survives a restart and a second browser, because it is in the URL the operator holds |
| **Choose the carrier per act — a dispatch act uses the flash, everything else the query** | The act is not what repeats. A receipt for any act, landed on a refreshing page, repeats. Keying on the act would put a future receipt on the URL of a page that refreshes, which is the exact defect this rule closes |
| **Give `flashStore` a TTL, so a repeat is bounded rather than prevented** | A bound of any length still shows the operator the same receipt more than once, and a TTL short enough to prevent it races the redirect. Single consume needs no clock |
| **Add the rule as a clause on [ADR-0130](./0130-scroll-restore-is-hardened-by-a-same-url-prg-plus-full-url-key-contract.md)** | ADR-0130 rules scroll restore. Its flash exists to keep a typed error value off the URL, and the reason here is repetition on a self-refreshing page, which ADR-0130 never considers. Filing it there would put a rule about receipt repetition inside a document about scroll position. §1 gets the bounding sentence ADR-0058 owes it instead |
