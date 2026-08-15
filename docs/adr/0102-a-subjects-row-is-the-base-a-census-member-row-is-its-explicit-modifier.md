# ADR-0102: A `Subjects` row is the base; a census member row is its explicit modifier, never a default

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#162 A member list and the estate listing must not share a row component by default](https://github.com/winniel123/verge-asm/issues/162)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md) and
[#74](https://github.com/winniel123/verge-asm/issues/74) already settled that a rule's census
member list and the `Subjects` listing are different objects and may not be the same object: a
member row may never carry a `Citation`; a `Subjects` row must. A member list carries a
denominator (its header count **is** `list.length`, exact) and no search
(#74 decision 8 — a filtered member view is two numbers over one population disagreeing on
screen, [#28](https://github.com/winniel123/verge-asm/issues/28)'s hazard). `Subjects` carries a
search (ADR-0072 decision 3 — refusing it manufactures a false absence at the text box) and no
denominator (estate completeness is unmeasurable — the "closest loss" in ADR-0072's own
alternatives table).

Both rows share real structural ground: a subject's key is a link that carries no state and
approves nothing, neither carries a per-row control, and both are ordered by the subject rather
than by attention
([#51](https://github.com/winniel123/verge-asm/issues/51) §4, ADR-0072's consequences, #74
decision 7). That is enough shared shape that a build session will reach for one row component
for both, exactly as [#51](https://github.com/winniel123/verge-asm/issues/51) §8 warned for a
different pair (the extension's census and a `Proposal` queue) and
[#123](https://github.com/winniel123/verge-asm/issues/123) then watched happen: Variant A drew
one table with one `state` column and one action column, and eleven addresses that were never
meant to carry a control acquired an `Approve` button apiece — "the per-address approval queue
[ADR-0002](./0002-ownership-gates-probing.md) rejected... arriving through a shared component
rather than through a decision."

\#51 §8 recorded the general shape for that pair: *if the two ever share a component, that
component must not carry per-row controls by default — the queue is the special case and the
census is the base.* That pair's hazard sits on one axis — an **affordance** (does the row grant
the operator an act?) — and the base/special-case split follows directly: the special case is
whichever shape is a strict superset of the other's capability, so the base is the shape with
nothing added, and sharing defaults to it.

This ticket's pair does not have that shape. Listed against the three axes that actually differ:

| | Census member row | `Subjects` row |
| --- | --- | --- |
| Search | refused (#74 decision 8) | required (ADR-0072 decision 3) |
| Denominator | locked to `list.length` (#28, #74 decision 1) | refused (ADR-0072, #28) |
| `Citation` | refused | required |

Every axis flips. Neither row is a strict subset of the other's behaviour — each has exactly one
thing the other lacks and lacks one thing the other has. "Which is the base" cannot be answered
by capability containment the way #51 §8 answered it for controls; it has to be answered by
asking which unconfigured render is the less dangerous mistake, since some render has to happen
when a build session reuses the component and forgets to say which list it is drawing.

## Decision

**A `Subjects` row is the base. A census member row is an explicit, atomic modifier that a
caller must request by name — never a default, and never three independently settable facets.**

**1. The base is the `Subjects` shape.** A row component with nothing configured renders: a
subject-key link with no per-row control, ordered by the subject, a `Citation` cell, search
available on the list it sits in, and **no denominator anywhere near it**. This is deliberate and
asymmetric with #51 §8's ruling, not an application of it: the two mistakes a forgotten
configuration can produce are not equally bad.

- **Base leaking onto a census member** (a build session forgets to switch modes on a `Signals`
  card) renders a `Citation` that duplicates the rule's own provenance and a search box atop a
  list whose header promises an exact count. Both are wrong, and both are the kind of wrong a
  screenshot catches: a search box where none belongs, or a header reading `2,904` above forty
  visible rows, is #74 decision 8's defect reproduced almost verbatim — visible, local, and caught
  by the same eyeball test #123 used to find its own two real rendering defects.
- **Member leaking onto `Subjects`** (the reverse forgotten switch) renders a `Subjects` screen
  with a denominator — a claim of estate completeness the product refuses everywhere else in the
  model. This is [#28](https://github.com/winniel123/verge-asm/issues/28)'s hazard in its most
  dangerous form: not a local rendering bug but a false assertion about the shape of the
  operator's whole attack surface, sitting on the screen [#10](https://github.com/winniel123/verge-asm/issues/10)
  made the second-position lookup for "what do I own." It also silently disables search, which
  ADR-0072 decision 3 already named as manufacturing a false absence at the text box — a subject
  the operator measured gone would read as "no results," which is a source erroring and producing
  an observation of absence.

  A wrong denominator on `Subjects` is not locally recoverable the way a stray search box on a
  census card is; it is exactly the class of defect this map keeps guarding against by name
  (ADR-0072's alternatives table, #51 §3 and §5, #74 decisions 1 and 9). The base must be the
  configuration that cannot produce it by omission, so the base is `Subjects`, not the member.

**2. The three axes are one discriminant, not three independent options.** A component that
exposed `showCitation`, `allowSearch` and `hasDenominator` as three separately settable props
would let a build session turn on any one without the other two — a `Subjects` row with a
denominator, or a member row with a `Citation`, are combinations neither ADR-0072 nor #74 ever
licenses, and independent toggles are exactly how #123's Variant A happened: one shared shape,
configured one property at a time, drifting into a state nobody chose deliberately. The component
takes a single required discriminant with exactly two values — call it the row's **kind**,
`subject` or `member` — and each value fixes all three facets atomically. There is no third value
and no partial state.

**3. `member` is requested, never inherited.** Whatever screen renders a rule's census
(`Signals`, per #74) must pass `kind="member"` explicitly on every row it draws. Nothing about
the component infers it from context (row count, presence of a rule prop, page URL); inference is
exactly the implicit channel #51 §8 and #123 both ruled out for controls, and it is no safer here
for a data claim than it was there for an affordance.

**4. The shared invariants stay shared, unconditionally.** Regardless of `kind`, a row's key is a
link and nothing else, carries no per-row control, and the list it sits in is ordered by the
subject and never by attention or a rule's verdict. These do not vary by mode because nothing in
ADR-0072 or #74 makes them vary — they are the actual common ground the two screens share, and
reusing one component for them is not a hazard the way the differing three axes are.

## Consequences

- **A build ticket for either screen states its row's `kind` and gets the other two facets for
  free, correctly, without restating them.** `Signals`' census card need only say `kind="member"`
  per row; it does not also have to remember to hide `Citation` and disable search — the mode
  bundle does that.
- **A code review has one thing to check on a new caller of this component: which `kind` did it
  pass, and does the screen match.** A `Subjects`-shaped row appearing anywhere outside the
  `Subjects` screen, or a `member`-shaped row anywhere outside a rule's census, is now a one-line
  diff to spot rather than a rendering to audit for three independent facts.
- **This binds [#161](https://github.com/winniel123/verge-asm/issues/161) and
  [#167](https://github.com/winniel123/verge-asm/issues/167)/[#168](https://github.com/winniel123/verge-asm/issues/168).**
  #161 draws a subject's own page, reached from a `Subjects` row's key link per
  [#74](https://github.com/winniel123/verge-asm/issues/74)'s `Citation`-hop precedent — that key
  link is the `kind="subject"` row's only interactive surface, and #161 should treat it as
  already specified rather than re-deriving what the link may carry. #167/#168's Seeds-screen
  prototypes render member-list-shaped surfaces (the `custody extension` census, a `Seed`'s
  address-scope membership) — those are census-shaped populations with their own denominator and
  no `Citation` and no search, matching `kind="member"`'s bundle, not `kind="subject"`'s, even
  though nothing on the Seeds screen is a rule's census in #74's sense. If either ticket needs a
  third `kind`, that is new ground this ADR does not cover and should be flagged rather than
  forced into one of these two.
- **This does not reach `Signals`' own layout, `Coverage`'s panels, or the Seeds screen's IA.**
  Only the row-component contract shared by two listings that were explicitly asked whether they
  could share one.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **Census member is the base; `Subjects` is the modifier** | The naive application of #51 §8's "poorer capability is the base" pattern, and wrong here because the axes are not a capability superset — member is not "`Subjects` plus restrictions," it differs on `Citation` and search in the *opposite* direction from denominator. Worse, its failure mode on omission is the map's worst-guarded defect: a forgotten mode switch renders `Subjects` with a denominator, asserting estate completeness the product refuses everywhere else |
| **Three independent boolean props** (`showCitation`, `allowSearch`, `hasDenominator`) | Reopens exactly what #123's Variant A demonstrated for the control axis: a shared shape configured one property at a time drifts into combinations nobody ruled on, because nothing stops a caller setting two of three correctly and forgetting the third. ADR-0072 and #74 never license a `Subjects` row with a denominator or a member row with a `Citation`, and independent props make both reachable by omission |
| **No shared component — two separate row implementations** | The closer call. It removes the mode-discriminant hazard entirely by removing the shared surface, and is a legitimate reading of "must not share a row component by default" taken literally. It loses because the two rows are not actually different in the parts that would justify duplicating them: link-only key, no per-row control, ordered by the subject are identical logic on both screens, and two independent copies of that logic is its own drift risk — the map's own merge notes name exactly this shape (an amendment landing in one copy and not the other) as a recurring defect class. A discriminated single component with no default carries the shared logic once and forces the differing three axes to be chosen, which is the cheaper failure mode to catch in review |
| **Default `kind` inferred from row count or URL** | Silent inference was the channel #51 §8 and #123 both closed off for the control axis; inferring `kind` from ambient context (e.g., "more than N rows means census") reintroduces an unversioned, screen-side heuristic in the exact shape ADR-0024 refuses generally |

## Thin ground

- **The severity ranking between the two omission failures is argued, not measured.** A missing
  `Subjects` denominator is ruled the worse defect because the map repeatedly names false estate
  completeness as its central hazard (#28, cited from ADR-0072's alternatives table, #51 §3/§5,
  #74 decisions 1 and 9) while nothing on the map gives comparable weight to a stray search box or
  a duplicated `Citation` on a census card. That comparison is reasoning from what the map has
  chosen to repeat, not from an incident or a user test. If a real build session finds the
  member-row omission is in practice invisible and ships unnoticed for a release, the severity
  ranking — and therefore which shape is the base — should be re-argued.
- **Whether #167/#168's Seeds-screen census populations are drawn from this same row component at
  all is asserted here, not ruled by either ticket.** They are member-shaped by the criteria this
  ADR states (denominator, no `Citation`, no search), but neither prototype ticket has been read
  against this ADR by its own author, and the consequences section above is a forward note rather
  than a joint ruling.
