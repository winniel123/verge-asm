package remoteexec

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

var ErrNoBinary = errors.New("remoteexec: no prober binary for the remote platform")

type BinaryProvider interface {
	Binary(goos, goarch string) (io.ReadCloser, error)
}

type DirBinaryProvider struct {
	Dir      string
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
	// A cross-arch request is refused, never served a binary the host cannot run (packaging §1.5).
	if p.Fallback != "" && goos == runtime.GOOS && goarch == runtime.GOARCH {
		if f, err := os.Open(p.Fallback); err == nil { // #nosec G304 (Fallback is the operator-configured VERGE_PROBER_PATH, the same trusted path the local ExecProber runs)
			return f, nil
		}
	}
	return nil, ErrNoBinary
}
