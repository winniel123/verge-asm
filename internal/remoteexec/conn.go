// Package remoteexec pushes the measurement binary to a provisioned prober over SSH
// and exec's it there, so an internet Vantage measures from its own position rather
// than from the instance host (ADR-0001, ADR-0103, #683).
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

// SSH runs one command per session, so every method here opens a session of its own.

type Conn interface {
	Output(ctx context.Context, cmd string) ([]byte, error)
	Run(ctx context.Context, cmd string, stdin io.Reader, stdout, stderr io.Writer) (ExitResult, error)
	RemoteAddr() net.Addr
	Close() error
}

type ExitKind int

const (
	ExitExited ExitKind = iota
	ExitSignalled
	ExitContextCancelled
)

type ExitResult struct {
	Kind   ExitKind
	Code   int
	Signal string
}

type Target struct {
	Addr            string
	Username        string
	PrivateKey      []byte
	HostKeyCallback ssh.HostKeyCallback
	Timeout         time.Duration
}

func Dial(ctx context.Context, t Target) (Conn, error) {
	signer, err := ssh.ParsePrivateKey(t.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("remoteexec: parse private key: %w", err)
	}
	// The caller's HostKeyCallback is enforced as given: a host is never blindly trusted.
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

func runSession(ctx context.Context, sess *ssh.Session, cmd string) (ExitResult, error) {
	if err := sess.Start(cmd); err != nil {
		return ExitResult{Kind: ExitExited, Code: -1}, err
	}
	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()
	// The error mirrors os/exec: a non-zero exit or a signal is a failure too (#867).
	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		_ = sess.Close()
		return ExitResult{Kind: ExitContextCancelled}, ctx.Err()
	case err := <-done:
		return classifyExit(err), err
	}
}

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
	// An ExitMissingError is the server closing the channel with no status, not a clean exit.
	if errors.As(err, &missingErr) {
		return ExitResult{Kind: ExitSignalled}
	}
	// An unclassifiable error is an honest "no clean exit", never a fabricated success (#867).
	return ExitResult{Kind: ExitExited, Code: -1}
}
