package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/winniel123/verge-asm/internal/commentlint/strip"
)

func runStrip(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("strip", flag.ContinueOnError)
	flags.SetOutput(stderr)
	write := flags.Bool("write", false, "rewrite each file and save the manifest")
	manifest := flags.String("manifest", strip.DefaultManifest, "where --write saves the residue manifest")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	named := false
	flags.Visit(func(f *flag.Flag) { named = named || f.Name == "manifest" })
	if named && !*write {
		fmt.Fprint(stderr, "commentlint strip: --manifest needs --write, and a dry run prints the manifest to stdout\n")
		usage(stderr)
		return 2
	}
	paths := flags.Args()
	if err := checkPathArgs(paths); err != nil {
		fmt.Fprintf(stderr, "commentlint strip: %v\n", err)
		return 2
	}
	if len(paths) == 0 {
		fmt.Fprint(stderr, "commentlint strip: name at least one path\n")
		usage(stderr)
		return 2
	}

	var residue []strip.Record
	deleted, changed := 0, 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "commentlint strip: %v\n", err)
			return 2
		}
		res, err := strip.File(p, src)
		if err != nil {
			fmt.Fprintf(stderr, "commentlint strip: %s: %v\n", p, err)
			return 2
		}
		residue = append(residue, res.Residue...)
		deleted += len(res.Deleted)
		if !res.Changed() {
			continue
		}
		changed++
		if !*write {
			continue
		}
		if err := os.WriteFile(p, res.Source, 0o644); err != nil {
			fmt.Fprintf(stderr, "commentlint strip: %v\n", err)
			return 2
		}
	}

	lines, err := manifestLines(residue)
	if err != nil {
		fmt.Fprintf(stderr, "commentlint strip: %v\n", err)
		return 2
	}
	if *write {
		if err := saveManifest(*manifest, lines); err != nil {
			fmt.Fprintf(stderr, "commentlint strip: %v\n", err)
			return 2
		}
	} else {
		stdout.Write(lines)
	}

	if *write {
		// A later run replaces the manifest, so a package-by-package sweep
		// names its own --manifest path. Printing it keeps that visible.
		fmt.Fprintf(stdout, "commentlint strip: deleted %d block(s) in %d file(s), %d block(s) of residue in %s\n",
			deleted, changed, len(residue), *manifest)
		return 0
	}
	fmt.Fprintf(stdout, "commentlint strip: would delete %d block(s) in %d file(s), %d block(s) of residue\n",
		deleted, changed, len(residue))
	return 0
}

func manifestLines(residue []strip.Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, r := range residue {
		if err := enc.Encode(r); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func saveManifest(path string, lines []byte) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, lines, 0o644)
}
