// Package migrations embeds the goose migration files into the web binary, so
// the binary that applies them carries them rather than reading a mounted path
// (packaging-and-configuration.md §5.1).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
