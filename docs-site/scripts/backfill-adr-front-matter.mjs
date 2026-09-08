#!/usr/bin/env node
// One-shot legacy backfill, deleted in the same PR (adr-governance §10, #1738)
import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { ADR_DIR, ADR_FILE, numberedSections } from "./check-adr-sections.mjs";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const HEADER = /^- \*\*([^*]+?):\*\*\s*(.*)$/;
const H1_PREFIX = /^ADR-(\d{4}):\s+/;
const DATE = /^\d{4}-\d{2}-\d{2}/;
const ADR_REF = /ADR-(\d{4})(?:\]\([^)]*\))?/g;
const CLAUSE_RUN = /^(?:\s*(?:,|and|\/)?\s*§(\d+(?:\.\d+)*))+/;
const PROOF = 'proof: {none: "predates the governance SPEC"}';

// The #1639 kind table. `keep` is a qualified key: the edge is written and the line stays.
// `inverse` writes the edge on the target. `sweep` decides `source`. `prose` writes nothing.
const K = (target, opts = {}) => ({ target, ...opts });
const KINDS = new Map([
  ["status", K("status")],
  ["date", K("date")],
  ["ticket", K("ticket")],
  ["tickets", K("ticket")],
  ["map", K("map")],
  ["parent map", K("map")],
  ["not a sub-issue of any map", K("nomap", { sweep: true })],
  ["pr that deleted the comment", K("pr", { sweep: true })],
  ["pr that deleted the comments", K("pr", { sweep: true })],
  ["sweep pr that deleted the comment", K("pr", { sweep: true })],
  ["sweep prs that deleted the comments", K("pr", { sweep: true })],
  ["sweep pr that compressed the comment", K("pr", { sweep: true })],
  ["sweep pr that rewrote the comment", K("pr", { sweep: true })],
  ["found by", K("prose", { sweep: true })],
  ["also found by", K("prose")],
  ["origin", K("prose")],
  ["relates to", K("prose")],
  ["decides, from three sides", K("prose")],
  ["decisions", K("prose")],
  ["dedup", K("prose")],
  ["re-files, design-first", K("prose")],
  ["recorded independently by four more sweeps", K("prose")],
  ["split out of", K("prose")],
  ["upstream", K("prose")],
  ["continues", K("prose")],
  ["rests on", K("rests-on")],
  ["supplies a premise for", K("rests-on", { inverse: true })],
  ["supplies the ground for", K("rests-on", { inverse: true })],
  ["generalises", K("rests-on")],
  ["inherits the runtime constraint of", K("rests-on")],
  ["preserves", K("rests-on")],
  ["withdrawal convention", K("rests-on")],
  ["depends on", K("rests-on")],
  ["implements", K("rests-on")],
  ["inherits the runtime of", K("rests-on")],
  ["instance of", K("rests-on")],
  ["confirms without amending", K("rests-on", { keep: true })],
  ["extends, and withdraws nothing in", K("rests-on", { keep: true })],
  ["keeps and extends, withdraws nothing", K("rests-on", { keep: true })],
  ["keeps, withdraws nothing", K("rests-on", { keep: true })],
  ["rests on, and applies", K("rests-on", { keep: true })],
  ["rests on, by analogy only", K("rests-on", { keep: true })],
  ["bounded by", K("bounds", { inverse: true })],
  ["constrained by", K("bounds", { inverse: true })],
  ["bounds", K("bounds")],
  ["constrains", K("bounds")],
  ["bounded by, and not an amendment to", K("bounds", { inverse: true, keep: true })],
  ["bound at the seam by", K("bounds", { inverse: true, keep: true })],
  ["bounded by (the gate below is unchanged)", K("bounds", { inverse: true, keep: true })],
  ["bounded by, and it bounds", K("bounds", { inverse: true, both: true, keep: true })],
  ["reaches, and is not satisfied by", K("bounds", { keep: true })],
  ["rules what three adrs assumed", K("bounds", { keep: true })],
  ["amends", K("amends")],
  ["discharges", K("amends")],
  ["narrows", K("amends")],
  ["withdraws a clause of", K("amends")],
  ["supersedes in part", K("amends")],
  ["refines", K("amends")],
  ["sharpens", K("amends")],
  ["completes", K("amends")],
  ["superseded in part by", K("amends", { inverse: true })],
  ["adds a site to, and does not close", K("amends", { keep: true })],
  ["amends in effect", K("amends", { keep: true })],
  ["amends one bullet of", K("amends", { keep: true })],
  ["amends/reverses", K("amends", { keep: true })],
  ["extends, and bounds one phrase in", K("amends", { keep: true })],
  ["reverses, narrowly", K("amends", { keep: true })],
  ["withdraws, in part", K("amends", { keep: true })],
  ["rests on, and bounds one clause in", K("amends", { keep: true, also: "rests-on" })],
  ["retires", K("retires")],
  ["sibling of", K("sibling")],
  ["adjacent", K("sibling")],
  ["read with", K("sibling")],
  ["follows", K("sibling")],
  ["sibling of, and not ruled by", K("sibling", { keep: true })],
  ["sibling, for the data seam", K("sibling", { keep: true })],
  ["not bound by", K("prose")],
  ["spec", K("prose")],
  ["findings", K("prose")],
  ["leaves untouched", K("prose")],
  ["measured by", K("prose")],
  ["spec content", K("prose")],
  ["also stated, uncited, at two other leaves", K("prose")],
  ["amended in place by", K("prose")],
  ["bounded away from", K("prose")],
  ["corrects", K("prose")],
  ["corrects, at the site that states it", K("prose")],
  ["not gap 1", K("prose")],
  ["research", K("prose")],
  ["supersedes", K("prose")],
  ["the enumeration itself", K("prose")],
  ["upholds", K("prose")],
  ["what is unsettled", K("prose")],
]);
const PROSE_PREFIXES = ["why not a section on", "why not an amendment to"];

// Hand-read facts the header lines do not state mechanically.
const HAND = {
  // ADR-0006 has no header block; its date is the file's first commit (15aeac9)
  6: { date: "2026-08-13" },
  // "Ticket: #349 (map: #348)" names the map inside the Ticket line
  115: { ticket: [349], map: 348 },
  // ADR-0109 and ADR-0116 record their supersession in a Status line (adr-governance §10)
  145: { relations: [{ kind: "supersedes", adr: 109 }, { kind: "supersedes", adr: 116 }] },
};
// An amends line that names its clause in prose, not beside the link.
const CLAUSE_HAND = new Map([["122:amends:118", "1"]]);

const pad = (n) => String(n).padStart(4, "0");
const report = { dropped: [], skippedRefs: [], noClause: [], kept: [], mutual: [], notes: [] };

function kindOf(key) {
  const k = key.trim().toLowerCase();
  if (KINDS.has(k)) return KINDS.get(k);
  if (PROSE_PREFIXES.some((p) => k.startsWith(p))) return K("prose");
  throw new Error(`unknown header key: ${key}`);
}

function targetsOf(value) {
  const out = [];
  const seen = new Set();
  ADR_REF.lastIndex = 0;
  let m;
  while ((m = ADR_REF.exec(value)) !== null) {
    const adr = Number(m[1]);
    const before = value.slice(Math.max(0, m.index - 12), m.index).replace(/[\s[]+$/, "");
    // "per ADR-0058" and "under ADR-0058" name the withdrawal convention, not a target
    if (/\b(per|under|see)$/i.test(before)) {
      report.skippedRefs.push({ adr, before: value.slice(Math.max(0, m.index - 30), m.index + 8) });
      continue;
    }
    if (seen.has(adr)) continue;
    seen.add(adr);
    const rest = value.slice(m.index + m[0].length);
    const run = CLAUSE_RUN.exec(rest);
    const clauses = run ? [...run[0].matchAll(/§(\d+(?:\.\d+)*)/g)].map((x) => x[1]) : [];
    out.push({ adr, clauses });
  }
  return out;
}

const hasNonAdrLink = (value) => /\]\((?!\.\/\d{4}-)[^)]*\)/.test(value) || /(^|[^/\w])#\d+/.test(value.replace(/\[ADR-\d{4}[^\]]*\]\([^)]*\)/g, ""));
const issuesOf = (value) => {
  const linked = [...value.matchAll(/issues\/(\d+)/g)].map((x) => Number(x[1]));
  if (linked.length > 0) return [...new Set(linked)];
  return [...new Set([...value.matchAll(/#(\d+)/g)].map((x) => Number(x[1])))];
};
const pullsOf = (value) => [...new Set([...value.matchAll(/pull\/(\d+)/g)].map((x) => Number(x[1])))];

function parse(name) {
  const raw = readFileSync(join(ROOT, ADR_DIR, name), "utf8");
  if (raw.startsWith("---")) throw new Error(`${name} already has front matter`);
  const lines = raw.split("\n");
  if (!lines[0].startsWith("# ")) throw new Error(`${name} does not open with an H1`);
  let end = 1;
  while (end < lines.length && !lines[end].startsWith("## ")) end++;
  const items = [];
  let i = 1;
  while (i < end) {
    if (!lines[i].startsWith("- ")) {
      items.push({ start: i, stop: i + 1, header: null });
      i++;
      continue;
    }
    let j = i + 1;
    while (j < end && /^\s+\S/.test(lines[j])) j++;
    const m = HEADER.exec(lines[i]);
    const value = m ? [m[2], ...lines.slice(i + 1, j).map((l) => l.trim())].join(" ") : null;
    items.push({ start: i, stop: j, header: m ? { key: m[1].trim(), value } : null });
    i = j;
  }
  const number = Number(ADR_FILE.exec(name)[1]);
  return { name, number, raw, lines, end, items, sections: numberedSections(raw), h1: lines[0].slice(2).trim() };
}

function main() {
  const dry = process.argv.includes("--dry");
  const names = readdirSync(join(ROOT, ADR_DIR))
    .filter((n) => ADR_FILE.test(n))
    .sort();
  const adrs = new Map(names.map((n) => {
    const a = parse(n);
    return [a.number, a];
  }));

  // First pass: fields and forward edges per ADR, plus inverse edges queued on the target.
  const edges = new Map([...adrs.keys()].map((n) => [n, []]));
  const fields = new Map();
  const consumed = new Map();
  // Returns false only when the line's edge has no front matter form, so the line must stay.
  const addEdge = (from, kind, adr, clause, why, inverse = false) => {
    if (adr === from) {
      report.dropped.push(`${pad(from)}: ${kind} targets itself (${why})`);
      return true;
    }
    const target = adrs.get(adr);
    if (!target) {
      report.dropped.push(`${pad(from)}: ${kind} ${pad(adr)} is not on disk (${why})`);
      return true;
    }
    const list = edges.get(from);
    if (inverse && list.some((e) => e.kind === kind && e.adr === adr)) return true;
    let c = clause ?? null;
    if (kind === "supersedes" || kind === "sibling") c = null;
    if (c !== null && !target.sections.has(c)) {
      report.notes.push(`${pad(from)}: ${kind} ${pad(adr)} §${c} does not resolve (target numbers ${[...target.sections].join(", ") || "nothing"}), clause dropped`);
      c = null;
    }
    if ((kind === "amends" || kind === "retires") && c === null && target.sections.size > 0) {
      c = CLAUSE_HAND.get(`${from}:${kind}:${adr}`) ?? null;
      if (c === null) {
        report.noClause.push(`${pad(from)}: ${kind} ${pad(adr)} names no clause and the target numbers ${[...target.sections].join(", ")}, so the line stays as prose (${why})`);
        return false;
      }
    }
    if (!list.some((e) => e.kind === kind && e.adr === adr && e.clause === c)) list.push({ kind, adr, clause: c });
    return true;
  };

  for (const a of adrs.values()) {
    const f = { ticket: [], map: null, pr: null, date: null, sweep: [], hasMap: false };
    const gone = new Set();
    for (const it of a.items) {
      if (!it.header) continue;
      const { key, value } = it.header;
      const kind = kindOf(key);
      const label = `${pad(a.number)} "${key}"`;
      if (kind.sweep) f.sweep.push(key);
      if (kind.target === "status" || kind.target === "date" || kind.target === "nomap") {
        if (kind.target === "date") f.date = DATE.exec(value)?.[0] ?? null;
        if (kind.target === "status" && value !== "Accepted") report.notes.push(`${label} Status text dropped: ${value}`);
        gone.add(it);
        continue;
      }
      if (kind.target === "ticket") {
        f.ticket.push(...issuesOf(value));
        gone.add(it);
        continue;
      }
      if (kind.target === "map") {
        const ids = issuesOf(value);
        if (ids.length !== 1) throw new Error(`${label} names ${ids.length} issues`);
        f.map = ids[0];
        f.hasMap = true;
        gone.add(it);
        continue;
      }
      if (kind.target === "pr") {
        const ids = pullsOf(value);
        if (ids.length === 1 && f.pr === null) {
          f.pr = ids[0];
          gone.add(it);
        } else report.kept.push(`${label} kept: ${ids.length} PRs on the line`);
        continue;
      }
      if (kind.target === "prose") continue;

      const targets = targetsOf(value);
      if (targets.length === 0) {
        report.kept.push(`${label} kept: no ADR target`);
        continue;
      }
      let expressed = true;
      for (const t of targets) {
        if (kind.inverse) {
          if (t.clauses.length > 0) report.notes.push(`${label} §${t.clauses.join(", §")} on an inverse edge dropped`);
          expressed = addEdge(t.adr, kind.target, a.number, null, `inverse of ${label}`, true) && expressed;
          if (kind.both) expressed = addEdge(a.number, kind.target, t.adr, null, label) && expressed;
        } else {
          const clauses = t.clauses.length > 0 ? t.clauses : [null];
          for (const c of clauses) expressed = addEdge(a.number, kind.target, t.adr, c, label) && expressed;
          if (kind.also) expressed = addEdge(a.number, kind.also, t.adr, null, label) && expressed;
        }
      }
      if (kind.keep) report.kept.push(`${label} kept: qualified key`);
      else if (!expressed) report.kept.push(`${label} kept: an edge has no clause`);
      else if (hasNonAdrLink(value)) report.kept.push(`${label} kept: a target is not an ADR`);
      else gone.add(it);
    }
    fields.set(a.number, f);
    consumed.set(a.number, gone);
  }

  for (const [n, hand] of Object.entries(HAND)) {
    const f = fields.get(Number(n));
    if (hand.date) f.date = hand.date;
    if (hand.ticket) f.ticket = hand.ticket;
    if (hand.map) f.map = hand.map;
    for (const r of hand.relations ?? []) addEdge(Number(n), r.kind, r.adr, r.clause ?? null, "hand");
  }

  // An edge lives once: a mutual same-kind pair keeps the lower-numbered acting ADR's edge.
  for (const [from, list] of edges) {
    for (const e of [...list]) {
      const back = edges.get(e.adr).find((x) => x.kind === e.kind && x.adr === from);
      if (back && from > e.adr) {
        list.splice(list.indexOf(e), 1);
        report.mutual.push(`${pad(from)} ${e.kind} ${pad(e.adr)} dropped: ${pad(e.adr)} already writes it`);
      }
    }
  }

  const sourceRows = [];
  let written = 0;
  for (const a of adrs.values()) {
    const f = fields.get(a.number);
    if (!f.date) throw new Error(`${pad(a.number)} has no date`);
    const source = f.sweep.length > 0 ? "sweep" : f.hasMap ? "grilling" : "fix";
    const because = f.sweep.length > 0 ? f.sweep.join("; ") : f.hasMap ? "Map line" : "no Map line, no sweep kind";
    sourceRows.push(`| ADR-${pad(a.number)} | ${source} | ${because} |`);

    const title = a.h1.replace(H1_PREFIX, "");
    const slug = a.name.slice(5, -3);
    const fm = ["---", `number: ${a.number}`, `title: ${JSON.stringify(title)}`, `slug: ${/^[a-z0-9-]+$/.test(slug) ? slug : JSON.stringify(slug)}`, `date: ${f.date}`, "status: accepted", `source: ${source}`];
    const ticket = [...new Set(f.ticket)];
    if (ticket.length === 1) fm.push(`ticket: ${ticket[0]}`);
    else if (ticket.length > 1) fm.push(`ticket: [${ticket.join(", ")}]`);
    if (f.map !== null) fm.push(`map: ${f.map}`);
    if (f.pr !== null) fm.push(`pr: ${f.pr}`);
    fm.push(PROOF);
    const rel = edges.get(a.number);
    if (rel.length > 0) {
      fm.push("relations:");
      for (const r of rel) fm.push(`  - {kind: ${r.kind}, adr: ${r.adr}${r.clause !== null ? `, clause: ${JSON.stringify(r.clause)}` : ""}}`);
    }
    fm.push("---", "");

    const gone = consumed.get(a.number);
    const region = [];
    for (const it of a.items) if (!gone.has(it)) region.push(...a.lines.slice(it.start, it.stop));
    const body = [];
    for (const l of region) if (!(l.trim() === "" && (body.length === 0 || body[body.length - 1].trim() === ""))) body.push(l);
    while (body.length > 0 && body[body.length - 1].trim() === "") body.pop();
    const rest = a.lines.slice(a.end);
    const out = [...fm, a.lines[0], "", ...body, ...(body.length > 0 && rest.length > 0 ? [""] : []), ...rest].join("\n");
    if (!dry) writeFileSync(join(ROOT, ADR_DIR, a.name), out);
    written++;
  }

  const counts = {};
  for (const r of sourceRows) counts[r.split("|")[2].trim()] = (counts[r.split("|")[2].trim()] ?? 0) + 1;
  console.log("## Source table\n\n| ADR | source | because |\n|---|---|---|");
  console.log(sourceRows.join("\n"));
  console.log(`\nCounts: ${JSON.stringify(counts)}\n`);
  const edgeCount = [...edges.values()].reduce((n, l) => n + l.length, 0);
  const byKind = {};
  for (const l of edges.values()) for (const e of l) byKind[e.kind] = (byKind[e.kind] ?? 0) + 1;
  console.log(`## Edges: ${edgeCount} ${JSON.stringify(byKind)}\n`);
  for (const [k, v] of Object.entries(report)) {
    console.log(`## ${k} (${v.length})`);
    for (const line of v) console.log(`- ${typeof line === "string" ? line : JSON.stringify(line)}`);
    console.log("");
  }
  console.log(`${dry ? "would write" : "wrote"} ${written} files`);
}

main();
