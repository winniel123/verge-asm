package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/winniel123/verge-asm/internal/commentlint/rule"
)

const columnTag = rule.RuleColumnOverCap

type overCap struct {
	path    string
	line    int
	columns int
	class   rule.Class
}

func overCapBlocks(path string, findings []rule.Finding) []overCap {
	var out []overCap
	for _, f := range findings {
		// The class is the one predicate, so the report reads it rather than repeating it (#1466).
		if f.Rule != rule.RuleColumnOverCap {
			continue
		}
		out = append(out, overCap{path: path, line: f.Line, columns: f.Columns, class: f.Class})
	}
	return out
}

func reportColumns(stdout io.Writer, overs []overCap) {
	counts := map[rule.Class]int{}
	for _, o := range overs {
		// A repair agent greps this line, so its shape is an interface (SPEC §4.4).
		fmt.Fprintf(stdout, "%s %s:%d %d %s\n", columnTag, o.path, o.line, o.columns, o.class)
		counts[o.class]++
	}
	fmt.Fprintf(stdout, "%s: %d block(s) over %d columns\n", columnTag, len(overs), rule.ColumnCap)
	classes := make([]rule.Class, 0, len(counts))
	for c := range counts {
		classes = append(classes, c)
	}
	sort.Slice(classes, func(i, j int) bool {
		if counts[classes[i]] != counts[classes[j]] {
			return counts[classes[i]] > counts[classes[j]]
		}
		return classes[i] < classes[j]
	})
	for _, c := range classes {
		fmt.Fprintf(stdout, "%s by class: %s %d\n", columnTag, c, counts[c])
	}
}
