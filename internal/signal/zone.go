package signal

import (
	"strings"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

func DeclaredNames(content, origin string) map[string]bool {
	// A pragmatic reader: $INCLUDE, $GENERATE and RDATA change no owner name, so none is evaluated.
	out := map[string]bool{}
	// Keyed the way every Name is: label sequence, ASCII-lowercased, no trailing dot (ADR-0055).
	cur := rw.CanonicalName(strings.TrimSuffix(origin, "."))
	lastOwner := ""

	for _, raw := range strings.Split(content, "\n") {
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "$") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 && strings.EqualFold(fields[0], "$ORIGIN") {
				cur = qualify(fields[1], cur)
			}
			continue
		}

		var owner string
		if line[0] == ' ' || line[0] == '\t' {
			owner = lastOwner
		} else {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			owner = qualify(fields[0], cur)
			lastOwner = owner
		}
		// A wildcard denotes a set of names rather than one, so it is a subject nowhere (ADR-0060).
		if owner == "" || leftmostWildcard(owner) {
			continue
		}
		out[owner] = true
	}
	return out
}

func qualify(token, origin string) string {
	if token == "@" {
		return origin
	}
	if strings.HasSuffix(token, ".") {
		return rw.CanonicalName(token)
	}
	if origin == "" {
		return rw.CanonicalName(token)
	}
	return rw.CanonicalName(token + "." + origin)
}

func stripComment(line string) string {
	// A ; inside a quoted TXT never reaches owner-name position, which is all this reader consumes.
	if i := strings.IndexByte(line, ';'); i >= 0 {
		return line[:i]
	}
	return line
}

func leftmostWildcard(name string) bool {
	return name == "*" || strings.HasPrefix(name, "*.")
}
