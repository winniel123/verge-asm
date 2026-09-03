// Package remoteexec pushes the measurement binary to a provisioned prober over
// SSH and exec's it there, so an internet Vantage measures from its OWN position
// rather than from the instance host (ADR-0001's "exec'd locally for internal
// measurement and pushed over SSH for external vantages", ADR-0103, #683, P0.8).
//
// The push/exec/observe logic is written against the narrow Conn seam below rather
// than *ssh.Client, so every rule — the uname arch check that gates the push, the
// binary-select refusal, the SSH_CLIENT egress read, and the job-spec-in/NDJSON-out
// exec — is unit-testable with an in-memory fake and no live SSH server. The pushed
// binary is the same cmd/prober the instance runs locally, so it carries the shared
// identifiable probe User-Agent (internal/measure.ProbeUserAgent) on its HTTP leg
// unchanged; this package never issues a probe of its own.
package remoteexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// Conn is one established SSH connection to a prober host. Each method runs exactly
// one remote command — SSH runs one command per session — so the production adapter
// opens a fresh ssh.Session per call and a fake can answer each command in memory.
type Conn interface {
	// Output runs cmd and returns its stdout. It is the read side: `uname` for the
	// arch check and the SSH_CLIENT read for egress. cmd is always a constant this
	// package supplies (never operator or spec input), so it carries no injected data.
	Output(ctx context.Context, cmd string) ([]byte, error)
	// Run streams stdin into cmd, cmd's stdout into stdout, and cmd's stderr into
	// stderr. It is the write/exec side: piping the binary to `cat > path` on push,
	// then exec'ing the pushed path with the job spec on stdin and reading NDJSON back
	// — never the spec on argv (ADR-0001). cmd is again a constant plus this package's
	// own generated temp path.
	//
	// Run surfaces all three of SSH's native channels: the caller supplies the stderr
	// sink (nil discards it) and receives a typed ExitResult classifying how the remote
	// command ended (#867). The returned error is non-nil for a session-level failure
	// AND for a non-zero exit or a signal (mirroring os/exec's cmd.Run), so a caller
	// that only checks err still treats a failed command as a failure; ExitResult
	// supplies the typed detail the transcript records on every outcome.
	Run(ctx context.Context, cmd string, stdin io.Reader, stdout, stderr io.Writer) (ExitResult, error)
	// RemoteAddr is the transport peer address of the established connection — the
	// address the instance actually dialled to reach the prober, observed locally at
	// connect (no remote command). It is the presented DIALLED address the Vantage-class
	// derivation reads (#710), captured "by construction" exactly as SSH_CLIENT is the
	// egress the prober reports. It may be nil where the underlying transport exposes no
	// peer address, in which case the fact is left unobserved rather than fabricated.
	RemoteAddr() net.Addr
	Close() error
}

type ExitKind int

const (
	// ExitExited: the command ran to completion and returned Code. Code is -1 when the
	// server reported no status (an honest "no clean exit", never a fabricated success).
	ExitExited ExitKind = iota
	ExitSignalled
	ExitContextCancelled
)

// ExitResult is the typed outcome of one remote Run. Exactly one field is meaningful,
// selected by Kind: Code for ExitExited, Signal for ExitSignalled, neither otherwise.
type ExitResult struct {
	Kind   ExitKind
	Code   int    // valid when Kind == ExitExited
	Signal string // valid when Kind == ExitSignalled
}

type Target struct {
	Addr            string // host:port
	Username        string
	PrivateKey      []byte
	HostKeyCallback ssh.HostKeyCallback
	Timeout         time.Duration
}

// Dial establishes the production SSH connection and returns a Conn over it. It
// reuses the repo's key material and trust-on-first-use host-key pinning exactly as
// the latency prober does (cmd/worker/vantages.go): the host key is pinned/enforced
// by HostKeyCallback, never blindly accepted.
func Dial(ctx context.Context, t Target) (Conn, error) {
	signer, err := ssh.ParsePrivateKey(t.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("remoteexec: parse private key: %w", err)
	}
	cfg := &ssh.ClientConfig{
		User:            t.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: t.HostKeyCallback,
		Timeout:         t.Timeout,
	}
	d := net.Dialer{Timeout: t.Timeout}
	netConn, err := d.DialContext(ctx, "tcp", t.Addr)
	if err != nil {
		return nil, fmt.Errorf("remoteexec: dial %s: %w", t.Addr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, t.Addr, cfg)
	if err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("remoteexec: ssh handshake %s: %w", t.Addr, err)
	}
	return &clientConn{client: ssh.NewClient(sshConn, chans, reqs)}, nil
}

type clientConn struct{ client *ssh.Client }

func (c *clientConn) Output(ctx context.Context, cmd string) ([]byte, error) {
	sess, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	var stdout bytes.Buffer
	sess.Stdout = &stdout
	// Output is the read side (uname / SSH_CLIENT); it needs only stdout, so the typed
	// exit result is discarded and any non-zero exit stays an error, exactly as before.
	if _, err := runSession(ctx, sess, cmd); err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func (c *clientConn) Run(ctx context.Context, cmd string, stdin io.Reader, stdout, stderr io.Writer) (ExitResult, error) {
	sess, err := c.client.NewSession()
	if err != nil {
		return ExitResult{Kind: ExitExited, Code: -1}, err
	}
	defer sess.Close()
	sess.Stdin = stdin
	sess.Stdout = stdout
	sess.Stderr = stderr
	return runSession(ctx, sess, cmd)
}

func (c *clientConn) RemoteAddr() net.Addr { return c.client.RemoteAddr() }

func (c *clientConn) Close() error { return c.client.Close() }

// runSession starts cmd on sess and waits, honouring ctx: a cancelled context kills
// the remote command and closes the session rather than blocking on a silent host. It
// returns the typed ExitResult alongside the error so a caller capturing a transcript
// classifies the ending on every outcome. The error is preserved unchanged (nil only on
// a clean exit-0), so err-only callers keep their existing behaviour.
func runSession(ctx context.Context, sess *ssh.Session, cmd string) (ExitResult, error) {
	if err := sess.Start(cmd); err != nil {
		// The command never started, so there is no exit status to report.
		return ExitResult{Kind: ExitExited, Code: -1}, err
	}
	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()
	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		_ = sess.Close()
		return ExitResult{Kind: ExitContextCancelled}, ctx.Err()
	case err := <-done:
		return classifyExit(err), err
	}
}

// classifyExit maps an ssh.Session.Wait error into the typed ExitResult. A nil error is
// a clean exit-0. An *ssh.ExitError carries either a signal name (signalled) or an exit
// status (exited). An *ssh.ExitMissingError — the server closed the channel with no exit
// status — reads as signalled with an empty name. Any other error (a transport or
// stdout-sink failure, e.g. the 64 MiB guard tripping) leaves the outcome unknown, read
// as exited(-1): an honest "no clean exit", never a fabricated success.
func classifyExit(err error) ExitResult {
	if err == nil {
		return ExitResult{Kind: ExitExited, Code: 0}
	}
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		if sig := exitErr.Signal(); sig != "" {
			return ExitResult{Kind: ExitSignalled, Signal: sig}
		}
		return ExitResult{Kind: ExitExited, Code: exitErr.ExitStatus()}
	}
	var missingErr *ssh.ExitMissingError
	if errors.As(err, &missingErr) {
		return ExitResult{Kind: ExitSignalled}
	}
	return ExitResult{Kind: ExitExited, Code: -1}
}
