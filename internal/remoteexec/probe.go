package remoteexec

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

// Commands are constants and the spec travels on stdin, never argv or operator input (ADR-0001).

const (
	cmdUnameS     = "uname -s"
	cmdUnameM     = "uname -m"
	cmdReadEgress = "printenv SSH_CLIENT"
)

type Facts struct {
	Platform   Platform
	Egress     string
	HasEgress  bool
	Dialled    string
	HasDialled bool
}

func Inspect(ctx context.Context, conn Conn) (Facts, error) {
	plat, err := remotePlatform(ctx, conn)
	if err != nil {
		return Facts{}, err
	}
	f := Facts{Platform: plat}
	if out, err := conn.Output(ctx, cmdReadEgress); err == nil {
		if egress, ok := parseEgress(string(out)); ok {
			f.Egress, f.HasEgress = egress, true
		}
	}
	// Observed locally at connect with no remote command: dialled is known by construction (#710).
	if dialled, ok := normalizeDialled(conn.RemoteAddr()); ok {
		f.Dialled, f.HasDialled = dialled, true
	}
	return f, nil
}

func remotePlatform(ctx context.Context, conn Conn) (Platform, error) {
	unameS, err := conn.Output(ctx, cmdUnameS)
	if err != nil {
		return Platform{}, fmt.Errorf("remoteexec: uname -s: %w", err)
	}
	unameM, err := conn.Output(ctx, cmdUnameM)
	if err != nil {
		return Platform{}, fmt.Errorf("remoteexec: uname -m: %w", err)
	}
	return parsePlatform(string(unameS), string(unameM))
}

func Probe(ctx context.Context, conn Conn, binaries BinaryProvider, spec wire.JobSpec) (wire.ProbeResult, error) {
	plat, err := remotePlatform(ctx, conn)
	if err != nil {
		return wire.ProbeResult{}, err
	}

	// The uname check gates the push, so a mismatched binary is never streamed (ADR-0103).
	bin, err := binaries.Binary(plat.GOOS, plat.GOARCH)
	if err != nil {
		return wire.ProbeResult{}, fmt.Errorf("remoteexec: select binary for %s/%s: %w", plat.GOOS, plat.GOARCH, err)
	}
	defer bin.Close()

	path, err := tempPath()
	if err != nil {
		return wire.ProbeResult{}, err
	}

	// The pushed binary is the instance's own cmd/prober, carrying measure.ProbeUserAgent (P0.11).
	if _, err := conn.Run(ctx, "cat > "+path+" && chmod 0700 "+path, bin, io.Discard, io.Discard); err != nil {
		// The push predates the measured exec, so a pre-exec failure carries no transcript (#867).
		return wire.ProbeResult{}, fmt.Errorf("remoteexec: push binary: %w", err)
	}
	defer func() { _, _ = conn.Output(ctx, "rm -f "+path) }()

	var stdin bytes.Buffer
	if err := wire.EncodeJobSpec(&stdin, spec); err != nil {
		return wire.ProbeResult{}, err
	}
	// Copied before Run drains the buffer, or the transcript's sent scope is empty (spec §3).
	sent := append([]byte(nil), stdin.Bytes()...)
	// A hostile or compromised prober must not OOM the worker, so its stdout is capped (#772).
	stdout := wire.NewLimitedBuffer(wire.MaxProberStdout)
	var stderr bytes.Buffer

	start := time.Now()
	exit, runErr := conn.Run(ctx, path, &stdin, stdout, &stderr)
	dur := time.Since(start)

	// The worker stamps the job id and applies the head and tail store caps at persist (spec §3.2).
	t := &wire.ProberTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: spec.Kind, Duration: dur},
		SentScope:       sent,
		Stdout:          stdout.Bytes(),
		Stderr:          stderr.Bytes(),
		Outcome:         proberOutcome(exit),
		StdoutOverflow:  stdout.Overflowed(),
	}

	if runErr != nil {
		return wire.ProbeResult{Transcript: t}, fmt.Errorf("remoteexec: exec prober: %w", runErr)
	}

	sc := wire.NewObservationScanner(bytes.NewReader(stdout.Bytes()))
	var obs []wire.Observation
	for sc.Next() {
		obs = append(obs, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		return wire.ProbeResult{Transcript: t}, fmt.Errorf("remoteexec: decode prober output: %w", err)
	}
	return wire.ProbeResult{Observations: obs, Transcript: t}, nil
}

func proberOutcome(exit ExitResult) wire.ProberOutcome {
	switch exit.Kind {
	case ExitContextCancelled:
		return wire.ProberContextCancelled{}
	case ExitSignalled:
		return wire.ProberSignalled{Signal: exit.Signal}
	default:
		return wire.ProberExited{Code: exit.Code}
	}
}

func tempPath() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("remoteexec: temp name: %w", err)
	}
	return "/tmp/verge-prober-" + hex.EncodeToString(b[:]), nil
}
