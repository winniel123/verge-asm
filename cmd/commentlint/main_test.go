package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no subcommand", nil, 2},
		{"an unknown subcommand", []string{"lint"}, 2},
		{"verify without --base", []string{"verify", "internal/p/a.go"}, 2},
		{"verify with an unknown flag", []string{"verify", "--bogus"}, 2},
		{"verify over an out-of-scope path", []string{"verify", "--base", "HEAD", "docs/spec/comment-policy.md"}, 0},
		{"verify over a refused path", []string{"verify", "--base", "HEAD", "design-system/preview/index.html"}, 2},
		{"--in-scope-only drops the refused path", []string{"verify", "--base", "HEAD", "--in-scope-only", "design-system/preview/index.html"}, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if got := run(c.args, &stdout, &stderr); got != c.want {
				t.Errorf("exit is %d, want %d (stdout %q, stderr %q)", got, c.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestUnknownSubcommandNamesItself(t *testing.T) {
	var stdout, stderr bytes.Buffer
	run([]string{"strip"}, &stdout, &stderr)
	if !strings.Contains(stderr.String(), `unknown subcommand "strip"`) {
		t.Errorf("stderr is %q, want it to name the subcommand", stderr.String())
	}
}
