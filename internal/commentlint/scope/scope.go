package scope

import (
	"path"
	"path/filepath"
	"strings"
)

type Verdict int

const (
	OutOfScope Verdict = iota
	InScope
	Refused
)

var surfaceExts = map[string]bool{
	".go":   true,
	".mjs":  true,
	".ts":   true,
	".jsx":  true,
	".tmpl": true,
	".css":  true,
}

func Classify(name string) Verdict {
	p := strings.TrimPrefix(path.Clean(filepath.ToSlash(name)), "./")
	ext := strings.ToLower(path.Ext(p))
	// A changed .html or .astro file is a scoping error wherever it sits, so
	// the refusal outranks every other exclusion (SPEC §6.7, #1133).
	if ext == ".html" || ext == ".astro" {
		return Refused
	}
	if excludedTree(p) {
		return OutOfScope
	}
	switch ext {
	case ".sql":
		if strings.HasPrefix(p, "db/queries/") {
			return InScope
		}
		return OutOfScope
	}
	if surfaceExts[ext] {
		return InScope
	}
	return OutOfScope
}

func excludedTree(p string) bool {
	if strings.HasPrefix(p, "prototypes/") || strings.HasPrefix(p, "internal/db/") {
		return true
	}
	if strings.HasPrefix(p, "db/migrations/") && strings.EqualFold(path.Ext(p), ".sql") {
		return true
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "node_modules" {
			return true
		}
	}
	return false
}
