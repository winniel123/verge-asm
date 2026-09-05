package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/winniel123/verge-asm/internal/commentlint/rule"
	"github.com/winniel123/verge-asm/internal/commentlint/scope"
	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

type violation struct {
	path string
	rule.Finding
}

func runLint(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("lint", flag.ContinueOnError)
	flags.SetOutput(stderr)
	github := flags.Bool("github", false, "print GitHub Actions annotations and a step summary")
	inScopeOnly := flags.Bool("in-scope-only", false, "drop every path outside the sweep's surfaces")
	columnReport := flags.Bool("column-report", false,
		"list the blocks over SPEC §4.4's column cap; reports only, and never changes the exit code")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	paths := flags.Args()
	if err := checkPathArgs(paths); err != nil {
		fmt.Fprintf(stderr, "commentlint lint: %v\n", err)
		return 2
	}

	files, err := resolveLintFiles(paths, *inScopeOnly)
	if err != nil {
		fmt.Fprintf(stderr, "commentlint lint: %v\n", err)
		return 2
	}

	var found []violation
	var overs []overCap
	lexFailures := 0
	for _, p := range files {
		res, err := lexFile(p)
		var unsupported *surface.UnsupportedError
		// A surface with no lexer is not a lex failure, and §6.7 folds no skip into the count.
		if errors.As(err, &unsupported) {
			continue
		}
		if err != nil {
			lexFailures++
			fmt.Fprintf(stderr, "commentlint lint: %s: %v\n", p, err)
			continue
		}
		for _, f := range rule.Lint(res, strings.HasSuffix(p, "_test.go")) {
			found = append(found, violation{path: p, Finding: f})
		}
		if *columnReport {
			overs = append(overs, overCapBlocks(p, res)...)
		}
	}

	if *github {
		reportGithub(stdout, len(files), found, lexFailures)
	} else {
		for _, v := range found {
			fmt.Fprintf(stdout, "%s:%d -> %s\n", v.path, v.Line, v.Rule)
		}
		fmt.Fprintf(stdout, "commentlint lint: %d violation(s) across %d file(s), %d lex failure(s)\n",
			len(found), len(files), lexFailures)
	}
	if *columnReport {
		reportColumns(stdout, overs)
	}
	if lexFailures > 0 {
		return 2
	}
	if len(found) > 0 {
		return 1
	}
	return 0
}

func resolveLintFiles(paths []string, inScopeOnly bool) ([]string, error) {
	if len(paths) == 0 {
		// An empty changed set lints nothing, because --in-scope-only never walks the tree (§6.3).
		if inScopeOnly {
			return nil, nil
		}
		return walkInScope(".")
	}
	var out []string
	for _, p := range paths {
		// A directory lexes as an unsupported surface, so it used to report a clean zero (#1466).
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			under, err := walkInScope(p)
			if err != nil {
				return nil, err
			}
			if len(under) == 0 {
				return nil, fmt.Errorf("%q holds no in-scope file", p)
			}
			out = append(out, under...)
			continue
		}
		verdict := scope.Classify(p)
		if verdict == scope.InScope {
			out = append(out, p)
			continue
		}
		if !inScopeOnly && verdict == scope.OutOfScope {
			out = append(out, p)
		}
	}
	return out, nil
}

func lexFile(p string) (surface.Result, error) {
	lexer, err := surface.For(p)
	if err != nil {
		return surface.Result{}, err
	}
	src, err := os.ReadFile(p)
	if err != nil {
		return surface.Result{}, err
	}
	return lexer.Lex(src)
}

func walkInScope(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		// A path made relative to a non-root walk cannot be reopened (#1466).
		rel := filepath.ToSlash(p)
		if scope.Classify(rel) == scope.InScope {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func checkPathArgs(paths []string) error {
	for _, p := range paths {
		// Go's flag package stops at the first path, so a later flag is silently ignored (#1133).
		if strings.HasPrefix(p, "-") {
			return fmt.Errorf("%q is a flag, and every flag goes before the first path", p)
		}
	}
	return nil
}
