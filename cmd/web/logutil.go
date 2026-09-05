package main

import "strings"

func logSafe(s string) string {
	// gosec's G706 tracker does not follow this sanitizer, so each wrapped call needs a waiver.
	return strings.NewReplacer("\r", "\\r", "\n", "\\n").Replace(s)
}
