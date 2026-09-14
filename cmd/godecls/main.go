// Command godecls emits the top-level declaration inventory the citations gate reads
// (SPEC docs/spec/citation-anchors.md §7.3).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
)

type entry struct {
	Names []string `json:"names"`
	Error string   `json:"error,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			usage(stderr)
			return 2
		}
	}
	paths := args
	if len(paths) == 0 {
		var err error
		if paths, err = readLines(stdin); err != nil {
			fmt.Fprintf(stderr, "godecls: %v\n", err)
			return 2
		}
	}
	out := make(map[string]entry, len(paths))
	for _, p := range paths {
		names, err := inventory(p)
		if err != nil {
			out[p] = entry{Error: err.Error()}
			continue
		}
		out[p] = entry{Names: names}
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
		"  godecls <paths...>\n"+
		"  godecls            # one path per line on stdin\n")
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

func inventory(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	names := []string{}
	add := func(n string) {
		// A blank identifier declares no name a citation can ever reach.
		if n != "" && n != "_" {
			names = append(names, n)
		}
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			add(funcName(d))
		case *ast.GenDecl:
			// A parenthesised group indents its specs, so the walk reads the spec, not the line.
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					add(s.Name.Name)
				case *ast.ValueSpec:
					for _, id := range s.Names {
						add(id.Name)
					}
				}
			}
		}
	}
	return names, nil
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
