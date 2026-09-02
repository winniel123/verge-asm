package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSampleSplitsTheTwoPopulations(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "internal/p/a.go", "package p\n\n// F reports the estate.\nfunc F() {}\n")
	writeFile(t, dir, "internal/p/a_test.go", "package p\n\n// TestF reports the estate.\nfunc TestF() {}\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"sample", "--population", "production"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Fatalf("exit is %d, want 0 (stderr %q)", got, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "internal/p/a.go:3") {
		t.Errorf("the production sheet misses the production block:\n%s", out)
	}
	if strings.Contains(out, "a_test.go") {
		t.Errorf("the production sheet reached a test file:\n%s", out)
	}

	stdout.Reset()
	if got := runWith([]string{"sample", "--population", "test"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Fatalf("exit is %d, want 0 (stderr %q)", got, stderr.String())
	}
	out = stdout.String()
	if !strings.Contains(out, "internal/p/a_test.go:3") {
		t.Errorf("the test sheet misses the test block:\n%s", out)
	}
	if strings.Contains(out, "In-scope Go files read: 2") {
		t.Errorf("the test sheet counted the production file:\n%s", out)
	}
}

func TestSampleSkipsAnExcludedTree(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "prototypes/p.go", "package p\n\n// F reports the estate.\nfunc F() {}\n")
	writeFile(t, dir, "internal/db/q.go", "package db\n\n// F reports the estate.\nfunc F() {}\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"sample"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Fatalf("exit is %d, want 0 (stderr %q)", got, stderr.String())
	}
	if !strings.Contains(stdout.String(), "admits for deletion: 0") {
		t.Errorf("an excluded tree reached the sample:\n%s", stdout.String())
	}
}

func TestSampleRejectsAnUnknownPopulation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"sample", "--population", "staging"}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
		t.Errorf("exit is %d, want 2", got)
	}
	if !strings.Contains(stderr.String(), "production or test") {
		t.Errorf("stderr is %q, want it to name both populations", stderr.String())
	}
}

func TestSampleRejectsAFourthRound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"sample", "--round", "4"}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
		t.Errorf("exit is %d, want 2", got)
	}
	if !strings.Contains(stderr.String(), "1, 2 or 3") {
		t.Errorf("stderr is %q, want it to name the round bound", stderr.String())
	}
}

func TestSampleRejectsAFlagAfterAPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"sample", "p.go", "--round", "2"}, &stdout, &stderr, stubGit(nil, nil)); got != 2 {
		t.Errorf("exit is %d, want 2", got)
	}
}

func TestSampleDropsANonGoPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "db/queries/scan.sql", "-- name: X :one\nSELECT 1;\n")

	var stdout, stderr bytes.Buffer
	if got := runWith([]string{"sample", "db/queries/scan.sql"}, &stdout, &stderr, stubGit(nil, nil)); got != 0 {
		t.Fatalf("exit is %d, want 0 (stderr %q)", got, stderr.String())
	}
	if !strings.Contains(stdout.String(), "In-scope Go files read: 0") {
		t.Errorf("a SQL path reached the Go sample:\n%s", stdout.String())
	}
}
