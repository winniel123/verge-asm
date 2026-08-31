//go:build windows

package queue

import "os"

// signalName is the windows twin of the unix build: windows has no POSIX signals,
// so a prober process never reads back a signal name. It exists only so the queue
// package compiles and its tests run on a windows dev host; the worker itself
// builds for linux.
func signalName(*os.ProcessState) string { return "" }
