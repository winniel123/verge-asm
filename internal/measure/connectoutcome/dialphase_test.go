package connectoutcome

import (
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"os"
	"syscall"
	"testing"
)

func TestClassifyDialErrorSplitsThePhase(t *testing.T) {
	dialErr := func(inner error) error {
		return &net.OpError{Op: "dial", Net: "tcp", Err: inner}
	}
	readErr := func(inner error) error {
		return &net.OpError{Op: "read", Net: "tcp", Err: inner}
	}

	cases := []struct {
		name            string
		err             error
		wantOutcome     TLSOutcome
		wantUnreachable bool
	}{
		{
			name:            "a refused connect never reached a peer",
			err:             dialErr(os.NewSyscallError("connect", syscall.ECONNREFUSED)),
			wantOutcome:     NoTLS,
			wantUnreachable: true,
		},
		{
			name:            "a connect-phase reset never reached a peer",
			err:             dialErr(os.NewSyscallError("connect", errors.New("connection reset by peer"))),
			wantOutcome:     NoTLS,
			wantUnreachable: true,
		},
		{
			name:            "a connect timeout never reached a peer",
			err:             dialErr(os.ErrDeadlineExceeded),
			wantOutcome:     NoTLS,
			wantUnreachable: true,
		},
		{
			name:            "the egress guard's socket refusal never reached a peer",
			err:             dialErr(errors.New("connectoutcome: refusing to dial non-globally-reachable address 127.0.0.1")),
			wantOutcome:     NoTLS,
			wantUnreachable: true,
		},
		{
			name:            "a handshake that stalls after the connect reached a peer",
			err:             readErr(os.ErrDeadlineExceeded),
			wantOutcome:     NoTLS,
			wantUnreachable: false,
		},
		{
			name:            "a plaintext peer reached a peer",
			err:             errors.New("tls: first record does not look like a TLS handshake"),
			wantOutcome:     NoTLS,
			wantUnreachable: false,
		},
		{
			name:            "a handshake EOF reached a peer",
			err:             io.EOF,
			wantOutcome:     NoTLS,
			wantUnreachable: false,
		},
		{
			name:            "a TLS alert reached a peer and refused",
			err:             errors.New("remote error: tls: handshake failure"),
			wantOutcome:     TLSRefused,
			wantUnreachable: false,
		},
		{
			name:            "an unclassifiable handshake failure reached a peer",
			err:             errors.New("something we have never seen"),
			wantOutcome:     NoTLS,
			wantUnreachable: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, unreachable := classifyDialError(tc.err)
			if outcome != tc.wantOutcome {
				t.Errorf("outcome = %q, want %q", outcome, tc.wantOutcome)
			}
			if unreachable != tc.wantUnreachable {
				t.Errorf("unreachable = %v, want %v", unreachable, tc.wantUnreachable)
			}
		})
	}
}

func TestHandshakeInvalidTargetIsUnreachable(t *testing.T) {
	res := NetHandshaker{}.Handshake(context.Background(), netip.AddrPort{}, "")
	if res.Outcome != NoTLS || !res.Unreachable {
		t.Errorf("Handshake(zero) = %q unreachable=%v, want %q unreachable=true", res.Outcome, res.Unreachable, NoTLS)
	}
}
