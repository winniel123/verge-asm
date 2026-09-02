package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/winniel123/verge-asm/internal/commentlint/sample"
)

func runSample(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sample", flag.ContinueOnError)
	flags.SetOutput(stderr)
	population := flags.String("population", string(sample.Production), "production or test")
	round := flags.Int("round", 1, "which sampling round this sheet records")
	size := flags.Int("size", sample.Size, "how many blocks the gate sample holds")
	supplement := flags.Int("supplement", sample.SupplementCap, "how many blocks cover a missed class")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	paths := flags.Args()
	if err := checkPathArgs(paths); err != nil {
		fmt.Fprintf(stderr, "commentlint sample: %v\n", err)
		return 2
	}
	pop := sample.Population(*population)
	if pop != sample.Production && pop != sample.Test {
		fmt.Fprintf(stderr, "commentlint sample: --population is production or test, not %q\n", *population)
		return 2
	}
	// §3.9 runs at most three rounds, so a fourth is a typo rather than a gate.
	if *round < 1 || *round > 3 {
		fmt.Fprintf(stderr, "commentlint sample: --round is 1, 2 or 3, not %d\n", *round)
		return 2
	}

	files, err := resolveSampleFiles(paths)
	if err != nil {
		fmt.Fprintf(stderr, "commentlint sample: %v\n", err)
		return 2
	}

	var admitted []sample.Item
	read := 0
	for _, p := range files {
		if sample.PopulationOf(p) != pop {
			continue
		}
		read++
		src, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "commentlint sample: %v\n", err)
			return 2
		}
		items, err := sample.Admitted(p, src)
		if err != nil {
			fmt.Fprintf(stderr, "commentlint sample: %s: %v\n", p, err)
			return 2
		}
		admitted = append(admitted, items...)
	}

	seed := sample.Seed(*round, pop)
	drawn := sample.Draw(admitted, *size, seed)
	sheet := sample.Sheet{
		Population: pop,
		Round:      *round,
		Files:      read,
		Admitted:   admitted,
		Drawn:      drawn,
		Extra:      sample.Supplement(admitted, drawn, *supplement, seed),
	}
	if err := sample.Render(stdout, sheet); err != nil {
		fmt.Fprintf(stderr, "commentlint sample: %v\n", err)
		return 2
	}
	return 0
}

func resolveSampleFiles(paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths, err := walkInScope(".")
		if err != nil {
			return nil, err
		}
		return goOnly(paths), nil
	}
	return goOnly(paths), nil
}

func goOnly(paths []string) []string {
	var out []string
	for _, p := range paths {
		if strings.EqualFold(path.Ext(strings.ReplaceAll(p, `\`, "/")), ".go") {
			out = append(out, p)
		}
	}
	return out
}
