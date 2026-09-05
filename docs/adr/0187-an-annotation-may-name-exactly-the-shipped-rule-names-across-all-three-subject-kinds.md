# ADR-0187: an `Annotation` may name exactly the shipped rule names, across all three subject kinds

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1301 ADR gaps: internal/signal](https://github.com/winniel123/verge-asm/issues/1301), gap 5
- **PR that deleted the comment:** [#1302](https://github.com/winniel123/verge-asm/pull/1302)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md), which makes an
  `Annotation` a declaration about one `(subject, signal-name)` pair whose whole effect is on the
  message — it fixes the **shape** of the key and bounds neither half — and
  [ADR-0092](./0092-an-operator-dials-movement-is-not-a-cause-and-an-annotation-never-lapses.md),
  which rules that an `Annotation` never lapses, its subject withdraws, and that a returning subject
  is the same pair. That ADR reasons about the pair throughout and bounds neither half either
- **Sibling of, and not ruled by:**
  [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md). It
  rules the **subject** half of the same key: a key is the thing denoted, and its normalisation may
  never move. This ADR rules the **signal-name** half. The two halves of one key, ruled separately,
  and neither contains the other
- **Bounded by:** [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md), whose v1 table
  enumerates the rules. This ADR bounds a set of names by the shipped rules and enumerates none of
  them itself, so a rule admitted there changes what an `Annotation` may name with no edit here

## Context

`internal/signal/corpus.go:56` and `internal/signal/rules.go:35` each carried the rule, in
declaration position, until #1302 compressed both. The `corpus.go` block:

```go
// AllRuleNames returns the names of all seventeen shipped rules in EvaluateCorpus
// order — the set an `Annotation` may name, so the Signals declare form offers
// exactly the rules that exist across all three subject kinds and nothing an
// operator could accept a firing on that never fires.
```

And `rules.go`, on the one-line alias `RuleNames`:

```go
// RuleNames returns the names of every shipped rule across all three subject
// kinds, in EvaluateCorpus order — the set an Annotation may name, so the Signals
// declare form offers exactly the rules that exist and nothing an operator could
// accept a firing on that never fires. It is AllRuleNames; the seventeen-rule set
// is the one the form and the acceptance guard both read.
```

**The rule was stated twice, in two files, and cited nothing either time.** #1302 deleted both
statements.

### The pair has two halves and only one was ruled

`ADR-0016` established the pair. `ADR-0092` reasoned about it at length. `ADR-0051` ruled the
**subject** half in full: the key is the thing denoted, and its normalisation may never move, which
is what makes a withdrawn-then-returning subject the same pair.

Neither ADR says anything about the **signal-name** half. `CONTEXT.md`'s `Annotation` entry states
the key, the six riders, the three barred routes and the withdrawal semantics, and it never says
which names may fill the second slot. So the half of the key that decides whether the acceptance can
ever take effect was the unruled half.

### There is no rule table, so nothing structural bounds the set

The rules are Go values compiled into the binary. `All()` returns five, `AllEndpointRules()` returns
ten and `AllServiceRules()` returns two — seventeen, in `EvaluateCorpus` order, which
`internal/signal/endpoint.go:62` records as ADR-0024's table order and forbids resorting.

The store carries no catalogue to point at. `db/migrations/20400_annotation.sql:22` declares
`signal_name TEXT NOT NULL` with one unique index on `(subject_key, signal_name)` and no foreign
key, because there is no table to reference. So an unbounded name half is not merely permitted by
the schema, it is the schema's default.

### What an unbounded name half would produce

An `Annotation` whose name is not a shipped rule is inert by construction. Its whole effect is to
mute the message on a `not-fired` → `fired` `Transition` for that pair (ADR-0016), and no such
transition can occur for a rule that never evaluates. The operator declares an accepted risk, sees a
row appear on `Signals`, and is protected from nothing.

**It is worse than inert on the console.** Three read paths degrade rather than refuse:

| Reader | What an unknown name produces |
| --- | --- |
| `annotationViews`, `cmd/web/signals.go:709` | `Orphan: !population[a.SignalName][a.SubjectKey]` is `true`, because no census carries that rule. The row is marked as naming a **withdrawn subject** |
| `signal.SubjectKindFor`, `internal/signal/corpus.go:25` | returns `""`, so `subjectHref` at `cmd/web/messages.go:341` falls to its default and the row's link is empty |
| `signal.SeverityFor`, `internal/signal/severity.go:30` | returns `(SevInfo, false)`. The row is badged `info` |

So the console would tell the operator their **subject** went away, when what was wrong was the
name they typed. That is the failure the bound prevents, and it is a wrong story rather than an
absent one.

### The two readers, verified — and the record is loose on one of them

The deleted `rules.go` block named *the form and the acceptance guard* as the two readers. Both
exist. They are bound by the same set in two different ways, and only one of them calls
`RuleNames`.

**The acceptance guard is the direct reader.** `knownRule` at `cmd/web/annotations.go:81` walks
`signal.RuleNames()` and returns false for anything else. `declareAnnotation` calls it at
`cmd/web/annotations.go:42` and refuses the submission. `s.store.CreateAnnotation` has exactly one
caller in the tree, at `cmd/web/annotations.go:52`, behind that check. **One write path, one guard.**

**The declare form is bound indirectly, through the census.** The form posts a hidden field —
`<input type="hidden" name="signal" value="{{.RuleID}}">` at
`design-system/templates/signals.tmpl:305` — and `RuleID` is set at `cmd/web/signals.go:384` from
the rule of the census row the operator opened. Every census comes from
`signal.EvaluateCorpus`, which walks the same three registries `AllRuleNames` walks. So the form can
only offer a rule that evaluated. It does not call `AllRuleNames`; the one call in `signals.go`, at
line 411, uses its length to size the `ruleVersions` map and reads the three registries directly.

**The correction matters, because it is why the guard is load-bearing.** The form is not a
whitelist, it is a rendering. The endpoint is a plain form POST reading `r.FormValue("signal")`, so
any client can post any string to it. The set is the bound, and the guard is where the bound is
enforced.

## Decision

> **An `Annotation`'s signal-name half may be exactly the names of the shipped rules, across all
> three subject kinds — one set, the one `AllRuleNames` returns. One set bounds both readers: the
> console's declare form, which can only submit a rule its own census rendered, and the acceptance
> guard, which refuses every other name at the single write path. An operator therefore cannot
> accept a firing on a rule that never fires.**

Five limbs.

### 1. The set is the shipped rules, and it is one set across the three subject kinds

Not a set per subject kind. `AllRuleNames` concatenates `All()`, `AllEndpointRules()` and
`AllServiceRules()` in `EvaluateCorpus` order, and that concatenation is the whole bound.

The set is **derived, never enumerated**. Nothing in this ADR, in the store, or in the console lists
the seventeen names. A rule admitted under ADR-0024 joins the set by being in a registry, and a rule
withdrawn leaves it the same way. The count is a dated fact about the registries and is not a term
of this rule.

### 2. Two readers, one set, and the guard is where the bound holds

The declare form offers a rule its census rendered. The acceptance guard admits a name in the set.
Both are bounded by the same three registries, so the two can never disagree about which rules
exist.

**The guard is where the bound is enforced**, because the form is a rendering rather than a
whitelist and the endpoint accepts a posted string. A second write path for annotations — an API, an
import, a fixture loader — takes the same guard. It does not get its own list.

### 3. The bound is at the write path and not in the schema

`signal_name` stays `TEXT NOT NULL` with no foreign key and no `CHECK`.

The rules are release-coupled Go values
([ADR-0004](./0004-signals-are-release-coupled-rules.md)). A database constraint over them would be
a second copy of the release's rule set, maintained by hand in migrations, and it would have to move
in the same deploy as the binary or refuse writes the binary allows. That is the hand-maintained
table shape [ADR-0009](./0009-verge-core-is-a-union.md) deleted once already, rebuilt in SQL.

### 4. What the bound does not reach: a rule withdrawn after the row is written

The guard runs once, at declaration. A row written against a rule that a later release withdraws or
renames survives, and nothing re-checks it.

**This is a defect and it is not fixed here.** §Context's table is what such a row produces: the
console marks it as naming a withdrawn **subject**, empties its link and badges it `info`. The
operator is told the wrong thing about their own estate.

It is not fixed here because the repair is a product decision this ADR does not hold. The row must
not be deleted — an `Annotation` is the operator's prose and deleting it is the suppression three
decisions have refused. Marking it needs a second marker beside `Orphan`, with its own copy, and a
marker is a rendering rule that belongs with the `Signals` screen. **It ships as its own ticket.**

### 5. Why the set is not partitioned by subject kind

An `Annotation` carries a subject key and a name, and nothing else identifies the pair.
`SubjectKindFor` at `internal/signal/corpus.go:25` recovers the kind from the **name**, by the same
walk over the same three registries.

So a per-kind name set would be a second structure saying what the name already says, kept in step
with it by hand. It would also buy nothing at the guard: the subject half is free text either way,
and ADR-0051 already rules what a subject key is.

## Consequences

- **This ADR changes no Go code.** The guard, the single write path and the form's binding through
  the census are all as ruled. It records the constraint at the layer that owns it.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Annotation` entry gains one clause.** The entry states the
  key's shape and every rider on it and never bounds the name half. Recorded in this batch's
  manifest. No existing clause in that entry is invalidated by this ruling — the pair, the six
  riders, the barred routes and the withdrawal semantics all stand unchanged.
- **`internal/signal/corpus.go`'s surviving comment gains a citation and one correction.** It says
  *the declare form offers exactly these*. The form is bounded through the census and the guard is
  the reader that walks the set, so the comment names the wrong reader. Recorded in the manifest.
- **A withdrawn or renamed rule leaves annotation rows the console describes wrongly.** §4 states
  it. **It ships as its own ticket**, and the ticket owns both the marker and its copy.
- **Nothing enforces the single-write-path premise.** If a second producer of `annotation` rows is
  added without calling `knownRule`, no check fires. `CreateAnnotation` has one caller today and a
  test asserting that is cheap. It is not opened here.
- **The set stays derived, so admitting a rule costs nothing at this seam.** ADR-0024's table can
  grow and no document, migration or console list has to follow it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A foreign key to a `signal_rule` table** | There is no such table and creating one makes the store hold a second copy of a release-coupled Go value (ADR-0004). It would have to be seeded in the same deploy as the binary, and a deploy that seeded late would refuse writes the running binary accepts. It also mints a stored object for something the model deliberately keeps compiled in |
| **A `CHECK` constraint enumerating the seventeen names** | A migration for every rule added, renamed or withdrawn, and the constraint is the one copy of the list a reviewer cannot see from the rule's own file. It is [ADR-0009](./0009-verge-core-is-a-union.md)'s deleted hand-maintained table, rebuilt in SQL, where nobody would look for it. It would also reject the row rather than the submission, so the operator gets a 500 instead of the guard's message |
| **Bound the set per subject kind, and validate the name against the subject's kind** | `SubjectKindFor` recovers the kind from the name over the same three registries, so a per-kind set restates what the name already carries and adds a second structure to keep in step. It also cannot validate the pair: the subject half is free text, so a `name` key under an endpoint rule is admissible either way and only the census can tell |
| **Validate on read rather than on write** | The read is every render of `Signals`, and the row is already written by then. It converts a refusable submission into a permanent row that some screen has to explain. §4 shows what explaining it costs — a second marker and its own copy — which is the price of the failure, not a way to avoid it |
| **Accept any string — the form is the only producer** | The form is a rendering, not a whitelist. It posts a hidden field and `declareAnnotation` reads `r.FormValue("signal")` from a plain form POST, so any client can submit any name. Without the guard the model's own claim — an `Annotation` mutes the pair's next firing — is false for every row nobody can fire |
| **Rule it on [ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md)** | That ADR's subject is what an `Annotation` **does**: it moves a message and never a number, and three of its four conceivable jobs are barred. This is a question about what a valid key **is**, which ADR-0016 states as a shape and leaves open on both halves. ADR-0051 already took the subject half out to its own document, and the name half follows it |
| **Rule it on [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)** | Its ruling is about denotation and normalisation — a key is the thing named, and the normalisation that produces it may never move. A rule name is not normalised and denotes nothing in the estate. The two halves of the pair share a key and not an argument |
| **Say nothing and leave it to the guard** | The guard is nine lines in `cmd/web` and reads as an input-validation convenience. Nothing said it was carrying a model constraint, so a second write path — an API, a fixture loader, a bulk import — would reasonably skip it, and the rows it wrote would be indistinguishable from an estate that had lost its subjects |
