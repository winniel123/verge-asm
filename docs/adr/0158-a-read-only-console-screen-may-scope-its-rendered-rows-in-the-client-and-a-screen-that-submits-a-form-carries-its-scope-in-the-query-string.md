# ADR-0158: A read-only console screen may scope its rendered rows in the client, and a screen that submits a form carries its scope in the query string

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1348 ADR gaps: cmd/web graph.go and inventory.go (#1211)](https://github.com/winniel123/verge-asm/issues/1348), gap 1
- **PR that deleted the comment:** [#1347](https://github.com/winniel123/verge-asm/pull/1347)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounds:** [ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md)'s *"no server-side search or pagination on `/inventory` in v1 … It is unbuilt, not designed against"* clause, at that clause's own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [ADR-0131](./0131-the-console-is-vanilla-server-rendered-prg-and-the-htmx-stack-is-withdrawn.md), which ratifies server-rendered PRG plus inline vanilla JavaScript and refuses only a fetch/swap layer; [ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md) decision 4, which rules a filter legal exactly where its predicate is a value the row already renders; [ADR-0130](./0130-scroll-restore-is-hardened-by-a-same-url-prg-plus-full-url-key-contract.md) §3, which preserves the query across a mutating action

## Context

[`cmd/web/inventory.go`](../../cmd/web/inventory.go) carried this, until #1347 deleted it:

```go
// HasGap reports whether the subject currently holds at least one Gap facet — a
// timeline whose value the system cannot state. The Inventory "Gaps only"
// client-side scope (SPEC-CHANGE #13, package v3.2.4) reads it off each rendered
// row to hide subjects that hold no Gap, without a server round-trip.
```

**The citation is dead.** `design-system/SPEC-CHANGE.md` is not in the tree. [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md) withdrew the SPEC-CHANGE collision protocol on 2026-08-28. The `#13` a reader follows resolves to an unrelated issue. This is [`comment-policy.md`](../spec/comment-policy.md) §8.3 shape 2.

**A survivor one file over states the opposite rule.** [`cmd/web/signals.go:198`](../../cmd/web/signals.go) reads *"The shell ships no client-side table machinery, so filter and sort state rides the query string."* Read as a claim about the shell, that sentence is false. Inventory ships a full client-side table toolbar. Nothing on disk says which screen gets which treatment, so a later session reading the two sites cannot tell a deliberate deviation from a defect.

**What the tree holds, read on 2026-09-05.**

| Fact | Site |
| --- | --- |
| `/inventory` and `/inventory/export` are the only Inventory routes, and both are `GET` | [`cmd/web/handlers.go:384-385`](../../cmd/web/handlers.go) |
| `inventory.tmpl` contains no `<form>` element | [`design-system/templates/inventory.tmpl`](../../design-system/templates/inventory.tmpl) |
| The toolbar holds kind, `Gaps only`, `Hide proxy edge`, a text filter, density, and column toggles. Every one of them hides or shows already-rendered DOM | `inventory.tmpl:110-136`, `:177-266` |
| `Gaps only` reads the rendered gap badge, not a datum. The script sets `data-gap="1"` from `tr.querySelector(".inv-gapbadge")` | `inventory.tmpl:184`, from the badge at `:150` |
| `inventorySubject.HasGap` is called by no template and by no other Go file. One test calls it | [`cmd/web/inventory.go:43`](../../cmd/web/inventory.go), `cmd/web/inventory_test.go:286` |
| The server windows each group to 25 subjects unless `?all=<kind>` expands one | `cmd/web/inventory.go:62`, `:461-475`, `:504` |
| Signals puts its filter and sort state in the query string, and it submits three `POST` forms | `cmd/web/signals.go:198-205`, `signals.tmpl:169`, `:297`, `:302`, `:341` |

**Inventory is the only screen with a client-side table scope.** No other template in `design-system/templates/` filters `tbody` rows. `shell.tmpl:367` filters the command palette, which renders no table and holds no row.

**No document rules this.** `docs/adr/`, `docs/spec/`, `docs/guides/`, `docs/research/` and `CONTEXT.md` state no rule about where a console filter runs. ADR-0072 decision 4 rules which predicate is legal and says nothing about where the predicate is evaluated.

## Decision

> **A console screen that submits no form MAY hold its filter and sort state as a client-side view scope over the rows the server already rendered. A console screen that submits a form MUST carry that state in the query string. The scope's predicate is a value the row already renders, and the scope narrows the rendered window only, never the estate.**

Four limbs.

### 1. The line is the form, and the ground is the redirect

ADR-0130 §3 preserves the query across a mutating action. A `POST` answers `303` to the GET URL the form was submitted from, and that URL carries the query. So the query string is the one carrier that survives a console action. A client-side scope does not survive it. On a screen with a `POST`, a client scope is discarded on every act, and the operator lands back on an unscoped list.

Inventory submits nothing. Its two routes are `GET`, and its template holds no `<form>`. There is no round trip to preserve state across, so the query string buys nothing and the client scope loses nothing.

Signals submits three `POST` forms. Its state therefore rides the query string, and `signals.go:198` is right about Signals.

### 2. This is not a fetch/swap layer, so ADR-0131 admits it

ADR-0131 ratifies *"server-rendered PRG plus inline vanilla JavaScript"* and refuses *"no in-place fetch/swap layer — htmx or bespoke"*. The Inventory toolbar issues no request. It sets `style.display` on rows the server already sent. It is inline vanilla JavaScript over static markup, which is the architecture ADR-0131 ratified, not the layer it refused.

The Inventory toolbar is therefore an admitted case and not a defect.

### 3. The rendered cell is the predicate's only carrier

ADR-0072 decision 4 rules a filter legal *"exactly where its predicate is a value the row already renders"*. A client scope satisfies that test by construction when it reads the DOM, because the DOM is what the row renders. `Gaps only` reads the gap badge. It is legal.

A hidden per-row datum is a second carrier for the same predicate, and it is invisible to the operator. A filter that narrows on it narrows on something the row does not render, which is what ADR-0072 decision 4 forbids. So the datum may not become the carrier.

`inventorySubject.HasGap` is that datum. It is unreachable from the template today, and it stays that way. The template does not gain a `data-gap` attribute rendered from it.

### 4. The scope narrows the window, not the estate

`windowInventoryGroups` caps each group at `inventoryGroupWindow` = 25 subjects. The cap landed with [#756](https://github.com/winniel123/verge-asm/issues/756). `Total` stays whole, so the group badge and the *"Show all N"* link state the group rather than the window.

A client scope can only reach the rows the server sent. `Gaps only` over a capped group therefore hides the non-gap rows inside the window and cannot reveal a gap in the 26th subject. The escape is the *"Show all N"* link, which is a server round trip carrying `?all=<kind>`.

This bound is a property of every client-side scope on this repo, not an Inventory quirk. A screen that needs a scope over the whole population needs a server-side predicate, which puts it under limb 1's second half.

## Consequences

- **[ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md) gains a bounding sentence** at its *"Estate-wide grouping"* Consequences bullet. That bullet says `/inventory` has no pagination and that a search or scope filter is *"unbuilt, not designed against"*. Both halves have moved. #756 built the per-group window, and the scope filter is built in the client. ADR-0058 requires the correction at that clause's site.
- **[`cmd/web/inventory.go`](../../cmd/web/inventory.go) gains one comment** beside the `windowInventoryGroups` call, stating limb 4's bound and citing this ADR. That call is the only place in Go where the bound is decided.
- **`inventorySubject.HasGap` is dead production code.** No template and no other Go file calls it. One test pins it. This ADR rules it must not become the carrier, so nothing will call it. Its removal, and its test's, is a separate change and is not made here.
- **[`cmd/web/signals.go:198`](../../cmd/web/signals.go) is narrowed to the screen and cites this ADR.** It stated limb 1's second half for Signals with a *"the shell ships no client-side table machinery"* ground that reads wider than this ADR rules and wider than the tree supports: Inventory's toolbar is exactly that machinery. [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) requires the correction at the clause's own site, so it lands here rather than as a later edit. The comment now gives the form as the ground.
- **No production behaviour changes.** Inventory and Signals already have the shapes this ADR states.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** *View scope* and *toolbar* are console-shell terms, not product-domain terms. The glossary carries terms and this is a rule.
- **A new screen states which half of limb 1 it is under.** A screen that grows its first `POST` moves under the second half, and its client scope becomes a defect at that moment.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Rule the client-side scope a defect and move Inventory's toolbar to the query string** | Reads `signals.go:198` as a rule about the shell when nothing ever ruled it one. It also costs six server round trips on a pure read screen — kind, gaps, proxy, text, density, columns — to buy a durability that no `POST` on this screen can consume. ADR-0131 already ratified inline vanilla JavaScript over rendered markup, which is exactly what the toolbar is |
| **Rule every console filter client-side, and withdraw `signals.go:198`** | Loses Signals' filter and sort on every annotation and every exclusion, because ADR-0130 §3 preserves the query and preserves nothing else. It also loses the shareable Signals URL. The screen with the actions is the screen that most needs its scope to survive an action |
| **Make `inventorySubject.HasGap` the carrier and render it as a `data-gap` attribute** | Puts the predicate in a hidden attribute rather than in the cell, which is ADR-0072 decision 4's refused shape read at the DOM. It also gives one predicate two definitions, in Go and in the badge markup, which can disagree. The badge is already the honest carrier and is already there |
| **Lift the 25-row window so the client scope reaches the whole estate** | Trades a stated bound for an unbounded page on a corpus ADR-0041 never compacts. #756 built the window for that reason. The bound is cheap to state and the page length is not |
| **Say nothing and let each screen keep its own wording** | The state #1348 recorded. Two adjacent files state opposite rules, one of them cites a document that no longer exists, and no reader can tell the deviation from the defect |
| **Write the rule as a `docs/spec` section on the console shell** | No such section exists, and one clause does not carry a spec file. The rule is a decision about two screens with a named ground, which is what an ADR is on this repo |
