package remoteexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure"
	"github.com/winniel123/verge-asm/internal/wire"
)

type fakeConn struct {
	outputs    map[string]string
	execStdout string
	execStderr string
	execExit   ExitResult
	execErr    error
	pushErr    error
	remoteAddr net.Addr

	ran      []string
	outCmds  []string
	pushed   *bytes.Buffer
	pushedIn *bytes.Buffer
	execPath string
}

func (c *fakeConn) Output(_ context.Context, cmd string) ([]byte, error) {
	c.outCmds = append(c.outCmds, cmd)
	if v, ok := c.outputs[cmd]; ok {
		return []byte(v), nil
	}
	if strings.HasPrefix(cmd, "rm -f ") {
		return nil, nil
	}
	return nil, errors.New("fakeConn: no output for " + cmd)
}

func (c *fakeConn) Run(_ context.Context, cmd string, stdin io.Reader, stdout, stderr io.Writer) (ExitResult, error) {
	c.ran = append(c.ran, cmd)
	switch {
	case strings.HasPrefix(cmd, "cat > "):
		if c.pushErr != nil {
			return ExitResult{Kind: ExitExited, Code: -1}, c.pushErr
		}
		c.pushed = &bytes.Buffer{}
		_, _ = io.Copy(c.pushed, stdin)
		return ExitResult{Kind: ExitExited, Code: 0}, nil
	default:
		c.execPath = cmd
		c.pushedIn = &bytes.Buffer{}
		_, _ = io.Copy(c.pushedIn, stdin)
		_, _ = io.WriteString(stdout, c.execStdout)
		if stderr != nil {
			_, _ = io.WriteString(stderr, c.execStderr)
		}
		return c.execExit, c.execErr
	}
}

func (c *fakeConn) RemoteAddr() net.Addr { return c.remoteAddr }

func (c *fakeConn) Close() error { return nil }

type staticBinaries struct {
	goos, goarch string
	body         string
}

func (b staticBinaries) Binary(goos, goarch string) (io.ReadCloser, error) {
	if goos == b.goos && goarch == b.goarch {
		return io.NopCloser(strings.NewReader(b.body)), nil
	}
	return nil, ErrNoBinary
}

func linuxAmd64Conn(execStdout string) *fakeConn {
	return &fakeConn{
		outputs:    map[string]string{cmdUnameS: "Linux\n", cmdUnameM: "x86_64\n"},
		execStdout: execStdout,
	}
}

func TestProbePushesMatchingBinaryAndExecs(t *testing.T) {
	obsLine, err := oneObservationJSON(wire.Observation{Batch: "b1", Kind: "connect-outcome"})
	if err != nil {
		t.Fatal(err)
	}
	conn := linuxAmd64Conn(obsLine)
	bins := staticBinaries{goos: "linux", goarch: "amd64", body: "ELF-AMD64-BINARY"}

	res, err := Probe(context.Background(), conn, bins, wire.JobSpec{Batch: "b1", Kind: "connect-outcome"})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	obs := res.Observations
	if len(obs) != 1 || obs[0].Batch != "b1" || obs[0].Kind != "connect-outcome" {
		t.Fatalf("observations = %+v, want one b1/connect-outcome", obs)
	}
	// The success path carries a transcript too, not only the failure paths (#867).
	tr, ok := res.Transcript.(*wire.ProberTranscript)
	if !ok {
		t.Fatalf("transcript = %T, want *wire.ProberTranscript", res.Transcript)
	}
	if tr.Kind != "connect-outcome" {
		t.Errorf("transcript kind = %q, want connect-outcome", tr.Kind)
	}
	if string(tr.Stdout) != obsLine {
		t.Errorf("transcript stdout = %q, want the NDJSON the prober wrote %q", tr.Stdout, obsLine)
	}
	if conn.pushedIn == nil || !bytes.Equal(tr.SentScope, conn.pushedIn.Bytes()) {
		t.Errorf("sent scope = %q, want the bytes drained on stdin", tr.SentScope)
	}
	if got, ok := tr.Outcome.(wire.ProberExited); !ok || got.Code != 0 {
		t.Errorf("outcome = %+v, want exited(0)", tr.Outcome)
	}
	if conn.pushed == nil || conn.pushed.String() != "ELF-AMD64-BINARY" {
		t.Errorf("pushed binary = %q, want the amd64 body", conn.pushed)
	}
	if len(conn.ran) != 2 || !strings.HasPrefix(conn.ran[0], "cat > /tmp/verge-prober-") {
		t.Fatalf("run sequence = %v, want push then exec", conn.ran)
	}
	if conn.execPath != strings.TrimPrefix(strings.SplitN(conn.ran[0], " && ", 2)[0], "cat > ") {
		t.Errorf("exec path %q does not match the pushed path in %q", conn.execPath, conn.ran[0])
	}
	if !containsPrefix(conn.outCmds, "rm -f /tmp/verge-prober-") {
		t.Errorf("pushed binary was not cleaned up: %v", conn.outCmds)
	}
}

func TestProbeCapturesTranscriptOnExecError(t *testing.T) {
	// Raw output is highest-value exactly when the job failed (#867, spec §3).
	conn := linuxAmd64Conn("")
	conn.execStderr = "panic: prober boom\n"
	conn.execExit = ExitResult{Kind: ExitExited, Code: 3}
	conn.execErr = errors.New("ssh: exit status 3")
	bins := staticBinaries{goos: "linux", goarch: "amd64", body: "ELF"}

	res, err := Probe(context.Background(), conn, bins, wire.JobSpec{Batch: "b1", Kind: "connect-outcome"})
	if err == nil {
		t.Fatal("Probe returned nil error for a failed exec")
	}
	tr, ok := res.Transcript.(*wire.ProberTranscript)
	if !ok {
		t.Fatalf("transcript = %T, want *wire.ProberTranscript on the error path", res.Transcript)
	}
	if string(tr.Stderr) != "panic: prober boom\n" {
		t.Errorf("transcript stderr = %q, want the captured crash text", tr.Stderr)
	}
	if got, ok := tr.Outcome.(wire.ProberExited); !ok || got.Code != 3 {
		t.Errorf("outcome = %+v, want exited(3)", tr.Outcome)
	}
	if len(res.Observations) != 0 {
		t.Errorf("observations = %+v, want none on the error path", res.Observations)
	}
}

func TestProbeCapturesTranscriptOnDecodeFailure(t *testing.T) {
	const garbage = "this is not valid ndjson\n"
	conn := linuxAmd64Conn(garbage)
	bins := staticBinaries{goos: "linux", goarch: "amd64", body: "ELF"}

	res, err := Probe(context.Background(), conn, bins, wire.JobSpec{Batch: "b1", Kind: "connect-outcome"})
	if err == nil {
		t.Fatal("Probe returned nil error for undecodable output")
	}
	tr, ok := res.Transcript.(*wire.ProberTranscript)
	if !ok {
		t.Fatalf("transcript = %T, want *wire.ProberTranscript on decode failure", res.Transcript)
	}
	if string(tr.Stdout) != garbage {
		t.Errorf("transcript stdout = %q, want the verbatim undecodable output", tr.Stdout)
	}
}

func TestProbeNoTranscriptWhenPushFails(t *testing.T) {
	conn := linuxAmd64Conn("")
	conn.pushErr = errors.New("cat: disk full")
	bins := staticBinaries{goos: "linux", goarch: "amd64", body: "ELF"}

	res, err := Probe(context.Background(), conn, bins, wire.JobSpec{Batch: "b1", Kind: "connect-outcome"})
	if err == nil {
		t.Fatal("Probe returned nil error for a failed push")
	}
	if res.Transcript != nil {
		t.Errorf("transcript = %+v, want nil when the push failed before any exec", res.Transcript)
	}
}

func TestProberOutcomeMapping(t *testing.T) {
	cases := []struct {
		name string
		exit ExitResult
		want wire.ProberOutcome
	}{
		{"exited", ExitResult{Kind: ExitExited, Code: 7}, wire.ProberExited{Code: 7}},
		{"signalled", ExitResult{Kind: ExitSignalled, Signal: "KILL"}, wire.ProberSignalled{Signal: "KILL"}},
		{"context-cancelled", ExitResult{Kind: ExitContextCancelled}, wire.ProberContextCancelled{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := proberOutcome(tc.exit); got != tc.want {
				t.Errorf("proberOutcome(%+v) = %+v, want %+v", tc.exit, got, tc.want)
			}
		})
	}
}

func TestClassifyExit(t *testing.T) {
	// The ssh error types have unexported fields, so only these two ends are constructible.
	if got := classifyExit(nil); got != (ExitResult{Kind: ExitExited, Code: 0}) {
		t.Errorf("classifyExit(nil) = %+v, want exited(0)", got)
	}
	if got := classifyExit(errors.New("transport reset")); got != (ExitResult{Kind: ExitExited, Code: -1}) {
		t.Errorf("classifyExit(non-ssh) = %+v, want exited(-1)", got)
	}
}

func TestProbeArchCheckRefusesMismatch(t *testing.T) {
	conn := &fakeConn{outputs: map[string]string{cmdUnameS: "Linux\n", cmdUnameM: "aarch64\n"}}
	bins := staticBinaries{goos: "linux", goarch: "amd64", body: "ELF-AMD64-BINARY"}

	_, err := Probe(context.Background(), conn, bins, wire.JobSpec{Batch: "b1", Kind: "connect-outcome"})
	if !errors.Is(err, ErrNoBinary) {
		t.Fatalf("Probe err = %v, want ErrNoBinary", err)
	}
	if conn.pushed != nil {
		t.Error("a binary was pushed despite the arch mismatch")
	}
	for _, c := range conn.ran {
		if strings.HasPrefix(c, "cat > ") {
			t.Errorf("push command %q ran despite the arch mismatch", c)
		}
	}
}

func TestProbeRefusesUnknownPlatform(t *testing.T) {
	conn := &fakeConn{outputs: map[string]string{cmdUnameS: "Darwin\n", cmdUnameM: "x86_64\n"}}
	bins := staticBinaries{goos: "linux", goarch: "amd64", body: "x"}
	if _, err := Probe(context.Background(), conn, bins, wire.JobSpec{Batch: "b", Kind: "k"}); err == nil {
		t.Fatal("Probe accepted an unrecognised platform")
	}
	if conn.pushed != nil {
		t.Error("a binary was pushed for an unrecognised platform")
	}
}

func TestInspectReadsPlatformAndEgress(t *testing.T) {
	conn := &fakeConn{
		outputs: map[string]string{
			cmdUnameS:     "Linux\n",
			cmdUnameM:     "x86_64\n",
			cmdReadEgress: "203.0.113.5 54321 22\n",
		},
		remoteAddr: &net.TCPAddr{IP: net.ParseIP("198.51.100.7"), Port: 22},
	}
	f, err := Inspect(context.Background(), conn)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if f.Platform.Label != "linux · x86_64" {
		t.Errorf("platform label = %q, want %q", f.Platform.Label, "linux · x86_64")
	}
	if f.Platform.GOOS != "linux" || f.Platform.GOARCH != "amd64" {
		t.Errorf("platform go ids = %s/%s, want linux/amd64", f.Platform.GOOS, f.Platform.GOARCH)
	}
	if !f.HasEgress || f.Egress != "203.0.113.5" {
		t.Errorf("egress = %q (has=%v), want 203.0.113.5", f.Egress, f.HasEgress)
	}
	if !f.HasDialled || f.Dialled != "198.51.100.7" {
		t.Errorf("dialled = %q (has=%v), want 198.51.100.7", f.Dialled, f.HasDialled)
	}
}

func TestInspectMissingDialledIsBlank(t *testing.T) {
	conn := &fakeConn{outputs: map[string]string{
		cmdUnameS:     "Linux\n",
		cmdUnameM:     "x86_64\n",
		cmdReadEgress: "203.0.113.5 54321 22\n",
	}}
	f, err := Inspect(context.Background(), conn)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if f.HasDialled || f.Dialled != "" {
		t.Errorf("dialled = %q (has=%v), want blank", f.Dialled, f.HasDialled)
	}
}

func TestInspectMissingEgressIsBlank(t *testing.T) {
	conn := &fakeConn{outputs: map[string]string{
		cmdUnameS:     "Linux\n",
		cmdUnameM:     "aarch64\n",
		cmdReadEgress: "\n",
	}}
	f, err := Inspect(context.Background(), conn)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if f.HasEgress || f.Egress != "" {
		t.Errorf("egress = %q (has=%v), want blank", f.Egress, f.HasEgress)
	}
	if f.Platform.GOARCH != "arm64" {
		t.Errorf("arm64 host mapped to %q", f.Platform.GOARCH)
	}
}

func TestProbeReusesSharedUserAgent(t *testing.T) {
	if measure.ProbeUserAgent != "verge-asm-prober (+https://github.com/winniel123/verge-asm)" {
		t.Fatalf("shared probe UA drifted: %q", measure.ProbeUserAgent)
	}
	if !strings.Contains(measure.ProbeUserAgent, "verge-asm") {
		t.Error("probe UA is not identifiable")
	}
}

func containsPrefix(ss []string, prefix string) bool {
	for _, s := range ss {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func oneObservationJSON(obs wire.Observation) (string, error) {
	var buf bytes.Buffer
	if err := wire.EncodeObservation(&buf, obs); err != nil {
		return "", err
	}
	return buf.String(), nil
}
