import { markerInventory } from "./marker.mjs";

// sqlc reads an indented marker too, and a missed marker reds a correct citation (SPEC §7.3).
const QUERY_NAME = /^[ \t]*-- name: (\S+)/;

export const SQLC_ROW = {
  name: "sqlc",
  vocabulary: "`-- name:` query",
  // A goose migration declares no names, so the extension is not the key (SPEC §3.2 rule 2).
  matches: (path) => path.startsWith("db/queries/") && path.endsWith(".sql"),
  inventory: (repoRoot, paths) => markerInventory(repoRoot, paths, QUERY_NAME),
};
