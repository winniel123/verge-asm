// Package migrations embeds the goose migration files into the web
// binary, so the binary that applies them at startup carries them rather
// than reading from a mounted path (packaging-and-configuration.md §5.1:
// nothing that should appear in the audit trail lives in the environment).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
