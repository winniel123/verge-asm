package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/winniel123/verge-asm/internal/commentlint/verify"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "commentlint: unknown subcommand %q\n", args[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, "usage: commentlint verify --base <ref> [--in-scope-only] [paths...]\n")
}

func runVerify(args []string, stdout, stderr io.Writer) int {
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

	report := verify.Run(fs.Args(), *inScopeOnly, gitShow(*base), os.ReadFile)
	for _, f := range report.Findings {
		if f.Status == verify.Clean || f.Status == verify.Skipped {
			continue
		}
		fmt.Fprintf(stdout, "%s: %s: %s\n", f.Path, f.Status, f.Detail)
	}
	fmt.Fprintf(stdout, "commentlint verify: %d clean, %d changed, %d lex failed, %d refused\n",
		report.Count(verify.Clean), report.Count(verify.Changed),
		report.Count(verify.LexFailed), report.Count(verify.Refused))
	return report.Exit()
}

func gitShow(base string) verify.ReadFunc {
	return func(name string) ([]byte, error) {
		out, err := exec.Command("git", "show", base+":"+name).Output()
		if err != nil {
			return nil, fmt.Errorf("git show %s:%s: %w", base, name, err)
		}
		return out, nil
	}
}
