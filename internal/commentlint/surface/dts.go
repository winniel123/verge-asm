package surface

import "strings"

func declarationFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(strings.ReplaceAll(name, `\`, "/")), ".d.ts")
}

func markDTSFields(comments []rawComment, next []int, toks []Token) {
	depth := braceDepths(toks)
	for k := range comments {
		i := next[k]
		// §4.3 carves out field prose and not the file, so a block outside a body is not it (#1406).
		if i >= len(toks) || depth[i] < 1 {
			continue
		}
		comments[k].field = fieldSignature(toks, i)
	}
}

func braceDepths(toks []Token) []int {
	out := make([]int, len(toks)+1)
	d := 0
	for i, t := range toks {
		out[i] = d
		if t.Kind != JSOp {
			continue
		}
		switch t.Text {
		case "{":
			d++
		case "}":
			d--
		}
	}
	out[len(toks)] = d
	return out
}

func fieldSignature(toks []Token, i int) bool {
	if toks[i].Kind == JSIdent && toks[i].Text == "readonly" && fieldName(toks, i+1) {
		i++
	}
	if !fieldName(toks, i) {
		return false
	}
	i++
	if i < len(toks) && toks[i].Kind == JSOp && toks[i].Text == "?" {
		i++
	}
	return i < len(toks) && toks[i].Kind == JSOp && toks[i].Text == ":"
}

func fieldName(toks []Token, i int) bool {
	if i >= len(toks) {
		return false
	}
	switch toks[i].Kind {
	case JSIdent, JSString, JSNumber:
		return true
	}
	return false
}
