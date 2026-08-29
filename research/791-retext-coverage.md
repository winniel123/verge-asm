# Research 791 — retext/remark coverage for the 5 mechanical STE rules

Ticket: winniel123/verge-asm#791 (part of map #790). Question Q5 and Q7.

Method: primary sources only. The unified, remark, retext, and syntax-tree GitHub
orgs, their READMEs, package source, and the npm registry. No blog posts. Each
claim cites its source URL.

## One correction to the premise

The ticket text guessed that `retext-pos` wraps `en-pos`. That guess is wrong.
`retext-pos` depends on **`pos`** (dariusk/pos-js), a JavaScript port of
FastTag, from Eric Brill's rule set. `en-pos` (finnlp/en-pos) is a **separate,
unrelated** tagger. retext does not use it. Both matter to this survey, so this
document keeps them distinct.

- retext-pos README and `package.json`: depends on `pos`, adds
  `node.data.partOfSpeech` on Word nodes.
  https://github.com/retextjs/retext-pos
- retext-pos v5.0.0 registry entry (deps: `pos`, `unist-util-visit`,
  `nlcst-to-string`, `@types/nlcst`): https://registry.npmjs.org/retext-pos

## Verdict table

| # | Rule | Plugin that covers it | Precision to expect | What it needs |
|---|------|----------------------|---------------------|---------------|
| a | No semicolons | None | High | Plain tree-walk over PunctuationNode |
| b | Sentence-length cap | None needed | High (deterministic) | Count WordNode children of each SentenceNode |
| c | Noun-cluster cap (4+ nouns) | None | **Low to moderate — hedge** | POS tags from `retext-pos` |
| d | Simple tenses (flag compound forms) | None (retext-passive is the closest pattern) | **Moderate at best — hedge** | POS tags, or a curated participle wordlist |
| e | No phrasal verbs | None | **Low without a wordlist, moderate with one** | Curated phrasal-verb wordlist |

No plugin exists for `retext-phrasal`, `retext-sentence-length`, `retext-tense`,
or `retext-noun`. The agent confirmed their absence from the retextjs org and
from npm search. https://github.com/orgs/retextjs/repositories

## Per-rule detail

### (a) No semicolons — deterministic

No retext or remark plugin targets the semicolon. None is needed. In the nlcst
prose tree a semicolon is a **PunctuationNode** with value `;`. The check is a
plain walk: `visit(tree, 'PunctuationNode')` and match the value.

Precision is high. The false-positive risk stays low because non-prose never
reaches this tree. Fenced code drops out. Inline code becomes an opaque Source
node (see Q7 below). Emoji parse as SymbolNode, not PunctuationNode, so they do
not trigger the check. Match the PunctuationNode value, not a regex over raw
source, to hold this precision.

- nlcst node model (PunctuationNode, SymbolNode):
  https://github.com/syntax-tree/nlcst

### (b) Sentence-length cap — deterministic

No plugin is needed. nlcst gives a **SentenceNode** directly. Count its
**WordNode** children. `visit(tree, 'SentenceNode', ...)` then filter children
by type `WordNode`.

`retext-readability` exists but does **not** count sentence length. It runs 7
readability formulas (Dale-Chall, Automated Readability, Coleman-Liau, Flesch,
Gunning-Fog, SMOG, Spache) and flags a sentence when 4 of 7 judge it hard. Word
count is an internal input, not an exposed cap. Do not use it as a proxy for the
20-word and 25-word caps in the style standard. https://github.com/retextjs/retext-readability

Precision is high for a word-count cap. Hedge slightly on sentence boundaries.
Abbreviations and decimals can mis-split a sentence in the tokenizer.

- nlcst SentenceNode/WordNode: https://github.com/syntax-tree/nlcst

### (c) Noun-cluster cap — POS-dependent, hedge

No plugin exists. The route is POS tagging. Use `retext-pos` to attach
`data.partOfSpeech` to each WordNode. Then, inside each SentenceNode, find a run
of 4 or more WordNodes whose tag starts with `NN` (Penn Treebank: NN, NNS, NNP,
NNPS). The mechanism is `retext-pos` plus `unist-util-visit`.

Precision is low to moderate. Hedge hard. The `pos` tagger is a lightweight
rule-based Brill-lineage tagger, not a neural one. The `pos` README publishes
**no accuracy figure**. Per-token error compounds across a 4-token run. Gerunds
and adjectival nouns (for example "running configuration") mistag often. Treat
the output as advisory. Expect both false positives and misses.

- retext-pos, Penn Treebank tags on WordNode: https://github.com/retextjs/retext-pos
- `pos` tagger, no accuracy claim: https://github.com/dariusk/pos-js

### (d) Simple tenses — no tense plugin, hedge

No plugin targets tense. No `retext-tense` exists. The closest pattern is
`retext-passive`, but it covers passive voice only, and it does so with a
**curated wordlist** (auxiliaries plus a participle list in `list.js`), not POS
tagging. https://github.com/retextjs/retext-passive

Two routes exist:
1. POS. Flag `have`, `has`, or `had` directly before a past participle **VBN**.
2. A curated participle wordlist, keyed off `have`/`has`/`had`, in the
   retext-passive style.

Precision is moderate at best. Hedge. VBN against VBD (past participle against
simple past, both end in "-ed") is a classic Brill-tagger confusion, so the pure
POS route over-flags and under-flags. The curated wordlist route is more
predictable, but the team must own and maintain the participle list. No primary
source claims reliable tense classification from `pos`.

### (e) No phrasal verbs — wordlist route, hedge

No plugin exists. No `retext-phrasal` exists. `retext-simplify` overlaps only by
accident. It is a curated wordlist of complex words with simpler suggestions
(for example "utilize" to "use"), and it includes some multiword entries, but it
is not a phrasal-verb detector and does not claim that coverage.
https://github.com/retextjs/retext-simplify

A curated phrasal-verb list (verb lemma plus particle) is the one reliable
route. POS can gate it (a verb tag before an RP particle or IN), but `pos` does
not tag the RP particle class well, and the literal-versus-phrasal distinction
("set up the ladder" against "walk up the hill") needs semantics a Brill tagger
does not supply. Precision is low without a wordlist and moderate with one.
Expect false positives on literal verb-plus-preposition sequences.

## Q7 — Markdown-AST prose extraction (confirmed)

Yes. The remark-to-retext bridge isolates prose and drops code by default. The
chain is `unified().use(remarkParse).use(remarkRetext, ...)`. `remark-retext`
runs on `mdast-util-to-nlcst`. https://github.com/remarkjs/remark-retext

Node handling, verified against the `mdast-util-to-nlcst` README and its source
(`lib/index.js`):

- **Tables** (`table`, `tableRow`, `tableCell`): always ignored. In the default
  `ignore` array.
- **Inline code** (`inlineCode`): always treated as "source". It becomes an
  opaque nlcst Source node. It is not tokenized into words or punctuation, so it
  contributes no words and no stray semicolons.
- **Fenced code** (`code`), **HTML** (`html`), and **front-matter**
  (`yaml`/`toml`): no handler in the dispatch, so they drop silently and never
  reach retext.
- **Blockquote** (`blockquote`): it has children, so the walker recurses into it
  and analyzes its prose **by default**. To skip blockquotes, as Q7 requires,
  add `blockquote` to the `ignore` option. It is not skipped for free.
- **Front-matter caveat:** `yaml`/`toml` nodes appear only if you add
  `remark-frontmatter`. Either way, to-nlcst has no handler for them, so they do
  not leak into prose.

Net: for rules (a) and (b) you get clean prose with code, inline code, tables,
and front-matter already excluded. Blockquotes need one line of `ignore`
config. https://github.com/syntax-tree/mdast-util-to-nlcst

Tokenizer: the nlcst tree (Sentence, Word, Punctuation, Symbol, WhiteSpace)
comes from **parse-latin** (7.0.0), which `retext-english`/`retext-latin` wrap.
https://registry.npmjs.org/parse-latin

## POS feasibility for the prototype (#792)

- `retext-pos` attaches `node.data.partOfSpeech` to each WordNode. The tags are
  Penn Treebank (NN, NNS, NNP, VB, VBD, **VBN**, VBZ, VBP, **MD**, IN, DT, and
  more). It tags per sentence for a little context. Read the tag during a
  `unist-util-visit` walk over WordNodes.
- The tagger is `pos`/FastTag: lightweight, rule-based, Brill-lineage, not
  neural. Its README publishes no accuracy number. https://github.com/dariusk/pos-js
- The tag set is good enough to **attempt** rule (c) (runs of NN*) and rule (d)
  (`have`/`has`/`had` plus VBN). Precision stays limited. Hedge. VBN-against-VBD
  confusion and noun/gerund/adjective mistags are the exact failure modes of a
  Brill tagger, and error compounds across a multi-token pattern.
- `en-pos` (the other tagger, not used by retext) claims 96.43% smoothed and
  94.4% unsmoothed on the Penn Treebank test. That figure is token-level on
  clean corpora, not phrase-boundary accuracy on these docs, and it needs custom
  wiring. https://github.com/finnlp/en-pos

## Maintenance status (npm registry)

- `retext` core: v9.0.0, 2023-09-06. Stable, quiet. https://registry.npmjs.org/retext
- `retext-pos`: v5.0.0, 2023-09-07. https://registry.npmjs.org/retext-pos
- `pos` (the real tagger): v0.4.2, **2016-09-16 — stale**. https://registry.npmjs.org/pos
- `en-pos`: v1.0.16, **2017-04-09 — stale**. https://registry.npmjs.org/en-pos
- `parse-latin`: v7.0.0, current. https://registry.npmjs.org/parse-latin

## Recommendation for the SPEC (Q5, Q8) and the prototype (#792)

- Rules (a) and (b) are clean deterministic tree-walks on the code-free nlcst
  prose tree. Ship these in v1 with high confidence. This matches Q8, which
  already lists semicolons and sentence-length as deterministic.
- Rule (e) (phrasal verbs) needs a curated wordlist. Q8 already assumes one.
- Rules (c) and (d) have no plugin. They can lean on `retext-pos` POS tags, but
  the tagger is lightweight, unmaintained since 2016, and publishes no accuracy
  figure. Precision will carry noise. Q8 already gates both behind the #792
  prototype. This research supports that gate. Do not promote (c) or (d) to a
  hard error until the prototype proves precision on the fixture corpus (Q11).
- A curated participle wordlist is a fallback for rule (d) if the POS route
  proves too noisy in the prototype.
