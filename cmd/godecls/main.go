// Command godecls emits the top-level declaration inventory the citations gate reads
// (SPEC docs/spec/citation-anchors.md §7.3).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
)

type entry struct {
	Names []string            `json:"names"`
	Spans map[string][][2]int `json:"spans,omitempty"`
	Error string              `json:"error,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("godecls", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { usage(stderr) }
	dir := fs.String("root", ".", "the directory every path resolves against")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		var err error
		if paths, err = readLines(stdin); err != nil {
			fmt.Fprintf(stderr, "godecls: %v\n", err)
			return 2
		}
	}
	// os.Root refuses a path that leaves the root, symlinks included, so no argument
	// reaches a file outside the tree this command inventories.
	root, err := os.OpenRoot(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "godecls: %v\n", err)
		return 2
	}
	defer root.Close()

	out := make(map[string]entry, len(paths))
	for _, p := range paths {
		names, spans, err := inventory(root, p)
		if err != nil {
			out[p] = entry{Error: err.Error()}
			continue
		}
		out[p] = entry{Names: names, Spans: spans}
	}
	body, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(stderr, "godecls: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "%s\n", body)
	return 0
}

func usage(w io.Writer) {
	fmt.Fprint(w, "usage:\n"+
		"  godecls [--root DIR] <paths...>\n"+
		"  godecls [--root DIR]              # one path per line on stdin\n"+
		"\nEvery path is read under --root, which defaults to the working directory.\n")
}

func readLines(r io.Reader) ([]string, error) {
	var lines []string
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for s.Scan() {
		if line := s.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, s.Err()
}

func inventory(root *os.Root, path string) ([]string, map[string][][2]int, error) {
	src, err := root.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	mode := parser.SkipObjectResolution | parser.ParseComments
	file, err := parser.ParseFile(fset, path, src, mode)
	if err != nil {
		return nil, nil, err
	}
	names := []string{}
	spans := map[string][][2]int{}
	add := func(n string, node ast.Node, doc *ast.CommentGroup) {
		// A blank identifier declares no name a citation can ever reach.
		if n == "" || n == "_" {
			return
		}
		names = append(names, n)
		// The region is what a disambiguating snippet must sit inside (SPEC §3.4).
		start := fset.Position(node.Pos()).Line
		// A doc comment reads as part of the declaration, and a false red is the fatal direction.
		if doc != nil {
			start = fset.Position(doc.Pos()).Line
		}
		end := fset.Position(node.End()).Line
		spans[n] = append(spans[n], [2]int{start, end})
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			add(funcName(d), d, d.Doc)
		case *ast.GenDecl:
			// A parenthesised group indents its specs, so the walk reads the spec, not the line.
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					add(s.Name.Name, s, specDoc(d, s.Doc))
				case *ast.ValueSpec:
					for _, id := range s.Names {
						add(id.Name, s, specDoc(d, s.Doc))
					}
				}
			}
		}
	}
	return names, spans, nil
}

func specDoc(d *ast.GenDecl, own *ast.CommentGroup) *ast.CommentGroup {
	if own != nil {
		return own
	}
	// An unparenthesised declaration carries its doc on the GenDecl, never on the spec.
	if d.Lparen == token.NoPos {
		return d.Doc
	}
	return nil
}

func funcName(d *ast.FuncDecl) string {
	if d.Name == nil {
		return ""
	}
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	recv := receiverName(d.Recv.List[0].Type)
	if recv == "" {
		return ""
	}
	// SPEC §3.3 rule 1 spells a method `Receiver.Method`, with no star and no parentheses.
	return recv + "." + d.Name.Name
}

func receiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverName(t.X)
	case *ast.IndexExpr:
		return receiverName(t.X)
	case *ast.IndexListExpr:
		return receiverName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
