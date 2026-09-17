# Citation anchors

- **Status:** Accepted — spec content for [#1931](https://github.com/winniel123/verge-asm/issues/1931)
- **Governs:** the form of a citation that names a place inside another file, and the required check that verifies it
- **Wayfinder map:** [Map: a drift-proof citation into a source file (#1931)](https://github.com/winniel123/verge-asm/issues/1931)
- **Triage:** [#1916](https://github.com/winniel123/verge-asm/issues/1916)

A document cites a place in another file. Today it names a line number, and that anchor drifts.
Every insertion above the cited line moves the place, and the citing document is a different file.
The pull request that shifts the line never opens that document.

No check reports the drift. The citation extractor's path pattern in
`docs-site/scripts/citations/extract.mjs` holds no colon, so a `path:NNN` token never becomes a
candidate. The token is invisible. It is not judged and passed.

This SPEC rules three things. It names the citation form (§3 to §6). It says what happens to the
existing population (§8). It says what a checker verifies, and which required check owns the rule
(§7).

**This document is a SPEC only.** It converts no citation and it writes no checker. A downstream
`/to-tickets` and implement effort builds both. This document follows STE-flavored mode, per the
[documentation style standard](documentation-style-standard.md) §2.

Seven tickets on map [#1931](https://github.com/winniel123/verge-asm/issues/1931) settled the
content below. Each ticket holds the measurements and the rejected alternatives behind its ruling.
§11 lists the facts this SPEC rests on.

---

## 1. Purpose and scope

### 1.1 What this SPEC decides

| Part | Section |
| --- | --- |
| Terms | §2 |
| The citation form, and the row table that gives an anchor its vocabulary | §3 |
| The Markdown row | §4 |
| The refusal of a line anchor | §5 |
| The ADR Decision proposal block's `Site` field | §6 |
| What the checker asserts, and which required check owns it | §7 |
| The existing population, the burn-down that ended it, and what retires the stage that replaced it | §8 |
| Non-goals | §9 |
| The change this SPEC makes to `docs/spec/adr-governance.md` | §10 |
| Established facts | §11 |

### 1.2 The boundary

The rule reaches the files `docs-site/scripts/citations/scope.mjs` already walks. That set is
`docs/adr`, `docs/spec`, `docs/agents`, `docs/guides`, `design-system`, `docs-site`, and six root
files. `docs/research` sits outside it, per [#1450](https://github.com/winniel123/verge-asm/issues/1450).

The rule and its enforcement share one boundary. A rule a required check cannot reach is not a rule.

The boundary is the scope module, never a list of directories copied into this SPEC. A later change
to that module moves the rule with it.

§6 adds one surface outside the tree: the `Site` field of an ADR Decision proposal block in a
pull-request body.

### 1.3 Out of scope

- **A bare-prose path that carries no backticks.** The extractor inspects inline code and link
  targets only.
- **A line anchor in code comment prose.** Fact F8 measures one instance repo-wide.
- **Repairing a dead path.** Fact F4 measures one dead path across the whole surface.
- **Repairing the content of a degraded anchor.** §8.4 rules that such an anchor keeps its path and
  loses its line. The record of those anchors feeds a separate effort.
- **The 122 line anchors in `docs/research`.** §1.2 puts them outside the boundary.
- **Renumbering or re-heading a target file** so that a citation can reach it. A citation never
  edits its target.

---

## 2. Terms

| Term | Meaning |
| --- | --- |
| Citation | An inline-code span or a Markdown link target that names a path in this repository. |
| Target | The file a citation names. |
| Anchor | The part of a citation after `#`. It names a place inside the target. |
| Line anchor | The retired form, `` `path:NNN` ``. A line number names the place. |
| Declared name | A name the target's own language declares: a Go top-level declaration, a sqlc query name, a template define name, or a Markdown heading. |
| Containment anchor | One token the target file contains. A checker verifies containment, not declaration. |
| Row | One line of the §3.2 table. A row keys on a path predicate, and it fixes the anchor vocabulary for that predicate. |
| Degrade | To drop an anchor and keep the bare path, because no anchor is derivable. |
| Burn-down list | The staged exemption list of §8.2, now retired. It held the line anchors the sweep had not yet converted. |
| Staged form | A line-anchor form the scanner reports and §5 does not refuse. §8.2 names them. |
| Staged count | The number of in-scope tokens of a staged form. An empty one retires the stage, per §8.2. |
| Held | Of a token: the derivation writes neither an anchor nor a bare path for it, so the sweep leaves it exactly as it found it. |

---

## 3. The form

### 3.1 The rule

**A citation that names a place in another file is one inline-code span. The anchor names the
declaration that encloses the place, as the target's own language declares it. The anchor is
omitted when the target's language declares nothing there.**

```
`internal/queue/hot.go#hotCore`
```

The anchor names a **region**, not a point. This is the property that kills the drift. An insertion
inside a declaration moves no anchor. A rename or a deletion breaks the anchor, and §7.1 makes a
broken anchor turn a required check red.

### 3.2 The row table

A row keys on a **path predicate**, never on a bare extension. The predicate follows the tool that
owns the file.

| Path predicate | The anchor names | Anchors today |
| --- | --- | --- |
| `**/*.go` | the enclosing top-level `func`, `type`, `var`, or `const` | 911 |
| `db/queries/**/*.sql` | the `-- name:` query name | 24 |
| `design-system/templates/**/*.tmpl` | the `{{define}}` name | 41 |
| Markdown | a heading slug. §4 rules the spelling. | 23 |
| any other target | nothing. The token is a bare path. | 50 |

The rows that carry an anchor vocabulary cover 999 of the 1,049 resolving anchors, 95%.

Four rules govern the table.

1. **The table is open.** §3.1 is the rule, and the table is its worked-out application. A target
   kind with no row takes a bare path. A later edit of this SPEC adds a row. A missing row never
   blocks a citation.
2. **A target that declares no name MAY carry a containment anchor.** The anchor is one token the
   target file contains. `db/migrations/**/*.sql` is covered by this rule, and it earns no
   exception. A goose migration declares no names, and a cited migration runs 31 lines at the
   median.
3. **A containment anchor must occur at least once. It need not be unique.** Containment proves
   that the token still exists. **It does not prove that the token names the cited place.** A
   reader who trusts a containment anchor further than that is misled.
4. **A declared name is MUST. A containment anchor is MAY.** A row with a vocabulary fixes the
   spelling of an anchor an author writes. §7.2 states the limit of that MUST.

### 3.3 Spelling, omission, and degradation

1. **A Go method is spelled `Receiver.Method`.** It carries no pointer star and no parentheses.
2. **The anchor is omitted** when no declaration encloses the place, or when the claim is about the
   file itself. The token is then a bare path, which the `citations` check judges today.
3. **An anchor that is not derivable degrades to a bare path.** The author keeps the path and drops
   the false precision. **An author never guesses an anchor, and a sweep never guesses one.** See
   §8.4.
4. **The anchor uses the extractor's own character class**, which is `[A-Za-z0-9_.@+-]` and `/`. It
   admits no space and no quote. Every measured declared name and every measured heading slug fits
   that class.

### 3.4 Disambiguation

One document may cite the same `(target, anchor)` pair more than once, and mean two different
places inside one declaration. Such a citation MAY carry a disambiguating snippet.

The snippet is a second code span, immediately after the anchor span. It holds a verbatim substring
of **one** line inside the declaration. Whitespace collapses for matching.

The pairing is safe because the rule reads "the code span immediately after an **anchor** span". An
anchor span identifies itself by its `#`. A bare path beside a second code span is a common and
unrelated shape, and the rule does not reach it.

The rule is kind-blind. It reaches every row of §3.2.

### 3.5 Both spellings

The rule binds the inline-code span and the Markdown link fragment, for **every** row. So
`[hotCore](internal/queue/hot.go#hotCore)` carries the same anchor vocabulary as the code span, and
one verifier reads both.

The extractor emits a link citation and a code citation into one stream with the same raw value.
Binding one spelling alone would add machinery to preserve a hole.

---

## 4. The Markdown row

A cross-file Markdown citation uses the `` `path#anchor` `` form of §3. It does not use `§`.

**The anchor is a `github-slugger` heading slug.** A slug resolves as a real link on GitHub and in
`docs-site`. A dotted `§3.1` resolves in neither.

Six rules govern the row.

1. **Any heading level is citable, H1 to H6.** A numbered heading earns no privilege. 23 of 23
   resolving Markdown anchors sit under some heading, and about 7 sit under a numbered one. Ten
   cited targets carry no numbered section at all.
2. **`ADR-nnnn §n.n` survives beside this row, and the two are disjoint.** The key is how the
   citation names its target.

   | Form | Target named by | Verified by |
   | --- | --- | --- |
   | `ADR-nnnn §n.n` | ADR number | `adr-sections`, today |
   | `` `path#slug` `` | path | the rule of §7 |

3. **A target with no heading degrades to a bare path**, per §3.3. A citation never re-heads its
   target.
4. **A de-duplicated slug is refused.** `github-slugger` appends `-1` when one file repeats a
   heading. That suffix is positional. A third copy of the heading, inserted above, makes the old
   `#consequences-1` point at the new heading **in silence**. It does not fail. That is the drift
   this SPEC exists to kill.
5. **The anchor is verified by a heading inventory, never by containment.** The slug is derived, so
   no heading holds it verbatim.
6. **The inventory slugs every heading, H1 to H6, in document order, on the inline-markup-stripped
   label.** Both halves are load-bearing. `docs-site/scripts/check-links.mjs` records why a subset
   fails: slugging a subset diverges the de-duplication counter from the renderer. Stripping
   matches what both renderers slug, and it changes 83 of 3,878 measured headings.

---

## 5. The refusal

**A new line anchor is refused inside the §1.2 boundary.** A citation names no line.

The refusal covers a line anchor in any position. An inline-code span, a Markdown link target, and a
line fragment:

```
`path/to/file.md:247`
](path/to/file.md:247)
#L247   #L247-L260
```

The installed base outside the code span is zero. The `#L` form is the one to expect next, because
GitHub's own "copy permalink" produces it.

**One carve-out.** A citation the gate resolves as `on-ref` is exempt. A line number pinned to a
named ref cannot drift, because the ref is frozen. That is not the defect this SPEC repairs. The
carve-out is narrow, and it is machine-detectable. The extractor requires the words *branch*,
*commit*, *ref*, *revision* or *tag* in the same block. It exports the helper that finds them.

A commit-pinned permalink stays an escape hatch for the narrow historical case. **It is not the
form.** It resolves forever and it is wrong forever, and no check can report that the live code
moved on.

---

## 6. The ADR Decision proposal block's `Site` field

`docs/spec/adr-governance.md` §2 requires a `Site` field in every ADR Decision proposal block. So
the repo mints a fresh anchor on every ADR pull request.

**The `Site` field takes the `` `path#anchor` `` form of §3.** The row table picks the vocabulary
by path predicate. An underivable target degrades to a bare path. `Site` earns no exemption.

Four rules bound this.

1. **Reach.** This SPEC reaches the `Site` field and nothing else in a pull-request body. It does
   not rule free prose in a body. It does not reach the GitHub issue a human opens from a ticked
   block. The ADR file that issue becomes sits in the tree, so §7.4 already covers it.
2. **Base tree.** A `Site` anchor resolves against the pull request's **merge ref**. That is the
   tree the checkout already hands every `doclint` job. The anchor names the place as it stands
   after the pull request lands, never before.
3. **Enforcement.** The `citations` check verifies `Site`. **So that check asserts over the whole
   tree and over one named pull-request-body field.** This widens §7.4, and the widening is
   deliberate. `citations` is the only job that already holds all three parts. It holds the
   resolver, the row table, and the Go parse of §7.3.
4. **No conversion.** The rule is new-blocks-only. The burn-down list of §8.2 never gains a `Site`
   entry. All four merged blocks carry an unticked `Ratified`, so `docs/spec/adr-governance.md` §7
   makes them refusals. The rule starts over an empty population.

Two of the four merged `Site` anchors had already drifted when this was measured. See fact F10.

---

## 7. The checker

### 7.1 The checker asserts two things

**Arm A, refusal.** No new line anchor inside the §1.2 boundary. This is §5.

**Arm B, resolution.** Every anchor a citation writes must resolve against its row in §3.2.

Both arms are necessary. Arm B alone leaves the invisibility of fact F1 open, because a fresh
`:NNN` token stays invisible to the extractor. Arm A alone freezes 945 anchors that no check reads,
and fact F3 measures at least 129 of them as already stale.

### 7.2 The checker asserts correctness, never presence

**A bare path stays legal for every target kind, forever.** §3.2 rule 4 binds the **spelling** of
an anchor an author writes. It does not require an author to write one.

This reading is the only one that keeps §8.4 free. Under the other reading, every anchor the sweep
cannot derive converts to a bare path and goes red at once. The sweep's own output would then need
a permanent exemption, which §8.2 refuses.

A pull request with no ADR Decision proposal block passes §6. A `Site` field that carries a bare
path passes forever.

### 7.3 The rule lives in `citations`, and that job gains a Go toolchain

The `citations` check grows both arms. It is required today, it reads the whole tree, and it
already owns path resolution and the exemption machinery.

**A column-0 regex cannot verify a Go anchor.** It fails in both directions, measured over all 685
tracked Go files:

- **False negatives.** 588 of 8,803 citable anchors sit inside a parenthesised `const` or `var`
  group, so the declaring line is indented. Across the Go files citations actually name, 342 of
  3,574 resolving anchors are grouped, **9.57%**.
- **False positives.** 14 column-0 lines match a declaration regex and declare nothing. Each sits
  inside a raw-string literal that embeds Go source as a test fixture. A regex checker would accept
  a fabricated anchor.

Both faults are fatal for a required check. One is a false red on 9.57% of the cited Go surface.
The other is a silent pass on a declaration that does not exist.

So the Go row needs a real `go/parser` parse. `docs-site`
declares no Go parser and cannot gain one cheaply. **The `citations` job adds one `setup-go` step,
and the Node checker shells out to a small Go command that emits a declaration inventory.**

The `-- name:` and `{{define}}` rows do not need a parse. A column-0 marker is the real declaration
syntax for both. All 85 `{{define}}` names and all 291 sqlc query names are distinct repo-wide.

**The SPEC names the job and the property together.** The rule is enforced by a required check that
reads the whole tree, and that check is `citations` today. Naming the job is house style here, and
`docs/spec/adr-governance.md` §9 lists CI checks by name. Stating the property beside the name means
that a CI rename breaks one sentence, not the rule.

This choice needs **no ruleset change**. A new `setup-go` step reads `go-version-file: .go-version`,
as all nine existing steps do. `scripts/check-go-pins.sh` reads no workflow file, so a hardcoded
version would drift with nothing to catch it.

### 7.4 The checker reads the whole tree

A pull request that renames or deletes a Go declaration breaks an anchor in a document it never
opens. A diff-scoped check cannot see that class, and that class is the defect this SPEC repairs.

This is why the `citations` job carries no paths filter today.

§6 rule 3 adds the one pull-request-body field to this reach.

### 7.5 The refusal uses its own scanner

Arm A does **not** widen the extractor's path pattern to admit a colon.

Three reasons. The pattern is a delicate gate, and §3.3 rule 4 already spends its one safe
character on `#`. A widened class would report a line anchor as "dead: no such path in the tree",
which names the wrong fault. And the burn-down list is not the concept
`docs-site/scripts/citations/exemptions.json` holds.

Arm A reads inline-code and link nodes directly, and it covers the three positions of §5.

### 7.6 Arm B judges an anchor only where the path resolves

`docs-site/scripts/citations/classify.mjs` assigns seven non-`ok` statuses. Arm B judges an anchor
only on a citation whose path resolves in the tracked tree.

- For `foreign` the tree is not ours to parse.
- For `untracked` and `withdrawn` no file exists.
- For `dead` the path failure is the real fault, and a second error on one token is noise.

**The pass-over is listed, never silent.** `docs-site/scripts/check-citations.mjs` already carries
the rule in its own words. Every bucket the gate passes over is listable, so no suppression is
silent. Arm B inherits an existing verbose bucket.

### 7.7 An unparseable target is operator error

`docs-site/scripts/check-citations.mjs` already separates two failures. An unreadable document exits
**2**. A dead path exits **1**.

**An unparseable target file, or an absent `go` binary, takes exit 2.** It is not a violation, and
it is not a pass.

The rejected posture reports and does not violate. That posture is correct for an unreachable ref,
because a shallow clone genuinely cannot judge that claim. An unparseable Go file is a claim the
gate should judge and could not. That is the invisibility of fact F1 under a friendlier name.

---

## 8. The existing population

### 8.1 Conversion is required

**The 945 in-scope anchors convert.** 803 sit in `docs/adr`, and 142 sit in `docs/spec`. See §11
fact F9.

**An anchor inside a landed ADR converts without a relation, without a marker, and without a later
ADR while it still names the same declaration.** `docs/spec/adr-governance.md` §3 rules that
repointing a stale anchor to that declaration is a factual repair, and a factual repair needs no
relation. ADR-2086 rules the line, and [citation anchor repair](citation-anchor-repair.md) §6.3
carries the same limit.

**An anchor that now names a different declaration was never inside that authorisation.** It
changes what the ADR asserts, so it is not a repair. It runs through a later ADR under
`docs/spec/adr-governance.md` §4, and a human does it. **The limit binds at conversion time and
after it.** §8.2's list is retired, and the conversion of
[#2120](https://github.com/winniel123/verge-asm/issues/2120) still has line anchors to reach. A
conversion that would write an anchor on a different declaration is outside this authorisation, and
§8.4 degrades that anchor instead.

**803 was never a blanket over the anchors already written.** The history arm of
[citation anchor repair](citation-anchor-repair.md) §7 is the instrument that sorts them, and that
SPEC's §9 facts F9 to F11 measure it. **The arm is evidence and not a proof**, so no verdict below
settles an anchor on its own.

- **Judged consistent.** The witnessed line sat in the declaration the anchor names. That is
  evidence the authorisation covers it. That SPEC's §7 shape 3 bounds the evidence: an anchor
  already wrong when its number landed leaves no change for the arm to find.
- **Judged a drift candidate.** The witnessed line sat somewhere other than the declaration the
  anchor names. That is evidence of the different-declaration case, and that SPEC's fact F11
  measures a false-positive class where the number landed just outside the declaration the prose
  means. A human reads each one under [citation anchor repair](citation-anchor-repair.md) §6.3, and
  that reading is what puts the anchor inside the authorisation or outside it.
- **Not judged.** The arm reports an anchor unwitnessed when no revision of the citing line spells a
  number, and when the witness spells a different count of numbers than the line cites. It reports
  an anchor unreadable when it cannot resolve the witnessed line at all. The authorisation over such
  an anchor is unproven rather than wrong, and this section asserts nothing about it.

**This section states no count of the three classes.** The arm re-measures on every run, and it
reports a different figure as the tree moves. §11 fact F9's 803 stands unchanged: it counts the
tokens the conversion had to reach, not the anchors this section authorises.

Refusal alone was rejected. It freezes 945 anchors that no check reads, and fact F3 measures at
least 129 of them as already stale.

### 8.2 The refusal lands first, and a newly scanned form is staged before it is refused

The refusal of §5 lands **before** the conversion. Conversion first was rejected, because no ratchet
exists during the sweep, and the gap re-admits new tokens.

The list is seeded with the exact in-scope line anchors measured when the check lands. The check
refuses every line anchor the list does not hold. The sweep deletes entries as it converts.

- **What sizes the work:** the list's length, readable at any moment.
- **What proves it complete:** the list is empty.

Four rules govern the list.

1. **It is a new file.** It does not reuse `docs-site/scripts/citations/exemptions.json`. That
   file's entries claim "this path is absent, and here is why". A burn-down entry claims something
   else: "this token is legal until the sweep reaches it".
2. **One entry per `(document, token)` pair.**
3. **No `reason` field.** A reason field is what turns a burn-down into a permanent exemption. A
   permanent exemption is rejected, because the list then never empties. The completion proof above
   dies with it.
4. **A stale entry fails the check.** An entry the scanner no longer finds is a converted token
   nobody deleted, or a typo. Without this rule a forgotten entry silently re-licenses that exact
   token.

A conversion pull request edits the document and the list together. That is correct, because they
are one act.

**One pull request for the whole sweep is rejected.** 87 files, plus a new check, plus this SPEC is
too large to review. This SPEC rules that the sweep is staged. It does not design the per-pull-request
slices. `/to-tickets` cuts those.

**The sweep is complete, and the list retired.** Four conversion tickets emptied it, and
[#1980](https://github.com/winniel123/verge-asm/issues/1980) then deleted the file, its loader and
the stale-entry rule of rule 4. The check consults no list, and `--prune-list` is gone with the
rest. The rules above record why the list took the shape it did. They govern no live file.

**A second stage replaced the list.**
[#2120](https://github.com/winniel123/verge-asm/issues/2120) widened Arm A's scanner to two more
spellings of the retired form, and it staged both behind the ruling above. A scan hit now carries a
**form**. The `path` form spells a whole path. The `file` form spells a slashless file name. The
`bare` form spells no path at all, and it leans on a path the prose already named.

**The stage defers enforcement. It amends no rule.** §5 refuses a new line anchor in every form,
and this section changes nothing about that. The check enforces §5 for the `path` form alone. It
reports a `file` or a `bare` token and refuses neither.
`docs-site/scripts/check-citations.mjs` holds that split in one function, and both the document arm
and the Site arm read it. A reader of §5 who wants to know what the check enforces today reads this
section.

**The staged count is every in-scope token of a staged form.** It is not a measure of sweep work,
and it is not a burn-down of the sweep alone. It counts what the check would turn red if the stage
retired today, which is the one quantity that decides whether the stage may retire.

**The stage retires when the staged count is zero.** The split function is then deleted, the check
enforces §5 for every form, and one deletion retires both arms.

**Zero is a moment, not a steady state, so the emptying and the deletion are one pull request.**
The retired list was a ratchet: a token the list did not hold was refused the day it was written.
The stage replaced the list with a report, and a report stops nothing. So a document merged during
the stage may mint a `file` or a `bare` token freely, and the count rises again. It stood at 986
when the stage landed and at 990 when F11 was first measured, and ADR-2159 minted the difference. Any
sequence that empties the count in one pull request and deletes the split function in a later one
lets a third pull request re-open the stage in between.

**A staged token leaves the count by one of three routes.** Only the first belongs to the sweep.

1. **Conversion.** §8.3's derivation reads an anchor out of the token, and the sweep writes it.
2. **A reader's repair.** The token is a citation, and the derivation holds it because the document
   alone does not prove the path. The derivation holds every `bare` token, because §8.4's
   degradation would write a citation with no path at all. A reader names the path, or rewords the
   sentence. That work is [#2156](https://github.com/winniel123/verge-asm/issues/2156).
3. **A matcher repair.** The token is no citation, so no edit to the document is correct. The
   scanner stops matching it. A port number in prose is the measured instance, fact F11 records it,
   and that repair is [#2185](https://github.com/winniel123/verge-asm/issues/2185).

**Naming the three routes is what makes zero reachable.** Conversion alone can never empty the
count. The derivation holds every `bare` token, and a port is no citation that any conversion can
take. A retirement condition only conversion can satisfy is a target the instrument cannot reach,
and a gate that reports an unreachable target teaches a reader to ignore it.

**The held figure bounds routes 2 and 3 together. It is not the staged count.** A dry run of
`npm run sweep:line-anchors` splits the staged population into derived, degraded and held. Held is
every token the derivation writes nothing for, so it holds route 2 and route 3 in one bucket. A
reader who wants route 2 alone subtracts the route-3 population F11 measures. A reader reads the
held figure from the sweep, never from the check, and it is a component of the staged count rather
than a rival to it.

**Excluding a held token from the staged count was rejected.** The count would then read zero while
tokens the check refuses after retirement were still in the tree, and the instrument would certify
a retirement that turns a required check red on the merge that performs it. Held work is human work,
not absent work.

**Retiring the stage on a condition other than zero was also rejected.** Every such condition names
a population the check keeps reporting and has agreed to ignore. That is the permanent exemption
rule 3 above already refuses, and it kills the completion proof for the second time.

### 8.3 A citation converts, the extractor changes first

`#` joins the extractor's character class **before** any citation converts.

A conversion that lands first is invisible exactly as a line anchor is invisible today, and a
half-finished sweep then hides itself. The extractor's classifier already cuts a value at `#`, so
the path arm resolves the token the moment the class admits it.

No code span in the corpus carries a `#` today, so the change collides with nothing.

### 8.4 An anchor the sweep cannot derive degrades

Two groups resist mechanical conversion. Fact F3 counts at least 129 anchors whose line is provably
wrong, so no correct target reads out of them. §3.2's last row leaves about 50 anchors with no
anchor vocabulary.

Such an anchor keeps its path and loses the false precision. **The sweep never guesses a target.**

The sweep records each degraded anchor. That record feeds a separate content-repair effort. §1.3
puts the repair itself out of scope.

Hand-repairing each one was rejected. It turns a mechanical sweep into 129 or more judgement calls.

### 8.5 A conversion that repeats an anchor in the token's own address is held

A region is coarser than a line, so two lines inside one declaration derive one anchor. Where both
tokens sit in one address, the pair collapses. The document keeps its claim, and the two citations
stop supporting it.

ADR-0188 is the measured case. Its sentence names two statements in `EdgeFanout.overExtension`, and
the next two sentences read *the first* and *the second*. Both words lose their referent when the
pair collapses.

**The address is the span the listener carve-out of #2185 already reads**: the sentence for prose,
and the table cell, or the row where that cell holds no prose.
`docs-site/scripts/citations/lineanchor.mjs#addressScope` names it, and one walk serves both rules.
A repeat across two addresses is not held. Each address states its own claim, and a reader reads one
at a time.

**The sweep holds every token in a collapsing group, never one of them.** Converting one member does
part the pair. It also writes an address that is half converted, and no rule asks for that.

**One token spelled twice in one address collapses nothing.** The source already names one line
twice, so the conversion takes a reader nothing it had. A group holds only where it spells two
tokens or more. A third token that repeats the pair's anchor still collapses against it, and the
whole group then holds.

**An abbreviation ends no sentence here.** The address index is a grouping key, not the text window
the listener carve-out reads, so a wrong split drops a hold this section requires. The prose arm
therefore splits less often than the carve-out does. `e.g.` and `cf.` open no new address, while a
citation that closes a sentence, such as `` `pdf.go:113`. ``, still does.

This is route 2 of §8.2. A reader names a different target, or rewords the sentence, and the tokens
then leave the staged count. The reader performs that rewrite. The sweep may not, because §8.2's
route 1 converts anchors and asserts nothing new.

Two alternatives were rejected.

1. **Convert and accept the repeat.** The claim stays true, and the citations stop proving it. In
   ADR-0188 the prose then refers to two targets a reader cannot tell apart.
2. **Hold on the source line rather than the address.** A Markdown line is a wrap position, not a
   unit of meaning. ADR-0188's sentence wraps, so a line rule misses the case that proves the rule.

---

## 9. Non-goals

**No renderer resolves an anchor.** A citation is text. A renderer MAY link the path, because the
`citations` check already verifies it. A renderer MAY pass the anchor through as a link fragment,
where the target's own renderer honours it. A renderer MUST NOT resolve an anchor to a line.
Resolving one re-derives the line anchor this SPEC removes, and a containment anchor has no single
destination to resolve to.

Four notes bound the clause.

1. **It bans a mechanism, not an outcome.** It forbids a line number. It permits a link to the
   path, because the path is already verified, so a path link cannot drift.
2. **It says "a renderer", never `docs-site`.** The reason rests on the form, not on today's
   ingestion boundary, so the ruling does not expire when that boundary moves. It also reaches
   GitHub's own render of the tree.
3. **One clause covers a bare path and an anchored path.** A split rule would link one and leave
   the other plain, and a reader could not tell why.
4. **A Markdown heading slug resolves for free**, because a blob fragment honours a heading slug.
   The clause permits that and requires nothing.

**No check enforces this clause.** It is prose only. The forbidden behaviour would live in
`docs-site/src/pipeline/render.jsx`, which the `citations` check never reads. The `docs-site` job is
not a required check. And §7.2 asserts correctness, never presence, while the rendered anchor
population is zero.

The clause lives here and nowhere else. `docs-site/PIPELINE.md` documents what the pipeline does. A
note there would describe absent behaviour.

---

## 10. What this SPEC changes in `docs/spec/adr-governance.md`

Two documents that state one rule will drift. This SPEC is the single authority on the citation
form, so the pull request that lands it makes two edits.

1. **§3's citation paragraph becomes a pointer.** That paragraph states the cross-file citation
   rule for `docs/adr/` today. It keeps the sentence that repointing a stale anchor is a factual
   repair, because §4 of that SPEC depends on it.
2. **§2's terms table spells the field `` Site (`path#anchor`) ``.** A field-shape table that spells
   no shape fails its own job.

---

## 11. Established facts

Facts F1 to F8 were measured during triage of
[#1916](https://github.com/winniel123/verge-asm/issues/1916), on `main` at `072ff24`. F9 was
measured when map [#1931](https://github.com/winniel123/verge-asm/issues/1931) was cut. F10 was
measured on 2026-09-14. **Do not re-measure F1 to F10.** F11 was measured on 2026-09-16, and it is
the one fact below that a later session re-reads. Its own paragraph says why.

**F1. The extractor never sees a line anchor.** The citation extractor gates every inline-code
candidate through one path pattern, and that pattern's character classes hold no colon. So a
`path:NNN` token never becomes a candidate. A bare path in the same span resolves normally.

**F2. The surface is 1,070 tokens across 87 files.** That counts backticked `path:NNN` tokens in
tracked `docs/**/*.md`. A larger figure of 1,353 counts bare-prose occurrences too, and those are
out of the extractor's reach. **Never size work against 1,353.**

**F3. At least 12.1% are already stale.** Of the 1,070 tokens, 82 cite a blank line. A further 26
name a line past the end of the file, and 21 name a path that no longer resolves. That is 129
provable failures.
Treat 129 as a floor, not a rate. A citation that names a wrong but non-blank line scores as live.

**F4. There is no path-integrity hole.** Strip the line number from all 1,070 tokens. Then 1,048
classify `ok`, 20 classify `foreign`, and 1 classifies `dead`. The colon blindness hides a stale
line number. It smuggles no dead path past a required check.

**F5. A `§` answer alone would reach 1.0% of the surface.** Of the 1,049 anchors that resolve, 1,024
(97.6%) name a source file, not Markdown. The split is `.go` 911, `.sql` 47, `.tmpl` 41, `.md` 25,
everything else 25. Only 10 Markdown anchors sit under a numbered heading.

**F6. A path plus `§` is verified by nothing.** The `adr-sections` citation finder matches no
pattern for a `§` hung on a path. A deliberately bogus `§99.99` on the same path also matches
nothing. `ADR-0144 §99.99` matches and fails correctly.

**F7. A symbol anchor is derivable for most Go citations.** 857 of 911 Go anchors sit inside a named
top-level declaration. 845 of 850 enclosing declarations are unique within their own file. A
word-boundary grep of the bare name returns exactly one line for only 297 of 850. **A checker must
parse. It must not grep.**

**F8. Code comment prose holds one line anchor repo-wide.** So the surface is the documents, not the
comments. This SPEC does not reach `docs/spec/comment-policy.md`.

**F9. The reachable boundary holds 945 anchors, not 1,070.** 803 sit in `docs/adr` and 142 in
`docs/spec`. 122 sit in `docs/research`, outside the boundary. `docs/agents`, `docs/guides`,
`design-system`, `docs-site` and the root files hold zero.

**F10. Two of the four merged `Site` anchors had already drifted.** Each was read at its own merge
commit and on `main`. One drifted by two lines, from a comment line to a type declaration. A second
moved from a `continue` statement to the `for` statement above it. A third of the four is the pull
request that converted ADR-0144's line anchors, and its own body minted a fresh line anchor.

**F11. The staged population is 986 tokens across 60 files.** 522 spell the `file` form and 464
spell the `bare` form. The tokens divide 818 in `docs/adr` and 168 in `docs/spec`. A dry run of the
sweep derives an anchor for 306, degrades 140, and holds 540. The count read 990 across 63 files
until the matcher stopped counting 4 tokens that were no citation. Each of those 4 sat outside
`docs/adr` and `docs/spec`, and each named port 8080 of the `web` service. No conversion repairs a
port, so §8.2 route 3 took them. The `docs/guides` and `README.md` families now hold no staged
token.

**F11 is the one fact to re-measure.** §8.2 makes retirement conditional on reading the staged
count as zero, and §8.2 also rules that the count rises with every document merged during the
stage. The figure above was 986 when the stage landed. Read it from the check, and read the split
from a dry run of the sweep. **Every other fact above stays frozen.**

**A method note.** A later re-derivation found 1,067 tokens, where F2 states a total of 1,070
tokens. It counted 46 `.sql` targets and 23 `.md` targets, where F5 states 47 and 25 of them. A
second count found 796 citations naming a Go file, where F7's method found 850 of them.

**Treat neither figure as a correction to the other.** Both put the Go arm at roughly 200 files. The
deltas change no ruling here.
