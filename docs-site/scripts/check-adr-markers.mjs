#!/usr/bin/env node
import { writeFileSync } from "node:fs";
import { dirname, join, posix, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { ADR_DIR, FENCE, headings } from "./check-adr-sections.mjs";
import { MARKER_LINE, incomingEdges, loadAdrs, plannedMarkers, validate } from "./check-adr-index.mjs";

const DEFAULT_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const H1_PREFIX = /^ADR-\d{4,}:\s+/;
const KIND_TEXT = {
  missing: "a relation without a marker",
  stray: "a marker without a relation",
  edited: "a hand-edited marker",
  other: "differs from the regeneration",
};

const pad = (n) => String(n).padStart(4, "0");

class AnchorError extends Error {}

function adrLink(a) {
  const title = a.front?.title ?? (a.h1 ? a.h1.title.replace(H1_PREFIX, "") : a.stem);
  const file = a.front ? `${pad(a.number)}-${a.front.slug}.md` : a.name;
  return `[ADR-${pad(a.number)}: ${title}](./${file})`;
}

export function renderMarker(m, adrs, target) {
  if (m.kind === "withdrawn") {
    const w = target.front.withdrawal;
    const to = w["moved-to"];
    let moved;
    if (to === "none" || to === null || to === undefined) moved = "Nothing replaces it.";
    else if (Number.isInteger(to)) moved = `Moved to ${adrLink(adrs.get(to))}.`;
    else moved = `Moved to [${to}](${posix.relative(ADR_DIR, to)}).`;
    const reason = String(w.reason).replace(/\.$/, "");
    return `> **Withdrawn** ${w.date}: ${reason}. ${moved} <!-- adr-marker withdrawn -->`;
  }
  const from = adrs.get(m.from);
  const tail = `by ${adrLink(from)}, ${from.front.date}. <!-- adr-marker ${m.kind} ${m.from} -->`;
  if (m.kind === "amends") return `> **Amended** ${tail}`;
  if (m.kind === "retires") return `> **Retired**, with no replacement, ${tail}`;
  if (m.kind === "supersedes") return `> **Superseded** ${tail}`;
  throw new Error(`no marker for ${m.kind}`);
}

// A fenced sentinel is a quoted example, so it is neither stripped nor counted (adr-governance §5)
function sentinelLines(lines) {
  const out = new Set();
  let fenced = false;
  lines.forEach((line, i) => {
    if (FENCE.test(line)) fenced = !fenced;
    else if (!fenced && MARKER_LINE.test(line)) out.add(i);
  });
  return out;
}

function stripped(lines) {
  const sentinels = sentinelLines(lines);
  const out = [];
  for (let i = 0; i < lines.length; i++) {
    if (!sentinels.has(i)) {
      out.push(lines[i]);
      continue;
    }
    if (lines[i + 1] === "") i++;
  }
  return out;
}

function anchors(a, lines, incoming) {
  const all = headings(lines.join("\n"));
  const h1 = all.find((h) => h.level === 1);
  const byLine = new Map();
  for (const m of plannedMarkers(a, incoming)) {
    const h = m.clause ? all.find((x) => x.number === m.clause) : h1;
    if (!h) throw new AnchorError(`${a.file}: no heading to anchor the ${m.kind} marker under`);
    const at = h.line - 1;
    byLine.set(at, [...(byLine.get(at) ?? []), m]);
  }
  return byLine;
}

export function regenerate(a, incoming, adrs) {
  const clean = stripped(a.body.split("\n"));
  const plan = anchors(a, clean, incoming);
  const out = [];
  for (let i = 0; i < clean.length; i++) {
    out.push(clean[i]);
    const ms = plan.get(i);
    if (!ms) continue;
    for (const m of ms) out.push("", renderMarker(m, adrs, a));
    if (i + 1 < clean.length && clean[i + 1] !== "") out.push("");
  }
  return a.frontRaw + out.join("\n");
}

export function classify(want, have) {
  const w = want !== null && MARKER_LINE.test(want);
  const h = have !== null && MARKER_LINE.test(have);
  if (w && have === null) return "missing";
  if (h && want === null) return "stray";
  if (w && h) return "edited";
  return "other";
}

// Every line number is the on-disk one, so an inserted line names the line it goes before.
function lcsOps(want, have) {
  const n = want.length;
  const m = have.length;
  const table = Array.from({ length: n + 1 }, () => new Int32Array(m + 1));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      table[i][j] = want[i] === have[j] ? table[i + 1][j + 1] + 1 : Math.max(table[i + 1][j], table[i][j + 1]);
    }
  }
  const ops = [];
  let i = 0;
  let j = 0;
  while (i < n || j < m) {
    if (i < n && j < m && want[i] === have[j]) {
      i++;
      j++;
    } else if (j < m && (i >= n || table[i][j + 1] >= table[i + 1][j])) {
      ops.push({ line: j + 1, want: null, have: have[j] });
      j++;
    } else {
      ops.push({ line: j + 1, want: want[i], have: null });
      i++;
    }
  }
  return ops;
}

export function hunks(wantText, haveText) {
  const ops = lcsOps(wantText.split("\n"), haveText.split("\n"));
  const out = [];
  for (const op of ops) {
    const prev = out[out.length - 1];
    if (prev && prev.want === null && op.have === null && prev.have !== "" && op.want !== "") {
      prev.want = op.want;
    } else if (prev && prev.have === null && op.want === null && prev.want !== "" && op.have !== "") {
      prev.have = op.have;
    } else out.push({ ...op });
  }
  const real = out.filter((h) => (h.want ?? "") !== "" || (h.have ?? "") !== "");
  return (real.length > 0 ? real : out).map((h) => ({ line: h.line, kind: classify(h.want, h.have), want: h.want, have: h.have }));
}

const shown = (line) => (line === null ? "<none>" : line === "" ? "<blank>" : line);

export function report(diff) {
  const lines = [];
  for (const h of diff.hunks) {
    lines.push(`${diff.file}:${h.line}: ${KIND_TEXT[h.kind]}`);
    lines.push(`  want: ${shown(h.want)}`);
    lines.push(`  have: ${shown(h.have)}`);
  }
  return lines;
}

export function run(repoRoot, { write }) {
  const adrs = loadAdrs(repoRoot);
  const problems = validate(adrs, repoRoot);
  if (problems.length > 0) return { code: 2, adrs, problems, diffs: [], wrote: [] };

  const incoming = incomingEdges(adrs);
  const stale = [];
  for (const a of adrs.values()) {
    let want;
    try {
      want = regenerate(a, incoming.get(a.number) ?? [], adrs);
    } catch (err) {
      if (!(err instanceof AnchorError)) throw err;
      problems.push({ file: a.file, rule: "markers", message: err.message });
      continue;
    }
    if (want !== a.raw) stale.push({ a, want });
  }
  if (problems.length > 0) return { code: 2, adrs, problems, diffs: [], wrote: [] };

  if (write) {
    for (const { a, want } of stale) writeFileSync(join(repoRoot, a.file), want);
    return { code: 0, adrs, problems, diffs: [], wrote: stale.map(({ a }) => a.file) };
  }
  const diffs = stale.map(({ a, want }) => ({ file: a.file, hunks: hunks(want, a.raw) }));
  return { code: diffs.length > 0 ? 1 : 0, adrs, problems, diffs, wrote: [] };
}

function main() {
  const argv = process.argv.slice(2);
  const write = argv.includes("--write");
  const r = run(DEFAULT_ROOT, { write });
  for (const p of r.problems) console.error(`check:adr-markers: ${p.message}`);
  for (const d of r.diffs) for (const line of report(d)) console.error(line);
  if (r.diffs.length > 0) console.error("check:adr-markers: run `npm run write:adr-markers`");
  for (const f of r.wrote) console.log(`check:adr-markers wrote ${f}`);
  if (r.code === 0) {
    const withFront = [...r.adrs.values()].filter((a) => a.front).length;
    console.log(`check:adr-markers OK — ${r.adrs.size} ADR(s), ${withFront} with front matter, ${r.wrote.length} written.`);
  }
  process.exit(r.code);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
