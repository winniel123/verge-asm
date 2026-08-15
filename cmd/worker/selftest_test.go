package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/winniel123/verge-asm/internal/wire"
)

func buildProberForTest(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "prober")

	cmd := exec.Command("go", "build", "-o", out, "github.com/winniel123/verge-asm/cmd/prober")
	cmd.Env = os.Environ()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build prober: %v\n%s", err, output)
	}
	return out
}

func TestExecProbeRoundTrip(t *testing.T) {
	proberPath := buildProberForTest(t)

	got, err := execProbe(context.Background(), proberPath, wire.JobSpec{Batch: "startup-selftest", Kind: "noop"})
	if err != nil {
		t.Fatalf("execProbe: %v", err)
	}
	if got.Batch != "startup-selftest" || got.Kind != "noop" {
		t.Fatalf("unexpected observation: %+v", got)
	}
}

func TestExecProbeMissingBinary(t *testing.T) {
	_, err := execProbe(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), wire.JobSpec{})
	if err == nil {
		t.Fatal("expected an error execing a missing binary")
	}
}
