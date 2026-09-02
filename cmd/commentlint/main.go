package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/winniel123/verge-asm/internal/commentlint/verify"
)

type git struct {
	verifyRef func(ref string) error
	show      func(ref, name string) ([]byte, error)
	read      verify.ReadFunc
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWith(args, stdout, stderr, realGit())
}

func runWith(args []string, stdout, stderr io.Writer, g git) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "lint":
		return runLint(args[1:], stdout, stderr)
	case "strip":
		return runStrip(args[1:], stdout, stderr)
	case "sample":
		return runSample(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr, g)
	default:
		fmt.Fprintf(stderr, "commentlint: unknown subcommand %q\n", args[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, "usage:\n"+
		"  commentlint lint   [--github] [--in-scope-only] [paths...]\n"+
		"  commentlint strip  [--write] [--manifest PATH] <paths...>\n"+
		"  commentlint sample [--population production|test] [--round N] [--size N] [paths...]\n"+
		"  commentlint verify --base <ref> [--in-scope-only] <paths...>\n")
}

func runVerify(args []string, stdout, stderr io.Writer, g git) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	base := fs.String("base", "", "ref holding the pre-sweep content")
	inScopeOnly := fs.Bool("in-scope-only", false, "drop every path outside the sweep's surfaces")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *base == "" {
		fmt.Fprint(stderr, "commentlint verify: --base is required\n")
		usage(stderr)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		fmt.Fprint(stderr, "commentlint verify: name at least one path\n")
		usage(stderr)
		return 2
	}
	for _, p := range paths {
		// Go's flag package stops at the first path, so a later flag would
		// reach the tool as a path and switch itself off in silence (#1133).
		if strings.HasPrefix(p, "-") {
			fmt.Fprintf(stderr, "commentlint verify: %q is a flag, and every flag goes before the first path\n", p)
			return 2
		}
	}
	if err := g.verifyRef(*base); err != nil {
		fmt.Fprintf(stderr, "commentlint verify: %v\n", err)
		return 2
	}

	readBase := func(name string) ([]byte, error) { return g.show(*base, name) }
	report := verify.Run(paths, *inScopeOnly, readBase, g.read)
	for _, f := range report.Findings {
		if f.Status == verify.Clean || f.Status == verify.Skipped {
			continue
		}
		fmt.Fprintf(stdout, "%s: %s: %s\n", f.Path, f.Status, f.Detail)
	}
	fmt.Fprintf(stdout, "commentlint verify: %d clean, %d changed, %d lex failed, %d refused, %d skipped\n",
		report.Count(verify.Clean), report.Count(verify.Changed), report.Count(verify.LexFailed),
		report.Count(verify.Refused), report.Count(verify.Skipped))
	return report.Exit()
}

func realGit() git {
	return git{verifyRef: gitVerifyRef, show: gitShow, read: os.ReadFile}
}

func gitVerifyRef(ref string) error {
	if _, err := runGit("rev-parse", "--verify", "--quiet", ref+"^{commit}"); err != nil {
		return fmt.Errorf("--base %s names no commit", ref)
	}
	return nil
}

func gitShow(ref, name string) ([]byte, error) {
	out, err := runGit("show", ref+":"+name)
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", ref, name, err)
	}
	return out, nil
}

func runGit(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return out, nil
}
