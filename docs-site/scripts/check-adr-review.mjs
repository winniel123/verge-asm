#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { basename, dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import {
  ADR_DIR,
  ADR_FILE,
  FENCE,
  LEGACY_MAX,
  MARKER_LINE,
  decisionBlock,
  headings,
  splitFrontMatter,
} from "./check-adr-sections.mjs";

const API = "https://api.github.com";
const PAGE = 100;

export const REVIEW_MARKER = /^\s*<!-- adr-review sha=([0-9a-f]{40}) verdict=(pass|fail) -->[ \t]*(?:\r?\n|$)/;

const short = (sha) => sha.slice(0, 7);

export function parseMarker(body) {
  if (typeof body !== "string") return null;
  const m = REVIEW_MARKER.exec(body);
  return m ? { sha: m[1], verdict: m[2] } : null;
}

function adrFilesWithStatus(files, status) {
  return files
    .filter((f) => f.status === status)
    .map((f) => f.filename)
    .filter((name) => dirname(name) === ADR_DIR && ADR_FILE.test(basename(name)))
    .sort();
}

export const addedAdrFiles = (files) => adrFilesWithStatus(files, "added");
export const modifiedAdrFiles = (files) => adrFilesWithStatus(files, "modified");

export const adrNumber = (file) => Number(ADR_FILE.exec(basename(file))[1]);

const QUOTE = /^\s{0,3}>/;
const SENTINEL = /<!-- adr-marker\b/;

export function legacyQuoteRuns(text) {
  const lines = text.split("\n");
  const runs = [];
  let run = [];
  let fenced = false;
  const flush = () => {
    if (run.length > 0 && !run.some((i) => SENTINEL.test(lines[i]))) runs.push(run.map((i) => i + 1));
    run = [];
  };
  lines.forEach((line, i) => {
    if (FENCE.test(line)) {
      fenced = !fenced;
      flush();
    } else if (!fenced && QUOTE.test(line)) run.push(i);
    else flush();
  });
  flush();
  return runs;
}

export function patchHunks(patch) {
  const out = [];
  let cur = null;
  for (const line of String(patch).split("\n")) {
    const at = /^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line);
    if (at) {
      cur = { changes: [] };
      cur.after = Number(at[1]);
      out.push(cur);
      continue;
    }
    if (!cur || line.startsWith("\\")) continue;
    const kind = line[0] ?? " ";
    if (kind === "+" || kind === "-") cur.changes.push({ kind, text: line.slice(1), at: cur.after });
    if (kind !== "-") cur.after++;
  }
  return out;
}

function toolWritten(changes) {
  // check-adr-markers regenerates every sentinel line, so it is the gate on this hunk (#2231)
  const marker = (c) => MARKER_LINE.test(c.text);
  return changes.some(marker) && changes.every((c) => c.text === "" || marker(c));
}

const correctable = (text) => QUOTE.test(text) && !SENTINEL.test(text);

export function markerScope(patch, after) {
  const hunks = patchHunks(patch);
  const runs = legacyQuoteRuns(after);
  const runOf = new Map();
  runs.forEach((run, i) => run.forEach((line) => runOf.set(line, i)));
  const fresh = new Set();
  for (const h of hunks) for (const c of h.changes) if (c.kind === "+") fresh.add(c.at);
  // ADR-2159 reaches a blockquote that was already there, so a run this patch wrote whole is out
  const standing = runs.map((run) => run.some((line) => !fresh.has(line)));
  const inStandingRun = (line) => runOf.has(line) && standing[runOf.get(line)];

  const offences = [];
  let edits = 0;
  for (const h of hunks) {
    if (h.changes.length === 0 || toolWritten(h.changes)) continue;
    edits++;
    const named = h.changes.filter((c) => c.text.trim() !== "");
    // A blank line carries no sentence, so it rides on the named change beside it (#2231)
    if (named.length === 0) {
      offences.push(...h.changes.map((c) => ({ line: c.at, kind: c.kind, text: c.text })));
      continue;
    }
    for (const c of named) {
      const ok =
        c.kind === "+"
          ? inStandingRun(c.at)
          : correctable(c.text) && (inStandingRun(c.at) || inStandingRun(c.at - 1));
      if (!ok) offences.push({ line: c.at, kind: c.kind, text: c.text });
    }
  }
  return { offences, edits };
}

function normalise(text) {
  // A PR body typed in the GitHub web UI arrives with CRLF line endings
  return text
    .split(/\r?\n/)
    .map((l) => l.replace(/\s+$/, ""))
    .join("\n")
    .trim();
}

const ATTRIBUTION_LINE = /^\s*\u{1F916} Generated with \[Claude Code\]\(\S+\)\s*$/u;
const SESSION_LINE = /^\s*https:\/\/claude\.ai\/code\/session_\S+\s*$/;

function withoutAttribution(prBody) {
  const lines = prBody.split(/\r?\n/);
  const skipBlank = (i) => {
    while (i > 0 && lines[i - 1].trim() === "") i--;
    return i;
  };
  let end = skipBlank(lines.length);
  // The session link sits below the attribution line, so the trailer is a block and not one line
  if (end > 0 && SESSION_LINE.test(lines[end - 1])) end = skipBlank(end - 1);
  if (end === 0 || !ATTRIBUTION_LINE.test(lines[end - 1])) return prBody;
  return lines.slice(0, skipBlank(end - 1)).join("\n");
}

export function bodyDecision(prBody) {
  if (typeof prBody !== "string") return null;
  // A last-position ## Decision would absorb the PR body's attribution trailer (#2101).
  const body = withoutAttribution(prBody);
  const has = headings(body).some((h) => h.level === 2 && h.title === "Decision");
  return has ? normalise(decisionBlock(body).text) : null;
}

function firstDifference(want, have) {
  const a = want.split("\n");
  const b = have.split("\n");
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    if (a[i] !== b[i]) return { line: i + 1, file: a[i] ?? "<end>", body: b[i] ?? "<end>" };
  }
  return null;
}

const amendsRoute = (file) =>
  `change it through a later ADR declaring {kind: amends, adr: ${adrNumber(file)}} (adr-governance §3, §8)`;

function modificationProblems(file, entry, readAdr) {
  const after = readAdr(file);
  if (after === null) return { problems: [`cannot read ${file} at the head SHA`], edits: 0 };
  if (typeof entry.patch !== "string") {
    return { problems: [`${file} carries no diff, so row M1 cannot be judged; ${amendsRoute(file)}`], edits: 0 };
  }
  const { offences, edits } = markerScope(entry.patch, after);
  const problems = offences.slice(0, 3).map((o) => {
    const verb = o.kind === "+" ? "adds" : "deletes";
    const shown = o.text.trim() === "" ? "<blank>" : o.text;
    return `${file}:${o.line} ${verb} a line outside a standing legacy marker blockquote, which row M1 refuses above ${LEGACY_MAX}: ${amendsRoute(file)}\n  ${o.kind}${shown}`;
  });
  if (offences.length > 3) problems.push(`${file}: ${offences.length - 3} further line(s) fail row M1`);
  return { problems, edits };
}

export function evaluate({ headSha, files, comments, prBody, readAdr }) {
  const added = addedAdrFiles(files);
  const modified = modifiedAdrFiles(files);
  const problems = [];
  const reviewed = [...added];

  for (const file of modified) {
    // 227 and below keeps the legacy in-file amendment route, so no M row applies (#2231)
    if (adrNumber(file) <= LEGACY_MAX) continue;
    const r = modificationProblems(file, files.find((f) => f.filename === file), readAdr);
    problems.push(...r.problems);
    if (r.problems.length === 0 && r.edits > 0) reviewed.push(file);
  }

  if (added.length > 1) problems.push(`adds ${added.length} ADR files (${added.join(", ")}); one ADR per PR`);
  else if (reviewed.length > 1) {
    problems.push(`reviews ${reviewed.length} ADR files (${reviewed.join(", ")}); one ADR per PR`);
  }

  if (reviewed.length === 0) return { code: problems.length > 0 ? 1 : 0, added, modified, reviewed, problems };

  const markers = comments.map((c) => parseMarker(c.body)).filter((m) => m && m.sha === headSha);
  if (markers.length === 0) {
    problems.push(`no adr-review marker for head ${short(headSha)}; run the adr-review skill on this PR`);
  } else if (markers.length > 1) {
    problems.push(`${markers.length} adr-review markers for head ${short(headSha)}; post one comment per SHA. Do not delete a marker`);
  } else if (markers[0].verdict !== "pass") {
    problems.push(`the adr-review verdict for head ${short(headSha)} is fail`);
  }

  if (added.length === 1) {
    const file = added[0];
    const raw = readAdr(file);
    const want = raw === null ? null : normalise(decisionBlock(splitFrontMatter(raw).body).text);
    const have = bodyDecision(prBody);
    if (want === null) problems.push(`cannot read ${file} at the head SHA`);
    if (have === null) problems.push("the PR body has no ## Decision section");
    if (want !== null && have !== null && want !== have) {
      const d = firstDifference(want, have);
      problems.push(
        `the PR body's ## Decision differs from ${file} at Decision block line ${d.line}\n  file: ${d.file}\n  body: ${d.body}`,
      );
    }
  }

  return { code: problems.length > 0 ? 1 : 0, added, modified, reviewed, problems };
}

async function pages(path, token, fetchImpl) {
  const out = [];
  for (let page = 1; ; page++) {
    const headers = { Accept: "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28" };
    if (token) headers.Authorization = `Bearer ${token}`;
    const res = await fetchImpl(`${API}${path}?per_page=${PAGE}&page=${page}`, { headers });
    if (!res.ok) throw new Error(`GET ${path} page ${page}: HTTP ${res.status}`);
    const batch = await res.json();
    out.push(...batch);
    if (batch.length < PAGE) return out;
  }
}

export async function gather({ repo, number, token, fetchImpl = fetch }) {
  const files = await pages(`/repos/${repo}/pulls/${number}/files`, token, fetchImpl);
  const comments = await pages(`/repos/${repo}/issues/${number}/comments`, token, fetchImpl);
  return { files, comments };
}

export async function headContents({ repo, sha, paths, token, fetchImpl = fetch }) {
  const out = new Map();
  for (const path of paths) {
    const headers = { Accept: "application/vnd.github.raw", "X-GitHub-Api-Version": "2022-11-28" };
    if (token) headers.Authorization = `Bearer ${token}`;
    const url = `${API}/repos/${repo}/contents/${path.split("/").map(encodeURIComponent).join("/")}?ref=${sha}`;
    const res = await fetchImpl(url, { headers });
    if (res.status === 404) {
      out.set(path, null);
      continue;
    }
    if (!res.ok) throw new Error(`GET contents/${path}: HTTP ${res.status}`);
    out.set(path, await res.text());
  }
  return out;
}

async function main() {
  const eventPath = process.env.GITHUB_EVENT_PATH;
  const repo = process.env.GITHUB_REPOSITORY;
  if (!eventPath || !repo) {
    console.error("check:adr-review: GITHUB_EVENT_PATH and GITHUB_REPOSITORY are required");
    process.exit(2);
  }
  const event = JSON.parse(readFileSync(eventPath, "utf8"));
  const pr = event.pull_request;
  if (!pr) {
    console.error("check:adr-review: the event carries no pull_request");
    process.exit(2);
  }

  const token = process.env.GITHUB_TOKEN;
  let gathered;
  let contents;
  try {
    gathered = await gather({ repo, number: pr.number, token });
    // The patch numbers a line in the head file, and the checkout is the merge ref (#2231)
    const paths = [...addedAdrFiles(gathered.files), ...modifiedAdrFiles(gathered.files)];
    contents = await headContents({ repo, sha: pr.head.sha, paths, token });
  } catch (err) {
    console.error(`check:adr-review: ${err.message}`);
    process.exit(2);
  }

  const readAdr = (file) => contents.get(file) ?? null;
  const r = evaluate({ headSha: pr.head.sha, prBody: pr.body ?? "", readAdr, ...gathered });

  for (const p of r.problems) console.error(`check:adr-review: ${p}`);
  if (r.code === 0) {
    const what =
      r.reviewed.length === 0 ? "no ADR file to review" : `${r.reviewed[0]} reviewed at ${short(pr.head.sha)}`;
    console.log(`check:adr-review OK — PR #${pr.number}, ${what}.`);
  }
  process.exit(r.code);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
