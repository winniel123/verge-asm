import { GO_ROW } from "./rows/go.mjs";
import { SQLC_ROW } from "./rows/sqlc.mjs";
import { TEMPLATE_ROW } from "./rows/template.mjs";
import { MARKDOWN_ROW } from "./rows/markdown.mjs";

// A row keys on a path predicate, because the predicate follows the tool that owns the
// file. A bare extension does not.
export const ROWS = [GO_ROW, SQLC_ROW, TEMPLATE_ROW, MARKDOWN_ROW];

// The table is open, so a missing row never blocks a citation (SPEC §3.2 rule 1).
export function rowFor(path, rows = ROWS) {
  return rows.find((row) => row.matches(path)) ?? null;
}
