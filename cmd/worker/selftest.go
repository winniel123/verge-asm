package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/winniel123/verge-asm/internal/wire"
)

// execProbe execs the prober binary at proberPath, writes spec to its
// stdin, and decodes the first NDJSON observation it writes back. This is
// the exec-locally half of ADR-0001's job-spec-in/NDJSON-out contract.
func execProbe(ctx context.Context, proberPath string, spec wire.JobSpec) (wire.Observation, error) {
	var stdin bytes.Buffer
	if err := wire.EncodeJobSpec(&stdin, spec); err != nil {
		return wire.Observation{}, err
	}

	cmd := exec.CommandContext(ctx, proberPath)
	cmd.Stdin = &stdin

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return wire.Observation{}, fmt.Errorf("worker: exec prober: %w (stderr: %s)", err, stderr.String())
	}

	sc := wire.NewObservationScanner(&stdout)
	if !sc.Next() {
		if err := sc.Err(); err != nil {
			return wire.Observation{}, fmt.Errorf("worker: decode prober output: %w", err)
		}
		return wire.Observation{}, fmt.Errorf("worker: prober produced no output")
	}
	return sc.Observation(), nil
}
