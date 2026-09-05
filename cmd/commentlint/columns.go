package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/winniel123/verge-asm/internal/commentlint/rule"
	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

const columnTag = "column-over-cap"

type overCap struct {
	path    string
	line    int
	columns int
	class   rule.Class
}

func overCapBlocks(path string, res surface.Result) []overCap {
	var out []overCap
	scan := func(blocks []surface.Block) {
		for _, b := range blocks {
			if b.Columns <= rule.ColumnCap {
				continue
			}
			c := rule.Classify(b)
			if !rule.CapBinds(c) {
				continue
			}
			out = append(out, overCap{path: path, line: b.StartLine, columns: b.Columns, class: c})
		}
	}
	scan(res.Blocks)
	scan(res.Trailing)
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
