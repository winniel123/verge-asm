//go:build windows

package queue

import "os"

func signalName(*os.ProcessState) string { return "" }
