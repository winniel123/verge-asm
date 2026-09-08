#!/usr/bin/env node
// PROTOTYPE for #1645. Throwaway. Never merge. Lives on prototype/adr-governance only.
//
// Reads YAML front matter from docs/adr/*.md, validates it against the schema #1641
// fixed, derives status from incoming relations, writes docs/adr/index.json, and
// writes one marker blockquote per amends/retires/supersedes relation under the
// affected heading, per #1644.
//
//   node scripts/prototype-adr-index.mjs --write   # write index.json and the markers
//   node scripts/prototype-adr-index.mjs --check   # regenerate both and diff, exit 1 on any difference
//   node scripts/prototype-adr-index.mjs           # --check, and print the index rows

import { existsSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import yaml from "js-yaml";
import { numberedSections } from "./check-adr-sections.mjs";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const ADR_DIR = "docs/adr";
const INDEX_FILE = `${ADR_DIR}/index.json`;
const LEGACY_MAX = 227;

const ADR_FILE = /^(\d{4,})-(.+)\.md$/;
const FENCE = /^\s{0,3}(?:```|~~~)/;
const HEADING = /^(#{1,6})\s+(?:(\d+(?:\.\d+)*)[.)]?\s+)?(\S.*?)\s*$/;
const FRONT = /^---\r?\n([\s\S]*?)\r?\n---\r?\n/;
const MARKER = /^> .*<!-- adr-marker (amends|retires|supersedes|withdrawn)(?: (\d+))? -->\s*$/;
const H1_PREFIX = /^ADR-\d{4,}:\s+/;

const REQUIRED = ["number", "title", "slug", "date", "status", "source", "proof"];
const STATUSES = new Set(["accepted", "withdrawn"]);
const SOURCES = new Set(["grilling", "fix", "sweep"]);
const CLAUSE_RULE = {
  amends: "required",
  retires: "required",
  supersedes: "forbidden",
  sibling: "forbidden",
  "rests-on": "optional",
  bounds: "optional",
};
const DERIVES = { amends: "amended", retires: "amended", supersedes: "superseded" };
const RANK = { accepted: 0, amended: 1, superseded: 2, withdrawn: 3 };
const H1_ORDER = { withdrawn: 0, supersedes: 1, amends: 2, retires: 2 };

const pad = (n) => String(n).padStart(4, "0");

// ---------------------------------------------------------------- load

function headings(lines) {
  const out = [];
  let fenced = false;
  lines.forEach((line, i) => {
    if (FENCE.test(line)) {
      fenced = !fenced;
      return;
    }
    if (fenced) return;
    const m = HEADING.exec(line);
    if (m) out.push({ line: i, level: m[1].length, number: m[2] ?? null, title: m[3] });
  });
  return out;
}

function load() {
  const adrs = new Map();
  for (const name of readdirSync(join(ROOT, ADR_DIR)).sort()) {
    const m = ADR_FILE.exec(name);
    if (!m) continue;
    const raw = readFileSync(join(ROOT, ADR_DIR, name), "utf8");
    const fm = FRONT.exec(raw);
    const a = {
      file: `${ADR_DIR}/${name}`,
      name,
      number: Number(m[1]),
      stem: m[2],
      raw,
      frontRaw: fm ? fm[0] : "",
      // The default YAML schema turns an unquoted 2026-09-07 into a Date, and CORE_SCHEMA has no timestamp type.
      front: fm ? yaml.load(fm[1], { schema: yaml.CORE_SCHEMA }) : null,
      body: fm ? raw.slice(fm[0].length) : raw,
    };
    a.lines = a.body.split("\n");
    a.headings = headings(a.lines);
    a.sections = numberedSections(a.body);
    a.h1 = a.headings.find((h) => h.level === 1) ?? null;
    adrs.set(a.number, a);
  }
  return adrs;
}

// ---------------------------------------------------------------- validate

function validate(adrs) {
  const problems = [];
  const fail = (a, msg) => problems.push(`${a.file}: ${msg}`);

  for (const a of adrs.values()) {
    const f = a.front;
    if (!f) continue;

    for (const k of REQUIRED) if (f[k] === undefined) fail(a, `front matter lacks \`${k}\``);
    if (f.number !== a.number) fail(a, `number ${f.number} != filename ${a.number}`);
    if (f.slug !== a.stem) fail(a, `slug \`${f.slug}\` != filename stem \`${a.stem}\``);
    const h1Title = a.h1 ? a.h1.title.replace(H1_PREFIX, "") : null;
    if (h1Title !== f.title) fail(a, `title != H1: \`${f.title}\` vs \`${h1Title}\``);
    if (!STATUSES.has(f.status)) fail(a, `status \`${f.status}\` is not accepted|withdrawn`);
    if (!SOURCES.has(f.source)) fail(a, `source \`${f.source}\` is not grilling|fix|sweep`);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(String(f.date))) fail(a, `date \`${f.date}\` is not YYYY-MM-DD`);

    const proofKeys = f.proof && typeof f.proof === "object" ? Object.keys(f.proof) : [];
    if (proofKeys.length !== 1 || !["test", "ticket", "none"].includes(proofKeys[0])) {
      fail(a, `proof must be exactly one of test|ticket|none, got ${JSON.stringify(f.proof)}`);
    } else if (proofKeys[0] === "test") {
      const [path, name] = String(f.proof.test).split("::");
      if (!existsSync(join(ROOT, path))) fail(a, `proof test path ${path} does not exist`);
      else if (!readFileSync(join(ROOT, path), "utf8").includes(name)) {
        fail(a, `proof test ${path} does not name ${name}`);
      }
    }

    if (a.number > LEGACY_MAX) {
      if (f.ticket === undefined) fail(a, `above ${LEGACY_MAX}, \`ticket\` is required`);
      else if (f.ticket !== a.number) fail(a, `above ${LEGACY_MAX}, number must equal ticket`);
      if (f.title.split(/\s+/).length > 16) fail(a, `title is over 16 words`);
      if (!a.h1 || !H1_PREFIX.test(a.h1.title)) fail(a, `H1 must carry the ADR-NNNN: prefix`);
      const first = a.headings.find((h) => h.level === 2);
      if (!first || first.title !== "Decision") fail(a, `first ## must be Decision`);
      const d = decisionBlock(a);
      if (d && d.words > 150) fail(a, `Decision block is ${d.words} words, cap is 150`);
    }

    if (f.status === "withdrawn") {
      const w = f.withdrawal ?? {};
      for (const k of ["date", "reason", "moved-to"]) {
        if (w[k] === undefined) fail(a, `withdrawn without withdrawal.${k}`);
      }
    }

    const seen = new Set();
    for (const r of f.relations ?? []) {
      const rule = CLAUSE_RULE[r.kind];
      if (!rule) {
        fail(a, `relation kind \`${r.kind}\` is not one of the six`);
        continue;
      }
      const target = adrs.get(r.adr);
      if (!target) {
        fail(a, `relation ${r.kind} targets ADR-${r.adr}, which is not on disk`);
        continue;
      }
      const key = `${r.kind}:${r.adr}:${r.clause ?? ""}`;
      if (seen.has(key)) fail(a, `duplicate relation ${key}`);
      seen.add(key);
      if (r.clause !== undefined) {
        if (typeof r.clause !== "string") fail(a, `clause ${r.clause} must be a string`);
        if (rule === "forbidden") fail(a, `${r.kind} forbids a clause`);
        else if (!target.sections.has(String(r.clause))) {
          fail(a, `${r.kind} ADR-${r.adr} §${r.clause}: target numbers ${[...target.sections].join(", ") || "nothing"}`);
        }
      } else if (rule === "required" && target.sections.size > 0) {
        fail(a, `${r.kind} ADR-${r.adr} needs a clause, the target numbers headings`);
      }
      // one direction: the same edge on both sides fails
      const back = (target.front?.relations ?? []).find((t) => t.kind === r.kind && t.adr === a.number);
      if (back) fail(a, `${r.kind} ADR-${r.adr} is also written on ADR-${r.adr}; an edge lives once`);
    }
  }
  return problems;
}

// ---------------------------------------------------------------- derive

function decisionBlock(a) {
  const i = a.headings.findIndex((h) => h.level === 2 && h.title === "Decision");
  if (i < 0) return null;
  const end = a.headings[i + 1]?.line ?? a.lines.length;
  const text = a.lines
    .slice(a.headings[i].line + 1, end)
    .filter((l) => !MARKER.test(l))
    .join("\n")
    .trim();
  return { words: text.split(/\s+/).filter(Boolean).length, text };
}

function incomingEdges(adrs) {
  const incoming = new Map();
  const push = (n, e) => incoming.set(n, [...(incoming.get(n) ?? []), e]);
  for (const a of adrs.values()) {
    for (const r of a.front?.relations ?? []) {
      push(r.adr, { kind: r.kind, from: a.number, clause: r.clause ?? null });
      // sibling is written once and shown on both sides
      if (r.kind === "sibling") push(a.number, { kind: "sibling", from: r.adr, clause: null });
    }
  }
  return incoming;
}

function derivedStatus(a, incoming) {
  if (a.front?.status === "withdrawn") return "withdrawn";
  let best = "accepted";
  for (const e of incoming) {
    const s = DERIVES[e.kind];
    if (s && RANK[s] > RANK[best]) best = s;
  }
  return best;
}

// ---------------------------------------------------------------- markers

function adrLink(a) {
  return `[ADR-${pad(a.number)}: ${a.front.title}](./${a.name})`;
}

function renderMarker(m, adrs) {
  if (m.kind === "withdrawn") {
    const w = m.withdrawal;
    let moved;
    if (w["moved-to"] === "none" || w["moved-to"] === null) moved = "Nothing replaces it.";
    else if (typeof w["moved-to"] === "number") moved = `Moved to ${adrLink(adrs.get(w["moved-to"]))}.`;
    else moved = `Moved to [${w["moved-to"]}](${relative(ADR_DIR, w["moved-to"])}).`;
    return `> **Withdrawn** ${w.date}: ${w.reason}. ${moved} <!-- adr-marker withdrawn -->`;
  }
  const from = adrs.get(m.from);
  const tail = `by ${adrLink(from)}, ${from.front.date}. <!-- adr-marker ${m.kind} ${m.from} -->`;
  if (m.kind === "amends") return `> **Amended** ${tail}`;
  if (m.kind === "retires") return `> **Retired**, with no replacement, ${tail}`;
  if (m.kind === "supersedes") return `> **Superseded** ${tail}`;
  throw new Error(`no marker for ${m.kind}`);
}

// Every marker the front matter implies for one target, grouped by the heading it sits under.
function plannedMarkers(a, incoming) {
  const byAnchor = new Map();
  const add = (line, m) => byAnchor.set(line, [...(byAnchor.get(line) ?? []), m]);
  if (a.front?.status === "withdrawn") add(a.h1.line, { kind: "withdrawn", withdrawal: a.front.withdrawal, from: 0 });
  for (const e of incoming) {
    if (!DERIVES[e.kind]) continue;
    const h = e.clause ? a.headings.find((x) => x.number === e.clause) : a.h1;
    add(h.line, e);
  }
  for (const [line, ms] of byAnchor) {
    const atH1 = line === a.h1.line;
    ms.sort((x, y) => (atH1 ? H1_ORDER[x.kind] - H1_ORDER[y.kind] : 0) || x.from - y.from);
  }
  return byAnchor;
}

// Strip every tool-written marker (and the blank line that follows it), then re-insert from the plan.
function regenerateBody(a, incoming, adrs) {
  const stripped = [];
  for (let i = 0; i < a.lines.length; i++) {
    if (MARKER.test(a.lines[i])) {
      if (a.lines[i + 1] === "") i++;
      continue;
    }
    stripped.push(a.lines[i]);
  }
  const clean = { ...a, lines: stripped, headings: headings(stripped) };
  const plan = plannedMarkers(clean, incoming);
  const out = [];
  for (let i = 0; i < stripped.length; i++) {
    out.push(stripped[i]);
    const ms = plan.get(i);
    if (!ms) continue;
    for (const m of ms) out.push("", renderMarker(m, adrs));
  }
  return out.join("\n");
}

// ---------------------------------------------------------------- index

function indexRows(adrs, incoming) {
  const rows = [];
  for (const a of adrs.values()) {
    if (!a.front) continue;
    const f = a.front;
    const inc = incoming.get(a.number) ?? [];
    const markers = [];
    const text = regenerateBody(a, inc, adrs).split("\n");
    text.forEach((l, i) => {
      const m = MARKER.exec(l);
      if (m) markers.push({ line: i + 1 + a.frontRaw.split("\n").length - 1, kind: m[1], from: m[2] ? Number(m[2]) : null });
    });
    rows.push({
      number: a.number,
      file: a.file,
      title: f.title,
      slug: f.slug,
      date: String(f.date),
      status: { set: f.status, derived: derivedStatus(a, inc) },
      source: f.source,
      proof: f.proof,
      ticket: f.ticket ?? null,
      map: f.map ?? null,
      pr: f.pr ?? null,
      withdrawal: f.withdrawal ?? null,
      relations: f.relations ?? [],
      incoming: inc,
      sections: [...a.sections],
      markers,
      decision: decisionBlock(a),
    });
  }
  return rows;
}

// ---------------------------------------------------------------- main

function main() {
  const argv = process.argv.slice(2);
  const write = argv.includes("--write");

  const adrs = load();
  const converted = [...adrs.values()].filter((a) => a.front).length;
  const problems = validate(adrs);
  if (problems.length > 0) {
    for (const p of problems) console.error(`front matter: ${p}`);
    process.exit(2);
  }
  const incoming = incomingEdges(adrs);

  const indexJson = JSON.stringify(
    { generated: "prototype-adr-index.mjs from YAML front matter in docs/adr/*.md", adrs: indexRows(adrs, incoming) },
    null,
    2,
  ) + "\n";

  const diffs = [];
  for (const a of adrs.values()) {
    const inc = incoming.get(a.number) ?? [];
    if (!a.front && inc.every((e) => !DERIVES[e.kind])) continue;
    const want = a.frontRaw + regenerateBody(a, inc, adrs);
    if (want === a.raw) continue;
    if (write) {
      writeFileSync(join(ROOT, a.file), want);
      console.log(`wrote markers: ${a.file}`);
    } else {
      const w = want.split("\n");
      const h = a.raw.split("\n");
      const at = w.findIndex((l, i) => l !== h[i]);
      diffs.push(`${a.file}:${at + 1}: marker set differs from the front matter\n  want: ${w[at]}\n  have: ${h[at] ?? "<eof>"}`);
    }
  }

  const indexPath = join(ROOT, INDEX_FILE);
  const have = existsSync(indexPath) ? readFileSync(indexPath, "utf8") : "";
  if (have !== indexJson) {
    if (write) {
      writeFileSync(indexPath, indexJson);
      console.log(`wrote ${INDEX_FILE}`);
    } else diffs.push(`${INDEX_FILE} is stale`);
  }

  if (!write && !argv.includes("--check")) console.log(indexJson);
  if (diffs.length > 0) {
    for (const d of diffs) console.error(d);
    process.exit(1);
  }
  console.log(`adr-index OK — ${adrs.size} ADR(s), ${converted} with front matter, ${problems.length} problem(s).`);
}

main();
