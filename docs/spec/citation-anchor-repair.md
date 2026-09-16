# Citation anchor repair

- **Status:** Proposed — spec content for the content-repair effort [citation anchors](citation-anchors.md) §8.4 defers
- **Governs:** the repair of a region anchor that resolves and names the wrong declaration, and the refinement of the drift guard that finds one
- **Parent effort:** [Implementation map: the drift-proof citation anchor (#1967)](https://github.com/winniel123/verge-asm/issues/1967)
- **Origin:** [#1977](https://github.com/winniel123/verge-asm/issues/1977), landed as [#2003](https://github.com/winniel123/verge-asm/pull/2003)

A citation names a region in another file. The sweep of
[#1975](https://github.com/winniel123/verge-asm/issues/1975) derived that region from the line
number the citation used to carry. It read the number against the target as the target stands
today. Where the citation had already drifted, the number landed inside a different declaration,
and the sweep wrote that declaration's name.

**A wrong region anchor resolves.** The `citations` check asserts that an anchor names a real
declaration. It cannot ask whether that declaration is the one the document means. So the check
passes the anchor, and no later run reports it. A wrong line number at least stayed wrong in a way
a reader could test. A wrong region anchor reads as verified.

[Citation anchors](citation-anchors.md) §8.4 degrades the drift it can prove. A blank line and a
line past the end of the file are provable. A line that moved onto another live declaration is not.
That SPEC names the remaining population and defers it: the degrade record "feeds a separate
content-repair effort", and §1.3 puts the repair itself out of the sweep's scope.

**This document is that effort's SPEC.** It rules three things. It refines the drift guard (§5). It
scopes the repair of what already landed (§6). It states what the guard cannot see (§7).

**This document is a SPEC only.** It repairs no citation and it writes no checker. A downstream
implement effort builds both. This document follows STE-flavored mode, per the
[documentation style standard](documentation-style-standard.md) §2.

---

## 1. Purpose and scope

### 1.1 What this SPEC decides

| Part | Section |
| --- | --- |
| Terms | §2 |
| The defect, and why no check reports it | §3 |
| The guard as it shipped, and its two false-positive classes | §4 |
| The refinement: an ordering rule and a review queue | §5 |
| The repair of `docs/spec`, and the audit that finds the work | §6 |
| The bound on each instrument | §7 |
| Non-goals | §8 |
| Established facts | §9 |

### 1.2 The boundary

The boundary is the scope module, `docs-site/scripts/citations/scope.mjs`. The rule reaches the
files that module already walks. It reaches no list of directories copied into a ticket.
`docs/research` sits outside it, per
[#1450](https://github.com/winniel123/verge-asm/issues/1450). A later change to that module moves
this rule with it.

This matches [citation anchors](citation-anchors.md) §1.2. One boundary serves both SPECs.

### 1.3 Out of scope

- **Repointing an anchor to the declaration the prose names.** §6.3 rules that a suspect anchor
  degrades. Choosing a new target is a judgement about what the document asserts. A human makes it.
  §6.4 states the route that human takes, and this SPEC repoints nothing itself.
- **The line anchors the sweep has not reached.** Three batch tickets remain on map
  [#1967](https://github.com/winniel123/verge-asm/issues/1967). They convert under the guard as it
  stands, refined or not.
- **A citation whose path no longer resolves.** ADR-0175 holds two. They are
  [#1980](https://github.com/winniel123/verge-asm/issues/1980)'s business, and §9 fact F7 records
  them.
- **The `citations` check's presence rule.** §7.2 of
  [citation anchors](citation-anchors.md) asserts correctness, never presence. This SPEC changes
  nothing there. A bare path stays legal for every target kind, forever.

---

## 2. Terms

| Term | Meaning |
| --- | --- |
| **Region anchor** | The `` `path#anchor` `` form [citation anchors](citation-anchors.md) §3 rules. |
| **Rival** | A declaration the target really has, which the citing line names, and which is not the anchor the sweep wrote. |
| **Suspect anchor** | A written anchor with a rival. The line and the prose disagree, so one of them is wrong. |
| **Corroborated anchor** | A written anchor the citing line also spells by name. The line and the prose agree. |
| **Unproven anchor** | A written anchor whose citing line names no declaration of the target. Nothing corroborates it and nothing contradicts it. |
| **The guard** | `docs-site/scripts/sweep/corroborate.mjs`, which reports a rival. |
| **The history arm** | `docs-site/scripts/sweep/history.mjs`, which reads the target's own history. |
| **Witness commit** | The newest commit whose revision of the citing line still spelled a line number for the cited path. |
| **Drift candidate** | An anchor whose cited line sat in another declaration at the witness commit. |
| **Crossed anchor** | A written anchor that names a declaration the citing document does not mean. A suspect anchor and a drift candidate are the two shapes an instrument reports. Neither report is a proof, so a human names the class. |

---

## 3. The defect

### 3.1 What the sweep did

The sweep split a retired token into a path and a line number. It resolved the path, took the
target's declaration inventory, and picked the innermost declaration enclosing that line. It wrote
that declaration's name.

The step is sound when the line number is live. It is unsound when the number has drifted, because
a drifted number still lands somewhere. The declaration it lands in is evidence of nothing.

### 3.2 Why no check reported it

Three rules compose into the gap.

1. **The extractor's path pattern held no colon**, so a `path:NNN` token never became a candidate.
   [Citation anchors](citation-anchors.md) §11 fact F1 measures this. The token was invisible, so no
   run ever judged the line number.
2. **The sweep's own degrade arm proves drift two ways only.** A blank cited line and a line past
   the end of the file are provable from the target alone. A line that moved onto another
   declaration is not.
3. **Arm B asks one question of an anchor.** Does the target declare this name? A wrong name that
   the target happens to declare answers yes.

So the conversion turned an unjudged wrong number into a judged wrong name.

### 3.3 Why this is worse than the form it replaced

A line anchor drifts in silence, and the drift stays detectable. A reader can open the target and
see that the cited line holds something else.

A region anchor that resolves reads as verified. The required check passed it. A reader has no
signal to look, and no run will ever raise one. The document now asserts a wrong fact with the
gate's endorsement.

This is the outcome [citation anchors](citation-anchors.md) §3.3 rule 3 forbids. That rule says an
author never guesses an anchor, and a sweep never guesses one.

---

## 4. The guard as it shipped

### 4.1 The rule

[#2003](https://github.com/winniel123/verge-asm/pull/2003) added `rivalName` in
`docs-site/scripts/sweep/corroborate.mjs`. It asks one question per token:

> Does the citing line spell, inside a code span, a name the target really declares, while spelling
> no name for the region the line number landed in?

If so, the line has drifted, and the token degrades to a bare path under
[citation anchors](citation-anchors.md) §8.4.

Three properties bound it.

1. **It guesses nothing.** It degrades. It never picks the rival, because picking one repairs
   content.
2. **Its reach is the citation's own source line.** A wider block collects a name from a sentence
   about another target.
3. **A hit carrying no citing line derives as it did before the guard.** The guard adds an arm and
   changes none.

### 4.2 What it caught

Over ADR-0001 to ADR-0176 the guard degraded 36 of 255 candidate conversions. §9 fact F4 records
the split. An audit written separately from the guard found 32 wrong anchors in the unguarded pass
and 0 in the guarded one.

ADR-0141 shows the shape. Its table has nine rows, and each row's first cell names the loop the row
is about. Six rows kept an anchor matching their own cell. Three degraded, because the cited line
had moved onto a neighbouring method.

### 4.3 Two false-positive classes

The guard is not exact. It degrades some sound derivations. Two classes are measured, and both make
the citing line name a declaration that is not the citation's target.

**Class A — the line names a second site, or a callee.** The rival belongs to a different claim in
the same line.

- ADR-0143's line cites the code that fills a field, then a new sentence names `rcodeName`.
  `rcodeName` is the callee, and it has its own declaration far from the cited range.
- ADR-0166's line cites one range and then names a second declaration with its own line number.
  The anchor the guard dropped was correct.

This class held a third member, and the history arm falsified it. `docs/spec/aperture-statement.md`
line 408 then read "`cmd/web/cold.go#server.coveragePage` holds `apertureMeters`". That reads as
the same shape: the rival is a declaration inside the written anchor. The witness commit
`060c880d` wrote that citation as a line number on `cmd/web/cold.go`. The number named line 246,
which at that commit was the `func apertureMeters(...)` declaration line itself. So the citation
named `apertureMeters`, and the written anchor was the declaration the drifted number later landed
in. The guard's report was a true positive. §9 fact F3 records the corrected verdict, and
[#2094](https://github.com/winniel123/verge-asm/issues/2094) holds that record.
[#2146](https://github.com/winniel123/verge-asm/issues/2146) repaired that site under §6.4, and
line 408 now names `apertureMeters`.

**The class survives on the two ADR sites above.** One falsified member is not the class, and §5.2
rule 2 still answers it.

**Class B — a route or a URL segment collides with a declaration name.** The identifier is not a Go
name at all.

- `docs/spec/audit-act.md` line 338 names the route `POST /onboarding`. The segment `onboarding`
  matches the declaration `server.onboarding`, so the guard reports a rival. The written anchor
  `cmd/web/onboarding.go#server.onboardingStep` is correct.

**The cost of a false positive is small and the cost of a miss is not.** A degraded citation keeps
its path and asserts nothing false. It loses precision a later pass can restore. A wrong anchor
asserts a false fact that no check will report. So the guard is deliberately biased toward
degrading, and §5 narrows it without reversing that bias.

---

## 5. The refinement

### 5.1 A rival spelled before the citation is a strong signal

The dominant true-positive shape puts the name first and the site second. Prose writes
``` `X` (`path`) ```. A table row writes the name in one cell and the site in the next.

Over the 36 degradations, 28 spell the rival before the citation on the line. Every one of those
that this SPEC's author read is a real drift. 8 spell it only afterwards, and those 8 are mixed.
§9 fact F5 records the mix.

**So the ordering is the discriminator, and it is not a proof.** A table that puts the site cell
first also puts the name afterwards. ADR-0149 and ADR-0169 do exactly that, and both are real
drifts.

### 5.2 The rule

1. **A rival spelled before the citation degrades the token.** This is the rule that shipped, and
   it stands.
2. **A rival spelled only after the citation enters a review queue.** The token converts as it
   would without the guard, and the tool records the pair for a human to read. A silent degrade
   here would drop a sound anchor, and a silent conversion would keep a wrong one. Neither is
   honest, so the tool reports rather than decides.
3. **An identifier inside a route span is not a declaration name.** A span holding a space or a
   leading `/` spells a route, a header or a command. It corroborates nothing and it contradicts
   nothing.

Rule 3 closes class B. Rule 2 bounds class A rather than closing it.

### 5.3 What the refinement must not do

**It must not lower the bias toward degrading.** Rule 2 applies to one measured shape. A builder who
finds a new false-positive class adds a rule for that shape. A builder never relaxes rule 1.

**It must not reimplement `rivalName`.** One derivation serves every caller, per
[#1975](https://github.com/winniel123/verge-asm/issues/1975). A second copy drifts from the first.

**It must not read the target alone.** The rule *any name the target does not declare is suspect* is
rejected. A citing line legitimately spells a package-qualified call, a struct field, a method-local
name, a rule slug, a column name and a template token, and the target declares none of them. That
rule would degrade sound anchors across the 301 unproven anchors of §9 fact F6. The test is the
tree, not the target. A rule that reads an absence as evidence asks the whole §1.2 boundary, in
every file and every row vocabulary, and only a name no declaration there carries is available to
it. This paragraph calls that the tree test.

**It must not read a common word's absence as evidence.** An identifier whose name is a common word,
or a common compound, carries too little distinctiveness for a position verdict. It scores
`unproven` however the tree reads, and no `suspect` verdict is available to it. The test is
mechanical, so a name that appears later is covered on the day it appears: the name is declared in
more than one unrelated package, or it matches a standard-library field name. A **distinctive**
identifier is the complement, and only a distinctive identifier reaches the `suspect` verdict the
tree test allows.

`notAfter` and `notBefore` are the measured cases, and they are why the tree test is not enough on
its own. `cmd/web/subjects.go#certValidity` and `internal/signal/endpoint.go#CertHorizon` spell both
as parameters, and every X.509 certificate carries both fields. No row inventory declares either
name, so the tree test alone would call both suspect. The tree holds each word for a reason
unrelated to the citation, so the absence is unreadable. `certExpiryWindow`, which §9 fact F9 names,
is the complement: one package declared it, nothing else in the tree spells it, and its withdrawal
reads.

**It must not raise a shape whose false-positive rate no fact measures.** This is the mirror of the
first rule. That rule forbids lowering the bias toward degrading, and a rule that raises `unproven`
to `suspect` moves the bias the other way. Such a rule reaches only the shape whose false-positive
rate a §9 fact records. A shape whose rate is assumed rather than counted stays `unproven`, and §5.2
rule 2 is where an uncertain one goes.

---

## 6. The repair

### 6.1 `docs/spec` already carries the defect

Ticket 9 of map [#1967](https://github.com/winniel123/verge-asm/issues/1967) converted `docs/spec`
before the guard existed. The guard now reports 4 suspect anchors there, and 3 of them are real.
§9 fact F3 names all four.

**The sweep tool's conversion arm cannot reach these tokens.** That arm converts a line anchor, and
these are written region anchors. The repair needs a reader over the tree's written anchors, which
is the audit arm §6.2 adds.

### 6.2 The audit reads the tree

The repair adds an audit arm to `docs-site/scripts/sweep-line-anchors.mjs`. It reads every written
region anchor inside the §1.2 boundary. For each one it calls `rivalName` with the citing line, the
target's row inventory, and the written anchor.

One arm on the existing command, never a new command. The sweep already owns the derivation, the row
dispatch and the path resolution.

The audit reports three counts per family: suspect, corroborated, and unproven. §9 facts F1 and F2
are its first output, so a builder can check the arm against them.

### 6.3 A suspect anchor degrades, and no tool repoints it

A suspect anchor keeps its path and drops the anchor. The citation then says less, and everything it
says is true.

Repointing it to the rival was rejected. The rival is the name the prose spells, and that is
evidence, not proof. §4.3 measures two classes where the rival is not the target. A tool that
repointed would write a new wrong anchor in exactly those cases, and it would carry the gate's
endorsement again.

**Changing what a document asserts is not a repair.** For an ADR that route runs through a later
ADR, per [adr governance](adr-governance.md) §4. Repointing a citation to a **different**
declaration can change an assertion. Degrading one cannot.

**The limit runs on the declaration, not on the act.** ADR-2086 rules that an anchor which moved
with its declaration still names that declaration. Repointing it there is a factual repair, it needs
no relation, and a sweep may do it. Repointing a suspect anchor to its rival is the other act. It
puts one declaration where another stood, so the document asserts something new. §4.3 measures two
classes where the rival is not the target, so no tool can tell the two acts apart here. A human
decides each one. Inside `docs/adr` the route is a later ADR's relation, per
[adr governance](adr-governance.md) §4, and §3 of that SPEC carries the same limit. That SPEC
governs no file under `docs/spec`, so it prescribes no route there. §6.4 states the route that
reaches `docs/spec`.

### 6.4 The route for a `docs/spec` anchor that crossed declarations

ADR-2086 §3 puts a crossed anchor in its second row. The document asserts something new, so a human
acts, and the act leaves a record. ADR-2086 §7 leaves `docs/spec` outside the ADR route, so the
relation of [adr governance](adr-governance.md) §4 is not that record here. This section names the
instrument that is.

**The record is the pull request, because a SPEC amends itself in place.** An ADR never does, per
[adr governance](adr-governance.md) §3, so its record has to be a later file. A SPEC is edited where
it stands, so the edit is the amendment. No relation is declared and no marker is written.

The route has three steps, and a human takes each one.

1. **Degrade first.** §6.3 stands. The anchor becomes a bare path, and the citation asserts nothing
   false from that moment. A degrade needs no reading and waits for no decision.
2. **Repair the sentence, and let the anchor follow it.** The human reads the citing sentence and the
   target, and states which declaration the sentence means. A rival the guard reports and a witness
   commit the history arm reads are both evidence for that reading. Neither one decides it, per §6.3
   and §7. The human then writes the sentence and its anchor in one commit. A new anchor under a
   sentence nobody re-read is the act §6.3 refuses.
3. **State the evidence in the pull request body.** One line carries the anchor that degraded, the
   declaration the repair names, and what chose it. The two steps may land in one pull request or in
   two, and the request that writes the new anchor carries that line.

**§6.3 keeps its sentence and gains a limit.** It rules the act a tool takes, and the act a human
takes on the anchor alone. It does not refuse a re-read sentence that carries a new anchor, because
that act re-states the claim rather than moving it.

**The declaration the sentence means may have left the target.** The citation is then wrong in its
path as well as its anchor, and the same commit writes both. A citation whose path no longer
resolves at all stays §1.3's case.

**A wrong rule is not a wrong citation.** This route repairs a citation. Where the reading finds the
SPEC's rule wrong, the crossed anchor is a symptom, and the rule changes by its own route.

**The rejected record is an in-tree sentence.** [adr governance](adr-governance.md) §5 keeps the
sentence rule of ADR-0058 for a spec target, so a note beside the repaired citation was available.
It was rejected because ADR-0058 rules a withdrawn mechanism, and a repaired citation withdraws
nothing. A note per repair would also state, in the SPEC's own prose, which line number once drifted.
The cost of the choice is that the tree shows a repaired anchor and a never-wrong anchor alike, and
the pull request holds the difference.

**No gate reports this work, and no count retires it.** §8 keeps every instrument out of the gate,
and §7 consequence 3 says why no list proves the anchors correct. §9 fact F3 records three real
suspect anchors in `docs/spec`. [#2146](https://github.com/winniel123/verge-asm/issues/2146)
repaired the first under this route, and the two `audit-act.md` rows have no owner. Fact F10
reports 30 more `docs/spec` drift candidates that no human has read, so those three are the first
cases and not the population.

---

## 7. The bound on any guard

**The guard proves nothing about an anchor whose line names no declaration.** It needs the document
to spell the target. Most citations do not.

§9 fact F6 measures the population. 114 of 126 written anchors in `docs/spec` and 187 of 219 in
`docs/adr` sit on a line that names no declaration of the target either way. Those anchors are
unproven. Some of them sit on a line that had drifted, and no reading of the citing document can
tell which.

Three consequences follow.

1. **A green `citations` run is not evidence that an anchor is right.** It is evidence that the name
   exists. §3.2 rule 3 is the whole of it.
2. **The unproven population has a second instrument, and the instrument is not a proof.** The
   history arm reads the target's own history.
   [#2015](https://github.com/winniel123/verge-asm/issues/2015) built it. It asks three mechanical
   questions per anchor. Which commit last wrote a line number on the citing line? Which
   declaration enclosed that number in the target at that commit? Is it the declaration the anchor
   names now? A different declaration is a drift candidate, and the citing line needs to name
   nothing. §9 facts F9, F10 and F11 measure it. The arm reports and writes nothing, and §8 keeps
   it out of the gate.
3. **No count retires this effort.** [Citation anchors](citation-anchors.md) §8.2 proves the sweep
   complete with an empty burn-down list. No list proves the anchors correct, because the
   correctness question has no list.

**The history arm has its own bound, and it is not the guard's.** The guard needs the citing
document to spell the target. The history arm needs the citation to have carried a line number that
a commit still holds. Five shapes defeat it.

1. **A citation no revision of its line ever wrote with a number.** The arm reports `unwitnessed`
   and judges nothing. F10 measures 75 of these.
2. **A citing line whose citation count and witness count disagree.** One line can cite one path
   twice, against a witness commit that spelled one number. No pairing follows from the counts, and
   the arm guesses none. F10 measures 86 of these, so this shape is the larger of the two.
3. **A citation that was wrong when it was written.** The arm measures change since the number was
   asserted. A number that was wrong on the day it landed leaves no change to find.
4. **A target the witness commit spells under another path.** The arm reads `<commit>:<path>` with
   the path as it stands today, so a rename before the witness reads as an absent target.
5. **A number that landed just outside the declaration the prose means.** The declaration then
   encloses no such line, and the arm calls the anchor a drift candidate. F11 measures this class,
   and it is the arm's one measured false positive.

**A drift candidate is evidence, and a human still reads it.** §6.3 rules that a suspect anchor
degrades and is never repointed. The same holds here, and for the same reason: the declaration the
arm names is where the number pointed, not what the document asserts.

---

## 8. Non-goals

**No check refuses an unproven anchor.** A refusal would fail on the 301 anchors of fact F6, and
almost all of them are sound. The guard runs at conversion time and in the audit. It is not a gate.

**No check runs the history arm.** The arm reports 172 drift candidates on the boundary, per fact
F10, and a hand read finds most of them real. A gate on that count would red every merge until a
human had read all 172, and it would red again on the false-positive class of F11. The arm is an
instrument for a repair effort, and the repair stays a human act under §6.3.

**No check refuses a bare path.** [Citation anchors](citation-anchors.md) §7.2 asserts correctness,
never presence. Every degrade this SPEC orders produces a bare path, so a presence rule would turn
the repair into a violation.

**No renderer resolves an anchor.** [Citation anchors](citation-anchors.md) §9 already rules this,
and this SPEC adds nothing.

**No ADR records this SPEC.** The decisions here are corrections to a mechanism that one pull
request built and the same pull request measured. A human opens the issue that becomes an ADR, per
`CLAUDE.md`.

**§6.4 is the one section that reason does not cover.** It rules a route rather than correcting the
guard, and it answers for `docs/spec` the question ADR-2086 §7 left open there. So the pull request
that added it carries a decision proposal block, per [adr governance](adr-governance.md) §7, and a
human decides whether an ADR follows.

---

## 9. Established facts

Every fact below was measured on 2026-09-14, against `main` at `5e661e0`. Each one is reproducible
with the guard the tree ships.

**F1 — `docs/spec` holds 126 written anchors on a target with an anchor vocabulary. 4 are suspect,
8 are corroborated, and 114 are unproven.**

**F2 — `docs/adr` holds 219 such anchors. 0 are suspect, 32 are corroborated, and 187 are
unproven.** The range converted under the guard, so a suspect anchor could not land.

**F3 — 3 of the 4 suspect anchors in `docs/spec` are real.**

| Document | Line | Anchor written | Rival | Verdict |
| --- | --- | --- | --- | --- |
| `docs/spec/audit-act.md` | 466 | `cmd/web/auth.go#initials` | `validateCredentials` | real |
| `docs/spec/audit-act.md` | 1510 | `cmd/web/auth.go#profileRelTime` | `sessionIP` | real |
| `docs/spec/aperture-statement.md` | 408 | `cmd/web/cold.go#server.coveragePage` | `apertureMeters` | real |
| `docs/spec/audit-act.md` | 338 | `cmd/web/onboarding.go#server.onboardingStep` | `server.onboarding` | class B |

The `aperture-statement.md` row read `class A` when this fact was first measured, on the guard's
evidence alone. The history arm overturned it, per §4.3: the witness `060c880d` named line 246 of
`cmd/web/cold.go`, which was the `func apertureMeters(...)` declaration line.
[#2146](https://github.com/winniel123/verge-asm/issues/2146) repaired that site under §6.4, and
the citation now names `apertureMeters`. This fact records the verdict alone.

**F4 — the guard degraded 36 of 255 candidate conversions over ADR-0001 to ADR-0176.** The
unguarded pass wrote 255 anchors. The guarded pass wrote 219 and degraded 141 in total.

**F5 — 28 of the 36 spell the rival before the citation, and 8 spell it only afterwards.** Of those
8, ADR-0149 line 48 and ADR-0169 lines 61 and 62 are real drifts. ADR-0143 line 26 and ADR-0166
line 88 are class A, and each lost a sound anchor.

**F6 — 301 of the 345 written anchors in the two families are unproven.** That is 114 in `docs/spec`
and 187 in `docs/adr`.

**F7 — two burn-down entries under ADR-0001 to ADR-0176 remain, and no conversion can clear them.**
Both name ~~`cmd/web/signals_cert_test.go`~~, which left the tree in
[#1732](https://github.com/winniel123/verge-asm/pull/1732). A bare path there would red the gate,
per [citation anchors](citation-anchors.md) §7.6, so the sweep held both.

**F8 — an independent audit found 32 wrong anchors in the unguarded pass and 0 in the guarded
pass.** The audit's rule was written separately from the guard, and it reads the pre-conversion line
rather than the written one.

A run on 2026-09-15 produced facts F9, F10 and F11. F9 ran against `c196bed^`, and F10 and F11
against `main` at `92f2db2`.

**F9 — the history arm reproduces 17 of the 18 wrong anchors
[#2011](https://github.com/winniel123/verge-asm/pull/2011) degraded.** The run reads `docs/adr` at
`c196bed^`, the tree before that conversion. It judges 600 anchors there, and it calls 216 drift
candidates. In each of the 17 the declaration it names is the declaration that pull request's human
review named as the intended target: `server.declareSeed`, `ListRecentDriftEvents`,
`settings-messages`, `MarkJobDone`, `subjectRulesFor`, `declaredNameTree`, `certDetailsFromValue`,
`certExpiryWindow`, `NewHTTPDoer`, `TestHTTPDoerRefusesRedirects` and
`TestRedirectIsRecordedButNotFollowed`. The 18th is ADR-0184 line 129. The arm never sees it. The
refined guard of §5.2 degrades that token first, so no conversion writes an anchor there for the
arm to judge. The two instruments together reach all 18.

**F10 — the arm judges 599 written anchors on the boundary, and calls 172 of them drift
candidates.** 427 are consistent. `docs/adr` holds 484 judged and 142 candidates. `docs/spec` holds
115 judged and 30 candidates. The root documents hold 2 anchors, and the arm judges neither. Read
against F6, this is a first rate on the unproven population, and it is not small.

The same run leaves 161 more anchors unwitnessed, and 5 more unreadable. Shape 1 of §7 holds 75 of
the unwitnessed, and shape 2 holds the other 86 of them. Four of the 5 unreadable sat on
uncommitted lines of this document. `git log -L` reads committed history, and it never reads the
working tree.

**F11 — a hand read of 20 of the 172 candidates found 1 false positive.** The sample takes a fixed
stride over the report in its own sort order, so no judgement of the reader chose the rows. The one
is ADR-0198 line 23. Its number named line 68 of `internal/queue/membership.go`, which was the blank
line immediately after `foldEstateTransitions`. That declaration is the one the prose means, and the
anchor is right. The other 19 are real.

Two of the 19 spell the intended declaration in the prose, and the guard reports neither. ADR-0157
line 62 puts that name on the line above the citation. `docs/spec/audit-act.md` line 19 names
`fillAuditSection`, and `cmd/web/settings.go` no longer declares it. The guard then reads the
strongest drift evidence available as no evidence at all.
