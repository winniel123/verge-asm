import { lineAnchorPattern } from "../citations/lineanchor.mjs";

// A degraded token keeps its path and loses the false precision (SPEC §8.4).
export function replacementFor(result) {
  return result.outcome === "anchor" ? `${result.value}#${result.anchor}` : result.value;
}

const GLUE = "[A-Za-z0-9_.@+/-]*";

// `a/b.go:265+` spells "onwards", and a region subsumes that, so the rewrite eats it (#1975).
function gluedPattern() {
  const { source, flags } = lineAnchorPattern();
  return new RegExp(`${source}${GLUE}`, flags);
}

export function trailingGlue(markdown, hit) {
  const slice = markdown.slice(hit.start, hit.end);
  const at = slice.indexOf(hit.token);
  if (at < 0) return "";
  return new RegExp(`^${GLUE}`).exec(slice.slice(at + hit.token.length))?.[0] ?? "";
}

// `internal/x.go:4` is a prefix of `internal/x.go:42`, so the scan's own pattern bounds the edit.
function replaceIn(text, replacements) {
  return text.replace(gluedPattern(), (m, token) => replacements.get(token) ?? m);
}

/**
 * Rewrite one document in place, inside the nodes the scan read.
 *
 * `conversions` holds one derived result per occurrence. An occurrence keys the edit, because
 * one token may convert at one site and hold at another. A following code span reads as a
 * snippet at one site and not at the other.
 */
export function rewriteDocument(markdown, conversions) {
  const edits = new Map();
  for (const r of conversions) {
    if (r.start === undefined || r.end === undefined) {
      throw new Error(`sweep: the scan gave no position for ${r.token}`);
    }
    const key = `${r.start}:${r.end}`;
    if (!edits.has(key)) edits.set(key, { start: r.start, end: r.end, replacements: new Map() });
    edits.get(key).replacements.set(r.token, replacementFor(r));
  }
  // Last edit first, so an earlier splice never moves a later offset.
  const ordered = [...edits.values()].sort((a, b) => b.start - a.start);
  let out = markdown;
  for (const { start, end, replacements } of ordered) {
    out = out.slice(0, start) + replaceIn(out.slice(start, end), replacements) + out.slice(end);
  }
  return out;
}
