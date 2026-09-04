//go:build unix

package main

import "golang.org/x/sys/unix"

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
