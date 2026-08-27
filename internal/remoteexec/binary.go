package remoteexec

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// ErrNoBinary is returned by a BinaryProvider when it holds no prober built for the
// requested platform. The push refuses on this error rather than shipping a binary
// the remote host cannot run — the arch check's whole job (packaging §1.5: the
// instance carries a prober for every matrix architecture and pushes the matching one).
var ErrNoBinary = errors.New("remoteexec: no prober binary for the remote platform")

// BinaryProvider supplies the statically-linked prober binary matched to a remote
// platform. Binary returns a fresh reader positioned at the start of the binary for
// (goos, goarch), or ErrNoBinary if none matches.
type BinaryProvider interface {
	Binary(goos, goarch string) (io.ReadCloser, error)
}

// DirBinaryProvider serves per-architecture prober binaries from a directory the
// instance image ships, named `prober-<goos>-<goarch>` (e.g. prober-linux-amd64).
// The instance carries one per matrix architecture, so an arm64 instance pushes to an
// amd64 host and vice versa. As a single-architecture fallback — a source checkout, or
// an image that shipped only its own arch — a Fallback path is served when the request
// matches the instance's own build architecture, so a same-arch push still works with
// no per-arch directory present.
type DirBinaryProvider struct {
	Dir string // directory of prober-<goos>-<goarch> binaries
	// Fallback is the instance's own-arch prober (VERGE_PROBER_PATH). Served only when
	// the requested (goos, goarch) is this build's own, never for a different arch.
	Fallback string
}

func (p DirBinaryProvider) Binary(goos, goarch string) (io.ReadCloser, error) {
	if p.Dir != "" {
		name := filepath.Join(p.Dir, fmt.Sprintf("prober-%s-%s", goos, goarch))
		if f, err := os.Open(name); err == nil { // #nosec G304 (name is a constant prefix + validated goos/goarch tokens from parsePlatform, never operator input)
			return f, nil
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	// No per-arch binary present. Only fall back to the own-arch binary, and only for
	// the instance's own platform — never hand back a mismatched binary.
	if p.Fallback != "" && goos == runtime.GOOS && goarch == runtime.GOARCH {
		if f, err := os.Open(p.Fallback); err == nil { // #nosec G304 (Fallback is the operator-configured VERGE_PROBER_PATH, the same trusted path the local ExecProber runs)
			return f, nil
		}
	}
	return nil, ErrNoBinary
}
