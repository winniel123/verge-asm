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

// JSX reads a `.jsx` file through esbuild's canonical form. No hand lexer can
// read JSX, because JSX text makes `//` literal (SPEC §5.3).
type JSX struct {
	Path string
}

const (
	JSXCanonical = "CANON"

	// A canonical-form token spans at most this many bytes, so a `verify`
	// failure names a readable run rather than the whole file.
	canonicalChunk = 200

	// esbuild ships with docs-site's dependency tree, so the sweep needs no Go
	// dependency for it (SPEC §6.1).
	esbuildRelPath = "docs-site/node_modules/.bin/esbuild"
)

// ErrNoEsbuild says the tool cannot judge a `.jsx` file here. §6.7 fails
// `verify` closed on it rather than reporting the file clean.
var ErrNoEsbuild = errors.New("commentlint needs the node esbuild that docs-site installs: run `npm ci` in docs-site")

func (j JSX) Lex(src []byte) (Result, error) {
	form, err := esbuildCanonical(j.Path, src, "jsx")
	if err != nil {
		return Result{}, err
	}
	skeleton := canonicalTokens(form)
	// esbuild strips every comment, so a deleted `eslint` directive would
	// leave the canonical form untouched. A line scan pins those back (§2.3).
	skeleton = append(skeleton, jsxDirectiveTokens(src)...)
	// esbuild reports no comment range, so the JSX blocks are empty and every
	// §3.6 JSX cell already reads `agent`.
	return Result{Skeleton: skeleton}, nil
}

// canonicalTokens chunks the canonical form. A token's line is its ordinal in
// that form, which is not a line of the source file.
func canonicalTokens(form []byte) []Token {
	var out []Token
	for _, line := range strings.Split(strings.ReplaceAll(string(form), "\r\n", "\n"), "\n") {
		for len(line) > canonicalChunk {
			out = append(out, Token{Kind: JSXCanonical, Text: line[:canonicalChunk], Line: len(out) + 1})
			line = line[canonicalChunk:]
		}
		if line != "" {
			out = append(out, Token{Kind: JSXCanonical, Text: line, Line: len(out) + 1})
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
	// esbuild keeps a comment that opens an object property or an array item,
	// so the plain transform is not comment-blind. --minify-whitespace drops
	// every comment and keeps every identifier (measured 2026-09-03, #1141).
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
