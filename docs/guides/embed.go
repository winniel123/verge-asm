// Package guides embeds the operator guides under docs/guides/ so the running
// binary can index them. The guides ARE the content store the Search screen's
// Documentation group reads (PARITY-CHART.md P2.5): the group's original drop
// rationale — "no content store" (#316) — no longer holds now that these guides
// exist, so they are embedded here and indexed server-side.
//
// Only the Markdown guides are embedded; this file itself is not. The embed
// pattern is colocated with the assets, mirroring db/migrations/embed.go.
package guides

import "embed"

//go:embed *.md
var FS embed.FS
