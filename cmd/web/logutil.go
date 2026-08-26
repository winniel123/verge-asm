package main

import "strings"

// logSafe neutralizes CR/LF in a value before it is interpolated into a log
// line, defeating log-injection / log-forging (CWE-117): a caller-controlled
// string can otherwise smuggle a newline and forge a second, attacker-authored
// log record. It replaces carriage return and line feed with their escaped
// two-character forms so the value stays on one line and stays legible.
//
// gosec's G706 taint tracker does not recognise this custom sanitizer, so a
// wrapped site still needs a companion `// #nosec G706 (sanitized via logSafe)`
// on the log call to close the alert.
func logSafe(s string) string {
	return strings.NewReplacer("\r", "\\r", "\n", "\\n").Replace(s)
}
