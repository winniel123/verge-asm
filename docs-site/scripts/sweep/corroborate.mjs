import { spellsPath } from "../citations/extract.mjs";

const IDENTIFIER = /[A-Za-z_][A-Za-z0-9_.]*/g;
const SPAN = /`([^`\n]*)`/g;

// A retired token spells its line two ways, and both hide the path from the path class (#1968).
const LINE_SUFFIX = /(?::\d+(?:-\d+)?|#L\d+(?:-[Ll]?\d+)?)$/;

// `retention.Retirer.Run` and `Retirer.Run` name one declaration, because a citing document
// qualifies a Go name by package and the row's inventory does not.
function sameName(a, b) {
  return a === b || a.endsWith(`.${b}`) || b.endsWith(`.${a}`);
}

// A route, a header and a command carry a slash, and `func Alpha()` carries none (SPEC §5.2).
function routeSpan(body) {
  return body.startsWith("/") || (/\s/.test(body) && body.includes("/"));
}

// The tail a path class cannot read: `cmd/web.sevLabel` and `cmd/web/signals.go` differ here.
function dottedTail(value) {
  const base = value.slice(value.lastIndexOf("/") + 1);
  const dot = base.lastIndexOf(".");
  return dot > 0 ? base.slice(dot).toLowerCase() : null;
}

// A qualified name spells `<package path>.<Name>`, and the tree holds that package path as a
// directory. `web-state/session.key` and `_gen/ARM64.rules` spell no directory, so both stay paths.
function namesPackage(value, env) {
  const cut = value.lastIndexOf("/");
  const base = value.slice(cut + 1);
  const dot = base.indexOf(".");
  if (cut < 0 || dot <= 0) return false;
  return env.tracked.dirs.has(`${value.slice(0, cut)}/${base.slice(0, dot)}`);
}

// A sibling citation spells a path, and a retired one hides it behind a line (SPEC §5.2 rule 3).
function citesPath(body, env) {
  const value = body.replace(LINE_SUFFIX, "");
  if (!spellsPath(value)) return false;
  const target = value.split("#")[0];
  const tail = dottedTail(target);
  // An extension the tree uses names a file, whatever else the span reads like.
  if (tail === null || env.extensions.has(tail)) return true;
  // Both signals must agree before rule 1 may reach inside a path-shaped span (#2013).
  return !namesPackage(target, env);
}

function reads(body, token, env) {
  // The citation's own span spells the path, and a path must corroborate nothing.
  if (body.includes(token)) return false;
  if (citesPath(body, env)) return false;
  return !routeSpan(body);
}

// A name spelled before the citation is the dominant true-positive shape (SPEC §5.1).
function identifiersOn(lineText, token, env) {
  const spans = [...lineText.matchAll(SPAN)];
  const own = spans.find((m) => m[1].includes(token));
  // A bare-prose copy of the token sits before the citation's own span, and orders nothing.
  const citeAt = own === undefined ? lineText.indexOf(token) : own.index;
  const out = [];
  for (const m of spans) {
    if (!reads(m[1], token, env)) continue;
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
 *
 * `env` is the citations environment, and its tracked tree tells a sibling path from a
 * package-qualified name.
 */
export function rivalName(lineText, token, names, region, env) {
  if (typeof lineText !== "string" || lineText === "") return UNPROVEN;
  // The reach is one line, because a block collects a name about another target.
  const idents = identifiersOn(lineText, token, env);
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
