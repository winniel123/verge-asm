# The aperture statement

- **Status:** Ready for `/to-tickets` — the terminal artefact of [SPEC: the aperture statement ships, and its port-tier line tells a name-only estate what it does not measure](https://github.com/winniel123/verge-asm/issues/1882)
- **Ticket:** [#1889 Write the aperture statement SPEC](https://github.com/winniel123/verge-asm/issues/1889)
- **Originating defect:** [#1854 The specified port-tier aperture line does not ship, so a name-only estate is told nothing](https://github.com/winniel123/verge-asm/issues/1854)
- **Decisions assembled:**
  - [#1883 the seven inputs](https://github.com/winniel123/verge-asm/issues/1883)
  - [#1884 the two lists](https://github.com/winniel123/verge-asm/issues/1884)
  - [#1885 the rules figure](https://github.com/winniel123/verge-asm/issues/1885)
  - [#1886 the frame](https://github.com/winniel123/verge-asm/issues/1886)
  - [#1887 the port-tier copy](https://github.com/winniel123/verge-asm/issues/1887)
  - [#1888 the stale counts](https://github.com/winniel123/verge-asm/issues/1888)
  - [#1890 the vantage-class line](https://github.com/winniel123/verge-asm/issues/1890)
  - [#1896 the `Vantage class` glossary entry](https://github.com/winniel123/verge-asm/issues/1896)
  - [#1898 the frame's promise](https://github.com/winniel123/verge-asm/issues/1898)
  - [#1903 the vantage-class remedy](https://github.com/winniel123/verge-asm/issues/1903)
  - [#1906 seven, not eight](https://github.com/winniel123/verge-asm/issues/1906)

This document is the map's destination. It is a plan-only spec, complete enough to hand to an
implementation session without that session re-litigating a decision the map already made. It does
not restate reasoning. A decision lives in exactly one place, its ticket. This document assembles
and narrates.

Every file and line anchor below is a **dated record** as of this document's composition
(2026-09-13). Verify an anchor against the tree before you edit.

Consult the `verge-asm-design` skill before you write any markup.

**How to read this**

1. Start at §1 for the defect and the shape of the fix.
2. Read §2 for the frame. Every line obeys it.
3. Read §3 and §4 for the two lines whose copy is fixed.
4. Read §5 and §6 for the computed value and the tests.
5. Read §7 for the four repairs that ride with the statement.
6. Read §8 for the acceptance criteria an implementation must meet.
7. Read a decision ticket for the reasoning behind any one rule.

---

## 1. The defect, and the shape of the fix

An estate that declares only a name scope measures nothing at the `hot` tier. Nothing on the
product says so. Every resolved address derives `ThirdParty`, and the custody gate refuses it
before the vantage class is read
([ADR-0019](../adr/0019-the-probing-gate-is-total-over-an-address.md)). The refusal is correct. The
silence is the defect ([#1854](https://github.com/winniel123/verge-asm/issues/1854)).

The surface that answers this is specified already and is drawn by nothing.
[ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md) names the **aperture statement**
and writes its port-tier line's figures.
[ADR-0095](../adr/0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)
adds the second figure and records against itself that *"the surface this ADR rules on has never
been drawn"*. `docs/research/safe-active-probing.md` §2.4 restates it. No template renders it.

**The fix is to build it.** The statement is a ledger on `/coverage`, one row per aperture input,
seven rows. This document fixes the frame, writes two of the seven rows in full, names the
computed value and its two renderers, and commissions the tests.

### 1.1 What this spec does not do

- It mints **no** `Coverage` element, **no** term, and **no** ADR. Triage round 2 of #1854 closed
  all three, because ADR-0095 already made the trade-off.
- It puts **no** estate count on the statement. The statement counts our own lists and our own
  rules. #44 decision 7 bars a count or a proportion of the operator's estate.
- It does **not** widen `unread` to absorb the second figure. ADR-0095 refuses that fusion by name.
- It puts **no** clock in any predicate ([ADR-1806](../adr/1806-a-census-is-computed-once-from-a-cause-frozen-basis-when-the-admitting-tier-has-drained.md)).

### 1.2 What it answers, in #1854's own words

| #1854 asked | The statement answers |
| --- | --- |
| An operator whose `hot` tier measures nothing can find out that it measured nothing. | The port-tier line reads `38 of 38 sensitive pairs unread` on a name-only estate (§3). |
| The statement names the cause, so the operator knows which act would change it. | The Remedy cell reads `Declare an address scope` with the reason that names both levers (§3.4). |
| A `hot` dispatch that enqueues no job does not render as complete with no further statement. | `Drained` gains a clause naming the surface that speaks (§7.2). The `Dispatch` row is not the surface. |

---

## 2. The frame

The frame is a **column contract**. It binds all seven rows, including the five whose copy is not
written here.

### 2.1 Shape and placement

The statement is a **four-column ledger**: `Input · Cadence · State · Remedy`. It renders as one
**full-width card above** Coverage's two-column grid. The meters card stays below and to the left.

A human chose this from three drawn takes on branch `chore/1886-aperture-statement-prototype`,
file `prototypes/aperture-statement/index.html`
([#1886](https://github.com/winniel123/verge-asm/issues/1886), PR
[#1899](https://github.com/winniel123/verge-asm/pull/1899)).

**Two alternatives were drawn and lost.** Seven free blocks with no shared columns lost to the
contract. Promoting the port-tier line into a headline over a roster of six also lost. It carried a
second cost the drawing made visible. A roster leaves no room for the per-line remedy §2.3 makes
mandatory. **Promotion and the remedy rule are incompatible.** A later session does not re-litigate
either. Both were built and rejected.

### 2.2 The column contract

Every row answers every column.

| Column | Holds |
| --- | --- |
| **Input** | The input's name. Fixed for all seven by §2.5. |
| **Cadence** | The cadence at which the input is re-asked, plus a reason line. |
| **State** | A state chip, then an **optional figure block**, then a detail line. |
| **Remedy** | A link with a label, or `none`. Either carries one reason sentence. |

**A state is not always an on/off.** Most inputs carry no operator toggle. The Cadence and State
cells state what is there, whatever shape it has.

**No fifth column is added.** The port-tier line's three figures stack inside its State cell
(§3.2). A figures column would sit empty on six of seven rows, which is the shape that made that
line read as an exception.

**Reasons ride the cell, never a tooltip.** All four cells render their reason inline.
`design-system/docs/DESIGN-NOTES.md#content-fundamentals-fixed--from-the-brief-not-restyled` records
[ADR-1875](../adr/1875-an-undo-control-renders-the-exact-lookup-instant-against-the-relative-timestamp-convention.md)
refusing a hover tooltip on the ground that it is unreachable by touch and keyboard.

### 2.3 The empty cell

**An empty cell renders `none` and its reason.** It is never blank, and the column is never
dropped. `none` is the literal shipped word.

**One word serves every empty cell, and a second is not needed.** Before the first batch, every
input has a declared value. Five are seeded by a migration or compiled into the release. The two
inputs folded from the `Seed` list are the custody gate and the control-probe population. Both fold
over an **empty** `Seed` list. An empty declaration is a declared state, not an absent one. So no
cell ever means *no value has been declared yet*. `none` carries only the case it already covers:
**no such value exists**.

Measured on this branch. `db/migrations/18801_measurement_scan.sql` and its eight siblings seed
every `Scan` row with a cadence. `db/migrations/18800_measurement_vantage.sql` seeds one
vantage. `db/migrations/` inserts no `Seed` row at all.

**Every line carries a Remedy cell.** A remedy on one row of seven teaches the operator that the
other six are not actionable ([#1890](https://github.com/winniel123/verge-asm/issues/1890)).

### 2.4 The source rule

**Every line reads declared configuration. No line reads a batch.**

A measured read makes a line non-constant and falsifies ADR-0044 decision 10's premise. One source
rule across seven lines is cheaper to hold than per-line provenance.

**The address-coverage test has exactly one binding.** A line that must know whether a declared
address scope covers an address calls `addressScopeCovered` (`cmd/web/vantageclass.go#server.addressScopeCovered`).
`cmd/web/vantageclass.go` refuses a second by name:

> One binding serves batch gating and every render, so a second predicate is refused (#711).

The statement computes no coverage test of its own.

**A fold over a derived class is not a second predicate.** §4.2's two booleans are folds over
`addressScopeCovered`'s output, not new coverage tests. A reader who meets a new boolean will reach
for #711. It is not engaged.

**The line discloses no history.** A reclassification caveat on the element built to be constant is
the estate count #44 decision 7 refuses, in a new costume.

### 2.5 The seven rows, in order

The Input column is fixed. The order is the ledger's, and `docs/spec/v1-spec.md#32-seeds--aperture` now states
the same seven in the same order, so the two lists compare member by member.

| # | Input | Copy fixed here |
| --- | --- | --- |
| 1 | Enabled sources | No — §9 |
| 2 | **Port and transport tiers** | **Yes — §3** |
| 3 | The custody gate | No — §9 |
| 4 | The queried qtype set | No — §9 |
| 5 | The TLS candidate set | No — §9 |
| 6 | **`Vantage class`** | **Yes — §4** |
| 7 | The control-probe population | No — §9 |

**An input is named for what an operator moves, not for the field that records it**
([ADR-0079](../adr/0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)).
This rule is not decoration. `v1-spec.md` §3.2 was wrong on three of these seven names. All three
drifted the same way, from the input toward the field that records it. Five of the seven are named
by record content and two by control. So a session with no rule in front of it renames the two odd
ones ([#1906](https://github.com/winniel123/verge-asm/issues/1906)).

**Seven, not eight.** The queried address scope is a **lever on** the custody gate, not a peer of
it. ADR-0079 decided this by name. `docs/adr/0047-…md:446` states the mechanism in general terms.
And the code carries one address dimension rather than two — `hotScopeRecord` and `coldScopeRecord`
each hold a single `Addresses` field (`internal/scan/hot.go#hotScopeRecord`,
`internal/scan/cold.go#coldScopeRecord`).

---

## 3. The port-tier line, in full

Source: [#1887](https://github.com/winniel123/verge-asm/issues/1887), with the figure placement
from [#1885](https://github.com/winniel123/verge-asm/issues/1885) and the union semantics from the
map's ruling 7.

### 3.1 The four cells

| Cell | Copy |
| --- | --- |
| **Input** | `Port and transport tiers` |
| **Cadence** | `daily · monthly`<br>reason: *"hot daily, cold monthly. Release-coupled: no operator dial exists."* |
| **State** | chip `hot on · cold off · udp no flag`<br>the figure block (§3.2)<br>detail: *"The cold tier's state is the shadow of an empty scope list, not a switch. UDP has no flag at all."* |
| **Remedy** | A link or `none`, plus one reason sentence (§3.4) |

The State cell holds the chip, then the figures, then the detail, in that order.

### 3.2 The three figures, in this fixed order

1. `{N} of 38 sensitive pairs unread`
2. `0 of 38 sensitive pairs the instrument cannot report as reached`
3. `0 of 17 rules unevaluable`

All three ship **verbatim** from ADR-0044 and ADR-0095. The three stack in a sunken sub-block
inside the State cell, under the chip.

**The order is a contract, not a layout.** Figures 1 and 2 partition one denominator, which is
ADR-0095's strongest reason for one line, so they stay adjacent. Figure 3 runs over a different
population with a different denominator, so it sits last. Putting figure 3 between them separates
the partition the single line exists to keep together.

**Figure 2 ships as a full sentence**, never as a label beside a value. Splitting a predicate from
its denominator is the shape ADR-0095 refused across two lines, and the same argument reaches two
cells.

**Figures 1 and 2 are not in conflict, and never were.** `docs/spec/v1-spec.md#63-coverage` separates
them by which side of the recorded scope a pair sits on. Figure 1 counts pairs **outside** the
recorded scope, an invitation the operator can act on. Figure 2 counts pairs **inside** it that the
instrument cannot report as reached. UDP is never probed, so no UDP pair is ever inside the
recorded scope, and figure 2 is `0`
([ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md)).

### 3.3 The substitution rule

```
N = 38   when no address scope is declared and no custody extension is on
N = 5    otherwise
```

**The numerator has two values in v1, never three.** With no address scope and no custody
extension, no address is walked, so no pair is read.
[ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union puts every TCP sensitive pair inside the
hot tier by construction. So with either lever present, only the 5 UDP pairs stay unread. There is
no partial case. The hot tier's enablement is the shadow of a non-empty scope list rather than a
switch ([#1883](https://github.com/winniel123/verge-asm/issues/1883)).

Nothing else moves any figure on this line in v1.

**The numerator reads declared configuration alone** — an address scope exists, or a custody
extension is on. It never reads what the tier actually targeted. A measured emptiness would make
the line non-constant.

**No copy is specified for `0`.** That state is unreachable while no UDP tier exists, and the floor
is `5`. Writing copy for an unreachable branch invites a later session to read the branch as
evidence the state is reachable. §6.2 commissions a test instead.

### 3.4 The remedy rule

| N | Remedy | Reason sentence |
| --- | --- | --- |
| `38` | link `Declare an address scope` → `/scope` | *"No declared scope reads a sensitive pair. An address scope, or a custody extension on a name scope, moves this figure."* |
| `5` | `none` | *"The 5 pairs still unread are UDP. No tier reads them, and no setting opens one."* |

**A pointer ships only where an act genuinely exists.** ADR-0044 permits this line a pointer *"because
here an action genuinely exists and recommending it tells the operator nothing false about
themselves."* On a healthy estate no act exists, because UDP carries no flag at all. A button to a
screen holding no relevant control is #1854's silence in a new costume.

**`Declare scope` is refused as the label.** `cmd/web/auth.go#firstRunChecklist` ships that label one card away,
on Coverage's own day-one checklist. It is true of a **name** scope, and a name scope moves nothing
on this line. The reason sentence carries the custody-extension lever instead, because the extension
is declared on the same screen (`design-system/templates/scope.tmpl#scope`).

### 3.5 The two renderings

**Name-only estate** — #1854's case:

```
Input     Port and transport tiers

Cadence   daily · monthly
          hot daily, cold monthly. Release-coupled: no operator dial exists.

State     [ hot on · cold off · udp no flag ]
          38 of 38 sensitive pairs unread
          0 of 38 sensitive pairs the instrument cannot report as reached
          0 of 17 rules unevaluable
          The cold tier's state is the shadow of an empty scope list, not a
          switch. UDP has no flag at all.

Remedy    [ Declare an address scope ]  → /scope
          No declared scope reads a sensitive pair. An address scope, or a
          custody extension on a name scope, moves this figure.
```

**Healthy estate** — the same install plus one address scope. Two cells differ, and no others:

```
State     38 of 38 sensitive pairs unread   →   5 of 38 sensitive pairs unread

Remedy    [ Declare an address scope ]      →   none
          No declared scope reads…          →   The 5 pairs still unread are UDP.
                                                No tier reads them, and no setting
                                                opens one.
```

### 3.6 Three things the prototype draws that do not ship

1. **The provenance line.** The prototype draws `ADR-0007 · ADR-0025 · ADR-0044` under each Input
   cell. The shipped Input cell reads `Port and transport tiers` alone. The design system addresses
   the operator about their estate, never about our decision record.
2. **ADR-0044's `verge-core` prose.** It explains the **denominator**, not the state, and the only
   cell that could hold it is the figure column §2.2 refuses. It lands in this document instead:
   ports outside `verge-core` produce no `Service`, so nothing on them is measured, evaluated or
   counted.
3. **The label `Declare scope`.** See §3.4.

---

## 4. The `Vantage class` line, in full

Source: [#1890](https://github.com/winniel123/verge-asm/issues/1890) for the cells and
[#1903](https://github.com/winniel123/verge-asm/issues/1903) for the remedy. The map's ruling 2 set
this spec's depth at the frame plus the port-tier line. This line's copy was fixed afterwards by two
tickets, so it is carried here rather than left in a closed tracker thread.

### 4.1 The four cells

| Cell | Copy |
| --- | --- |
| **Input** | `` `Vantage class` `` |
| **Cadence** | `none`<br>reason: the class is derived where it is used, so it carries no currency and needs no cadence |
| **State** | The derived class set, with a count per class: `1 internet · 2 internal`, `2 internal`, `unverified` |
| **Remedy** | §4.2 |

**The count is over declared vantages**, which is our own list, so #44 decision 7's bar on an estate
count does not reach it. It is the only cell that separates a one-prober install from a ten-prober
one. Both read `internet, internal` without it.

**The Cadence cell is `none` and not *every batch*.**
[ADR-0028](../adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) ties cadence to currency.
A value derived at the point of use is never stale. `CONTEXT.md`'s *re-verified every batch* is
already withdrawn for the same reason
([#1896](https://github.com/winniel123/verge-asm/issues/1896), PR
[#1904](https://github.com/winniel123/verge-asm/pull/1904)).

**The State cell reads the live derivation** through `addressScopeCovered` (§2.4). No row stores a
class. `internal/scan/scan.go` marks the `vantage.class` column vestigial, and
`cmd/web/auth.go#server.dashboardData` records that the chip is derived per read.

### 4.2 The remedy — a total function over two legs

The domain is not the class set. The declared class set has **eight** members once mixtures are
counted, and a clause per set covers none of them. The domain is the two `Exposure` legs
([ADR-0017](../adr/0017-exposure-needs-both-legs.md)), each an existential over the derived class:

- `hasInternet` — some declared vantage derives `internet`
- `hasInternal` — some declared vantage derives `internal`

`unverified` stops being a clause. It becomes what it is, a member that starts no leg
(`internal/queue/vantageclass.go#widenedClasses`).

| `hasInternet` | `hasInternal` | Remedy | Reason, in the cell |
| --- | --- | --- | --- |
| yes | yes | `none` | *"A vantage reads from each side of your boundary, so no class is missing."* |
| yes | no | `Provision a prober inside your estate` → `/settings?tab=vantages` | *"No declared address scope covers any prober, so no vantage starts the internal leg. Run a prober inside your estate, then declare its egress as an address scope."* |
| no | yes | `Provision a prober` → `/settings?tab=vantages` | *"No vantage presents an address outside your declared scopes, so no vantage starts the internet leg. `Exposure` needs an outside observer, unconditionally."* |
| no | no | `Provision a prober` → `/settings?tab=vantages` | *"No vantage presents an observed address, so neither leg has a reader. `Exposure` needs an outside observer first."* |

**Four cases, no configuration uncovered.** The acceptance is discharged by the shape rather than by
enumeration.

Four constraints a builder must not simplify away:

1. **`Add an internal vantage` is withdrawn.** No control makes a vantage internal. The prober form
   carries no class field (`cmd/web/probers.go#proberView`), and a vantage derives `internal` only when
   every address it presents is covered (`internal/exposure/exposure.go#VerifyClass`). On a name-only
   estate, adding a prober cannot move the cell.
2. **Row 2 must not lead with the egress declaration.** `design-system/templates/settings.tmpl#settings-vantages`
   tells the operator to declare a prober's egress as an address scope. Applied to an **internet**
   prober, that step flips it to `internal` and destroys the internet leg. The label names the
   necessary step, and the reason carries the second step in order.
3. **Rows 3 and 4 share one label and differ only in reason.** The act is identical. Two labels for
   one act would teach the operator that two different things are available.
   `Provision a prober` is the shipped card title verbatim
   (`design-system/templates/settings.tmpl#settings-vantages`). *"Exposure needs an outside observer,
   unconditionally"* is `cmd/web/auth.go#firstRunChecklist` verbatim.
4. **Row 4 is a fresh install, never an empty list.**
   `db/migrations/18800_measurement_vantage.sql` seeds one `local` vantage at `unverified` per
   install. The empty vantage list is unreachable and needs no copy.

**The target is real.** `cmd/web/handlers.go#server.handler` redirects `/settings/vantages` to
`/settings?tab=vantages`, and `message.KindVantageClass` already routes there
(`cmd/web/messages.go#messageLink`).

---

## 5. The computed value, and its two renderers

**The statement's value is computed once.** `/coverage` and the v1 API are two renderers of one
computation. An HTML-only statement moves #1854's silence rather than ending it.

### 5.1 The home

`cmd/web/cold.go#server.coveragePage` holds `apertureMeters`, the shipped precedent for exactly this shape. Build a
sibling of it that returns the seven rows as a typed slice.

One function feeds both renderers today. `cmd/web/cold.go#server.coveragePage` renders the meters into the
template, and `cmd/web/api_v1.go#server.apiCoverage` renders them into JSON.

Do not invent a second pattern beside a working one.

### 5.2 The two renderers

| Renderer | Site |
| --- | --- |
| `/coverage` | `design-system/templates/coverage.tmpl`, a new full-width card above the grid (§2.1) |
| `/api/v1/coverage` | `cmd/web/api_v1.go`, a new key on `apiCoverageResponse` beside `meters` |

**The API addition needs no compatibility ruling, and this was measured rather than assumed.** No
document in the tree states a field-stability contract for `/api/v1`.
[ADR-0123](../adr/0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md)
already licenses the addition in general terms: `/api/v1` endpoints are read mirrors of the reads
the HTML surface already wraps. The statement is such a read. No mutating verb is added, so the
bearer path's read-only property is untouched.

### 5.3 The statement and the meters coexist

The statement does not fuse with the per-`Seed` aperture meters. The meters count the estate over a
declared address range
([ADR-0120](../adr/0120-an-address-scope-meter-counts-what-the-batch-walked-over-its-declared-range-not-the-estate.md)).
The statement counts our own lists and is barred from an estate count. Fusing them puts an estate
count on the one element built to carry none.

They separate by placement and by micro-label. §7.4 moves the micro-label.

---

## 6. Derivation, and the three tests

### 6.1 Both denominators are derived, never typed

| Figure | Derivation | Live value |
| --- | --- | --- |
| `38` | `vergecore.Default().Count().Sensitive` (`internal/vergecore/vergecore.go#List.Count`, field at `:154`, filled at `:172`) | 38 |
| `17` | `len(signal.AllRuleNames())` (`internal/signal/corpus.go#AllRuleNames`) | 17 |
| `5` | `SensitivePairs()` (`internal/vergecore/vergecore.go#List.SensitivePairs`) filtered on `Transport == UDP` | 5 |

**The `5` numerator has no exported home.** `Counts.UDP` counts the UDP members of the **union**,
not of the sensitive half. The caller filters it
([#1884](https://github.com/winniel123/verge-asm/issues/1884)).

**A typed literal is the defect this rule exists to prevent.** ADR-0044's own prose corrected the
numerator four times.

### 6.2 The three tests an implementation ships

1. **The denominator assertion.** The rendered denominator equals
   `vergecore.Default().Count().Sensitive`, and the rules denominator equals
   `len(signal.AllRuleNames())`. This is the list-movement gate ruling 9 asked for. A typed `38`
   fails it the day the list moves.
2. **The floor test.** Figure 1's numerator is `38` or `5` and is never anything else. It is never
   `0` while no UDP tier exists.
3. **The domain-guard test.** Every rule sends configuration absence to `OutsideDomain` and never
   to `NotEvaluable`. It fails the day someone adds a rule that can never speak.

**No further list-movement test is commissioned.** Six tests already gate the two lists themselves,
two of them golden tests that re-parse the research notes and byte-compare against the shipped TSV
(`internal/vergecore/vergecore_test.go`, `internal/vergecore/golden_test.go#TestSensitiveHalfIsResearchNoteSection3` and `:204`,
`internal/signal/endpoint_test.go#TestEvaluateCorpusReturnsSeventeenRules`). Test 1 is the only new link needed, between the shipped
list and the rendered figure.

### 6.3 Why figure 3's numerator is the literal `0`

Deriving it was proposed and the code falsifies the premise. 16 `return OutsideDomain` guards sit
across the 17 rules (`internal/signal/rules.go#zoneDeclaredNameReturnsNameError.Eval`, `:118` are two of them). Configuration absence
routes to **outside the domain**, never to unevaluable. `NotEvaluable` arises only from measurement
outcomes, which §2.4 bars from this line.

So ADR-0095's *"does not move, and cannot"* is right as an absolute. Its reason is one the ADR did
not state: the domain guard, not the transport. A derived count that is provably always `0` would
fail on nothing, which is ruling 9's intent inverted. **Test 3 is the deliverable, not a derived
figure.**

---

## 7. The five repairs that ride with the statement

Two are prose and land in this document's own pull request. Three are markup or template copy and
land as implementation tickets.

### 7.1 ADR-0044's figure-constancy claim is struck — **landed here**

`docs/adr/0044-a-one-off-measurement-has-no-currency.md#narrow-is-not-the-failure-narrow-and-silent-is` claims that **every figure on this
line is unchanged on every shipped configuration**.

The **pairs** figure falsifies that clause. Under the union semantics of ruling 7, a name-only
estate reads `38 of 38` and a mixed estate reads `5 of 38`. Those are two shipped configurations
with two values.

The strike is intra-document, so
[ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
intra-document unit covers it. **No new ADR. No relation marker. No `adr-review` cycle.**

Three neighbouring sites stay untouched. All three are correct as written:

- `docs/adr/0044-…md:202` — *"the numerator is unchanged"*, true of the rules numerator.
- `docs/adr/0095-…md:61` — *"Does not move, and cannot"*, true as an absolute (§6.3).
- `docs/research/safe-active-probing.md#24-continuous-vs-periodic--the-schedule-is-the-real-answer` — a restating site whose specifying site did not move.

### 7.2 `CONTEXT.md`'s `Drained` entry gains one clause — **landed here**

The entry already states two things. A tier which is enabled and admits nothing is drained and not
pending. The openings it would carry never arrive. It never says what the operator is told. That
silence is #1854's third acceptance bullet.

The added clause names the surface and not the copy:

> **The aperture statement's port-tier line tells the operator.** That line states the unread pair
> count and the act that moves it. The `Dispatch` row is not the surface.

### 7.3 `CONTEXT.md` separates *unevaluable* from `not-evaluable` — **landed here**

`CONTEXT.md` owns `not-evaluable` as one member of the census over a `Predicate domain`. It has
never defined *unevaluable*, which is how one word came to do two jobs. Figure 3 ships the word on
`/coverage`, so the model defines it:

> *Unevaluable* is a property of a **rule** over our own rule set — a rule that can never speak.
> `not-evaluable` is a value about one **subject**. The two counts are not comparable and never
> fuse.

### 7.4 The `Aperture` micro-label moves to the statement — **specified**

`design-system/templates/coverage.tmpl#coverage` carries the micro-label `Aperture` over the title
*"What the last batch walked"*, rendering the ADR-0120 meters. After §5.3 the two elements read
different sources and move at different rates. The card's title already says what it is, so the
micro-label is the part that overreaches.

- The **statement** takes the micro-label `Aperture`.
- The **meters card** takes `Address scopes`, which names the denominator its meters actually have
  under ADR-0120. Its title is unchanged.

### 7.5 The `Rules` card is retitled — **specified**

`design-system/templates/coverage.tmpl#coverage` titles a card *"Unevaluable this batch"*. Its rows read
*"N subjects in its domain could not be read this batch"* (`cmd/web/cold.go`), and its empty
state reads *"Every rule could evaluate"* (`:190`). That is a per-subject count wearing a per-rule
name, which ADR-0095 refuses by name at `docs/adr/0095-…md:347`. It sits one card from figure 3.

- Title: **"Rules waiting on a reading"**.
- Empty state: **"Every rule read every subject in its domain."**
- The rows are unchanged. Figure 3 is unchanged and ships verbatim.

*Unevaluable* then carries one meaning on the page, and it is the statement's.

---

## 8. Acceptance

An implementation meets this spec when all of the following hold.

1. `/coverage` renders a four-column ledger with seven rows, above the grid, and every row fills
   every column.
2. A name-only estate reads `38 of 38 sensitive pairs unread` on the port-tier row, beside
   `Declare an address scope` → `/scope`.
3. An estate with one address scope reads `5 of 38` on that row, beside `none` and its reason.
4. `/api/v1/coverage` returns the same seven rows from the same computation, with no mutating verb
   added.
5. Both denominators are read from `vergecore.Default().Count().Sensitive` and
   `len(signal.AllRuleNames())`. Neither is typed.
6. The three tests of §6.2 ship and pass.
7. No cell renders blank. Every `none` carries its reason, inline and not in a tooltip.
8. No copy on the statement states a count or a proportion of the operator's estate.
9. The two template repairs of §7.4 and §7.5 ship.

---

## 9. What is left for a later effort

**Five rows' copy.** Rows 1, 3, 4, 5 and 7 of §2.5 need their Cadence, State and Remedy cells
written against this frame. Their Input names are fixed and their facts are measured in
[#1883](https://github.com/winniel123/verge-asm/issues/1883). Each lands as its own ticket on the
implementation map.

Three facts those tickets inherit:

- Only **two** inputs carry a genuine operator toggle — enabled sources through `source_state`, and
  the custody gate through `seed.custody_extension`. The `cold` tier's enable is derived from a
  non-empty scope list rather than set.
- Only `dns` and `zone` have cadence setters (`db/queries/measurement.sql#GetDnsCadenceSeconds`, `db/queries/zone.sql#GetZoneCadenceSeconds`).
  Every other cadence is release-coupled.
- The two `Offer`-class inputs and the control-probe population are deliberately un-toggleable. An
  offer the operator can narrow is a finding the operator can silence
  ([ADR-0030](../adr/0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md)).

**Out of scope for this spec and for its map:**

- **Opening a UDP tier.** It is the only act in v1's reach that moves figure 2 off `0`. It is a
  separate effort with its own consent and payload questions.
- **Pinning a historic `Vantage class`** — [#1895](https://github.com/winniel123/verge-asm/issues/1895).
  Every read applies the **current** predicate to historic rows, so the class has no history. The
  statement states present configuration and claims nothing about a past batch, so it does not wait
  on the repair.
- **The Vantages tab's class tag** — [#1908](https://github.com/winniel123/verge-asm/issues/1908),
  repaired. The tab rendered the vestigial `vantage.class` column, which no query ever writes, so it
  tagged every vantage `unverified`. It now derives the tag through the one `addressScopeCovered`
  binding. §4.2's remedy held either way. **The card title joined that ticket.** Row 2 sends the
  operator there for the **internal** direction, and rows 3 and 4 quote the title verbatim. So the
  title and the copy moved together, and the title is now `Provision a prober`.
- **The stale `Scan` count** — [#1911](https://github.com/winniel123/verge-asm/issues/1911),
  repaired. Nine `Scan` rows ship, and `docs/spec/v1-spec.md` §3.4's table carries all nine.
- **ADR-0144's stale line-anchored citation** — [#1912](https://github.com/winniel123/verge-asm/issues/1912).
