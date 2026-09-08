#!/usr/bin/env node
import { existsSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import yaml from "js-yaml";
import { ADR_DIR, ADR_FILE, LEGACY_MAX, headings, numberedSections } from "./check-adr-sections.mjs";

const DEFAULT_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
export const INDEX_FILE = `${ADR_DIR}/index.json`;

const FRONT = /^---\r?\n([\s\S]*?)\r?\n---\r?\n/;
const H1_PREFIX = /^ADR-(\d{4,}):\s+/;
const DATE = /^\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01])$/;
const CLAUSE = /^\d+(?:\.\d+)*$/;
export const MARKER_LINE = /^> .*<!-- adr-marker (amends|retires|supersedes|withdrawn)(?: (\d+))? -->\s*$/;

const REQUIRED = ["number", "title", "slug", "date", "status", "source", "proof"];
const STATUSES = new Set(["accepted", "withdrawn"]);
const SOURCES = new Set(["grilling", "fix", "sweep"]);
const PROOFS = ["test", "ticket", "none"];
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
const MARKER_ORDER = { withdrawn: 0, supersedes: 1, amends: 2, retires: 2 };
const TITLE_CAP = 16;
const DECISION_CAP = 150;

const pad = (n) => `ADR-${String(n).padStart(4, "0")}`;
const isInt = (v) => Number.isInteger(v);
const words = (s) => s.split(/\s+/).filter(Boolean).length;

export function splitFrontMatter(raw) {
  const m = FRONT.exec(raw);
  if (!m) return { front: null, frontRaw: "", frontLines: 0, body: raw, error: null };
  let front = null;
  let error = null;
  try {
    // The default schema turns an unquoted 2026-09-07 into a Date (adr-governance §3)
    front = yaml.load(m[1], { schema: yaml.CORE_SCHEMA });
  } catch (err) {
    error = err.reason ?? err.message;
  }
  return {
    front,
    frontRaw: m[0],
    frontLines: m[0].split("\n").length - 1,
    body: raw.slice(m[0].length),
    error,
  };
}

export function loadAdrs(repoRoot) {
  const adrs = new Map();
  let names;
  try {
    names = readdirSync(join(repoRoot, ADR_DIR));
  } catch {
    return adrs;
  }
  for (const name of names.sort()) {
    const m = ADR_FILE.exec(name);
    if (!m) continue;
    const raw = readFileSync(join(repoRoot, ADR_DIR, name), "utf8");
    const split = splitFrontMatter(raw);
    const body = split.body;
    const all = headings(body);
    adrs.set(Number(m[1]), {
      file: `${ADR_DIR}/${name}`,
      name,
      number: Number(m[1]),
      stem: name.slice(m[1].length + 1, -3),
      raw,
      ...split,
      headings: all,
      h1: all.find((h) => h.level === 1) ?? null,
      sections: numberedSections(body),
    });
  }
  return adrs;
}

export function decisionBlock(body) {
  const lines = body.split(/\r?\n/);
  const all = headings(body);
  const i = all.findIndex((h) => h.level === 2 && h.title === "Decision");
  let text;
  if (i >= 0) {
    const end = i + 1 < all.length ? all[i + 1].line - 1 : lines.length;
    text = lines.slice(all[i].line, end);
  } else {
    // ADR-0006 has no Decision heading, so its first paragraph stands in (#1641 §5)
    const h1 = all.find((h) => h.level === 1);
    let at = h1 ? h1.line : 0;
    while (at < lines.length && (lines[at].trim() === "" || MARKER_LINE.test(lines[at]))) at++;
    const start = at;
    while (at < lines.length && lines[at].trim() !== "") at++;
    text = lines.slice(start, at);
  }
  const joined = text
    .filter((l) => !MARKER_LINE.test(l))
    .join("\n")
    .trim();
  return { words: words(joined), text: joined };
}

function checkProof(a, f, repoRoot, fail) {
  const keys = f.proof && typeof f.proof === "object" && !Array.isArray(f.proof) ? Object.keys(f.proof) : [];
  if (keys.length !== 1 || !PROOFS.includes(keys[0])) {
    fail(a, "proof", `proof must be exactly one of test|ticket|none, got ${JSON.stringify(f.proof)}`);
    return;
  }
  const kind = keys[0];
  const value = f.proof[kind];
  if (kind === "none") {
    if (typeof value !== "string" || value.trim() === "") fail(a, "proof", "a none proof needs a reason string");
    return;
  }
  if (kind === "ticket") {
    if (!isInt(value)) fail(a, "proof", `ticket proof must be an integer, got ${JSON.stringify(value)}`);
    return;
  }
  const at = typeof value === "string" ? value.indexOf("::") : -1;
  const path = at > 0 ? value.slice(0, at) : "";
  const name = at > 0 ? value.slice(at + 2) : "";
  if (path === "" || name === "") {
    fail(a, "proof", `proof test must be <path>::<name>, got ${JSON.stringify(value)}`);
    return;
  }
  if (!path.endsWith(".go") && !path.endsWith(".mjs")) {
    fail(a, "proof", `proof test ${path} must be a .go or .mjs path`);
    return;
  }
  if (!existsSync(join(repoRoot, path))) {
    fail(a, "proof", `proof test path ${path} does not exist`);
    return;
  }
  const text = readFileSync(join(repoRoot, path), "utf8");
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  if (path.endsWith(".go")) {
    if (!name.startsWith("Test")) fail(a, "proof", `proof test ${name} is not a Test function`);
    else if (!new RegExp(`^func ${escaped}\\(`, "m").test(text)) {
      fail(a, "proof", `proof test ${path} does not declare func ${name}`);
    }
  } else if (!new RegExp(`\\btest\\(\\s*(["'\`])${escaped}\\1`).test(text)) {
    fail(a, "proof", `proof test ${path} holds no test("${name}")`);
  }
}

function checkRelations(a, f, adrs, fail) {
  if (f.relations === undefined) return;
  if (!Array.isArray(f.relations) || f.relations.some((r) => !r || typeof r !== "object")) {
    fail(a, "relations", "relations must be a list of {kind, adr, clause} rows");
    return;
  }
  const seen = new Set();
  for (const r of f.relations) {
    const rule = CLAUSE_RULE[r.kind];
    if (!rule) {
      fail(a, "relations", `relation kind \`${r.kind}\` is not one of the six`);
      continue;
    }
    if (!isInt(r.adr)) {
      fail(a, "relations", `relation ${r.kind}: adr must be an integer naming an ADR file, got ${JSON.stringify(r.adr)}`);
      continue;
    }
    if (r.adr === a.number) {
      fail(a, "relations", `relation ${r.kind} targets itself`);
      continue;
    }
    const target = adrs.get(r.adr);
    if (!target) {
      fail(a, "relations", `relation ${r.kind} targets ${pad(r.adr)}, which is not on disk`);
      continue;
    }
    const key = `${r.kind}:${r.adr}:${r.clause ?? ""}`;
    if (seen.has(key)) fail(a, "relations", `duplicate relation ${key}`);
    seen.add(key);
    const label = `${r.kind} ${pad(r.adr)}`;
    if (r.clause !== undefined && r.clause !== null) {
      if (typeof r.clause !== "string") fail(a, "relations", `${label}: clause ${r.clause} must be a string`);
      else if (!CLAUSE.test(r.clause)) {
        fail(a, "relations", `${label}: clause \`${r.clause}\` is not a dotted heading number`);
      } else if (rule === "forbidden") fail(a, "relations", `${r.kind} forbids a clause`);
      else if (target.sections.size === 0) {
        fail(a, "relations", `${label} §${r.clause}: ${pad(r.adr)} numbers no heading, so the relation is whole-ADR`);
      } else if (!target.sections.has(r.clause)) {
        fail(a, "relations", `${label} §${r.clause}: target numbers ${[...target.sections].join(", ")}`);
      }
    } else if (rule === "required" && target.sections.size > 0) {
      fail(a, "relations", `${label} needs a clause, the target numbers headings`);
    }
    const back = (Array.isArray(target.front?.relations) ? target.front.relations : []).some(
      (t) => t && t.kind === r.kind && t.adr === a.number,
    );
    if (back) fail(a, "relations", `${label} is also written on ${pad(r.adr)}; an edge lives once`);
  }
}

function checkWithdrawal(a, f, adrs, repoRoot, fail) {
  const w = f.withdrawal && typeof f.withdrawal === "object" ? f.withdrawal : {};
  for (const k of ["date", "reason", "moved-to"]) {
    if (w[k] === undefined || w[k] === null) fail(a, "schema", `withdrawn without withdrawal.${k}`);
  }
  if (w.date !== undefined && !DATE.test(String(w.date))) fail(a, "schema", `withdrawal.date \`${w.date}\` is not YYYY-MM-DD`);
  const to = w["moved-to"];
  if (to === undefined || to === null || to === "none") return;
  if (isInt(to)) {
    if (!adrs.has(to)) fail(a, "schema", `moved-to ${pad(to)} is not on disk`);
  } else if (typeof to !== "string") fail(a, "schema", `moved-to must be a path, an ADR number, or none`);
  else if (!existsSync(join(repoRoot, to))) fail(a, "schema", `moved-to ${to} does not exist`);
}

export function validate(adrs, repoRoot) {
  const problems = [];
  const fail = (a, rule, message) => problems.push({ file: a.file, rule, message: `${a.file}: ${message}` });

  for (const a of adrs.values()) {
    if (a.error) {
      fail(a, "schema", `front matter is not valid YAML: ${a.error}`);
      continue;
    }
    const f = a.front;
    if (f === null) continue;
    if (typeof f !== "object" || Array.isArray(f)) {
      fail(a, "schema", "front matter must be a YAML mapping");
      continue;
    }
    if (!a.body.startsWith("\n# ")) fail(a, "schema", "front matter must be followed by one blank line, then the H1");

    for (const k of REQUIRED) if (f[k] === undefined || f[k] === null) fail(a, "schema", `front matter lacks \`${k}\``);
    if (f.number !== undefined && f.number !== a.number) fail(a, "schema", `number ${f.number} != filename ${a.number}`);
    if (f.slug !== undefined && f.slug !== a.stem) fail(a, "schema", `slug \`${f.slug}\` != filename stem \`${a.stem}\``);

    const prefix = a.h1 ? H1_PREFIX.exec(a.h1.title) : null;
    if (prefix && Number(prefix[1]) !== a.number) fail(a, "schema", `H1 prefix ${pad(Number(prefix[1]))} names another number`);
    const h1Title = a.h1 ? a.h1.title.replace(H1_PREFIX, "") : null;
    if (f.title !== undefined && h1Title !== f.title) fail(a, "schema", `title != H1: \`${f.title}\` vs \`${h1Title}\``);

    if (f.status !== undefined && !STATUSES.has(f.status)) fail(a, "schema", `status \`${f.status}\` is not accepted|withdrawn`);
    if (f.source !== undefined && !SOURCES.has(f.source)) fail(a, "schema", `source \`${f.source}\` is not grilling|fix|sweep`);
    if (f.date !== undefined && !DATE.test(String(f.date))) fail(a, "schema", `date \`${f.date}\` is not YYYY-MM-DD`);
    if (f.ticket !== undefined && !isInt(f.ticket) && !(Array.isArray(f.ticket) && f.ticket.length > 0 && f.ticket.every(isInt))) {
      fail(a, "schema", `ticket must be an integer or a list of integers, got ${JSON.stringify(f.ticket)}`);
    }
    for (const k of ["map", "pr"]) {
      if (f[k] !== undefined && !isInt(f[k])) fail(a, "schema", `${k} must be an integer, got ${JSON.stringify(f[k])}`);
    }
    if (f.proof !== undefined) checkProof(a, f, repoRoot, fail);

    if (a.number > LEGACY_MAX) {
      if (f.ticket === undefined) fail(a, "schema", `above ${LEGACY_MAX}, \`ticket\` is required`);
      else if (f.ticket !== a.number) {
        fail(a, "schema", `above ${LEGACY_MAX}, number ${a.number} must equal ticket ${JSON.stringify(f.ticket)}`);
      }
      if (typeof f.title === "string" && words(f.title) > TITLE_CAP) {
        fail(a, "schema", `title is ${words(f.title)} words, cap is ${TITLE_CAP}`);
      }
      if (!prefix) fail(a, "schema", `H1 must carry the ${pad(a.number)}: prefix`);
      const first = a.headings.find((h) => h.level === 2);
      if (!first || first.title !== "Decision") fail(a, "decision-block", "first ## must be Decision");
      const d = decisionBlock(a.body);
      if (d.words > DECISION_CAP) fail(a, "decision-block", `Decision block is ${d.words} words, cap is ${DECISION_CAP}`);
    }

    if (f.status === "withdrawn") checkWithdrawal(a, f, adrs, repoRoot, fail);
    checkRelations(a, f, adrs, fail);
  }
  return problems;
}

export function derivedStatus(set, incoming) {
  if (set === "withdrawn") return "withdrawn";
  let best = "accepted";
  for (const e of incoming) {
    const s = DERIVES[e.kind];
    if (s && RANK[s] > RANK[best]) best = s;
  }
  return best;
}

export function incomingEdges(adrs) {
  const incoming = new Map();
  const push = (n, e) => incoming.set(n, [...(incoming.get(n) ?? []), e]);
  for (const a of adrs.values()) {
    if (!Array.isArray(a.front?.relations)) continue;
    for (const r of a.front.relations) {
      push(r.adr, { kind: r.kind, from: a.number, clause: r.clause ?? null });
      if (r.kind === "sibling") push(a.number, { kind: "sibling", from: r.adr, clause: null });
    }
  }
  for (const list of incoming.values()) list.sort((x, y) => x.from - y.from || x.kind.localeCompare(y.kind));
  return incoming;
}

export function plannedMarkers(a, incoming) {
  const out = [];
  if (a.front.status === "withdrawn") out.push({ kind: "withdrawn", from: null, clause: null });
  for (const e of incoming) if (DERIVES[e.kind]) out.push({ kind: e.kind, from: e.from, clause: e.clause });
  return out.sort(
    (x, y) => MARKER_ORDER[x.kind] - MARKER_ORDER[y.kind] || (x.from ?? 0) - (y.from ?? 0) || String(x.clause).localeCompare(String(y.clause)),
  );
}

export function buildIndex(adrs) {
  const incoming = incomingEdges(adrs);
  const rows = [];
  for (const a of [...adrs.values()].sort((x, y) => x.number - y.number)) {
    const f = a.front;
    if (!f) continue;
    const inc = incoming.get(a.number) ?? [];
    rows.push({
      number: a.number,
      file: a.file,
      title: f.title,
      slug: f.slug,
      date: String(f.date),
      status: { set: f.status, derived: derivedStatus(f.status, inc) },
      source: f.source,
      proof: f.proof,
      ticket: f.ticket ?? null,
      map: f.map ?? null,
      pr: f.pr ?? null,
      withdrawal: f.withdrawal ?? null,
      relations: f.relations ?? [],
      incoming: inc,
      sections: [...a.sections],
      markers: plannedMarkers(a, inc),
      decision: decisionBlock(a.body),
    });
  }
  return { generated: `check-adr-index.mjs from the front matter of ${ADR_DIR}/*.md`, adrs: rows };
}

export function renderIndex(index) {
  return `${JSON.stringify(index, null, 2)}\n`;
}

export function run(repoRoot, { write }) {
  const adrs = loadAdrs(repoRoot);
  const converted = [...adrs.values()].filter((a) => a.front).length;
  const problems = validate(adrs, repoRoot);
  if (problems.length > 0) return { code: 2, adrs, converted, problems, diffs: [], wrote: false };

  const want = renderIndex(buildIndex(adrs));
  const path = join(repoRoot, INDEX_FILE);
  const have = existsSync(path) ? readFileSync(path, "utf8") : null;
  if (have === want) return { code: 0, adrs, converted, problems, diffs: [], wrote: false };
  if (write) {
    writeFileSync(path, want);
    return { code: 0, adrs, converted, problems, diffs: [], wrote: true };
  }
  return { code: 1, adrs, converted, problems, diffs: [`${INDEX_FILE} is stale`], wrote: false };
}

function main() {
  const argv = process.argv.slice(2);
  const write = argv.includes("--write");
  const r = run(DEFAULT_ROOT, { write });
  for (const p of r.problems) console.error(`check:adr-index: ${p.message}`);
  for (const d of r.diffs) console.error(`check:adr-index: ${d}, run \`npm run write:adr-index\``);
  if (r.wrote) console.log(`check:adr-index wrote ${INDEX_FILE}`);
  if (r.code === 0) {
    console.log(`check:adr-index OK — ${r.adrs.size} ADR(s), ${r.converted} with front matter.`);
  }
  process.exit(r.code);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
