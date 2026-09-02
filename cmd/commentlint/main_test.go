package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func stubGit(refs map[string]bool, files map[string]string) git {
	return git{
		verifyRef: func(ref string) error {
			if refs[ref] {
				return nil
			}
			return fmt.Errorf("--base %s names no commit", ref)
		},
		show: func(ref, name string) ([]byte, error) {
			src, ok := files[name]
			if !ok {
				return nil, fmt.Errorf("path %s is absent at %s", name, ref)
			}
			return []byte(src), nil
		},
		read: func(name string) ([]byte, error) {
			src, ok := files[name]
			if !ok {
				return nil, fmt.Errorf("path %s is absent", name)
			}
			return []byte(src), nil
		},
	}
}

func TestRunRejectsUsageErrorsBeforeGit(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no subcommand", nil},
		{"an unknown subcommand", []string{"lint"}},
		{"verify without --base", []string{"verify", "internal/p/a.go"}},
		{"verify with an unknown flag", []string{"verify", "--bogus"}},
		{"verify with no path", []string{"verify", "--base", "main"}},
		{"a flag after the first path", []string{"verify", "--base", "main", "a.go", "--in-scope-only"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			g := stubGit(nil, nil)
			g.verifyRef = func(string) error {
				t.Fatal("a usage error reached git")
				return nil
			}
			if got := runWith(c.args, &stdout, &stderr, g); got != 2 {
				t.Errorf("exit is %d, want 2 (stdout %q, stderr %q)", got, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunVerifyExitCodes(t *testing.T) {
	files := map[string]string{
		"internal/p/a.go":                  "package p\n\nfunc F() {}\n",
		"docs/spec/comment-policy.md":      "# a\n",
		"design-system/preview/index.html": "<p>a</p>\n",
	}
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"an unknown base ref", []string{"verify", "--base", "bogus", "internal/p/a.go"}, 2},
		{"an unchanged Go file", []string{"verify", "--base", "main", "internal/p/a.go"}, 0},
		{"an out-of-scope path", []string{"verify", "--base", "main", "docs/spec/comment-policy.md"}, 0},
		{"a refused path", []string{"verify", "--base", "main", "design-system/preview/index.html"}, 2},
		{"--in-scope-only drops the refused path", []string{"verify", "--base", "main", "--in-scope-only", "design-system/preview/index.html"}, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			g := stubGit(map[string]bool{"main": true}, files)
			if got := runWith(c.args, &stdout, &stderr, g); got != c.want {
				t.Errorf("exit is %d, want %d (stdout %q, stderr %q)", got, c.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestSummaryReportsTheSkippedCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	files := map[string]string{"docs/a.md": "# a\n"}
	g := stubGit(map[string]bool{"main": true}, files)
	runWith([]string{"verify", "--base", "main", "docs/a.md"}, &stdout, &stderr, g)
	if !strings.Contains(stdout.String(), "1 skipped") {
		t.Errorf("summary is %q, want it to report the skipped count", stdout.String())
	}
}

func TestUnknownSubcommandNamesItself(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runWith([]string{"strip"}, &stdout, &stderr, stubGit(nil, nil))
	if !strings.Contains(stderr.String(), `unknown subcommand "strip"`) {
		t.Errorf("stderr is %q, want it to name the subcommand", stderr.String())
	}
}
