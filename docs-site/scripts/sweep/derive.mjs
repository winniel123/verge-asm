import { ROWS, rowFor } from "../citations/rows.mjs";
import { CONTAINMENT_ROW } from "../citations/rows/containment.mjs";
import { ANCHOR } from "../citations/extract.mjs";
import { holdsSnippet } from "../citations/armb.mjs";
import { classify } from "../citations/classify.mjs";
import { namesAnotherSite } from "./rewrite.mjs";
import { rivalName } from "./corroborate.mjs";

// A retired token spells its line two ways, and both carry an optional end (SPEC §5).
const SPLIT = /^(.*?)(?::(\d+)(?:-(\d+))?|#L(\d+)(?:-[Ll]?(\d+))?)$/;

export function splitToken(token) {
  const m = SPLIT.exec(token);
  if (!m) return null;
  const start = Number(m[2] ?? m[4]);
  const end = m[3] ?? m[5];
  return { value: m[1], fromLine: start, toLine: end === undefined ? start : Number(end) };
}

// The sweep proposes what the gate accepts, so it resolves the path the gate's own way (#1970).
function resolvePath(env, docFile, value) {
  const [result] = classify(env, docFile, [
    { raw: value, kind: "code", line: 1, withdrawn: false, prose: "", refs: [], snippet: null },
  ]);
  if (result === undefined) return { status: "unresolved" };
  return { status: result.status, path: result.path };
}

// The innermost region wins, because a heading nests and a citation names the nearest one.
function enclosing(spans, start, end) {
  let best = null;
  let tied = false;
  for (const [name, regions] of spans) {
    for (const [from, to] of regions) {
      if (from > start || end > to) continue;
      if (best === null || from > best.from) {
        best = { name, from, to };
        tied = false;
      } else if (from === best.from && name !== best.name) {
        tied = true;
      }
    }
  }
  // Two names on one declaration line name two places, and the sweep never picks one (§3.3 rule 3).
  if (tied) return { ambiguous: true };
  return best;
}

function degraded(entry, reason) {
  return { ...entry, outcome: "degraded", reason };
}

// A dead path is the path arm's business, and a bare path would red the gate (SPEC §7.6).
function held(entry, reason) {
  return { ...entry, outcome: "held", reason };
}

// A token is derived against one row's inventory, or it keeps its path and loses the line (§8.4).
function deriveOne(token, inventory, env) {
  const entry = inventory.get(token.path);
  if (entry === undefined) return degraded(token, "the inventory names no such target");
  if (entry.error) return degraded(token, `unreadable target: ${entry.error}`);
  // A target with no vocabulary MAY carry a containment anchor, and a MAY is never minted.
  if (!entry.spans) return degraded(token, "the target declares no anchor vocabulary");

  const lines = entry.lines ?? [];
  if (token.toLine > lines.length) {
    return degraded(token, `line ${token.toLine} is past the end of ${token.path} (${lines.length} lines)`);
  }
  if ((lines[token.fromLine - 1] ?? "").trim() === "") {
    return degraded(token, `line ${token.fromLine} of ${token.path} is blank`);
  }

  const region = enclosing(entry.spans, token.fromLine, token.toLine);
  if (region === null) {
    return degraded(token, `no ${token.row.vocabulary} in ${token.path} encloses the cited line`);
  }
  if (region.ambiguous) {
    return degraded(token, "two declarations share the enclosing region");
  }
  // A row may hold a name and still refuse its spelling, and the gate would then go red (SPEC §4).
  const refusal = entry.refused?.get(region.name);
  if (refusal) return degraded(token, refusal);
  // An anchor outside the extractor's character class is unreadable (SPEC §3.3 rule 4).
  if (!ANCHOR.test(region.name)) {
    return degraded(token, `\`${region.name}\` leaves the anchor character class`);
  }
  // A stale line resolves as a live one does, so a name the citing line spells overrules it.
  const judged = rivalName(token.lineText, token.token, entry.spans.keys(), region.name, env);
  if (judged.verdict === "suspect" && judged.position === "before") {
    return degraded(
      token,
      `the citing line names \`${judged.rival}\` before the citation, and line ${token.fromLine} ` +
        `of ${token.path} sits in ${region.name}, so the line has drifted`,
    );
  }
  // A silent degrade drops a sound anchor and a silent conversion keeps a wrong one (SPEC §5.2).
  const review = judged.verdict === "suspect" ? { rival: judged.rival, position: judged.position } : null;
  // Converting mints a snippet claim over the next code span (SPEC §3.4).
  if (token.snippet != null && !holdsSnippet(entry.lines, entry.spans.get(region.name), token.snippet)) {
    return degraded(
      token,
      `converting would read \`${token.snippet}\` as a snippet, and no line inside ` +
        `${region.name} holds it`,
    );
  }
  return { ...token, outcome: "anchor", anchor: region.name, ...(review ? { review } : {}) };
}

// One derivation for all four conversion tickets, so no batch re-invents the conversion (#1975).
export function derive(repoRoot, env, found, rows = ROWS) {
  const pending = [];
  const done = [];

  for (const hit of found) {
    const parts = splitToken(hit.token);
    // No bare path is recoverable here, so the token is never rewritten.
    if (parts === null) {
      done.push(held({ ...hit }, "the token spells no line"));
      continue;
    }
    const base = { ...hit, value: parts.value, fromLine: parts.fromLine, toLine: parts.toLine };
    // A reversed range satisfies the containment test against a region holding neither line.
    if (parts.fromLine > parts.toLine) {
      done.push(degraded(base, `the range ${parts.fromLine}-${parts.toLine} runs backwards`));
      continue;
    }
    // One anchor represents neither of two places, and the sweep never picks one (§3.3 rule 3).
    if (namesAnotherSite(hit.glue)) {
      done.push(degraded(base, `the token names another place at \`${hit.glue.trim()}\``));
      continue;
    }
    const { status, path } = resolvePath(env, hit.file, parts.value);
    if (status === "dead") {
      const why = `${parts.value} is no longer in the tree, so the bare path would red the gate`;
      done.push(held(base, why));
      continue;
    }
    // Arm B judges an anchor only where the path resolves, so the sweep proposes none (SPEC §7.6).
    if (status !== "ok") {
      done.push(degraded(base, `the path does not resolve in the tree (${status})`));
      continue;
    }
    const row = rowFor(path, rows);
    if (row === CONTAINMENT_ROW) {
      done.push(degraded({ ...base, path, row }, "the target declares no anchor vocabulary"));
      continue;
    }
    pending.push({ ...base, path, row });
  }

  const byRow = new Map();
  for (const token of pending) {
    if (!byRow.has(token.row)) byRow.set(token.row, []);
    byRow.get(token.row).push(token);
  }

  for (const [row, tokens] of byRow) {
    const paths = [...new Set(tokens.map((t) => t.path))].sort();
    let inventory;
    try {
      inventory = row.inventory(repoRoot, paths);
    } catch (err) {
      const detail = (err.message ?? "").split("\n")[0] || err.code || "it did not finish";
      for (const token of tokens) done.push(degraded(token, `the ${row.name} row could not run: ${detail}`));
      continue;
    }
    for (const token of tokens) done.push(deriveOne(token, inventory, env));
  }

  return done.sort(
    (a, b) => a.file.localeCompare(b.file) || a.line - b.line || a.token.localeCompare(b.token),
  );
}
