# ADR-0167: a design corpus a live read cannot produce is served as a pinned fixture, and the live path renders the honest projection

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1333 ADR gaps: cmd/web/devfixtures.go](https://github.com/winniel123/verge-asm/issues/1333), gap 2; and [#1339 ADR gaps: cmd/web/seeds.go](https://github.com/winniel123/verge-asm/issues/1339), gap 1 — one rule, filed twice
- **Sweep PRs that deleted the comments:** [#1335](https://github.com/winniel123/verge-asm/pull/1335), [#1340](https://github.com/winniel123/verge-asm/pull/1340)
- **Rests on:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md), which already rules the never-fabricate half — a screen with no backing data ships a design-system empty state, *"never fabricated data"* (Decision, line 40; Consequences, line 103). **This ADR does not restate that rule and does not re-rule it.** It rules only what ADR-0110 leaves open: what a dev build may serve *instead*
- **Rests on:** [ADR-0166](./0166-a-verge-dev-build-is-a-capture-affordance-and-every-gate-it-opens-is-unreachable-in-a-released-build.md), which rules `VERGE_DEV` affordances generally. A pinned fixture is one such affordance, and ADR-0166's bounds are not repeated here
- **Rests on:** [ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md), which makes `design-system/` an in-repo source of truth, so `fixtures.json` is a file this repo owns
- **Narrows:** [ADR-0120](./0120-an-address-scope-meter-counts-what-the-batch-walked-over-its-declared-range-not-the-estate.md), whose Consequences and Alternatives table state the never-fabricate sentence while attributing it to `design-system/SPEC-CHANGE.md`, a file that is not on disk
- **Not bound by:** [ADR-0158](./0158-a-read-only-console-screen-may-scope-its-rendered-rows-in-the-client-and-a-screen-that-submits-a-form-carries-its-scope-in-the-query-string.md), which rules where a *rendered* row set may be filtered. This ADR rules where the row set comes from, one step earlier

## Context

Two sweep tickets filed the same rule. #1333's gap 2 counted twelve statements of it across
`cmd/web/devfixtures.go`; #1339's gap 1 found it again at three sites in `cmd/web/seeds.go`. Both
attributed it to `design-system/SPEC-CHANGE.md`. That file is not on disk, and
[ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md)'s status line
withdraws the collision protocol that produced it.

**Both issue bodies frame the rule as "never fabricate". That framing is wrong, and it is corrected
on disk.** `docs/spec/comment-policy.md` §4.7 names the misfile by number: *"#1204 did exactly this:
it filed the never-fabricate rule as gap 2 of #1333, and `docs/adr/0110` line 74 states it."* The
substance holds and the pointer has since moved: line 74 is now the ADR-0109 bullet
[#1410](https://github.com/winniel123/verge-asm/issues/1410) struck through, and the rule reads at
lines 40 and 103 — *"design-system empty-states where a new screen has no backing data yet, never
fabricated data"*, in the Decision blockquote and again in Consequences. Two repairs already cite it
in the tree, `cmd/web/reports.go:25` and `cmd/web/settings.go:868`. **The never-fabricate rule has a
live home, and this ADR is not it.**

What neither ticket separated out is the other half of the same sentence, and that half is genuinely
unwritten: *the console does not merely refuse to invent the datum — under `devMode` it serves a
curated corpus in its place.* Refusing and substituting are different acts. ADR-0110 rules the
refusal. Nothing on disk rules the substitution, and `docs/spec/release-pipeline.md` — the one live
document that names `design-system/fixtures/fixtures.json` — names it only for version stamping
(`:315`, `:1191`, `:1199`).

**The mechanism as it stands today.** `devMode` is set once, from `VERGE_DEV`, at
`cmd/web/main.go:107` and assigned at `:117`. Twenty-two `xxxFixtureData` builders exist —
twenty-one in `cmd/web/devfixtures.go` (`:404` through `:2340`) and one at
`cmd/web/settings_fixtures.go:404`. **Not seventeen**, which is #1333's count. Every one of them is
reached only from inside an `if s.devMode` branch — nineteen call sites, among them `cold.go:131`,
`exposure.go:33`, `seeds.go:94`, `settings.go:274`, `auth.go:455` and `:1831`, and
`subjects.go:933`. The dev-only routes are registered inside one `if s.devMode` block at
`handlers.go:484-491`. **The gate holds with no exception.**

The corpus arrives by two mechanisms, and only one of them is what the deleted comments described.
Seven production sites read the package directly with
`fs.ReadFile(designfs.FS, "fixtures/fixtures.json")`: `chrome.go:188`, `devfixtures.go:1412`,
`:1903`, `:1975`, `:2198`, `:2318`, and `settings_fixtures.go:391`. Every other builder reads Go
package variables that **transcribe** a slice of the same file, and a drift test compares the two.
Seventeen such tests exist — fourteen in `cmd/web/devfixtures_test.go`, two in
`inventory_fixture_test.go`, one in `search_fixtures_test.go` — including
`TestScopeFixtureMatchesPackage` (`devfixtures_test.go:794`) and its siblings
`TestCoverageFixtureMatchesPackage` (`:326`) and `TestExposureFixtureMatchesPackage` (`:411`).

**The seeds.go comment was wrong about its own mechanism**, and the sweep inherited the error.
It said `seedsPage` *"serves the pinned fixtures.json → scope slice"*. It does not.
`scopeFixtureData` (`devfixtures.go:927`) reads `devScopeSeeds`, `devScopeCustody`,
`devScopeNameTree` and five more Go variables and never opens the JSON at runtime.
`TestScopeFixtureMatchesPackage` is the only thing connecting them. The surviving uncited line is
`cmd/web/seeds.go:93`.

## Decision

> **Where a screen's design corpus cannot be derived from live reads without inventing domain data,
> the `devMode` build serves a pinned slice of `design-system/fixtures/fixtures.json` and the live
> path renders the honest, usually emptier, projection. The two paths are separate. The live path is
> never bent to imitate the fixture.**

### 1. The qualifying test is a missing first-class read, not a sparse database

A screen qualifies only where the design's figure has **no first-class datum behind it**. ADR-0120's
Coverage meter is the worked case: the denominator is a first-class read and the numerator —
subjects the batch walked within a declared range — is not one yet. `cmd/web/seeds.go:397` states
the same shape for the custody census: *"No measured resolution numerator exists yet."*

An empty database is not a qualification. A screen whose reads all resolve and return nothing renders
the empty state ADR-0110 requires. **The fixture exists for the case where no read exists to be
empty.**

### 2. Two mechanisms, and the drift test is what licenses the second

**Read the package directly** where the shape permits it. `devfixtures.go:1902` states the reason:
*"Read straight from the package, so no second copy exists and no drift test is needed."* This is the
preferred form, because a corpus that exists once cannot drift.

**Transcribe into Go only with a drift test.** A transcription is a second copy of an authored
figure, and a second copy without a test is the fabrication risk arriving by the back door: the
screen would show a figure that is nobody's — neither the estate's nor the design's. **A
transcription with no drift test is a defect under this ruling**, not a style choice.

### 3. The live path is the product, and it never reads the fixture

No production read of `fixtures.json` sits outside a `devMode` branch. The live paths at
`cold.go:135`, `exposure.go:38` and `seeds.go:407` run their own reads and render what those reads
return: `exposurePage` renders `Withheld` when no vantage carries an internet leg, `coveragePage`
renders a census where ADR-0120's numerator is missing, and `renderSeeds` renders a zero custody
census. **These are the honest answers, and their emptiness is information.** The correct response to
a dev screen that is richer than the live one is to make the missing datum first-class, never to
widen the fixture's reach.

The one hybrid site is `applyInventoryFixtureCounts` (`cmd/web/inventory.go:481`), which overwrites
group totals on rows the live query produced. It sits inside the `devMode` branch at `:506-508` and
is pinned by `TestInventoryFixtureCountsMatchPackage`, so it is legal — and it is the shape to watch,
because one missed guard there is the failure this ruling exists to prevent.

### 4. What makes a fixture legitimate, and what makes it a lie

A pinned fixture is legitimate when **all four** hold.

| Test | Why |
| --- | --- |
| **Authored, not approximated.** Every figure traces to a slice of `fixtures.json` | An approximated datum shipped as fact is the drift the never-fabricate rule prevents (ADR-0110) |
| **Pinned.** Read from the package, or transcribed under a drift test | An unpinned copy becomes a third figure that no one authored |
| **Gated.** Reachable only through `devMode` | An operator who can see it will read it as their estate |
| **Displacing, not decorating.** It replaces the whole projection for that screen | A fixture mixed into a live row set produces a page where the operator cannot tell which figure is theirs |

A fixture becomes a lie the moment a value appears in it that is neither a live read nor an authored
design figure — the invented middle. That is the same act ADR-0110 forbids, arriving through the dev
build instead of the live one.

### 5. The bound: a capture input, never a source of truth an operator sees

A pinned fixture gives a capture, a template, and a review a stable subject. It is **input to a
picture of the console**, and authority for nothing else: not evidence, not a default, not a seed for
a real install. Nothing outside `cmd/web`'s dev branches may read it, no measurement may consult it,
and no operator-facing document may quote a figure from it as a fact about an estate. The
`-seed-fixtures` flag is the one legitimate crossing, and it refuses to run without `VERGE_DEV`
(`cmd/web/main.go:45-46`).

## Consequences

- **`cmd/web/seeds.go:93` and `cmd/web/devfixtures.go:25-27` gain a citation.** They are the two
  surviving uncited statements of this rule, and this ADR is their home.
- **ADR-0120 is repairable at two sites, and is not amended here.** Its lines `:64`-`:65` and `:79`
  attribute the never-fabricate sentence to `SPEC-CHANGE.md`. Under
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) the
  attribution is withdrawn at the site that states it, re-pointed at ADR-0110's Consequences for the
  refusal and at this ADR for the pinned-fixture half. That edit belongs to the ticket that owns
  `0120`.
- **One defect is exposed by §2 and is not fixed here.** `devRunningRunJobs`
  (`cmd/web/devfixtures.go:643-650`) transcribes the six job rows of
  `settings.scans.active[0].jobs` in `fixtures.json` (ids 912 to 917) and **no drift test compares
  them.** It is the only unpinned transcription in the tree.
- **`docs/spec/comment-policy.md` §4.7 needs a second row for the `SPEC-CHANGE` family.** Today it
  maps the whole family to ADR-0110, at a line number that has moved. That mapping is right for the
  refusal and incomplete for the substitution, which is why two tickets read it and still filed a gap.
- **A new screen is cheaper to leave honest than to fixture.** The four tests in §4 are the price of
  a fixture, and a screen that renders its real reads pays none of it.
- **Nothing about a live deployment changes.** No live read moves, no template moves, and no golden
  corpus is re-escrowed. This ADR states what the code already does at every site but one.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Derive the dev corpus from a seeded database instead of a fixture** — seed the estate so the live path produces the design's figures | The design's figures include values that no first-class read produces (ADR-0120's walked-subject numerator, `seeds.go:397`'s resolution numerator). Seeding cannot produce a number the model has no field for, so the seeder would have to write the figure directly — the fabrication, relocated into the database, where the drift test cannot see it |
| **Make the live path fall back to the fixture when a read is thin** | This is the failure the ruling names. An operator could not distinguish their estate from the design's, and the emptiness that `exposurePage`'s `Withheld` and `coveragePage`'s census exist to communicate would be papered over by exactly the invented data ADR-0110 forbids |
| **Drop the fixtures and capture the live screens as they render** | Costs the stable subject. A capture of a sparse estate is not a capture of the screen, and every reviewer would compare two things at once — the composition and whatever the database held that hour. `TestScopeFixtureMatchesPackage` and its sixteen siblings exist because the corpus must not move under the capture |
| **Transcribe every fixture into Go, dropping the runtime reads** | Turns seven single-copy corpora into seven second copies, each needing a drift test that does not exist yet. `devfixtures.go:1902` records the reason the other direction is preferred |
| **Read `fixtures.json` in the live path too, as a documented demo mode** | A demo mode an operator can enter shows fabricated figures to a human, which is the harm rather than a mitigation of it. Nothing distinguishes it from the fallback row above except intent, and intent does not reach the screen |
| **Rule this as an amendment to ADR-0110** | ADR-0110 rules the port and what a screen with no data shows. The substitution is a build-flag mechanism it never contemplated, and nothing about ADR-0110 has become false. Under ADR-0058's split that is a new ruling, not a correction |
