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
	grouped   = 1
	alsoHere  = 2
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

func write(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func decode(t *testing.T, out string) map[string]entry {
	t.Helper()
	var got map[string]entry
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	return got
}

func invoke(t *testing.T, stdin string, args ...string) (map[string]entry, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	if code != 0 {
		return nil, stderr.String(), code
	}
	return decode(t, stdout.String()), stderr.String(), code
}

func namesOf(t *testing.T, path string) map[string]bool {
	t.Helper()
	got, _, code := invoke(t, "", path)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	e, ok := got[path]
	if !ok {
		t.Fatalf("no entry for %s", path)
	}
	if e.Error != "" {
		t.Fatalf("error for %s: %s", path, e.Error)
	}
	set := map[string]bool{}
	for _, n := range e.Names {
		set[n] = true
	}
	return set
}

func TestInventoryNamesEveryTopLevelDeclaration(t *testing.T) {
	names := namesOf(t, write(t, "hot.go", sample))
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
	names := namesOf(t, write(t, "hot.go", sample))
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
	names := namesOf(t, write(t, "hot.go", sample))
	if names["fabricated"] {
		t.Error("a name inside a raw-string literal is not a declaration")
	}
}

func TestInventoryHoldsNoBlankIdentifierAndNoImport(t *testing.T) {
	names := namesOf(t, write(t, "hot.go", sample))
	if names["_"] {
		t.Error("a blank identifier declares no citable name")
	}
	if names["fmt"] {
		t.Error("an import is not a declaration")
	}
}

func TestUnparseableFileReportsAnError(t *testing.T) {
	path := write(t, "broken.go", "package x\n\nfunc (\n")
	got, _, code := invoke(t, "", path)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got[path].Error == "" {
		t.Fatal("an unparseable file must carry an error")
	}
	if got[path].Names != nil {
		t.Fatal("an unparseable file must carry no names")
	}
}

func TestAbsentFileReportsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.go")
	got, _, code := invoke(t, "", path)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got[path].Error == "" {
		t.Fatal("an absent file must carry an error")
	}
}

func TestDeclarationFreeFileCarriesAnEmptyList(t *testing.T) {
	path := write(t, "empty.go", "package x\n")
	got, _, code := invoke(t, "", path)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got[path].Error != "" {
		t.Fatalf("unexpected error: %s", got[path].Error)
	}
	if got[path].Names == nil || len(got[path].Names) != 0 {
		t.Fatalf("want an empty list, got %#v", got[path].Names)
	}
}

func TestStdinCarriesOnePathPerLine(t *testing.T) {
	a := write(t, "a.go", "package x\n\nfunc Alpha() {}\n")
	b := write(t, "b.go", "package x\n\nfunc Beta() {}\n")
	got, _, code := invoke(t, a+"\n"+b+"\n")
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

func TestHelpExitsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("want 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatal("want a usage line on stderr")
	}
}
