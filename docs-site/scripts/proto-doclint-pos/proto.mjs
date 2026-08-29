// THROWAWAY PROTOTYPE — wayfinder ticket #792.
//
// Question: do the two POS-dependent doc-lint rules reach acceptable precision
// on real verge-asm docs, using the `pos`/FastTag tagger (the tagger that
// retext-pos wraps)?
//   Rule C  noun-cluster cap  — flag a run of 4+ stacked nouns (Penn tag NN*).
//   Rule D  simple-tense      — flag have/has/had directly before a past
//                               participle (Penn tag VBN): present perfect and
//                               compound/auxiliary forms.
//
// It prints EVERY flag with its sentence and the per-word POS tags, so a human
// can score each flag true-positive vs false-positive by eye. It only measures
// whether the POS route is trustworthy enough to ship these two rules as hard
// errors (Q8). Deterministic rules (semicolons, sentence-length, phrasal verbs)
// are out of scope here.
//
// TOOLING NOTE (found while building this): the retext prose stack at the
// current versions (parse-latin 7 / parse-english 7 / nlcst bridge) DUPLICATES
// words on this environment ("a b c" tokenizes to a b b c c, "one, two, three"
// to one two two two three three three). Both the remark-retext bridge and a
// bare retext-english pipeline show it. So this prototype does NOT use retext.
// It extracts prose from the markdown AST, then calls the `pos` tagger directly
// with its own lexer. That is the same tagger retext-pos wraps, so it answers
// the precision question directly. The tooling duplication bug is a separate
// finding the SPEC must account for.
//
// Run:  npm install && node proto.mjs        (from this directory)

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'
import { unified } from 'unified'
import remarkParse from 'remark-parse'
import remarkFrontmatter from 'remark-frontmatter'
import remarkGfm from 'remark-gfm'
import { visitParents } from 'unist-util-visit-parents'
import pkg from 'pos'

const { Lexer, Tagger } = pkg
const REPO = resolve(process.cwd(), '..', '..', '..')

const FAMILIES = [
  { name: 'guides', glob: 'docs/guides' },
  { name: 'specs', glob: 'docs/spec' },
  { name: 'ADRs', glob: 'docs/adr' },
  { name: 'research', glob: 'docs/research' },
  { name: 'agents', glob: 'docs/agents' },
  { name: 'CONTEXT.md', file: 'CONTEXT.md' },
  { name: 'CLAUDE.md', file: 'CLAUDE.md' },
  { name: 'README.md', file: 'README.md' },
  { name: 'SECURITY.md', file: 'SECURITY.md' },
]

function mdFiles(dir) {
  const out = []
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry)
    if (statSync(p).isDirectory()) out.push(...mdFiles(p))
    else if (entry.endsWith('.md')) out.push(p)
  }
  return out
}
function filesForFamily(f) {
  if (f.file) return [join(REPO, f.file)]
  try {
    return mdFiles(join(REPO, f.glob))
  } catch {
    return []
  }
}

// ---- prose extraction ------------------------------------------------------
// Keep only text inside prose blocks. Skip code, inline code, tables,
// front-matter, blockquotes (frozen quoted source, per #791 + §3). Each block
// becomes one line so sentences do not run across headings/paragraphs.
const md = unified().use(remarkParse).use(remarkGfm).use(remarkFrontmatter, ['yaml', 'toml'])
const SKIP = new Set(['code', 'inlineCode', 'table', 'blockquote', 'html', 'yaml', 'toml'])

function extractBlocks(raw) {
  const tree = md.parse(raw)
  const blocks = []
  visitParents(tree, 'text', (node, ancestors) => {
    if (ancestors.some((a) => SKIP.has(a.type))) return
    const block = ancestors[ancestors.length - 1]
    const last = blocks[blocks.length - 1]
    if (last && last.block === block) last.parts.push(node.value)
    else blocks.push({ block, parts: [node.value] })
  })
  return blocks.map((b) => b.parts.join(' ').replace(/\s+/g, ' ').trim()).filter(Boolean)
}

// naive sentence split for readable context; the rules run per sentence
function sentences(block) {
  return block
    .split(/(?<=[.!?])\s+(?=[A-Z(])/)
    .map((s) => s.trim())
    .filter(Boolean)
}

// ---- rules -----------------------------------------------------------------
const lexer = new Lexer()
const tagger = new Tagger()
const NOUN = /^NN/
const HAVE = new Set(['have', 'has', 'had'])

function analyzeSentence(sentence, out) {
  const tagged = tagger.tag(lexer.lex(sentence)) // [[word, tag], ...]

  // Rule C — run of 4+ consecutive NN* tokens.
  let run = []
  const flush = () => {
    if (run.length >= 4) {
      out.nounClusters.push({
        cluster: run.map(([w]) => w).join(' '),
        tags: run.map(([w, t]) => `${w}/${t}`).join(' '),
        sentence,
      })
    }
    run = []
  }
  // A cluster extends only on a NN* token that is a real alphabetic word.
  // Punctuation the tagger mislabels NN (em-dash, pipe, slash) must BREAK the
  // run, not bridge it — "authors — one act" is not a noun cluster.
  const isWord = (w) => /[A-Za-z]/.test(w) && !/^[-|/]+$/.test(w)
  for (const tok of tagged) {
    if (NOUN.test(tok[1]) && isWord(tok[0])) run.push(tok)
    else flush()
  }
  flush()

  // Rule D — have/has/had immediately before a VBN participle.
  for (let i = 0; i < tagged.length - 1; i++) {
    const [aw] = tagged[i]
    const [bw, bt] = tagged[i + 1]
    if (HAVE.has(aw.toLowerCase()) && bt === 'VBN') {
      out.tenses.push({ hit: `${aw} ${bw}`, tags: `${aw}/${tagged[i][1]} ${bw}/${bt}`, sentence })
    }
  }
}

// ---- run -------------------------------------------------------------------
const totals = { nounClusters: 0, tenses: 0, files: 0 }
const report = []

for (const fam of FAMILIES) {
  const files = filesForFamily(fam)
  const famFlags = { nounClusters: [], tenses: [] }
  for (const file of files) {
    const rel = relative(REPO, file).replace(/\\/g, '/')
    const out = { nounClusters: [], tenses: [] }
    for (const block of extractBlocks(readFileSync(file, 'utf8'))) {
      for (const s of sentences(block)) analyzeSentence(s, out)
    }
    for (const f of out.nounClusters) famFlags.nounClusters.push({ ...f, file: rel })
    for (const f of out.tenses) famFlags.tenses.push({ ...f, file: rel })
    totals.files++
  }
  totals.nounClusters += famFlags.nounClusters.length
  totals.tenses += famFlags.tenses.length
  report.push({ fam: fam.name, files: files.length, ...famFlags })
}

// ---- output ----------------------------------------------------------------
const line = '─'.repeat(78)
console.log(line)
console.log('PROTO #792 — POS-rule precision on real verge-asm docs')
console.log(`tagger: pos/FastTag (the tagger retext-pos wraps) · files: ${totals.files}`)
console.log(line)

console.log(`\n### RULE C — noun-cluster cap (4+ stacked nouns): ${totals.nounClusters} flags\n`)
for (const r of report) {
  if (!r.nounClusters.length) continue
  console.log(`  ── ${r.fam} (${r.nounClusters.length}) ──`)
  for (const f of r.nounClusters) {
    console.log(`  [${f.file}]  "${f.cluster}"`)
    console.log(`    tags: ${f.tags}`)
    console.log(`    sent: ${f.sentence}`)
    console.log('')
  }
}

console.log(`\n### RULE D — simple-tense (have/has/had + VBN): ${totals.tenses} flags\n`)
for (const r of report) {
  if (!r.tenses.length) continue
  console.log(`  ── ${r.fam} (${r.tenses.length}) ──`)
  for (const f of r.tenses) {
    console.log(`  [${f.file}]  "${f.hit}"   (${f.tags})`)
    console.log(`    sent: ${f.sentence}`)
    console.log('')
  }
}

console.log(line)
console.log('per-family flag counts:')
for (const r of report) {
  console.log(
    `  ${r.fam.padEnd(12)} files=${String(r.files).padStart(3)}  nounCluster=${String(r.nounClusters.length).padStart(4)}  tense=${String(r.tenses.length).padStart(4)}`,
  )
}
console.log(line)
console.log(`TOTAL  nounCluster=${totals.nounClusters}  tense=${totals.tenses}`)
console.log(line)
