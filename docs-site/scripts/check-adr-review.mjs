#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { basename, dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { ADR_DIR, ADR_FILE, headings } from "./check-adr-sections.mjs";
import { decisionBlock, splitFrontMatter } from "./check-adr-index.mjs";

const DEFAULT_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const API = "https://api.github.com";
const PAGE = 100;

export const REVIEW_MARKER = /^\s*<!-- adr-review sha=([0-9a-f]{40}) verdict=(pass|fail) -->/;

const short = (sha) => sha.slice(0, 7);

export function parseMarker(body) {
  if (typeof body !== "string") return null;
  const m = REVIEW_MARKER.exec(body);
  return m ? { sha: m[1], verdict: m[2] } : null;
}

export function addedAdrFiles(files) {
  return files
    .filter((f) => f.status === "added")
    .map((f) => f.filename)
    .filter((name) => dirname(name) === ADR_DIR && ADR_FILE.test(basename(name)))
    .sort();
}

function normalise(text) {
  return text
    .split(/\r?\n/)
    .map((l) => l.replace(/\s+$/, ""))
    .join("\n")
    .trim();
}

export function bodyDecision(prBody) {
  if (typeof prBody !== "string") return null;
  const has = headings(prBody).some((h) => h.level === 2 && h.title === "Decision");
  return has ? normalise(decisionBlock(prBody).text) : null;
}

function firstDifference(want, have) {
  const a = want.split("\n");
  const b = have.split("\n");
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    if (a[i] !== b[i]) return { line: i + 1, file: a[i] ?? "<end>", body: b[i] ?? "<end>" };
  }
  return null;
}

export function evaluate({ headSha, files, comments, prBody, readAdr }) {
  const added = addedAdrFiles(files);
  const problems = [];
  if (added.length === 0) return { code: 0, added, problems };

  if (added.length > 1) problems.push(`adds ${added.length} ADR files (${added.join(", ")}); one ADR per PR`);

  const markers = comments.map((c) => parseMarker(c.body)).filter((m) => m && m.sha === headSha);
  if (markers.length === 0) {
    problems.push(`no adr-review marker for head ${short(headSha)}; run the adr-review skill on this PR`);
  } else if (markers.length > 1) {
    problems.push(`${markers.length} adr-review markers for head ${short(headSha)}; one comment per SHA, and nobody deletes one`);
  } else if (markers[0].verdict !== "pass") {
    problems.push(`the adr-review verdict for head ${short(headSha)} is fail`);
  }

  if (added.length === 1) {
    const file = added[0];
    const raw = readAdr(file);
    const want = raw === null ? null : normalise(decisionBlock(splitFrontMatter(raw).body).text);
    const have = bodyDecision(prBody);
    if (want === null) problems.push(`cannot read ${file} from the checkout`);
    if (have === null) problems.push("the PR body has no ## Decision section");
    if (want !== null && have !== null && want !== have) {
      const d = firstDifference(want, have);
      problems.push(
        `the PR body's ## Decision differs from ${file} at Decision block line ${d.line}\n  file: ${d.file}\n  body: ${d.body}`,
      );
    }
  }

  return { code: problems.length > 0 ? 1 : 0, added, problems };
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

  let gathered;
  try {
    gathered = await gather({ repo, number: pr.number, token: process.env.GITHUB_TOKEN });
  } catch (err) {
    console.error(`check:adr-review: ${err.message}`);
    process.exit(2);
  }

  const readAdr = (file) => {
    try {
      return readFileSync(join(DEFAULT_ROOT, file), "utf8");
    } catch {
      return null;
    }
  };
  const r = evaluate({ headSha: pr.head.sha, prBody: pr.body ?? "", readAdr, ...gathered });

  for (const p of r.problems) console.error(`check:adr-review: ${p}`);
  if (r.code === 0) {
    const what = r.added.length === 0 ? "no ADR file added" : `${r.added[0]} reviewed at ${short(pr.head.sha)}`;
    console.log(`check:adr-review OK — PR #${pr.number}, ${what}.`);
  }
  process.exit(r.code);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
