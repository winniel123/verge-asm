import { lineAnchorPattern } from "../citations/lineanchor.mjs";

// A degraded token keeps its path and loses the false precision (SPEC §8.4).
export function replacementFor(result) {
  return result.outcome === "anchor" ? `${result.value}#${result.anchor}` : result.value;
}

const GLUE = "[A-Za-z0-9_.@+/-]*";

// `a/b.go:978,997` names a second place, and the scan reads only the first (#1976).
const SITES = String.raw`(?:[,:]\s*:?\d+(?:-\d+)?)*`;

// A residue the rewrite leaves behind keeps a line number the ratchet can no longer see (#1976).
export function namesAnotherSite(glue) {
  return /[,:]\s*:?\d/.test(glue ?? "");
}

// `a/b.go:265+` spells "onwards", and a region subsumes that, so the rewrite eats it (#1975).
function gluedPattern() {
  const { source, flags } = lineAnchorPattern();
  return new RegExp(`${source}${GLUE}${SITES}`, flags);
}

// The glue follows this occurrence, and a search hands every copy the first one's glue (#2249).
export function trailingGlue(hit) {
  if (hit.text == null || hit.at == null) return "";
  const after = hit.text.slice(hit.at + hit.token.length);
  return new RegExp(`^${GLUE}${SITES}`).exec(after)?.[0] ?? "";
}

// `internal/x.go:4` is a prefix of `internal/x.go:42`, so the scan's own pattern bounds the edit.
function replaceIn(text, replacements) {
  return text.replace(gluedPattern(), (m, token) => replacements.get(token) ?? m);
}

// A link label may hold a code span, and the scan reads both. The inner splice would move the
// outer node's end, so an enclosed range merges into its enclosing one and the outermost wins.
function groupEdits(conversions) {
  const sorted = [...conversions].sort((a, b) => a.start - b.start || b.end - a.end);
  const groups = [];
  for (const r of sorted) {
    if (r.start === undefined || r.end === undefined) {
      throw new Error(`sweep: the scan gave no position for ${r.token}`);
    }
    const open = groups[groups.length - 1];
    if (open && r.start < open.end) {
      // A partial overlap is not a shape mdast produces, and a blind merge would widen the window.
      if (r.end > open.end) throw new Error(`sweep: ${r.token} overlaps without nesting`);
      open.members.push(r);
      continue;
    }
    groups.push({ start: r.start, end: r.end, members: [r] });
  }
  return groups.map((g) => {
    const replacements = new Map();
    for (const r of g.members) {
      const to = replacementFor(r);
      const seen = replacements.get(r.token);
      // One slice cannot spell one token two ways, so a split verdict rewrites nothing here.
      if (seen !== undefined && seen !== to) return { ...g, conflict: r.token };
      replacements.set(r.token, to);
    }
    return { ...g, replacements };
  });
}

/**
 * Rewrite one document in place, inside the nodes the scan read.
 *
 * `conversions` holds one derived result per occurrence. An occurrence keys the edit, because
 * one token may convert at one site and hold at another. A following code span reads as a
 * snippet at one site and not at the other.
 */
export function rewriteDocument(markdown, conversions) {
  const groups = groupEdits(conversions);
  const conflicts = groups.filter((g) => g.conflict);
  if (conflicts.length > 0) {
    throw new Error(`sweep: one span spells ${conflicts.map((g) => g.conflict).join(", ")} two ways`);
  }
  // Last edit first, so an earlier splice never moves a later offset.
  const ordered = groups.sort((a, b) => b.start - a.start);
  let out = markdown;
  for (const { start, end, replacements } of ordered) {
    out = out.slice(0, start) + replaceIn(out.slice(start, end), replacements) + out.slice(end);
  }
  return out;
}
