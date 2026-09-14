import { spellsPath } from "../citations/extract.mjs";

const IDENTIFIER = /[A-Za-z_][A-Za-z0-9_.]*/g;

// `retention.Retirer.Run` and `Retirer.Run` name one declaration, because a citing document
// qualifies a Go name by package and the row's inventory does not.
function sameName(a, b) {
  return a === b || a.endsWith(`.${b}`) || b.endsWith(`.${a}`);
}

// A span holding a space or a leading `/` spells a route, not a declaration (SPEC §5.2 rule 3).
function routeSpan(body) {
  return /\s/.test(body) || body.startsWith("/");
}

function reads(body, token) {
  // The citation's own span spells the path, and a path must corroborate nothing.
  if (body.includes(token)) return false;
  // A second citation on the line is a path too, and it names a target rather than a declaration.
  if (spellsPath(body)) return false;
  return !routeSpan(body);
}

// A name spelled before the citation is the dominant true-positive shape (SPEC §5.1).
function identifiersOn(lineText, token) {
  const citeAt = lineText.indexOf(token);
  const out = [];
  for (const m of lineText.matchAll(/`([^`\n]*)`/g)) {
    if (!reads(m[1], token)) continue;
    // A line that does not spell the citation orders nothing, so every name keeps the old bias.
    const before = citeAt < 0 || m.index < citeAt;
    for (const [ident] of m[1].matchAll(IDENTIFIER)) out.push({ ident, before });
  }
  return out;
}

const UNPROVEN = { verdict: "unproven", rival: null, position: null };
const CORROBORATED = { verdict: "corroborated", rival: null, position: null };

/**
 * The verdict the citing line passes on an anchor: `corroborated` where the line spells the
 * anchor's own region, `suspect` where it spells another declaration the target really has, and
 * `unproven` where it spells neither. A suspect verdict carries the rival and its position.
 */
export function rivalName(lineText, token, names, region) {
  if (typeof lineText !== "string" || lineText === "") return UNPROVEN;
  // The reach is one line, because a block collects a name about another target.
  const idents = identifiersOn(lineText, token);
  for (const { ident } of idents) {
    // The line spells the region, so the number and the prose agree and nothing is amiss.
    if (sameName(ident, region)) return CORROBORATED;
  }
  // One pass over an iterator would exhaust it, and every later identifier would see no name.
  const declared = [...names];
  const before = new Set();
  const after = new Set();
  for (const { ident, before: leads } of idents) {
    for (const name of declared) {
      if (name === region) continue;
      if (sameName(ident, name)) (leads ? before : after).add(name);
    }
  }
  // A sorted pick keeps one run's report byte-identical to the next.
  const pick = (set) => [...set].sort()[0];
  if (before.size > 0) return { verdict: "suspect", rival: pick(before), position: "before" };
  if (after.size > 0) return { verdict: "suspect", rival: pick(after), position: "after" };
  return UNPROVEN;
}
