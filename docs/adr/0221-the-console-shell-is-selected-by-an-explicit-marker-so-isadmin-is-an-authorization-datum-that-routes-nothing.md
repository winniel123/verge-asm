---
number: 221
title: "the console shell is selected by an explicit marker, so `IsAdmin` is an authorization datum that routes nothing"
slug: the-console-shell-is-selected-by-an-explicit-marker-so-isadmin-is-an-authorization-datum-that-routes-nothing
date: 2026-09-07
status: accepted
source: grilling
ticket: 1358
map: 1131
proof: {none: "predates the governance SPEC"}
relations:
  - {kind: rests-on, adr: 181, clause: "2"}
  - {kind: sibling, adr: 173}
---

# ADR-0221: the console shell is selected by an explicit marker, so `IsAdmin` is an authorization datum that routes nothing

- **Sibling of, and not ruled by:** [ADR-0173](./0173-the-api-access-tab-is-the-one-settings-surface-a-viewer-reads-and-it-carries-state-without-a-control.md) §1. That ADR rules a `viewer` carve-out on the tab identifier, and its renderer reads `IsAdmin` as an authorization flag. This ADR keeps that reading intact and takes the second, unrelated job away from the same key

## Context

`cmd/web` renders every console page from a `map[string]any`. `injectChrome` runs on every
render. It decides whether the page gets the console shell. The shell is the topnav, the palette,
the bell and the footer. Until this ADR the decision read like this, at `auth.go:1833` on
`c068bb9`:

```go
if _, isChrome := m["IsAdmin"]; !isChrome {
    return
}
```

**The presence of the key was the whole gate. The value of the key did not affect it.**

Three consequences followed.

**One key answered two unrelated questions.** The templates read `IsAdmin` as an authorization
flag at 30 sites across `settings.tmpl`, `scope.tmpl`, `signals.tmpl` and `rundetail.tmpl`. It
answers *may this account act?* The injector read the same key's presence to answer *which shell
wraps this page?* Those questions share no subject.

**Opting out of the shell was invisible at the call site.** `signinData` and `loginData` omit
`IsAdmin`, so every sign-in surface renders bare. `renderError` set `Account` and `IsAdmin` only
when `currentAccount` resolved. A signed-out error page therefore rendered bare, and a signed-in
one kept the topnav. Nothing at any of those omissions said so. `errors.go:14` carried the only
statement of the rule anywhere on disk:

```go
// The chrome appears only where the data map carries an "IsAdmin" key (#533).
```

**The clump repeated at 44 render sites across 20 files.** `"Title"`, `"Account"`, `"IsAdmin"`
and `"NavActive"` travelled together. 26 sites were production render paths and 18 were dev-only
fixtures. Deleting one inert key from that clump in #1355 took a 42-file edit.

The trap is what makes this a rule rather than a tidy-up. A helper that always stamps `IsAdmin`
reads as a pure refactor and is a behaviour change. It puts the console shell on the sign-in page
and on every signed-out error page. The Phase-A goldens crop to `<main>`. The chrome band is
therefore not pixel-gated, and no golden covers it.

No ADR stated the presence gate. ADR-0181 names the injector only for the static organisation
chip.

## Decision

> **The console shell is selected by an explicit marker on the render map. `injectChrome` reads
> that marker and nothing else to route. `IsAdmin` is an authorization datum, and the injector
> never reads it. Two named constructors build the render map, and only the shell one sets the
> marker.**

Four limbs.

### 1. The marker is a key of its own, and it carries no other meaning

`shellKey` is `"InShell"`. `injectChrome` gates on `m[shellKey].(bool)`. A page is in the console
shell because it says so, not because it happens to carry an authorization flag.

The injector keeps every other read it had: `Account`, `NavActive`, `Scanning`, `SignalCount`,
`Unread` and `BackURL`. This ADR moves the gate and moves nothing else.

### 2. Two constructors, and the pair is the whole vocabulary

```go
func pageData(acct db.Account, title, navActive string, rest ...map[string]any) map[string]any
func barePageData(title string) map[string]any
```

`pageData` sets the marker, the title, the account, the admin flag and the active nav identifier.
`barePageData` sets the title alone. A caller picks one, and the choice of constructor is the
choice of shell. The variadic `rest` merges a page's own keys, so a call site states its extra
keys once and never restates the clump.

### 3. `IsAdmin` keeps its authorization reading, unchanged

Every template branch on `IsAdmin` survives. `pageData` still computes `acct.Role == roleAdmin`,
and it still writes it under the same key. ADR-0173's API-access carve-out reads it exactly as it
did. What ends is the second job.

### 4. The bare surfaces are covered by tests, because the goldens cannot cover them

The goldens crop to `<main>`. The chrome band sits outside that crop, so a regression here is
invisible to them. Three tests in `cmd/web/chrome_gate_test.go` assert on the rendered HTML. The
sign-in page carries no chrome band. A signed-out error page carries none. A signed-in error page
carries one.

## Consequences

- **`errors.go:14` is withdrawn.** The sentence it stated is now false. The edit replaces it with
  the rule that survives: a signed-out reader has no console to return to.
- **26 production render sites call `pageData`.** `auth.go`, `cold.go`, `drift.go`, `errors.go`,
  `exposure.go`, `graph.go`, `inventory.go`, `messages.go`, `onboarding.go`, `rawoutput.go`,
  `reports.go`, `reports_schedule.go`, `scans.go`, `search.go`, `seeds.go`, `settings.go`,
  `signals.go` and `subjects.go`.
- **18 dev-only fixture sites carry the marker and nothing else.** A marker gate would otherwise
  strip the shell from every dev fixture page. Migrating them onto `pageData` is a separate
  ticket, as #1358 directs.
- **A future render map that forgets the marker renders bare.** That is the failure this shape
  chooses: a missing shell is visible on the first look, and a shell on a sign-in page is not.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** No domain term moves. `IsAdmin` keeps the
  meaning it always had, and the marker is a rendering detail with no estate meaning.
- **Nothing forces a new render site through a constructor.** A hand-built `map[string]any` still
  compiles. **A vet-style check that fails a render map carrying `"Account"` without the marker
  ships as its own ticket.**

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the presence gate and give the helper a `chrome bool` argument** | It makes the 26 production sites honest and leaves the defect in place. The injector still routes on an authorization datum, and every render map built outside the helper still carries the hidden switch. It also reads as a boolean parameter whose two values do unrelated things, which is the shape a named-constructor pair exists to replace |
| **Gate on the presence of `"Account"` instead** | Trades one presence gate for another. `Account` is also read as data by the injector and by the templates, so the same key would again answer two questions, and a page that wants the shell without an account could not ask for one |
| **Invert the marker, so a map opts *out* of the shell** | Every existing map without a marker would gain the shell, which is a behaviour change at every render site that is bare today and at every one added later by accident. The failure mode is a topnav on a sign-in page, which is exactly the regression this ADR exists to prevent |
| **Delete the gate and let each template decide** | `error-page` already branches on `.Chrome`, and moving the decision into 20 templates spreads one rule across a layer that cannot be tested without rendering. It also gives the injector no way to skip the store reads a bare page does not need |
| **Migrate the 18 dev-fixture sites in the same change** | #1358 fixes the production shape first and holds the fixtures back on purpose. The fixtures still need the marker, so they get the one key that keeps them rendering and no more. A mixed change would bury the 26 production edits in a diff twice its size |
