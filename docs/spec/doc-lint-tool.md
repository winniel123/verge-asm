# Documentation lint tool

- **Status:** Accepted — spec content for [#790](https://github.com/winniel123/verge-asm/issues/790)
- **Governs:** [Documentation style standard](documentation-style-standard.md) §4.1 and §4.4
- **Wayfinder map:** [Documentation lint tool SPEC (#790)](https://github.com/winniel123/verge-asm/issues/790)
- **Ticket:** [#793 Write the documentation lint tool SPEC](https://github.com/winniel123/verge-asm/issues/793)

This document specifies an advisory documentation lint tool. The tool checks the mechanical STE
rules from the [documentation style standard](documentation-style-standard.md). The standard defers
this tool in its §9. This SPEC is the deferred effort.

This document is a SPEC only. It does not build the tool. A downstream `/to-tickets` and implement
effort builds it. This document itself follows STE-flavored mode (see the style standard §2.1).

The style standard set the mechanical rules. Two frontier tickets then tested each rule against
real tooling:

- [#791](https://github.com/winniel123/verge-asm/issues/791) surveyed the retext and remark
  ecosystem for coverage of the five mechanical gates.
- [#792](https://github.com/winniel123/verge-asm/issues/792) ran the two part-of-speech (POS) rules
  over all 176 in-scope docs to measure precision.

Their findings set the v1 rule set below. This SPEC records the reason for each include and each
deferral, so the implement effort does not re-litigate a closed question.

---

## 1. Purpose and scope

### 1.1 What the tool is

The tool is an **advisory** documentation linter. It reports style violations. It never gates a
merge. It runs in two places:

1. A local command a writer runs by hand.
2. A non-blocking CI check.

The tool checks the mechanical STE rules only. It does not check the lexical rules. It does not
check the ADHD format subset. A rule ships only where a fixed algorithm reaches acceptable
precision on real docs.

### 1.2 What the tool is not

- **Not a merge gate.** The CI check is `continue-on-error`. A violation never blocks a merge. The
  human review in the style standard §7 stays the acceptance path.
- **Not a lexical checker.** The lexical rules need a dictionary the ASD-STE100 skill does not ship.
  The tool cannot check them.
- **Not an ADHD-format checker.** The ADHD subset is per-section. The tool cannot detect a
  procedural section, so it drops every section-dependent rule (see §7).

### 1.3 In-scope files

The tool lints the same file set the style standard §1.1 governs:

| Path | Family |
| --- | --- |
| `docs/adr/*` | ADRs |
| `docs/spec/*` | specs |
| `docs/agents/*` | agent docs |
| `docs/guides/*` | guides |
| `docs/research/*` | research |
| `CONTEXT.md` | domain model |
| `CLAUDE.md` | agent config |
| `README.md` | project overview |
| `SECURITY.md` | security policy |

The tool skips the style standard §1.2 out-of-scope paths: `docs/correspondence/`,
`docs/wayfinder/`, `CHANGELOG`, generated files, token files, and `docs/guides/embed.go`.

### 1.4 File path drives the check

The tool keys each file off its path. The path selects the family (the §1.3 table). The family
selects the STE mode (the style standard §3 table). The mode selects the rule set.

**A note on modes.** The style standard §4.1 mechanical gates apply in both Strict mode and
STE-flavored mode, with no difference. So for the v1 mechanical rules, every in-scope family gets
the same rule set. The mode split changes only the lexical rules, which the tool does not check.
The tool still reads the path for one reason: to decide whether a file is in scope at all.

---

## 2. The v1 rule set

The tool ships four rules in v1. Two frontier tickets set this list. Each row states the severity,
the detection method, and the source decision.

| Rule | Severity | Method | Style standard rule |
| --- | --- | --- | --- |
| No semicolons | **error** | Tree-walk for a `;` punctuation node | §4.1 rule 1 |
| Sentence-length cap | **error** | Count words per sentence | §4.1 rule 2 |
| No phrasal verbs | **error** | Curated wordlist match | §4.1 rule 5 |
| Simple tenses | **warning** | POS: `have`/`has`/`had` + past participle | §4.1 rule 4 |

Severity has two levels:

- **error** — a fully mechanical rule. A fixed algorithm decides it with no judgment residue. The
  tool reports it as an error.
- **warning** — a rule with a judgment residue. The algorithm is reliable, but the style standard
  allows a human to keep a flagged form. The tool reports it for a human to read. It is never an
  error.

The tool also carries a set of **candidate warnings** (§2.5). Each candidate needs the implement
effort to prove its detector on the fixture corpus before the effort enables it.

### 2.1 No semicolons — error

Flag every semicolon in prose text. The style standard §4.1 rule 1 makes any semicolon a failure.

**Method.** Walk the prose text nodes. Match a punctuation node with the value `;`. Do not run a
regular expression over the raw markdown source, because that would flag a semicolon inside a code
span. #791 confirmed this is a clean deterministic tree-walk. No plugin is needed.

**Precision.** High. Non-prose never reaches the prose tree (see §3). A fenced code block drops
out. An inline code span becomes an opaque source node. An emoji parses as a symbol, not
punctuation, so it does not trigger the rule.

### 2.2 Sentence-length cap — error

Flag a sentence above the word cap. The style standard §4.1 rule 2 sets two caps: 20 words for an
instruction, 25 words for a description.

**The tool cannot tell an instruction from a description**, because that split needs section
context the tool does not have. So the tool uses one universal error line:

- **25 words** is the universal error. A sentence above 25 words fails.
- **20 words** is an optional warning. The implement effort may add a 20-word warning as a second,
  softer signal.

**Method.** The prose tree gives a sentence node directly. Count its word-node children. #791
confirmed this is deterministic and needs no plugin. Do not use `retext-readability` as a proxy,
because it runs readability formulas, not a word-count cap.

**Precision.** High for the word count. Hedge slightly on sentence boundaries. An abbreviation or a
decimal number can mis-split a sentence in the tokenizer.

### 2.3 No phrasal verbs — error

Flag a phrasal verb (a verb plus a particle, such as "spin up" or "kick off"). The style standard
§4.1 rule 5 makes a phrasal verb a failure.

**Method.** A curated wordlist. Each entry is a verb lemma plus a particle. #791 found no
`retext-phrasal` plugin and no reliable POS route, because the `pos` tagger does not tag the
particle class well. A curated list is the one reliable route. The retext-passive plugin uses the
same wordlist pattern.

**The team owns the wordlist.** The wordlist is a maintained asset in the repo. The implement
effort seeds it from the phrasal verbs the style standard names and the ASD-STE100 skill lists. The
effort grows it as new false negatives appear.

**Precision.** Moderate. A curated list catches its entries reliably. It misses a phrasal verb not
on the list. It may over-flag a literal use ("set up the ladder" against "set up the server"). The
fixture corpus (§6) sets the acceptance bar for the seeded list.

### 2.4 Simple tenses — warning

Flag a compound verb form built on `have`, `has`, or `had` plus a past participle (for example
"have received", "has completed"). The style standard §4.1 rule 4 flags a compound form, but it
allows the form to stay when it carries current relevance or a hedge. That keep-decision is a
judgment residue.

**This rule is a warning, not an error.** The judgment residue is the reason. The tool flags the
form. A human decides whether to keep it. The tool never fails on it.

**Method.** POS tagging. Flag `have`, `has`, or `had` directly before a past participle. #792 ran
this rule with the `pos` tagger over all 176 in-scope docs. The result:

- Precision was about 100%. The run produced 634 flags across about 180 distinct participle forms.
  The scored sample found zero false positives.
- The `have`/`has`/`had` anchor plus the participle position is reliable. The grammatical position
  after the anchor forces the participle tag, so the tagger does not confuse it.
- Volume is high on clean docs. Many flags are legitimate current-relevance uses (for example
  "nobody has measured", "has been GA since 2025"). A hard error would be wrong. A warning is
  right.

**Do not use the remark-retext bridge for this rule.** See the tokenizer caveat in §4.3.

### 2.5 Candidate warnings — prove before enable

The style standard §4.2 lists four review prompts. Ticket Q2 classified three of them as warnings:
passive voice, one-instruction-per-sentence, and no-ellipsis. (The fourth, lists-for-sequences, is
section-dependent, so Q3 drops it from v1. See §7.)

These three are **candidates**, not shipped rules. #791 and #792 tested the five §4.1 mechanical
rules, not the §4.2 prompts. So no measured precision figure exists for the three candidates. The
implement effort treats each candidate the same way #792 treated the POS rules:

1. Build the detector.
2. Prove its precision on the fixture corpus (§6).
3. Enable it as a warning only if precision is acceptable. Defer it otherwise.

Detector notes for the effort:

- **Passive voice.** The most tractable candidate. The retext-passive plugin detects a passive form
  with a curated wordlist (a form of "be" plus a participle list). This is a known route.
- **One-instruction-per-sentence.** Higher risk. No known detector. The style standard §4.2 says
  some second instructions need a human to see them.
- **No-ellipsis.** Higher risk. Detecting a dropped subject, verb, or article needs a parse the
  `pos` tagger does not support reliably.

---

## 3. What the tool reads

The tool lints prose text only. It skips every non-prose region. #791 and #792 both confirmed the
skip list.

The tool skips:

- Fenced code blocks.
- Inline code spans.
- Tables.
- Front-matter.
- Blockquotes (a blockquote holds frozen quoted source, per the style standard §6).

**Method.** The markdown AST separates prose from these regions. The remark parser drops fenced
code, inline code, tables, and front-matter from the prose text by default. A blockquote needs one
explicit `ignore` rule, because the default keeps it. #791 recorded this exact behaviour.

The reason to skip is correctness, not speed. A semicolon in a shell command is valid shell. A long
line in a code sample is valid code. A quoted source sentence is frozen. A rule that read these
regions would flag valid, frozen, or out-of-scope text.

---

## 4. Architecture

### 4.1 Stack

The tool is a Node script built on the unified ecosystem (ticket Q5):

- **remark** parses the markdown into an AST (mdast).
- The tool extracts prose text nodes from the AST and skips the §3 non-prose regions.
- A prose-tree walk applies the deterministic rules (semicolons, sentence-length, phrasal-verbs).
- The `pos` tagger applies the simple-tense rule.

### 4.2 Location

The tool lives in the `docs-site/` lane (ticket Q5). The script sits in `docs-site/scripts/`,
next to `check-links.mjs`. A `package.json` script wires it, modeled on `check:links` (ticket Q10).
`check-links.mjs` is the reference for the file layout, the CLI shape, and the `file:line -> reason`
output format.

### 4.3 The tokenizer caveat

**Do not assume the `remark-retext` bridge works out of the box.** #792 found that the retext prose
stack duplicates words at the current package versions. The tokenizer turns "a b c" into "a b b c c".
Both the `remark-retext` bridge and a bare `retext-english` pipeline show the bug. A duplicated
word breaks every word-count and POS rule.

The implement effort picks one mitigation:

1. Use the `pos` tagger with a direct lexer. This is the path #792's prototype took. Its precision
   numbers used this path, so they hold.
2. Pin a working `parse-latin` or `parse-english` version that does not duplicate.
3. Add a de-duplication guard after tokenizing.

The prototype at `docs-site/scripts/proto-doclint-pos/` on branch `proto/792-pos-precision` shows
option 1 working. It is a throwaway. Do not ship it. Read it as a reference for the direct-lexer
path.

---

## 5. Interfaces

### 5.1 The local command

A writer runs the tool by hand. The command lints one of two targets (ticket Q6):

- **Named files.** The writer passes one or more file paths. The tool lints those files.
- **The whole tree.** With no path, the tool lints every in-scope file (§1.3).

The command prints one line per violation, in the `check-links.mjs` style:

```
docs/spec/example.md:42  ->  simple-tense  (warning: "has completed" — compound form, confirm current relevance)
```

The command exits non-zero when it finds an error-level violation. A warning alone does not change
the exit code, because a warning is advisory.

### 5.2 The CI check

A new dedicated `doclint` CI job runs the tool (ticket Q10). The job:

- Runs on a broad doc path filter that covers all nine in-scope families.
- Is `continue-on-error` (ticket Q9). The job never fails the workflow. It never blocks a merge.
- Scopes to the changed docs on a pull request (ticket Q6). It lints the diff, not the whole tree.

**CI output** is GitHub Actions annotations plus a job-log summary (ticket Q9). An annotation marks
each violation inline on the pull request. The job-log summary lists the counts by rule and by
severity. **No SARIF in v1.**

The `doclint` job is separate from the existing `docs-site` build job. The build job's gates are
required and blocking. The `doclint` job is advisory and non-blocking, so it must not share the
build job's blocking behaviour.

---

## 6. Acceptance

A fixture corpus is the tool's acceptance gate (ticket Q11). For each rule, the corpus holds two
sets:

1. **Must-flag** snippets. Each holds a real violation the rule must catch.
2. **Must-not-flag** clean snippets. Each holds valid prose the rule must leave alone.

A rule passes acceptance when it flags every must-flag snippet and flags no must-not-flag snippet.
The candidate warnings (§2.5) use the same gate to prove precision before the effort enables them.

**The disable directive.** An inline `<!-- doclint-disable-line -->` comment silences the tool on
the next line (ticket Q11). It exists for an unavoidable false positive. A writer uses it to keep a
phrasal verb that is a proper noun, or a long sentence that cannot split, for example. The tool
reads the directive and skips the marked line.

---

## 7. Deferred and out of scope

### 7.1 Deferred out of v1

- **Noun-cluster cap.** The style standard §4.1 rule 3 flags a run of four or more stacked nouns.
  #792 ran this rule over all 176 docs. True positives were near zero on clean docs. The `pos`
  tagger mistags a verb ("stores", "issues") and a number ("four", "six") as a noun. The per-token
  error compounds across the four-token window. The signal-to-noise ratio is too low to ship, even
  as a warning. The rule stays a human-review gate only. Revisit it when a better tagger is
  available. #791 noted `en-pos` claims higher accuracy, but it is stale and needs custom wiring.
- **Lists-for-sequences.** The style standard §4.2 asks a writer to turn three or more prose steps
  into a list. This rule is section-dependent. It applies inside a procedural section. Ticket Q3
  drops every section-dependent rule from v1, because the tool cannot detect a procedural section.
- **Section-dependent checking in general.** v1 checks whole-doc rules only (ticket Q3).
- **Annotation-based section tagging.** Ticket Q3 option b would let a writer tag a procedural
  section with an annotation. This would unlock the ADHD subset and the section-dependent rules.
  It is deferred until v1 lands.
- **Broader ADHD-format checking** beyond the STE rules.

### 7.2 Out of scope

These sit past this SPEC's destination. They return only as a fresh effort.

- **Lexical-rule and keep-modality automation.** Both need a dictionary or a human reading. The
  ASD-STE100 skill ships no dictionary.
- **The tool as a blocking merge gate.** Ticket Q4 ruled the tool advisory.
- **The tool's implementation and build.** This SPEC hands off to the downstream `/to-tickets` and
  implement effort.

---

## 8. Handoff

This SPEC is the destination of map #790. It reaches the destination and hands off.

The next effort is `/to-tickets` on this SPEC, then implement. That effort builds the tool. It
starts from §2 (the rule set), §4 (the architecture and the tokenizer caveat), and §6 (the fixture
corpus as the acceptance gate). It proves each candidate warning (§2.5) on the corpus before it
enables the candidate.
