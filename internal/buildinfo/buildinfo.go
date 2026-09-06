// Package buildinfo reports the version this binary was built as: the link-time
// stamp when a release build set one, and the operator's VERGE_VERSION otherwise
// (release-pipeline.md §3).
package buildinfo

import "github.com/winniel123/verge-asm/internal/env"

var version string

func Stamped() bool {
	return version != ""
}

func Version() string {
	// An env that outranked the stamp would let an operator relabel a released image (release-pipeline.md §3).
	if version != "" {
		return version
	}
	return env.OrDefault("VERGE_VERSION", "dev")
}
