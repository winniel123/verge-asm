package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const columnSample = `package p

func f() int {
	// the upstream page size is capped because a wider request drops the cursor and the caller recovers nothing
	// a short reason clause, because the cap holds
	return 1
}
`

func columnSampleFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(path, []byte(columnSample), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func runLintOut(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runLint(args, &stdout, &stderr)
	if stderr.Len() > 0 {
		t.Fatalf("stderr holds %q", stderr.String())
	}
	return stdout.String(), code
}

func TestLintDefaultPathReportsNoColumns(t *testing.T) {
	out, code := runLintOut(t, columnSampleFile(t))
	if code != 0 {
		t.Errorf("got exit %d, want 0", code)
	}
	if strings.Contains(out, columnTag) {
		t.Errorf("the default path printed a column report:\n%s", out)
	}
	if !strings.Contains(out, "0 violation(s)") {
		t.Errorf("got %q, want zero violations", out)
	}
}

func TestLintColumnReportNamesTheOverCapBlock(t *testing.T) {
	path := columnSampleFile(t)
	out, code := runLintOut(t, "-column-report", path)
	if code != 0 {
		t.Errorf("got exit %d, want 0: the report never changes the exit code", code)
	}
	want := columnTag + " " + path + ":4 112 why-note"
	if !strings.Contains(out, want) {
		t.Errorf("got %q, want a line %q", out, want)
	}
	if !strings.Contains(out, columnTag+": 1 block(s) over 100 columns") {
		t.Errorf("got %q, want a one-block total", out)
	}
	if !strings.Contains(out, columnTag+" by class: why-note 1") {
		t.Errorf("got %q, want a class breakdown", out)
	}
}

func TestLintColumnReportSkipsADirective(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skipped.go")
	src := "package p\n\n" +
		"//go:generate stringer -type=Kind -output=kind_string.go -linecomment -trimprefix=Kind -tags=none,more\n" +
		"type Kind int\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, code := runLintOut(t, "-column-report", path)
	if code != 0 {
		t.Errorf("got exit %d, want 0", code)
	}
	if !strings.Contains(out, columnTag+": 0 block(s) over 100 columns") {
		t.Errorf("got %q, want no over-cap block: a directive's columns are the tool's, not a comment's", out)
	}
}

func TestLintColumnReportNamesAnUnjudgedClass(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unjudged.go")
	src := "package p\n\nfunc f() {\n" +
		"\t// The console renders the table and then the footer and then the pager, in exactly that stated order.\n" +
		"\t_ = 1\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, code := runLintOut(t, "-column-report", path)
	if code != 0 {
		t.Errorf("got exit %d, want 0", code)
	}
	if !strings.Contains(out, "106 prose-other") {
		t.Errorf("got %q, want the prose-other block named: the cap binds a class no rule judges", out)
	}
}

func TestLintWalksADirectoryArgument(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(columnSample), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, code := runLintOut(t, "-column-report", dir)
	if code != 0 {
		t.Errorf("got exit %d, want 0", code)
	}
	if !strings.Contains(out, "across 1 file(s)") {
		t.Errorf("got %q, want the directory walked: lexing one as a file reported a false zero", out)
	}
	if !strings.Contains(out, columnTag+": 1 block(s) over 100 columns") {
		t.Errorf("got %q, want the walked file's over-cap block", out)
	}
}

func TestLintWalksAnAbsoluteDirectoryWithoutEscapingScope(t *testing.T) {
	abs, err := filepath.Abs("internal/db")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	var relOut, absOut bytes.Buffer
	relCode := runLint([]string{"-column-report", "internal/db"}, &relOut, &bytes.Buffer{})
	absCode := runLint([]string{"-column-report", abs}, &absOut, &bytes.Buffer{})
	if relCode != absCode {
		t.Errorf("got exit %d absolute and %d relative, want the same: sqlc output is out of scope", absCode, relCode)
	}
}

func TestLintRejectsADirectoryHoldingNoInScopeFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runLint([]string{"-column-report", t.TempDir()}, &stdout, &stderr); code != 2 {
		t.Errorf("got exit %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "holds no in-scope file") {
		t.Errorf("got %q, want the empty directory named: a silent zero reads as a clean bill", stderr.String())
	}
}
