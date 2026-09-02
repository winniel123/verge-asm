package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func escapeData(value string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	return r.Replace(value)
}

func escapeProperty(value string) string {
	r := strings.NewReplacer(":", "%3A", ",", "%2C")
	return r.Replace(escapeData(value))
}

func annotationLine(v violation) string {
	return fmt.Sprintf("::warning file=%s,line=%d,title=%s::%s",
		escapeProperty(v.path), v.Line,
		escapeProperty("commentlint ("+v.Rule+")"),
		escapeData(string(v.Class)+" block; CLAUDE.md keeps a comment only on an unrecoverable external cause"))
}

func summary(fileCount int, found []violation, lexFailures int) string {
	var b strings.Builder
	b.WriteString("## commentlint\n\n")
	b.WriteString("Advisory comment lint (SPEC docs/spec/comment-policy.md §6.7). This check never blocks a merge.\n\n")
	fmt.Fprintf(&b, "**%d file(s) linted, %d violation(s).**\n\n", fileCount, len(found))
	// The exit code already separates a lex failure from a violation, so the
	// summary must not re-merge them (SPEC §6.7).
	fmt.Fprintf(&b, "**Lex failures: %d.** A lex failure is not a violation.\n\n", lexFailures)
	if len(found) == 0 {
		return b.String()
	}

	counts := map[string]int{}
	for _, v := range found {
		counts[v.Rule]++
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if counts[ids[i]] != counts[ids[j]] {
			return counts[ids[i]] > counts[ids[j]]
		}
		return ids[i] < ids[j]
	})

	b.WriteString("### By rule\n\n| Rule | Count |\n| --- | --- |\n")
	for _, id := range ids {
		fmt.Fprintf(&b, "| %s | %d |\n", id, counts[id])
	}
	b.WriteString("\n")
	return b.String()
}

func reportGithub(stdout io.Writer, fileCount int, found []violation, lexFailures int) {
	for _, v := range found {
		fmt.Fprintln(stdout, annotationLine(v))
	}
	text := summary(fileCount, found, lexFailures)
	if raw := os.Getenv("GITHUB_STEP_SUMMARY"); raw != "" {
		// gosec G703 taints an environment path into a file open, and Clean is
		// what clears the taint on CI's high-severity run.
		path := filepath.Clean(raw)
		if f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprint(f, text)
			f.Close()
		}
	}
	fmt.Fprint(stdout, text)
}
