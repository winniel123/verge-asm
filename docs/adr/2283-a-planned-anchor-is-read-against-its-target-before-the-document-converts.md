---
number: 2283
title: "A planned anchor is read against its target before the document converts"
slug: a-planned-anchor-is-read-against-its-target-before-the-document-converts
date: 2026-09-17
status: accepted
source: fix
ticket: 2283
proof: {test: "docs-site/scripts/sweep-line-anchors.test.mjs::a rival wrapped onto the source line above is out of the guard's reach"}
---

# ADR-2283: A planned anchor is read against its target before the document converts

## Decision

> **A document converts only after a human reads every planned anchor against its target and against
> the citing prose.** [#2156](https://github.com/winniel123/verge-asm/issues/2156)'s zero-held bar
> gains a fourth test. A token the read rejects degrades to a bare path, and the rest of the document
> converts. The `--history` arm runs per slice as evidence for the read, never in place of it.
>
> A reader asks why a mechanical sweep needs a hand. Because both instruments passed ADR-0212's one
> wrong anchor. The guard reads the citation's own source line, and a wrap put the rival above it.
> The arm found no change, because the number was wrong when it landed.
>
> Rejected: hold the whole document when one anchor fails the read. That freezes its sound tokens in
> the form the conversion retires.
>
> Reversal writes an anchor no check will ever question.

## 1. Context

[#2156](https://github.com/winniel123/verge-asm/issues/2156) rules that the unit of conversion is a
document, never a token, because a partial conversion orphans the path-less tokens that lean on the
one converted. Its bar is three tests: every token in the document spells a path, the sweep resolves
that path, and the sweep derives an anchor for it. A document with one held token does not convert.

Every one of the three asks what the **sweep** can do. None asks whether the anchor is right.

`docs/spec/citation-anchor-repair.md` §3.1 already states why the derivation is unsound on a drifted
number: *"a drifted number still lands somewhere. The declaration it lands in is evidence of
nothing."* §3.2 records that the sweep proves drift two ways only, a blank cited line and a line past
the end of the file. A number that moved onto another live declaration is not provable from the
target alone.

So a document can hold zero held tokens and convert every one of them onto the wrong declaration.
§3.3 of that SPEC names the outcome: *"A region anchor that resolves reads as verified. The required
check passed it. A reader has no signal to look, and no run will ever raise one."*

ADR-2086 §4 and #2156's own acceptance already forbid the result. Nothing in the bar makes a session
look.

## 2. The measurement

The ADR-0177 to ADR-0198 slice, [#2282](https://github.com/winniel123/verge-asm/issues/2282), is
where the gap surfaced. Its reader read every planned anchor of the documents that reached the bar,
and landed one of them.

This section re-ran both mechanical instruments over the four documents that slice held on the
wrong-anchor ground. The run was taken on 2026-09-17, from `docs-site/`, against `main` at
`584ca10`. It reads 32 tokens.

| Document | Tokens | Held, no path | Degraded by the guard | Planned anchors | Wrong on the hand read | Reported by the history arm |
| --- | --- | --- | --- | --- | --- | --- |
| ADR-0184 | 23 | 2 | 8 | 13 | 2 | 0 of the 2 |
| ADR-0197 | 5 | 0 | 2 | 3 | 3 | 3 |
| ADR-0212 | 1 | 0 | 0 | 1 | 1 | 0 |
| ADR-0221 | 3 | 0 | 2 | 1 | 1 | 1 |

**The guard reports none of the 7, and it cannot.** A planned anchor is by definition a token the
guard passed, so the column would be zero whatever the corpus held. The finding is not the count. It
is that the bar's three tests read the guard's output and nothing else, and §4 of
`docs/spec/citation-anchor-repair.md` states the bounds that let a wrong anchor through it. Three of
those bounds account for the 7.

1. **The rival is declared in another file.** ADR-0197 §3 tabulates the producer's steps, and each
   row's first cell names a query or a method of `db.Queries`. `internal/queue/produce.go` declares
   none of them, so there is no rival to report. §7 states the bound: the guard needs the citing
   document to spell a declaration **of the target**.
2. **The rival sits on the source line above.** §4.1 property 2 bounds the guard's reach to the
   citation's own source line, because a wider block collects a name from a sentence about another
   target. A paragraph wrapped at 100 columns splits the pair. ADR-0212 does exactly that: line 61
   spells `fanOut`, and line 62 carries the citation.
3. **The prose names no live declaration.** ADR-0184's two wrong anchors sit in a marker sentence
   about what a pull request **deleted**. Both names it spells left the tree, so the target declares
   neither, and there is nothing for the guard to match.

**ADR-0184 also holds 2 path-less tokens, so it fails #2156's bar on the first test as well.** It is
in this table because supplying those two paths is the reader's ordinary first step, and the bar
would then pass the document with its 2 wrong anchors intact.

**The history arm reads a planned anchor already.** `historyCitations` in
`docs-site/scripts/sweep-line-anchors.mjs` pushes every token the conversion arm would convert into
the same judgement as a written one. So the arm was available to #2282, and running it costs one
command.

## 3. Why the history arm is not the fourth test

The arm asks whether the cited number named the same declaration at the commit that wrote it. That
question has an answer only where the number was right when it landed.

**ADR-0212 is the counter-example, and it is measured.** Line 62 reads *"calls `fanOut` — which
routes the cold kind to the streamed fan-out"*, and the citation is a line number on
`internal/queue/queue.go`. The witness commit is `39d00e9b`. At that commit the cited line was the
declaration line of `internal/queue/queue.go#Dispatcher.settleDue`, and it still is. The arm reports
`consistent`. The sweep's dry run derives `Dispatcher.settleDue`, with no degradation and no review
queue entry.

Both instruments are green, and the anchor names a method the sentence is not about. The sentence
means `internal/queue/queue.go#Dispatcher.fanOutStreamed`, which stands far from the cited line. The
number was wrong on the day it was written, so there is no change for the arm to find. This is shape
3 of `docs/spec/citation-anchor-repair.md` §7, and it is the first instance of that shape the tree
measures.

**ADR-0184's two wrong anchors defeat the arm a second way.** One cited number is past the end of
the target at the witness commit, so the arm reports `unreadable`. The other sits on a line whose
citation count and witness count disagree, which is shape 2, so the arm reports `unwitnessed`. The
arm judges neither, and the sweep would convert both onto live declarations —
`internal/message/render.go#artifactPeriod` and `internal/message/render.go#changeFamily`.

So the arm reports nothing on 3 of the 7 anchors the hand read rejects. A bar resting on the arm
alone converts ADR-0212 and ADR-0184.

## 4. The arm still runs, and it runs per slice

The arm named 6 drift candidates in ADR-0184 that #2283's hand read had not, and all 3 of ADR-0197's.
That is the complementarity `docs/spec/citation-anchor-repair.md` §9 fact F9 records, where the guard
and the arm together reach all 18 wrong anchors of
[#2011](https://github.com/winniel123/verge-asm/pull/2011) and neither reaches them alone.

So the arm is an input to the read and not a substitute for it. A slice runs `--history` over its
range, and the reader starts from the candidates it names. A candidate is evidence and not a proof,
per §7, so the reader still decides each one, and the reader still reads the anchors the arm passed.

The arm is not promoted to a gate. `docs/spec/citation-anchor-repair.md` §8 keeps it out of one,
because it reports 172 drift candidates on the boundary and a gate on that count reds every merge.
Nothing here changes that.

## 5. What a rejected token does

A token the read rejects degrades to a bare path. The rest of the document converts, and the document
is still the unit.

**A degrade inside a converting document is the act the tool already takes.** The guard degraded 36
of 255 candidate conversions over ADR-0001 to ADR-0176, per §9 fact F4, and those degradations landed
inside documents that converted.
[#2288](https://github.com/winniel123/verge-asm/pull/2288) is the nearest case: it converted ADR-0195
with 2 anchors and 1 degrade, in one document and one commit. So a hand degrade adds no new shape to
a document. It moves one decision from the guard to a reader, on a shape the guard cannot see.

**The reader degrades and never repoints.** `docs/spec/citation-anchor-repair.md` §6.3 rules this for
a written anchor, and ADR-2086 §4 rules it for a converted one. Picking the declaration the prose
names is a judgement about what the document asserts. Inside `docs/adr` that route is a later ADR's
relation, per `docs/spec/adr-governance.md` §4. It is not part of a conversion.

**Rejected: hold the whole document.** The argument for it is that a mixed document hides which
anchors a reader checked. It fails on two counts. Under this ADR a reader checked every anchor the
document carries, so there is nothing to hide. And a hold freezes that document's sound tokens in a
line-number form no check judges at all, which is the form
[#2120](https://github.com/winniel123/verge-asm/issues/2120) staged for retirement.

ADR-0184 is the one document in §2 where the two rulings differ. It plans 13 anchors and 2 are wrong,
so a hold refuses 2 wrong conversions at the price of 11 sound ones. On the other three every planned
anchor is wrong, and the two rulings do the same thing.

A path-less token is the one case that still holds the document, and #2156's bar already rules it.
The reader supplies its path first, or the document does not convert.

## 6. What this ADR does not rule

**Whether any of the four documents is repaired.** A degrade leaves a bare path, and the citation
then says less than the prose does. Repointing it is a later ADR's act under §5, and this ADR opens
no such ticket.

**The repeated-anchor hold.** ADR-0188 passes a hand read on all 3 of its planned anchors and is held
for another reason, which [#2286](https://github.com/winniel123/verge-asm/issues/2286) owns. A fourth
test that a document passes does not convert it.

**The refusal flip.** `stageOf` in `docs-site/scripts/check-citations.mjs` stays until the staged
count reaches zero. This ADR changes no count.

**Any refinement of the guard.** The three bounds of §2 are properties the SPEC states, not defects.
Widening the guard's reach past one source line would collect a name from a sentence about another
target, which §4.1 property 2 rejects on its own ground.

**Whether a slice may batch its reading.** #2156 slices by family and by range, and a reader may read
a whole slice at once. The rule is that every planned anchor is read, not that each one is read
alone.

## 7. Consequences

A slice of #2156 now carries three steps before `--write`: supply every path, run `--history` over
the range, and read every planned anchor against its target and its prose. The pull request records
the reading, the same way #2282's acceptance already asks.

The read costs a reader one open file per distinct target. The 18 planned anchors of §2 reach 11
targets, and ADR-0184 alone reaches 8 of them.

**No check enforces this.** The `citations` gate asserts that an anchor names a real declaration, and
§3.2 rule 3 is the whole of it. A session that skips the read still merges green. That is the
standing `docs/spec/citation-anchor-repair.md` §6.4's route already has, and the pull request body is
the record in both.

**The four documents keep their line anchors until a session reads them.** This ADR converts none of
them. It states what a converting session owes.

Reversal writes an anchor no check will ever question, on a document that reads as verified.

## 8. Proof

The proof names `a rival wrapped onto the source line above is out of the guard's reach`, in
`docs-site/scripts/sweep-line-anchors.test.mjs`. It runs one citing sentence twice against one
fixture, and the two runs differ only in where the sentence breaks.

| The citing line the guard reads | Outcome | What it locks |
| --- | --- | --- |
| the rival and the citation together | `degraded` | the guard reports the rival when it reaches it |
| the citation alone, the rival above | `anchor` | §2 bound 2, which is ADR-0212's shape |

**This suite is not run by any required check.** `test:sweep` appears in no workflow, because
[#1975](https://github.com/winniel123/verge-asm/issues/1975) keeps the sweep tool out of the required
checks' path. [#2279](https://github.com/winniel123/verge-asm/issues/2279) holds that gap. ADR-2277
§8 records it against the same suite, and it is not repaired here.

The measurement of §2 and §3 is reproducible with two commands from `docs-site/`, over the four
documents:

```sh
node scripts/sweep-line-anchors.mjs --history <paths>
node scripts/sweep-line-anchors.mjs <paths>
```
