//go:build !windows

package queue

import (
	"os"
	"syscall"
)

// signalName returns the name of the signal that killed the prober process, or ""
// when it exited normally (raw-job-output spec §1.2). It reads the platform
// WaitStatus the process's ProcessState carries. The worker builds for linux, so
// this is the production path; the windows twin returns "" so the package still
// compiles and tests run on a dev host.
func signalName(ps *os.ProcessState) string {
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return ws.Signal().String()
	}
	return ""
}
