package signal

import (
	"strings"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// DeclaredNames extracts the set of `Name`s an operator's zone file declares — the
// owner names of its records — keyed the way every Name is (label sequence,
// lower-cased over ASCII, no trailing dot; ADR-0055). It is what puts a name
// inside `zone-declared-name-returns-name-error`'s domain and decides
// `resolved-name-absent-from-zone`'s predicate.
//
// This is a pragmatic master-file reader, not a full RFC 1035 parser: it reads
// owner names, honours `$ORIGIN`, resolves relative names and `@` against the
// current origin, and carries a blank owner (leading whitespace) down from the
// previous record. It does not evaluate `$INCLUDE`, `$GENERATE`, parentheses
// spanning lines, or record RDATA — none of which changes which owner names the
// file declares. A leftmost `*` label is skipped: a wildcard denotes a set of
// names rather than one and is a subject nowhere (CONTEXT.md `Name`; ADR-0060).
func DeclaredNames(content, origin string) map[string]bool {
	out := map[string]bool{}
	cur := rw.CanonicalName(strings.TrimSuffix(origin, "."))
	lastOwner := ""

	for _, raw := range strings.Split(content, "\n") {
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}

		// A directive resets state rather than declaring a name.
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

// stripComment removes a `;` comment. It does not track quoted strings — a `;`
// inside a TXT RDATA quote is rare in owner-name position and never changes the
// owner token, which is all this reader consumes.
func stripComment(line string) string {
	if i := strings.IndexByte(line, ';'); i >= 0 {
		return line[:i]
	}
	return line
}

func leftmostWildcard(name string) bool {
	return name == "*" || strings.HasPrefix(name, "*.")
}
