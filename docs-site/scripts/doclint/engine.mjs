/*
 * doclint engine — the walking skeleton (#817).
 *
 * One path through parse -> prose extraction -> rule -> violation, shared by every
 * rule the later tickets (#818-#820) add. The engine owns three things:
 *
 *   1. parse()        — markdown -> mdast, on the unified stack (SPEC §4.1).
 *   2. extractProse() — the mdast text nodes that hold prose, with the SPEC §3
 *                       non-prose regions removed.
 *   3. lintMarkdown() — run a rule set over the prose and collect violations.
 *
 * A rule never touches the raw markdown source. It reads the prose text nodes this
 * engine hands it (SPEC §2.1). That is why a `;` inside a code span or a table cell
 * can never reach a rule: extractProse() drops those regions before any rule runs.
 */
import { unified } from "unified";
import remarkParse from "remark-parse";
import remarkGfm from "remark-gfm";
import remarkFrontmatter from "remark-frontmatter";
import { visit, SKIP } from "unist-util-visit";
import { visitParents } from "unist-util-visit-parents";

/*
 * The parser stack. remark-frontmatter turns a YAML front-matter block into a `yaml`
 * node the prose walk skips (SPEC §3). remark-gfm turns a pipe table into a `table`
 * node the prose walk skips. Without these two plugins the front-matter and the table
 * cells would parse as ordinary prose.
 */
const processor = unified()
  .use(remarkParse)
  .use(remarkFrontmatter)
  .use(remarkGfm);

/**
 * Parse markdown to an mdast tree. Position data is on, so every text node carries
 * its source line.
 * @param {string} markdown
 * @returns {import("mdast").Root}
 */
export function parse(markdown) {
  return processor.parse(markdown);
}

/*
 * The SPEC §3 non-prose regions. The prose walk stops at each type and never descends
 * into it:
 *   code       — a fenced or indented code block.
 *   inlineCode — an inline code span.
 *   table      — a GFM table (skip the whole subtree, so no cell text leaks).
 *   yaml       — a front-matter block.
 *   blockquote — frozen quoted source (skip the whole subtree; the default keeps it).
 *
 * The list is exactly SPEC §3, no more. A raw-HTML node (`html`) needs no entry here:
 * extractProse() collects `text` nodes only, and an HTML tag, comment, or block is an
 * `html` node, never a `text` node, so it never reaches a rule.
 */
const NON_PROSE = new Set(["code", "inlineCode", "table", "yaml", "blockquote"]);

/**
 * The prose text nodes of a tree, with every SPEC §3 non-prose region removed.
 * @param {import("mdast").Root} tree
 * @returns {import("mdast").Text[]}
 */
export function extractProse(tree) {
  const nodes = [];
  visit(tree, (node) => {
    if (NON_PROSE.has(node.type)) return SKIP; // do not descend into a non-prose region
    if (node.type === "text") nodes.push(node);
  });
  return nodes;
}

/*
 * Inline (span-level) node types. A prose sentence can run straight through one of
 * these — "the `--flag` option is set" is one sentence, not three — so an inline node
 * must never break a prose block. extractProseBlocks() skips past every inline ancestor
 * to find the block that owns a text node. The set holds every mdast/GFM inline type
 * that can wrap prose text. `inlineCode` is here too, but it is also non-prose (its
 * text never reaches a block), so it only ever matters as a boundary a block spans.
 */
const INLINE = new Set([
  "emphasis",
  "strong",
  "delete",
  "link",
  "linkReference",
  "footnoteReference",
  "inlineCode",
]);

/**
 * The nearest block-level ancestor of a text node — the paragraph, heading, or list
 * item that owns it. Walk the ancestors from nearest to farthest and return the first
 * one that is not an inline span. Two text nodes with the same block ancestor belong to
 * the same sentence run, even when an inline span splits them.
 * @param {import("mdast").Parent[]} ancestors  root-first ancestor chain.
 * @returns {import("mdast").Parent}
 */
function nearestBlock(ancestors) {
  for (let i = ancestors.length - 1; i >= 0; i--) {
    if (!INLINE.has(ancestors[i].type)) return ancestors[i];
  }
  return ancestors[0];
}

/**
 * A prose block: the joined prose text of one block-level node, with its start line.
 * @typedef {Object} ProseBlock
 * @property {string} value      the block prose, inline-span fragments joined by a space.
 * @property {number} startLine  the 1-based source line the block's first fragment starts on.
 */

/**
 * The prose text grouped into block-level runs, with every SPEC §3 non-prose region
 * removed. A rule that reasons about a whole sentence (the sentence-length cap, #818)
 * needs the block, not the loose text nodes, because a sentence can span several text
 * nodes that an inline span (emphasis, a link, an inline code span) split apart.
 *
 * Each block joins its fragments with a single space, so a sentence that runs across an
 * inline code span stays one sentence and its words on both sides still count. The join
 * never inserts a newline, and a soft-wrapped paragraph is already one text node that
 * keeps its own newlines, so a within-block line offset still maps back to a source line.
 * @param {import("mdast").Root} tree
 * @returns {ProseBlock[]}
 */
export function extractProseBlocks(tree) {
  const blocks = [];
  visitParents(tree, "text", (node, ancestors) => {
    // Skip a text node inside any non-prose region (SPEC §3), the same regions
    // extractProse() drops. A blockquote or table cell never becomes a prose block.
    if (ancestors.some((a) => NON_PROSE.has(a.type))) return;
    const container = nearestBlock(ancestors);
    const last = blocks[blocks.length - 1];
    // visitParents walks in document order, so the fragments of one block arrive back
    // to back. Extend the open block while its container holds; otherwise open a new one.
    if (last && last.container === container) {
      last.parts.push(node.value);
    } else {
      blocks.push({ container, parts: [node.value], startLine: startLineOf(node) });
    }
  });
  return blocks.map((b) => ({ value: b.parts.join(" "), startLine: b.startLine }));
}

/**
 * The 1-based source line a text node starts on. A text node can span several lines,
 * so a rule that scans the node value tracks the line itself, one increment per `\n`,
 * starting from this value.
 * @param {import("mdast").Text} node
 * @returns {number}
 */
export function startLineOf(node) {
  return node.position?.start?.line ?? 1;
}

/**
 * The 1-based source line an offset inside a text node value falls on: the node start line
 * plus one for every newline before the offset. A word-level rule that scans a node value
 * for a match (no-phrasal-verbs) uses this to point at the match's own line, not the node's
 * first line, because one text node can span several soft-wrapped source lines.
 * @param {import("mdast").Text} node
 * @param {string} value  the node value the rule scanned (node.value).
 * @param {number} offset the match index inside value.
 * @returns {number}
 */
export function lineAtOffset(node, value, offset) {
  let line = startLineOf(node);
  for (let i = 0; i < offset; i++) if (value[i] === "\n") line++;
  return line;
}

/**
 * A rule.
 * @typedef {Object} Rule
 * @property {string} name        the rule id, printed in the output.
 * @property {"error"|"warning"} severity  error changes the exit code; warning never does.
 * @property {(proseNodes: import("mdast").Text[], ctx: RuleContext) => {line: number, message: string}[]} check
 * @property {{mustFlag: string[], mustNotFlag: string[]}} [fixtures]  the SPEC §6 corpus.
 */

/**
 * The second argument every rule receives. A word-level rule reads proseNodes (the first
 * argument). A sentence-level rule reads ctx.blocks, because a sentence can span several
 * text nodes an inline span split apart.
 * @typedef {Object} RuleContext
 * @property {ProseBlock[]} blocks  the block-level prose runs (extractProseBlocks).
 */

/**
 * A violation, before a file path is attached.
 * @typedef {Object} Violation
 * @property {string} rule
 * @property {"error"|"warning"} severity
 * @property {number} line
 * @property {string} message
 */

/**
 * Run a rule set over one markdown document.
 * @param {string} markdown
 * @param {Rule[]} rules
 * @returns {Violation[]} sorted by line, then by rule name.
 */
export function lintMarkdown(markdown, rules) {
  const tree = parse(markdown);
  const proseNodes = extractProse(tree);
  const ctx = { blocks: extractProseBlocks(tree) };
  const violations = [];
  // De-dup identical findings. The output is line-granular (no column), so two matches
  // of one rule on one line — even in separate text nodes split by an inline code span —
  // are one line the reader reads once, not two identical rows or two CI annotations.
  const seen = new Set();
  for (const rule of rules) {
    for (const hit of rule.check(proseNodes, ctx)) {
      const key = `${rule.name} ${hit.line} ${hit.message}`;
      if (seen.has(key)) continue;
      seen.add(key);
      violations.push({
        rule: rule.name,
        severity: rule.severity,
        line: hit.line,
        message: hit.message,
      });
    }
  }
  violations.sort((a, b) => a.line - b.line || a.rule.localeCompare(b.rule));
  return violations;
}

/**
 * Format one violation in the SPEC §5.1 style: `file:line  ->  rule  (severity: message)`.
 * The severity word inside the parentheses is how the severity reaches the reader,
 * because the line has no separate severity column.
 * @param {Violation & {file: string}} v
 * @returns {string}
 */
export function formatViolation(v) {
  return `${v.file}:${v.line}  ->  ${v.rule}  (${v.severity}: ${v.message})`;
}
