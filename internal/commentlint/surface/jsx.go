package surface

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type JSX struct {
	Path string
}

const (
	JSXCanonical = "CANON"

	canonicalChunk = 200

	esbuildRelPath = "docs-site/node_modules/.bin/esbuild"
)

var ErrNoEsbuild = errors.New("commentlint needs the node esbuild that docs-site installs: run `npm ci` in docs-site")

func (j JSX) Lex(src []byte) (Result, error) {
	// No hand lexer can read JSX, because JSX text makes `//` literal (§5.3).
	form, err := esbuildCanonical(j.Path, src, "jsx")
	if err != nil {
		return Result{}, err
	}
	skeleton := canonicalTokens(form)
	// esbuild strips every comment, so a line scan pins the directives back (§2.3).
	skeleton = append(skeleton, jsxDirectiveTokens(src)...)
	// esbuild reports no comment range, so `verify` alone gates `.jsx` (§5.3, ruling 15).
	return Result{Skeleton: skeleton}, nil
}

func canonicalTokens(form []byte) []Token {
	var out []Token
	for _, line := range strings.Split(strings.ReplaceAll(string(form), "\r\n", "\n"), "\n") {
		// A chunk bounds the run a `verify` failure prints. Its line stays
		// zero, because the canonical form has no line of the source file.
		for len(line) > canonicalChunk {
			out = append(out, Token{Kind: JSXCanonical, Text: line[:canonicalChunk]})
			line = line[canonicalChunk:]
		}
		if line != "" {
			out = append(out, Token{Kind: JSXCanonical, Text: line})
		}
	}
	return out
}

func jsxDirectiveTokens(src []byte) []Token {
	var out []Token
	for i, line := range strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n") {
		for at := 0; at < len(line)-1; at++ {
			if line[at] != '/' || (line[at+1] != '/' && line[at+1] != '*') {
				continue
			}
			rest := line[at:]
			if jsDirective(rest) {
				out = append(out, Token{Kind: JSDirective, Text: strings.TrimSpace(rest), Line: i + 1})
			}
		}
	}
	return out
}

func esbuildCanonical(name string, src []byte, loader string) ([]byte, error) {
	bin, err := findEsbuild(name)
	if err != nil {
		return nil, err
	}
	// esbuild's plain transform is not comment-blind, so the minify flag is required (#1141).
	cmd := exec.Command(bin, "--loader="+loader, "--minify-whitespace")
	cmd.Stdin = bytes.NewReader(src)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("esbuild: %s", firstLine(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func findEsbuild(name string) (string, error) {
	dir, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	for dir = filepath.Dir(dir); ; {
		candidate := filepath.Join(dir, filepath.FromSlash(esbuildRelPath))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoEsbuild
		}
		dir = parent
	}
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return strings.TrimSpace(text[:i])
	}
	if text == "" {
		return "the transform failed and printed nothing"
	}
	return text
}
