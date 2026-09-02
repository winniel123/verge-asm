package strip

import (
	"fmt"
	"go/format"
	"path"
	"strings"

	"github.com/winniel123/verge-asm/internal/commentlint/rule"
	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

const DefaultManifest = ".commentlint/residue.jsonl"

type Record struct {
	File      string     `json:"file"`
	StartLine int        `json:"start_line"`
	EndLine   int        `json:"end_line"`
	Class     rule.Class `json:"class"`
	Signal    string     `json:"signal,omitempty"`
	Trailing  bool       `json:"trailing,omitempty"`
}

type Result struct {
	Source  []byte
	Deleted []Record
	Residue []Record
}

func (r Result) Changed() bool {
	return len(r.Deleted) > 0
}

func File(name string, src []byte) (Result, error) {
	// A strip that silently does nothing on a non-Go path is how a sweep agent
	// believes a slice is done (SPEC §6.4).
	if !strings.EqualFold(path.Ext(strings.ReplaceAll(name, `\`, "/")), ".go") {
		return Result{}, &UsageError{Path: name}
	}
	res, err := surface.Go{}.Lex(src)
	if err != nil {
		return Result{}, err
	}

	var out Result
	var cut []surface.Block
	for _, b := range res.Blocks {
		class, signal, deletable := rule.Deletable(b)
		rec := Record{File: name, StartLine: b.StartLine, EndLine: b.EndLine, Class: class, Signal: signal}
		if class == rule.Directive || class == rule.GeneratedHeader {
			continue
		}
		if !deletable {
			out.Residue = append(out.Residue, rec)
			continue
		}
		out.Deleted = append(out.Deleted, rec)
		cut = append(cut, b)
	}
	for _, b := range res.Trailing {
		class, signal, _ := rule.Deletable(b)
		if class == rule.Directive || class == rule.GeneratedHeader {
			continue
		}
		out.Residue = append(out.Residue, Record{
			File: name, StartLine: b.StartLine, EndLine: b.EndLine,
			Class: class, Signal: signal, Trailing: true,
		})
	}

	if len(cut) == 0 {
		out.Source = src
		return out, nil
	}
	formatted, err := format.Source(remove(src, cut))
	if err != nil {
		return Result{}, fmt.Errorf("gofmt after the delete pass: %w", err)
	}
	out.Source = formatted
	return out, nil
}

type UsageError struct {
	Path string
}

func (e *UsageError) Error() string {
	return fmt.Sprintf("strip deletes on the Go surface only, and %s is not Go", e.Path)
}

func remove(src []byte, cut []surface.Block) []byte {
	lines, endings := splitLines(string(src))
	dead := make([]bool, len(lines)+1)
	for _, b := range cut {
		for n := b.StartLine; n <= b.EndLine && n <= len(lines); n++ {
			dead[n] = true
		}
	}
	// A cut drops a blank line only where it would otherwise leave two
	// consecutive blank lines (SPEC §3.8).
	for _, b := range cut {
		before := previousLive(lines, dead, b.StartLine)
		after := nextLive(lines, dead, b.EndLine)
		if before == 0 || after == 0 {
			continue
		}
		if blank(lines[before-1]) && blank(lines[after-1]) {
			dead[after] = true
		}
	}

	var out strings.Builder
	for i, line := range lines {
		if dead[i+1] {
			continue
		}
		out.WriteString(line)
		out.WriteString(endings[i])
	}
	return []byte(out.String())
}

func previousLive(lines []string, dead []bool, from int) int {
	for n := from - 1; n >= 1; n-- {
		if !dead[n] {
			return n
		}
	}
	return 0
}

func nextLive(lines []string, dead []bool, from int) int {
	for n := from + 1; n <= len(lines); n++ {
		if !dead[n] {
			return n
		}
	}
	return 0
}

func blank(line string) bool {
	return strings.TrimSpace(line) == ""
}

func splitLines(src string) (lines, endings []string) {
	// .gitattributes checks this tree out as CRLF on Windows, so a cut carries
	// each line's own ending back out.
	for len(src) > 0 {
		i := strings.IndexByte(src, '\n')
		if i < 0 {
			return append(lines, src), append(endings, "")
		}
		line := src[:i]
		ending := "\n"
		if strings.HasSuffix(line, "\r") {
			line = strings.TrimSuffix(line, "\r")
			ending = "\r\n"
		}
		lines = append(lines, line)
		endings = append(endings, ending)
		src = src[i+1:]
	}
	return lines, endings
}
