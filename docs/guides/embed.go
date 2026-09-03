// Package guides embeds the operator guides under docs/guides/ so the running
// binary can index them. They are the content store the Search screen's
// Documentation group reads (PARITY-CHART.md P2.5, #316).
package guides

import "embed"

//go:embed *.md
var FS embed.FS
