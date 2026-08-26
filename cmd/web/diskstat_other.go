//go:build !unix

package main

// diskUsage has no portable implementation off the unix deployment target (dev on
// Windows), so it reports ok=false and the instance-health disk figure collapses rather
// than fabricate one — the same honest degradation the rest of the health tab uses.
func diskUsage(string) (used, total uint64, ok bool) {
	return 0, 0, false
}
