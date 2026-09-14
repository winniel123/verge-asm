package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `package hot

import "fmt"

type Queue struct{ n int }

func hotCore() int { return 1 }

func (q *Queue) Run() error { return nil }

func (q Queue) Len() int { return q.n }

const (
	grouped  = 1
	alsoHere = 2
)

var (
	table = map[string]int{}
	_     = fmt.Sprint
)

type (
	Pair struct{ A, B int }
	Trio struct{ A, B, C int }
)

var fixture = ` + "`" + `
package fake

func fabricated() {}
` + "`" + `

func Generic[T any](v T) T { return v }

type Box[T any] struct{ v T }

func (b *Box[T]) Get() T { return b.v }
`

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return name
}

func invoke(t *testing.T, dir, stdin string, args ...string) (map[string]entry, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(append([]string{"--root", dir}, args...), strings.NewReader(stdin), &stdout, &stderr)
	if code != 0 {
		return nil, stderr.String(), code
	}
	var got map[string]entry
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", stdout.String(), err)
	}
	return got, stderr.String(), code
}

func sampleNames(t *testing.T) map[string]bool {
	t.Helper()
	dir := t.TempDir()
	name := write(t, dir, "hot.go", sample)
	got, _, code := invoke(t, dir, "", name)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got[name].Error != "" {
		t.Fatalf("error for %s: %s", name, got[name].Error)
	}
	set := map[string]bool{}
	for _, n := range got[name].Names {
		set[n] = true
	}
	return set
}

func TestInventoryNamesEveryTopLevelDeclaration(t *testing.T) {
	names := sampleNames(t)
	for _, want := range []string{
		"Queue", "hotCore", "grouped", "alsoHere", "table", "Pair", "Trio",
		"fixture", "Generic", "Box",
	} {
		if !names[want] {
			t.Errorf("%q is missing from the inventory", want)
		}
	}
}

func TestInventorySpellsAMethodReceiverDotMethod(t *testing.T) {
	names := sampleNames(t)
	for _, want := range []string{"Queue.Run", "Queue.Len", "Box.Get"} {
		if !names[want] {
			t.Errorf("%q is missing from the inventory", want)
		}
	}
	for _, reject := range []string{"Run", "(*Queue).Run", "Queue.Run()", "*Queue.Run"} {
		if names[reject] {
			t.Errorf("%q must not be in the inventory", reject)
		}
	}
}

func TestInventoryHoldsNoNameFromARawStringLiteral(t *testing.T) {
	if sampleNames(t)["fabricated"] {
		t.Error("a name inside a raw-string literal is not a declaration")
	}
}

func TestInventoryHoldsNoBlankIdentifierAndNoImport(t *testing.T) {
	names := sampleNames(t)
	if names["_"] {
		t.Error("a blank identifier declares no citable name")
	}
	if names["fmt"] {
		t.Error("an import is not a declaration")
	}
}

func TestUnparseableFileReportsAnError(t *testing.T) {
	dir := t.TempDir()
	name := write(t, dir, "broken.go", "package x\n\nfunc (\n")
	got, _, code := invoke(t, dir, "", name)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got[name].Error == "" {
		t.Fatal("an unparseable file must carry an error")
	}
	if got[name].Names != nil {
		t.Fatal("an unparseable file must carry no names")
	}
}

func TestAbsentFileReportsAnError(t *testing.T) {
	got, _, code := invoke(t, t.TempDir(), "", "gone.go")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got["gone.go"].Error == "" {
		t.Fatal("an absent file must carry an error")
	}
}

func TestAPathThatLeavesTheRootReportsAnError(t *testing.T) {
	// os.Root is what keeps an argument from naming a file outside the tree.
	dir := t.TempDir()
	write(t, dir, "hot.go", sample)
	for _, escape := range []string{"../hot.go", filepath.Join(dir, "hot.go")} {
		got, _, code := invoke(t, filepath.Join(dir), "", escape)
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		if got[escape].Error == "" {
			t.Errorf("%q must not resolve", escape)
		}
	}
}

func TestDeclarationFreeFileCarriesAnEmptyList(t *testing.T) {
	dir := t.TempDir()
	name := write(t, dir, "empty.go", "package x\n")
	got, _, code := invoke(t, dir, "", name)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got[name].Error != "" {
		t.Fatalf("unexpected error: %s", got[name].Error)
	}
	if got[name].Names == nil || len(got[name].Names) != 0 {
		t.Fatalf("want an empty list, got %#v", got[name].Names)
	}
}

func TestStdinCarriesOnePathPerLine(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a.go", "package x\n\nfunc Alpha() {}\n")
	b := write(t, dir, "b.go", "package x\n\nfunc Beta() {}\n")
	got, _, code := invoke(t, dir, a+"\n"+b+"\n")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got))
	}
	if got[a].Names[0] != "Alpha" || got[b].Names[0] != "Beta" {
		t.Fatalf("got %#v", got)
	}
}

func TestAnAbsentRootExitsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"--root", filepath.Join(t.TempDir(), "gone"), "a.go"}
	if code := run(args, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("want 2, got %d", code)
	}
}

func TestHelpExitsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("want 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatal("want a usage line on stderr")
	}
}
