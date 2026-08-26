//go:build unix

package main

import "golang.org/x/sys/unix"

// diskUsage reports the used and total bytes of the filesystem holding path, on the unix
// deployment target (#633, WORK-ORDER-DOGFOOD-R1 item 3). It is a real Statfs of the
// running host — never a fabricated figure — so the instance-health disk bar shows the
// genuine volume state. An error (path gone, permissions) or a nonsensical read reports
// ok=false, and the surface collapses the figure rather than guess one.
func diskUsage(path string) (used, total uint64, ok bool) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, 0, false
	}
	bsize := uint64(st.Bsize) // #nosec G115 -- Bsize is the filesystem block size, always a small positive value
	total = uint64(st.Blocks) * bsize
	avail := uint64(st.Bavail) * bsize
	if total == 0 || avail > total {
		return 0, 0, false
	}
	return total - avail, total, true
}
