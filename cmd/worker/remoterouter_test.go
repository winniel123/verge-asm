package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/remoteexec"
	"github.com/winniel123/verge-asm/internal/wire"
)

type fakeVantageGetter struct {
	byID map[int64]db.Vantage
	err  error
}

func (f fakeVantageGetter) GetVantage(_ context.Context, id int64) (db.Vantage, error) {
	if f.err != nil {
		return db.Vantage{}, f.err
	}
	v, ok := f.byID[id]
	if !ok {
		return db.Vantage{}, errors.New("no such vantage")
	}
	return v, nil
}

type routerConn struct {
	obsLine string
	ran     []string
}

func (c *routerConn) Output(_ context.Context, cmd string) ([]byte, error) {
	switch cmd {
	case "uname -s":
		return []byte("Linux\n"), nil
	case "uname -m":
		return []byte("x86_64\n"), nil
	default:
		return nil, nil // rm -f, printenv, etc.
	}
}

func (c *routerConn) Run(_ context.Context, cmd string, stdin io.Reader, stdout, _ io.Writer) (remoteexec.ExitResult, error) {
	c.ran = append(c.ran, cmd)
	if strings.HasPrefix(cmd, "cat > ") {
		_, _ = io.Copy(io.Discard, stdin)
		return remoteexec.ExitResult{}, nil
	}
	_, _ = io.Copy(io.Discard, stdin)
	_, _ = io.WriteString(stdout, c.obsLine)
	return remoteexec.ExitResult{}, nil
}

func (c *routerConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("198.51.100.9"), Port: 22}
}

func (c *routerConn) Close() error { return nil }

type oneArchBinaries struct{}

func (oneArchBinaries) Binary(goos, goarch string) (io.ReadCloser, error) {
	if goos == "linux" && goarch == "amd64" {
		return io.NopCloser(strings.NewReader("BINARY")), nil
	}
	return nil, remoteexec.ErrNoBinary
}

func provisionedRow(id int64, hostKey string) db.Vantage {
	v := db.Vantage{
		ID:       id,
		Host:     pgtype.Text{String: "prober.example.com", Valid: true},
		Port:     pgtype.Int4{Int32: 22, Valid: true},
		Username: pgtype.Text{String: "prober", Valid: true},
	}
	if hostKey != "" {
		v.HostKey = pgtype.Text{String: hostKey, Valid: true}
	}
	return v
}

// A job with no vantage, and a resolver-only vantage (no prober host), both defer to
// the local prober (handled=false) — off-host routing is only for provisioned probers.
func TestRouterDefersLocalForNonProber(t *testing.T) {
	rt := &remoteProberRouter{
		store:    fakeVantageGetter{byID: map[int64]db.Vantage{1: {ID: 1}}}, // no host
		binaries: oneArchBinaries{},
		stateDir: t.TempDir(),
	}

	if _, handled, err := rt.ProbeVantage(context.Background(), pgtype.Int8{}, wire.JobSpec{}); handled || err != nil {
		t.Errorf("no-vantage: handled=%v err=%v, want deferred to local", handled, err)
	}
	if _, handled, err := rt.ProbeVantage(context.Background(), pgtype.Int8{Int64: 1, Valid: true}, wire.JobSpec{}); handled || err != nil {
		t.Errorf("resolver-only: handled=%v err=%v, want deferred to local", handled, err)
	}
}

// A provisioned prober whose host key is not yet pinned is refused (handled, but a
// transient error) — measurement never silently first-trusts a host.
func TestRouterRefusesUnpinnedProber(t *testing.T) {
	rt := &remoteProberRouter{
		store:    fakeVantageGetter{byID: map[int64]db.Vantage{2: provisionedRow(2, "")}},
		binaries: oneArchBinaries{},
		stateDir: t.TempDir(),
		dial: func(context.Context, remoteexec.Target) (remoteexec.Conn, error) {
			t.Fatal("dialled despite an unpinned host key")
			return nil, nil
		},
	}
	_, handled, err := rt.ProbeVantage(context.Background(), pgtype.Int8{Int64: 2, Valid: true}, wire.JobSpec{})
	if !handled || err == nil {
		t.Errorf("unpinned prober: handled=%v err=%v, want handled with a transient error", handled, err)
	}
}

// A provisioned, pinned prober is measured off-host: the router dials, pushes the
// arch-matched binary, exec's it and returns the observations it wrote.
func TestRouterProbesPinnedProberOffHost(t *testing.T) {
	var obsBuf bytes.Buffer
	if err := wire.EncodeObservation(&obsBuf, wire.Observation{Batch: "b7", Kind: "connect-outcome"}); err != nil {
		t.Fatal(err)
	}
	conn := &routerConn{obsLine: obsBuf.String()}

	stateDir := t.TempDir()
	// The router reads the private key off the worker volume before dialling.
	keyPath := vantageKeyPath(stateDir, 3)
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("PRIVATE-KEY"), 0o600); err != nil {
		t.Fatal(err)
	}

	rt := &remoteProberRouter{
		store:    fakeVantageGetter{byID: map[int64]db.Vantage{3: provisionedRow(3, "ssh-ed25519 AAAApinned")}},
		binaries: oneArchBinaries{},
		stateDir: stateDir,
		dial: func(_ context.Context, target remoteexec.Target) (remoteexec.Conn, error) {
			if string(target.PrivateKey) != "PRIVATE-KEY" {
				t.Errorf("dial got private key %q, want the worker-volume key", target.PrivateKey)
			}
			if target.Addr != "prober.example.com:22" {
				t.Errorf("dial addr = %q", target.Addr)
			}
			return conn, nil
		},
	}

	res, handled, err := rt.ProbeVantage(context.Background(), pgtype.Int8{Int64: 3, Valid: true}, wire.JobSpec{Batch: "b7", Kind: "connect-outcome"})
	if err != nil || !handled {
		t.Fatalf("pinned prober: handled=%v err=%v", handled, err)
	}
	if len(res.Observations) != 1 || res.Observations[0].Batch != "b7" {
		t.Fatalf("observations = %+v, want one b7", res.Observations)
	}
	// #867: the remote path now carries the verbatim transcript back — a ProberTranscript
	// with the exec's kind and a clean exited(0) outcome, no longer an absent (nil) shape.
	tr, ok := res.Transcript.(*wire.ProberTranscript)
	if !ok {
		t.Fatalf("remote Transcript = %T, want *wire.ProberTranscript (#867)", res.Transcript)
	}
	if tr.Kind != "connect-outcome" {
		t.Errorf("transcript kind = %q, want connect-outcome", tr.Kind)
	}
	if got, ok := tr.Outcome.(wire.ProberExited); !ok || got.Code != 0 {
		t.Errorf("transcript outcome = %+v, want exited(0)", tr.Outcome)
	}
	// The binary was pushed before the exec.
	if len(conn.ran) < 2 || !strings.HasPrefix(conn.ran[0], "cat > ") {
		t.Errorf("run sequence = %v, want a push then exec", conn.ran)
	}
}
