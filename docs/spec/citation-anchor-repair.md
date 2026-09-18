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
3. **An identifier inside a route span is not a declaration name.** A span holding a leading `/`,
   or holding both a space and a `/`, spells a route, a header or a command. It corroborates
   nothing and it contradicts nothing.
4. **An identifier cut out of a bare commit hash is not a declaration name.** A span whose whole
   body is 7 to 40 lowercase hexadecimal characters, at least one of them a digit, yields no
   candidate name.
5. **A name several declarations share picks out no rival.** An identifier that matches several
   declared names of the target picks out none of them, and the guard picks none either. Two
   spellings of one declaration are not several declarations. Where one matched name is the
   identifier itself, the line spells that declaration whole. Where one alone is the deepest, a
   qualified spelling reaches it past its own bare tail. Either one is the rival.

Rule 3 closes class B. Rule 2 bounds class A rather than closing it.

**A space alone does not spell a route, and rule 3 said it did when it shipped.** The rule as
written read `` `func Alpha() int` `` as a route, so a signature stopped corroborating and the next
rival degraded a sound anchor.
[#2010](https://github.com/winniel123/verge-asm/pull/2010) narrowed the test to ask for both a
space and a slash, and recorded the cause in its own body. The sentence above kept the wider
wording until [#2257](https://github.com/winniel123/verge-asm/issues/2257) found the two
disagreeing. The code is the record of that decision, and this sentence now matches it.

**Rule 4 answers a shape §9 fact F12 measured.** The identifier pattern starts on a letter, so it
cuts `fa97` out of `860fa97` and offers the tail as a candidate. §5.3 delegates a new
false-positive class to a builder, and this is one. The digit keeps the rule off `defaced`, which
is seven hexadecimal characters and also a word. The cost is a hash that spells no digit at all.
Six of the sixteen hexadecimal characters are letters. That costs about 1 seven-character hash in
960. A longer abbreviation is rarer still, at about 1 in 2600 for eight characters.

**Every rule above reads the target's own inventory, and that scoring inverts.** A name the target
still declares is a rival, so rule 1 degrades the token where the line spells it first. A name the
target has **dropped** matches no inventory entry, so it is no rival under §2 and the verdict is
`unproven`. The token then converts. So withdrawing the declaration scores weaker than keeping it,
and the strongest drift evidence the corroborator can hold reads as no evidence at all. §9 fact F11
names one site of that shape: `docs/spec/audit-act.md` line 19 spells `fillAuditSection`, and
`cmd/web/settings.go` no longer declares it.

ADR-2155 rules that the verdict stays `unproven`. It rejects the repair on §9 fact F12's rate, not
on the inversion, which it records as real. §5.3 states the boundary any such repair would have to
respect, and F12 states what is left inside it.

**Rule 5 answers the sealed union.** A Go interface declares a marker method, and every member of
the union declares the same method. ADR-0209 line 57 cites three such interfaces, and the citing
line spells `isCTOutcome()`. `internal/wire/transcript.go` declares that method on three receivers.
So `sameName` matched three names, the guard took the first in sort order, and it called
`CTContextCancelled.isCTOutcome` the rival of the interface that declares the method.
[#2285](https://github.com/winniel123/verge-asm/issues/2285) records the three sound anchors that
pick destroys. A rival is one declaration the target really has, and an arbitrary pick between twins is
not one. §3.3 rule 3 already refuses to pick between two names on one declaration line, and rule 5
is that refusal on the citing line. The exact-name arm holds rule 1's reach where the line spells a
declaration whole: `Alpha` beside `AAA.Alpha` still names `Alpha`. A qualified spelling reaches the
deepest name it ends with. `internal/act/render.go` declares `markedSubject` and
`AccountRef.markedSubject`, and a citing `act.AccountRef.markedSubject` still names the method. One
receiver still rivals, because one name picks out one declaration. §9 fact F13 measures the reach.

**Rule 5 reaches the rival arm alone.** The corroboration arm asks another question: does the line
name the anchor's own region? A member's own anchor is corroborated by the marker method it
declares, so a shared name supports that anchor rather than blurring it. Tightening the
corroboration arm would raise the bias toward degrading, and §5.3 closes a raise to a shape §9
measures. ADR-2277 §2 records the coupling: corroboration returns before the rival scan, so every
verdict it awards is a rival never scanned for. Rule 5 leaves that cost where it stands.

### 5.3 What the refinement must not do

**It must not lower the bias toward degrading.** Rule 2 applies to one measured shape. A builder who
finds a new false-positive class adds a rule for that shape. A builder never relaxes rule 1.

**It must not reimplement `rivalName`.** One derivation serves every caller, per
[#1975](https://github.com/winniel123/verge-asm/issues/1975). A second copy drifts from the first.

**It must not read the target alone.** The rule *any name the target does not declare is suspect* is
rejected. A citing line legitimately spells a struct field, a method-local name, a rule slug, a
column name and a template token. No declaration vocabulary carries any of those, so the target's
inventory declares none of them, and that rule would degrade sound anchors across the 301 unproven
anchors of §9 fact F6. The test is the tree, not the target. A rule that reads an absence as
evidence asks every declaration the tree carries, under every declaration vocabulary, and it is
closed to any name some inventory in the tree still carries. This section calls that the tree test.

**A declaration vocabulary is one that names.** The containment row matches every path and answers
whether a file holds a token. It enumerates no name, `rivalName` never sees it, and a rule that
counted it would find every word present and fire on nothing.

**It must not read a common word's absence as evidence.** An identifier whose name is a common word,
or a common compound, carries too little distinctiveness for a position verdict. It scores
`unproven` however the tree reads, and the tree test is closed to it. The test is mechanical, so a
name that appears later is covered on the day it appears. Either limb is enough.

1. The tree spells the name outside a citation, in a position no declaration vocabulary carries: a
   parameter, a local, or a struct field. The name is then absent from every inventory for a reason
   the vocabulary chose, not for a withdrawal.
2. The name matches a standard-library field or method name, with an exported initial folded.

A **distinctive** identifier is the complement. Only a distinctive identifier reaches the verdict the
tree test awards.

`notAfter` and `notBefore` are the measured cases, and they are why the tree test is not enough on
its own. `cmd/web/subjects.go#certValidity` spells `notAfter` as a parameter, and
`internal/signal/endpoint.go#CertHorizon` spells both. Every X.509 certificate carries both fields,
and `crypto/x509` spells them `NotAfter` and `NotBefore`. So both limbs hold, while no row inventory
declares either name and the tree test alone would call both suspect. The tree holds each word for a
reason unrelated to the citation, so the absence is unreadable. `certExpiryWindow`, which §9 fact F9
names, is the complement: no file outside `docs/` spells it, no limb holds, and its withdrawal
reads. These three measurements were taken on 2026-09-16, against `main` at `ffa10a8`.

**It must not raise a shape whose false-positive rate no fact measures.** This is the mirror of the
first rule. That rule forbids lowering the bias toward degrading, and a rule that raises `unproven`
to `suspect` moves the bias the other way. Such a rule reaches only the shape whose false-positive
rate a §9 fact records. A shape whose rate is assumed rather than counted stays `unproven`, and §5.2
rule 2 is where an uncertain one goes.

**§9 fact F12 measures the tree test, and this section states a boundary and no rule.** F3 and F5
measure the position guard, F8 the audit, F11 the history arm, and F12 the tree test's
false-positive rate on this boundary. **The rate F12 records is 14 false positives in 14, over an
empty true-positive population.** The two shapes this section leaves open are a retired line anchor
and a residue of five names. Rule 4 takes one of the five. So the residue reads as four names
against the repaired guard, and the rate reads as 13 of 13. Either rate is every raised name.

**ADR-2155 read that rate, and no tree test ships.** A rival the target no longer declares keeps
the verdict `unproven`. The rule that
[#2155](https://github.com/winniel123/verge-asm/issues/2155) proposed would have raised 13
occurrences over 11 anchors, every one a false positive, and would have recovered no fact. §4.3's
trade prices a false positive against a miss, and this boundary holds no miss to prevent.

**Two separate things emptied the true-positive population, and neither is a defect of this
section.** The rails above closed shapes A and C, which are 445 of the 459 names and are false
positives by construction. The names that would have been true positives left the corpus before the
run: [#2011](https://github.com/winniel123/verge-asm/pull/2011) degraded the ADR-0186 rows by hand,
so none of them is a written anchor's rival today. F12 records both.

So §2 needs no repair. A suspect anchor still takes its name from a rival, and a rival is still a
declaration the target really has. No verdict this SPEC rules awards `suspect` without one.

**A later run may reopen it.** The refusal rests on a rate over one corpus. F12 names the two
commands that reproduce its inputs, and a corpus that grows a true positive puts the question back.

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
a commit still holds. Six shapes defeat it.

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
6. **An edit to the citing line that leaves the number standing.** The witness is the newest
   revision that spelled a number, so a commit that rewrote the line for an unrelated reason
   becomes the witness. The arm then reads the target as it stood at that later commit, and a drift
   before the edit leaves no change to find. The arm reports `consistent`.

**Shape 6 is measured on ADR-0212 line 62.** The citing sentence names the streamed fan-out and
cites `` `queue.go:111` ``. `fa2c312` wrote that citation, and line 111 of
`internal/queue/queue.go` then sat inside `Dispatcher.fanOut`. `39d00e9` rewrote the same line to
convert a second token on it under
[#1979](https://github.com/winniel123/verge-asm/issues/1979), and line 111 by then sat inside
`Dispatcher.settleDue`. The arm takes `39d00e9` as the witness, so it reports `consistent`, and the
dry run derives `internal/queue/queue.go#Dispatcher.settleDue` with no degradation and no review
queue entry. Both instruments pass a wrong anchor. Verified 2026-09-18 against `main` at `51f4b18`.
[A planned anchor is read against its target before the document converts](../adr/2283-a-planned-anchor-is-read-against-its-target-before-the-document-converts.md)
§3.1 is the ruling that surfaced the shape, and §6 of it leaves this section's edit open.

**The line-anchor conversion is itself such an edit, and it reaches only the tokens it leaves
behind.** A `--write` run rewrites a citing line that carries several tokens, and it converts one of
them. The converted token spells no number afterwards, so the run is never that token's witness, and
the witness falls back to the revision that wrote the number. A token the run left numbered on the
same line keeps its number, so the run **is** that token's witness, and every drift before the run
is unreadable there. `39d00e9` above is
[#2020](https://github.com/winniel123/verge-asm/pull/2020) doing exactly that to `queue.go:111`, and
[#2011](https://github.com/winniel123/verge-asm/pull/2011) ran the same way. So a conversion erases
evidence on the tokens it has not reached, and F10 and F11 record that both runs precede them.

**The witness rule stands until a human rules on it, and this paragraph is the reason.** Two rival
rules close shape 6, and each answers ADR-0212. The arm reads the oldest revision that spelled a
number, which is the assertion the arm claims to measure change since. Or the arm reads every such
revision, and a disagreement between them is the drift. The second reads more history, and its
marginal cost is one more historical tree per extra revision: the arm runs one inventory per row
over every tree it holds, so an extra revision adds a path to that batch rather than a run. Neither
cost is measured, and either rule changes every count in §9 facts F9, F10 and F11. So this SPEC
lists the shape and leaves the rule alone.
[#2337](https://github.com/winniel123/verge-asm/issues/2337) carries the ruling and the
re-measurement that follows it. `docs-site/scripts/sweep-history.test.mjs` pins the shape, so a
change to the rule moves a test.

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

**This count is a floor, not a rate.** `92f2db2` already held the conversions of
[#2011](https://github.com/winniel123/verge-asm/pull/2011) and
[#2020](https://github.com/winniel123/verge-asm/pull/2020). Each run rewrote citing lines and
converted one token of each. §7 shape 6 then makes the run the witness for any token it left
numbered on the same line, so a drift before the run is unreadable there and a candidate there never
reaches this count. How many that hides is unmeasured. The runs do not blind the tokens they
converted: those spell no number afterwards, so the witness falls back to the revision that wrote
it. F9 carries no such note. It ran against `c196bed^`, the tree before both conversions.

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

**The sample is drawn from a population two conversion runs precede.** The 172 come from the F10
run, which post-dates the conversions of
[#2011](https://github.com/winniel123/verge-asm/pull/2011) and
[#2020](https://github.com/winniel123/verge-asm/pull/2020). §7 shape 6 substitutes a later witness
rather than dropping a row, so it changes the tree a verdict rests on and never the size of the
sampled set. This fact's own false positive shows where the mechanism stops: ADR-0198 line 23 sits
on a line [#2011](https://github.com/winniel123/verge-asm/pull/2011) rewrote, and the arm's witness
there is still `fa2c312`, because that run converted the token and left it spelling no number.
Verified 2026-09-18 against `main` at `51f4b18`.

A run on 2026-09-16 produced fact F12. It ran against `ec0404e`, the branch this fact lands from.
That branch is `main` at `ffa10a8` plus its own merges. Two commands reproduce the inputs.
`npm run sweep:line-anchors -- --audit` from `docs-site/` gives the population.
`go run ./cmd/godecls --root .` over `git ls-files '*.go'` gives the Go half of the tree index.

**F12 — the audit judges 626 written anchors and calls 512 unproven. 208 of them spell a name the
target does not declare, in a span the guard reads. §5.3 leaves the tree test open to 14 of the 459
names they spell. A hand read finds every one of the 14 a false positive.**

The 512 are 97 in `docs/spec`, 413 in `docs/adr` and 2 in the root documents. The 208 sit on 195
citing lines. They spell 459 such names between them, and 307 of the names are distinct. The other
304 anchors spell no name the guard reads. So no rule over the citing line reaches them. The shapes
below are exclusive, in §5.3's own order of closure. Each one counts a name the target lacks.

| Shape | What the name is | §5.3 | `docs/spec` | `docs/adr` | Total | Distinct |
| --- | --- | --- | --- | --- | --- | --- |
| A — declared elsewhere | some inventory in the tree carries it under `sameName` | closed | 14 | 110 | 124 | 90 |
| C — weak | limb 1 holds: a file the markdown row does not claim spells the token | closed | 55 | 266 | 321 | 205 |
| B — case-folded only | no inventory carries it, and one carries a case-folded spelling | **open** | 5 | 3 | 8 | 7 |
| D — residue | no inventory carries it folded or not, and no such file spells it | **open** | 2 | 4 | 6 | 5 |

§5.3 closes the tree test to a name some inventory still carries, and to a weak name. It closes it
to neither B nor D. **So B and D together are the population a rule of
[#2155](https://github.com/winniel123/verge-asm/issues/2155)'s shape raises.** That is 14
occurrences over 12 anchors. Seven of the 12 spell such a name before the citation, so rule 1 of
§5.2 degrades them. The other five spell one only afterwards, so they enter rule 2's review queue.
By syntax the 459 are 399 bare identifiers, 40 dotted names and 20 file names.

**All 14 are false positives.** No site below carries a region anchor. One would put this fact's
own lines into the population it counts.

| Site | The name | Why the anchor stands |
| --- | --- | --- |
| `docs/spec/golden-corpus.md` line 219 | `Bumped` | a table column header inside a two-word span, and the anchor is a heading slug |
| `docs/spec/comment-policy.md` line 2321 | `fa97` | the tail of the commit hash `860fa97`, which the identifier pattern cuts at the digits |
| ADR-0160 line 39, on both anchors | `account.password_hash` | a database column, and both anchors name the declarations the row is about |
| ADR-0179 line 33 | `signalRow.seenAge` | a struct field of the anchor's own type, so the prose supports the anchor rather than rivalling it |
| ADR-0147 line 23 | `assetsWatched` | a withdrawn local; the anchor's declaration still computes the stat, and the stale part is the expression the prose quotes |
| 6 more anchors, 8 occurrences | `artifactdoc.go`, `password.go` and 5 like them | shape B is a retired line anchor, and a file name is no declaration name |

So the tree test's measured false-positive rate on this boundary is 14 of 14. Its measured
true-positive population on this boundary is empty.

**The `fa97` row no longer reaches the population, and this fact keeps its figures.** §5.2 rule 4
now drops a span that spells a bare commit hash, so the `comment-policy.md` site yields no
candidate name. That site is one occurrence of shape D, on one anchor, and the hash is spelled
before the citation. Read against the repaired guard, the raised population is 13 occurrences over
11 anchors. Shape D falls to 5 occurrences and 4 distinct names. Six of the 11 spell their name
before the citation. This paragraph is arithmetic over the table above, not a second run. No rule
of [#2155](https://github.com/winniel123/verge-asm/issues/2155)'s shape turns on it. The
false-positive rate stays every raised name, at 13 of 13. The true-positive population stays empty.
A run that re-measures this fact supersedes both readings.

**Shape B is a retired line anchor, spelled as a bare basename.** §8.4 of
[citation anchors](citation-anchors.md) stages that form, and the conversion of
[#2120](https://github.com/winniel123/verge-asm/issues/2120) has not reached it. Each of the 8 sits
in a span such as `` `artifactdoc.go:147` ``. The guard reads it, because `citesPath` strips the
line suffix and finds a basename that spells no directory. So the span survives, and the basename
becomes a candidate. The name then folds on its extension onto a declared type.
`artifactdoc.go` folds onto the `Go` that `internal/commentlint/surface/golang.go` declares, and
`_integration_state.sql` folds onto the `SQL` of its sibling. A later conversion of that stage
empties shape B, and no rule is needed.

**The true positives left the corpus before it was measured.**
[#2011](https://github.com/winniel123/verge-asm/pull/2011) degraded the ADR-0186 rows by hand. So
no name they spell is a written anchor's rival today. This run scored them as names against its
tree index. Shape D holds `certExpiryWindow`, shape B holds `weakKeyOrSignature` and
`selfSignedOf`, and shape C holds `notAfter` and `notBefore`. So the tree test reaches three of
those five. It reaches two of the three for the wrong reason, because those declarations are alive
under an exported initial.

**Relocation is a shape of its own, and shape B holds none of it today.** `sameName` is
case-sensitive. So a live declaration reads as absent where the prose spells it with another
initial. `internal/signalfacts/signalfacts.go` declares `WeakKeyOrSignature` and `SelfSignedOf`,
and ADR-0186 spelled both with a lowercase initial. Neither is a written anchor's rival now, and
none of the 8 shape-B names is a relocation. Folding the case inside `sameName` is not the repair
either. Measured over all 459 names, a folding `sameName` makes 3 of them a rival of the anchor's
own target. They sit at ADR-0150 line 52, ADR-0189 line 83 and ADR-0193 line 59, and they spell
`connect`, `ReserveCTSlot` and `sct`. All three come before the citation, so all three would
degrade a sound anchor under §5.2 rule 1. None is a relocation.

**Limb 1 reads a dotted name two ways, and the raised population turns on it.** The table asks
whether the tree spells the whole dotted name. The other reading asks whether it spells the tail:
`seenAge` of `signalRow.seenAge`, and `password_hash` of `account.password_hash`. Under that
reading shape B empties, D falls from 6 to 3, and the raised population falls to 3 occurrences. The
false-positive rate is 14 of 14 under the first reading, and 3 of 3 under the second.

**What this run did not measure.** §5.3 calls a name weak when either limb holds, and this run
applied limb 1 alone. Limb 2 names a standard-library field or method, with an exported initial
folded. A hand read applied it over the 14, and no run applied it mechanically. A mechanical run
can only move a name out of B or D into C. So the raised population can only shrink. Shape A
already takes every name some inventory carries. So a name that reaches C is spelled in a position
no declaration vocabulary carries, which is what limb 1 asserts.

**How the shapes were derived.** The guard scored its own reads. For each span of an unproven
citing line, the run rebuilt the line with every other span blanked. It then called `rivalName`
once per identifier of that span, against a one-name inventory holding it. A `suspect` answer then
means the guard reads that name in that position. The blanking is what makes the answer sound.
`sameName` matches a suffix, so an un-isolated probe for `fetcher.go` answers `suspect` on a bare
`go` from any other span. An earlier run of this fact over-counted 10 names that way. No second
matcher was written, per §5.3, and the tree holds no harness for this run. So this paragraph is the
record of the method. The tree index took every tracked file under the four declaring rows. It
holds 9,163 names from 737 Go files, 298 from the `-- name:` queries, and 86 from the `{{define}}`
templates. It also holds 4,447 heading slugs from 480 markdown files. Limb 1 is a token search over
every tracked file the markdown row does not claim.

A run on 2026-09-17 produced fact F13. It ran against `main` at `9b1707e`, plus the branch that
ships §5.2 rule 5. Two commands reproduce it, both from `docs-site/`:
`npm run sweep:line-anchors -- --audit --report <file>` and
`npm run sweep:line-anchors -- --report <file>`.

**F13 — no verdict on today's corpus turns on a name several declarations share.** The audit judges
633 written anchors, and `rivalName` reads every one. 100 corroborate and return, so the rival scan
reads 533. The dry run reads 976 tokens and reaches a target for 332 of them. 35 of those
corroborate, so the rival scan reads 297. A shared name arises in none of the 830 scans.

Either side of rule 5 the audit report and the conversion plan carry the same verdict on every row.
The two runs that measured it ran before this section's own lines moved, and their reports are
byte-identical. A later run shifts the line number of a citation this document holds, and nothing
else.

**The shape is real, and it sits in the population no arm reads today.** ADR-0209 line 57 spells
three path-less tokens, and the sweep holds each one for a reader. Supplied with the three proven paths, the
guard before rule 5 degrades all three, naming `CTContextCancelled.isCTOutcome` the rival each
time. After rule 5 the same three derive `ProberOutcome`, `CTOutcome` and `ZoneOutcome`. How many of
the 459 path-less tokens sit on such a line is unmeasured.
[#2156](https://github.com/winniel123/verge-asm/issues/2156) supplies the paths that would count
them.
