//go:build !windows

package queue

import (
	"os"
	"syscall"
)

func signalName(ps *os.ProcessState) string {
	// The worker image builds GOOS=linux, so the windows twin is a compile stub, never a live path.
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return ws.Signal().String()
	}
	return ""
}
