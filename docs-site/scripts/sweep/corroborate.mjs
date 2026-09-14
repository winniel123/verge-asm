const IDENTIFIER = /[A-Za-z_][A-Za-z0-9_.]*/g;

// `retention.Retirer.Run` and `Retirer.Run` name one declaration, because a citing document
// qualifies a Go name by package and the row's inventory does not.
function sameName(a, b) {
  return a === b || a.endsWith(`.${b}`) || b.endsWith(`.${a}`);
}

// The citation's own span spells the path, and a path must corroborate nothing.
function identifiersOn(lineText, token) {
  const out = new Set();
  for (const [, body] of lineText.matchAll(/`([^`\n]*)`/g)) {
    if (body.includes(token)) continue;
    for (const [ident] of body.matchAll(IDENTIFIER)) out.add(ident);
  }
  return out;
}

/**
 * The declaration the citing line names, where the target really declares it and the line
 * spells no name for the region the line number landed in.
 */
export function rivalName(lineText, token, names, region) {
  if (typeof lineText !== "string" || lineText === "") return null;
  // The reach is one line, because a block collects a name about another target.
  const idents = identifiersOn(lineText, token);
  for (const ident of idents) {
    // The line spells the region, so the number and the prose agree and nothing is amiss.
    if (sameName(ident, region)) return null;
  }
  // One pass over an iterator would exhaust it, and every later identifier would see no name.
  const declared = [...names];
  const rivals = new Set();
  for (const ident of idents) {
    for (const name of declared) {
      if (name === region) continue;
      if (sameName(ident, name)) rivals.add(name);
    }
  }
  // A sorted pick keeps one run's report byte-identical to the next.
  return rivals.size === 0 ? null : [...rivals].sort()[0];
}
