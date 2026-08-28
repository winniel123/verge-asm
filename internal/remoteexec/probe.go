package remoteexec

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/winniel123/verge-asm/internal/wire"
)

// Commands are constants (uname / the SSH_CLIENT read) plus this package's own
// generated temp path — never operator or job-spec input — so the remote command
// strings carry no injected data. The job spec travels on stdin, never argv (ADR-0001).
const (
	cmdUnameS     = "uname -s"
	cmdUnameM     = "uname -m"
	cmdReadEgress = "printenv SSH_CLIENT"
)

// Facts are the lifecycle facts the worker observes off-host on the connect that pins
// a prober's host key: the remote platform (VantageCard's accepted-platform chip, and
// what the arch check matches the binary to) and the egress address the probe leaves
// from (SSH_CLIENT). A read that could not identify a fact leaves it zero rather than
// fabricating one; the caller persists only what was actually observed.
type Facts struct {
	Platform   Platform
	Egress     string
	HasEgress  bool
	Dialled    string
	HasDialled bool
}

// Inspect reads the prober's lifecycle facts over an established connection: `uname`
// for the platform and SSH_CLIENT for the egress. The platform is mandatory (an
// unidentifiable host is an error); the egress is best-effort (a host that does not
// export SSH_CLIENT collapses that chip, never fabricates an address).
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
	// The dialled address is observed off-host at connect — the SSH transport's peer
	// address (#710) — with no remote command, exactly what "known by construction"
	// means. Best-effort like egress: a peer address that will not parse leaves the
	// fact zero rather than fabricating one.
	if dialled, ok := normalizeDialled(conn.RemoteAddr()); ok {
		f.Dialled, f.HasDialled = dialled, true
	}
	return f, nil
}

// remotePlatform runs the uname arch check over the connection, mapping the kernel and
// machine names to Go identifiers and the chip label.
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

// Probe pushes the matching prober binary to the host and exec's it there, returning
// the observations it wrote — the SSH twin of the local ExecProber, honouring the same
// job-spec-in/NDJSON-out contract (ADR-0001). The order is load-bearing:
//
//  1. `uname` arch check — identify the remote platform.
//  2. select the binary built for THAT platform; refuse (ErrNoBinary) rather than push
//     a mismatched one — the arch check gates the push.
//  3. stream the binary to a fresh temp path and mark it executable.
//  4. exec it with the job spec on stdin, decoding the NDJSON observations it writes.
//  5. best-effort remove the pushed binary.
//
// The pushed binary is the instance's own cmd/prober, so it carries the shared
// identifiable probe User-Agent (measure.ProbeUserAgent) on its HTTP leg unchanged.
func Probe(ctx context.Context, conn Conn, binaries BinaryProvider, spec wire.JobSpec) ([]wire.Observation, error) {
	plat, err := remotePlatform(ctx, conn)
	if err != nil {
		return nil, err
	}

	bin, err := binaries.Binary(plat.GOOS, plat.GOARCH)
	if err != nil {
		return nil, fmt.Errorf("remoteexec: select binary for %s/%s: %w", plat.GOOS, plat.GOARCH, err)
	}
	defer bin.Close()

	path, err := tempPath()
	if err != nil {
		return nil, err
	}

	// Push: stream the binary to the temp path and make it executable in one command.
	if err := conn.Run(ctx, "cat > "+path+" && chmod 0700 "+path, bin, io.Discard); err != nil {
		return nil, fmt.Errorf("remoteexec: push binary: %w", err)
	}
	// Best-effort cleanup regardless of how the exec goes — the pushed binary is
	// disposable, one per invocation, so version skew is structurally impossible.
	defer func() { _, _ = conn.Output(ctx, "rm -f "+path) }()

	var stdin bytes.Buffer
	if err := wire.EncodeJobSpec(&stdin, spec); err != nil {
		return nil, err
	}
	// Fail-closed sink: the prober is untrusted (a compromised host, or a MITM
	// before the host key is pinned), so its stdout is capped at MaxProberStdout
	// during the streaming copy rather than buffered without bound — a hostile
	// prober cannot OOM the worker (#772). Exceeding the cap surfaces as a Run
	// error, driving the same retry/dead-letter path any exec failure does.
	stdout := wire.NewLimitedBuffer(wire.MaxProberStdout)
	if err := conn.Run(ctx, path, &stdin, stdout); err != nil {
		return nil, fmt.Errorf("remoteexec: exec prober: %w", err)
	}

	sc := wire.NewObservationScanner(bytes.NewReader(stdout.Bytes()))
	var obs []wire.Observation
	for sc.Next() {
		obs = append(obs, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("remoteexec: decode prober output: %w", err)
	}
	return obs, nil
}

// tempPath returns a fresh, collision-resistant remote path under /tmp for the pushed
// binary. The random suffix is this package's own — no operator or spec input reaches
// the command string.
func tempPath() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("remoteexec: temp name: %w", err)
	}
	return "/tmp/verge-prober-" + hex.EncodeToString(b[:]), nil
}
