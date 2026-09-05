# ADR-0181: the deployment is single-tenant, so no `Organisation` is modelled and the shell ships a static chip

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1334 ADR gaps: cmd/web/auth.go](https://github.com/winniel123/verge-asm/issues/1334), gap 2
- **Sweep PR that deleted the comment:** [#1337](https://github.com/winniel123/verge-asm/pull/1337)
- **Rests on:** [ADR-0001](./0001-stack-and-runtime.md), whose Context (line 10) carries *"an AGPL-3.0, self-hosted, single-tenant web application"* as the premise the stack was chosen against. It states the deployment shape and rules nothing about the domain or the console
- **Rests on:** [`docs/spec/v1-spec.md`](../spec/v1-spec.md) §1 (lines 26 and 48), which states the product fact — *"**Single-tenant, self-hosted** via `docker compose`. No multi-tenancy, billing, or hosted infrastructure"* — and defers hosted multi-tenant SaaS *"outside the map's destination outright"* (line 753). **This ADR does not restate that. It rules what §1 leaves open: what the domain holds, what the shell renders, and what a later tenancy change would actually cost**
- **Rests on:** [ADR-0167](./0167-a-design-corpus-a-live-read-cannot-produce-is-served-as-a-pinned-fixture-and-the-live-path-renders-the-honest-projection.md) §3, under which the `devMode` chip and the production chip are two paths by licence, not by accident
- **Not bound by:** [ADR-0073](./0073-an-operator-dial-carries-no-author-however-specific-its-target.md), the citation both deleted comments carried. It rules that an operator dial carries no author. The string `org` appears in it **zero times**
- **Narrows:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md) Decision, line 41, which enumerates *"org switcher"* inside the shell spec

## Context

`cmd/web/auth.go` stated this rule twice, and #1337 deleted both statements. One sat on
`injectChrome`, the other beside the literal — *"single-org deployment; static chip (ADR-0073,
switcher retired #33)"*. Neither statement survives; the constant does, at
`cmd/web/auth.go:1875`.

**The citation was wrong, not dead.** ADR-0073 is *An operator dial carries no author, however
specific its target*, Status Accepted, and it rules attribution on an `Annotation` and a mute. It
says nothing about organisations. [`docs/spec/comment-policy.md`](../spec/comment-policy.md) §4.7
grades this the worse failure: a dangling token *"names nothing a citation can reach"* and dies on
inspection, while a citation that resolves *"can invert the meaning of the surviving line"* and
survives a file-existence check. §8.3's rider is the operative one — **a source suppresses only
where it states the rule** — and ADR-0073 does not. The three issue ids beside it fail the same
test: #33 is *Does the claim/attestation/determinacy standard generalise to the other nine
signals?*, #28 is the Coverage screen prototype and #27 the CAIDA bundling question — all live,
none this rule's source, which §4.7 names a design-collision id.

**What the tree holds.** One production site fills the chip: `cmd/web/auth.go:1875`,
`Org: "self-hosted"`, a Go string literal with no read behind it. `chromeVM.Org`
(`cmd/web/chrome.go:20`) is its only carrier, and `design-system/templates/shell.tmpl:125` renders
it as `<span class="sh-orgchip">{{.Org}}</span>` — a span, styled at `:29`, with no control, no
form and no link. The second site is the `devMode` branch at `cmd/web/auth.go:1831`, which calls
`chromeFromFixture` (`cmd/web/chrome.go:203`); that reads `design-system/fixtures/fixtures.json`
line 4534, `"org": "acmecorp"`. **The two sites do not agree**, and under ADR-0167 §3 they are not
required to: the fixture is the design corpus and the live path renders the honest projection.

**No switcher markup ships.** `shell.tmpl` holds one residue — `"sh-org"` in the popover array at
`:312`, guarded by `if (!d) return;` at `:313`, and the word *org* in the JS comment at `:295`.
No element carries that id, so the code is inert. The switcher exists only in the kit:
`design-system/components/navigation/OrgSwitcher.jsx`, its `.d.ts` and its `.prompt.md`, plus
`TopNav.jsx:40`'s `orgs` branch, whose sibling at `:41` is the static chip the Go shell ported.
`ConsoleApp.jsx:95` passes no `orgs`, so the reference shell renders the static branch too.

**No tenancy column exists.** Seven of 39 files in `db/queries/` carry `account_id`, and all seven
are per-account auth state — sessions, personal tokens, recovery codes, password resets, SSO
identities, invites, message reads. Not one corpus query file — `subjects.sql`, `span.sql`,
`measurement.sql`, `custody.sql`, `seeds.sql`, `signals.sql`, `zone.sql`, `vantages.sql` — carries
any scoping column. The deployment's own configuration is a singleton by construction:
`db/migrations/23000_instance_config.sql:12` is `id BOOLEAN PRIMARY KEY DEFAULT true CHECK (id)`.
`CONTEXT.md` defines no `Organisation`.

## Decision

> **verge-asm models no organisation. One deployment serves one operator's estate. The console
> shell renders a fixed chip reading `self-hosted` and ships no switcher. The chip is a design
> element the shell's layout needs, filled with a constant. It is NOT a placeholder for a model
> that is coming.**

### 1. There is no `Organisation`, and the word is already spent

No `Organisation` type, no `organisation` table, no organisation column, no tenancy predicate on
any read. The chip's value is not a projection of anything.

The domain already spends the word. `CONTEXT.md:220` has the operator *"searching the org-name
box"*, and that org is a **registry** organisation — the string handed to ARIN
(`internal/proposer/arin.go:91`) and CAIDA (`internal/proposer/caida.go:50`) to produce a
`Proposal`. It is a third party's name for a network, never a tenant of this instance. A reader
who meets `chromeVM.Org` and reads it as that term is reading a fourth thing.

### 2. The chip is a constant, and a constant is the honest rendering

The shell's brand-then-context-then-nav row needs a context slot. `self-hosted` fills it with the
one true fact about the deployment. A control implies a choice, and there is exactly one estate, so
a control would offer a choice that does not exist. **The chip is finished, not stubbed.**

### 3. Tenancy is not a UI change, and this is the clause a later session needs

Because **no read carries a tenancy predicate**, adding an organisation is not scoped by the
console. It reaches every corpus read, every derivation and every message: a scope column on the
subject, span, measurement, custody, seed and signal tables and the predicate on each read of them;
the derivation and comparison path, whose comparability keys would have to admit the scope;
`internal/estate` membership and `internal/queue`'s folds; every `Signal` census, which is
re-derived live; delivery and report bodies, which name subjects; the backup allowlist
(ADR-0161); and the auth layer, which today knows accounts and roles and no scope at all.

**So nobody adds the switcher first.** A switcher over an unscoped corpus is a control that
appears to partition an estate it cannot partition — the false-reassurance failure, arriving
through the chrome. The order is: rule the tenancy model, scope the reads, and only then give the
chip a control.

### 4. The kit's switcher is kit IA, and the product refuses it

`OrgSwitcher.jsx` and `TopNav.d.ts:8`'s *"Org context chip (MSPs run many orgs)"* are the kit
describing a product this is not. v1-spec §1 already governs this: the design system is canonical
for the visual layer, *"Its IA and vocabulary are not"*, and where the kit and `CONTEXT.md` collide,
`CONTEXT.md` wins. The chip's **look** is the kit's and is kept. Its **affordance** is not.

## Consequences

- **The deleted comments do not come back.** The rule has a document, and the citation that pointed
  at the wrong ADR is retired rather than repaired.
- **ADR-0110's line 41 is now wrong at its own site** and must be withdrawn there under
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
  `ConsoleApp.jsx:95` passes no `orgs`, so the shell spec never instantiated the switcher the
  sentence names. This ADR does not edit that file.
- **`chromeVM.Org` keeps a name that collides with the registry-org term.** Renaming it is a code
  change, not a ruling.
- **The `sh-org` popover wiring at `shell.tmpl:312` is dead code**, inert behind its own guard.
- **No migration, no query and no golden moves.** The ruling describes the tree as it stands.
- **A future tenancy decision supersedes this ADR at this file**, and its own cost is §3.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Model an `Organisation` now, against a future need** | It prices the whole of §3 today for a destination v1-spec §1 line 753 puts *"outside the map's destination outright"*. A scope column on every corpus table enters the comparability key, so every `Span` becomes keyed on a dimension with exactly one value — an ADR-0007 term that partitions nothing while every read, fold and golden carries it. And a scope guessed before its requirement is guessed wrong: an MSP boundary, a business-unit boundary and a customer boundary are three different partitions of the same estate |
| **Remove the chip from the shell** | It deletes a fact the operator uses. `self-hosted` states the deployment posture on every screen, and v1-spec §1 makes that posture load-bearing — *"the instance is a high-value target"*. It also breaks the shell's layout contract against `TopNav.jsx:41` and the `screenshots/` ADR-0110 makes the visual ground truth, for a saving of one span |
| **Render the deployment's own hostname in the chip instead of a constant** | The hostname is not known. `VERGE_EXTERNAL_URL` (`cmd/web/main.go:131`) and `VERGE_PUBLIC_URL` (`cmd/worker/main.go:190`) both default to the empty string, so the common deployment has none to render and the chip would be blank. The remaining source is the `Host` header, which a fronted deployment lets a proxy set — [ADR-0159](./0159-an-unnamed-proxy-is-never-trusted-so-the-client-ip-is-the-immediate-peer-and-a-fronted-deployment-must-name-its-proxies.md) refuses to trust an unnamed proxy for a weaker fact than this. A chip that varies by request path is a chip whose value means nothing |
| **Ship the kit's `OrgSwitcher` bound to a one-element list** | A control that offers one option is an affordance that promises a second. It invites the reading §3 exists to kill — that the model is coming and only the UI is behind — and it puts a live control in front of reads that carry no scope |
| **Keep the chip and re-cite ADR-0073, repairing nothing** | The citation resolves, so a reader follows it to a ruling on annotation authorship and concludes the rule was decided somewhere they cannot find. That is the §4.7 failure, not a repair of it |
