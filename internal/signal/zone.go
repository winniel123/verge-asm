package signal

import (
	"strings"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

func DeclaredNames(content, origin string) map[string]bool {
	out := map[string]bool{}
	walkOwners(content, origin, func(owner string, _ []string) {
		out[owner] = true
	})
	return out
}

func DelegatedSubzones(content, origin string) map[string]bool {
	apex := rw.CanonicalName(origin)
	out := map[string]bool{}
	walkOwners(content, origin, func(owner string, rest []string) {
		if owner != apex && rrType(rest) == "NS" {
			out[owner] = true
		}
	})
	return out
}

func walkOwners(content, origin string, visit func(owner string, rest []string)) {
	// A pragmatic reader: $INCLUDE, $GENERATE and RDATA change no owner name, so none is evaluated.
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
		fields := strings.Fields(line)
		if line[0] == ' ' || line[0] == '\t' {
			owner = lastOwner
		} else {
			if len(fields) == 0 {
				continue
			}
			owner = qualify(fields[0], cur)
			lastOwner = owner
			fields = fields[1:]
		}
		// A wildcard denotes a set of names rather than one, so it is a subject nowhere (ADR-0060).
		if owner == "" || leftmostWildcard(owner) || highBit(owner) {
			continue
		}
		visit(owner, fields)
	}
}

func rrType(rest []string) string {
	// RFC 1035 §5.1 lets TTL and class appear in either order, and both are optional.
	for _, tok := range rest {
		if isTTL(tok) || isClass(tok) {
			continue
		}
		return strings.ToUpper(tok)
	}
	return ""
}

func isTTL(tok string) bool {
	return tok != "" && tok[0] >= '0' && tok[0] <= '9'
}

func isClass(tok string) bool {
	switch strings.ToUpper(tok) {
	case "IN", "CH", "HS", "CS":
		return true
	}
	return false
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

func highBit(name string) bool {
	// A high-bit octet typed in a text form has two readings, so it is refused (ADR-0055).
	for i := 0; i < len(name); i++ {
		if name[i] >= 0x80 {
			return true
		}
	}
	return false
}
