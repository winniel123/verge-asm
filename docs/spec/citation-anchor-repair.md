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
| The bound on what any guard can prove | §7 |
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
- `docs/spec/aperture-statement.md` line 408 reads "`cmd/web/cold.go#server.coveragePage` holds
  `apertureMeters`". The anchor is right, and the rival is a declaration inside it.

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

---

## 6. The repair

### 6.1 `docs/spec` already carries the defect

Ticket 9 of map [#1967](https://github.com/winniel123/verge-asm/issues/1967) converted `docs/spec`
before the guard existed. The guard now reports 4 suspect anchors there, and 2 of them are real.
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

### 6.3 A suspect anchor degrades, and is never repointed

A suspect anchor keeps its path and drops the anchor. The citation then says less, and everything it
says is true.

Repointing it to the rival was rejected. The rival is the name the prose spells, and that is
evidence, not proof. §4.3 measures two classes where the rival is not the target. A tool that
repointed would write a new wrong anchor in exactly those cases, and it would carry the gate's
endorsement again.

**Changing what a document asserts is not a repair.** For an ADR that route runs through a later
ADR, per [adr governance](adr-governance.md) §4. Repointing a citation to a different declaration
can change an assertion. Degrading one cannot.

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
2. **The unproven population needs a different instrument.** A candidate is the target's own history:
   a citation added in one commit, against a target changed later, is a drift candidate. This SPEC
   does not rule that instrument. It records the gap.
3. **No count retires this effort.** [Citation anchors](citation-anchors.md) §8.2 proves the sweep
   complete with an empty burn-down list. No list proves the anchors correct, because the
   correctness question has no list.

---

## 8. Non-goals

**No check refuses an unproven anchor.** A refusal would fail on the 301 anchors of fact F6, and
almost all of them are sound. The guard runs at conversion time and in the audit. It is not a gate.

**No check refuses a bare path.** [Citation anchors](citation-anchors.md) §7.2 asserts correctness,
never presence. Every degrade this SPEC orders produces a bare path, so a presence rule would turn
the repair into a violation.

**No renderer resolves an anchor.** [Citation anchors](citation-anchors.md) §9 already rules this,
and this SPEC adds nothing.

**No ADR records this SPEC.** The decisions here are corrections to a mechanism that one pull
request built and the same pull request measured. A human opens the issue that becomes an ADR, per
`CLAUDE.md`.

---

## 9. Established facts

Every fact below was measured on 2026-09-14, against `main` at `5e661e0`. Each one is reproducible
with the guard the tree ships.

**F1 — `docs/spec` holds 126 written anchors on a target with an anchor vocabulary. 4 are suspect,
8 are corroborated, and 114 are unproven.**

**F2 — `docs/adr` holds 219 such anchors. 0 are suspect, 32 are corroborated, and 187 are
unproven.** The range converted under the guard, so a suspect anchor could not land.

**F3 — 2 of the 4 suspect anchors in `docs/spec` are real.**

| Document | Line | Anchor written | Rival | Verdict |
| --- | --- | --- | --- | --- |
| `docs/spec/audit-act.md` | 466 | `cmd/web/auth.go#initials` | `validateCredentials` | real |
| `docs/spec/audit-act.md` | 1510 | `cmd/web/auth.go#profileRelTime` | `sessionIP` | real |
| `docs/spec/aperture-statement.md` | 408 | `cmd/web/cold.go#server.coveragePage` | `apertureMeters` | class A |
| `docs/spec/audit-act.md` | 338 | `cmd/web/onboarding.go#server.onboardingStep` | `server.onboarding` | class B |

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
