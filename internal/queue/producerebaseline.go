package queue

import (
	"sort"
	"time"

	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
)

func rebaselineMessages(observedAt time.Time, changes []spanChange) []*message.Message {
	// One message per alerting derivation per release, never per affected subject (ADR-0008).
	moved := map[string][]string{}
	var order []string
	for _, c := range changes {
		if c.Opened || len(c.PrevVector) == 0 || !alertingDerivation(c.Facet) {
			continue
		}
		if c.PrevVector.Equal(c.Vector) {
			continue
		}
		if _, seen := moved[c.Facet]; !seen {
			order = append(order, c.Facet)
		}
		moved[c.Facet] = unionLeaves(moved[c.Facet], drift.MovedLeaves(c.PrevVector, c.Vector))
	}
	var msgs []*message.Message
	for _, facet := range order {
		m := message.Rebaseline(facet, moved[facet], facet == resolutionwalk.FacetResolution, observedAt)
		if m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

func alertingDerivation(facet string) bool {
	// Only membership (ADR-0031) and the flagship (ADR-0029) read a derivation into a message.
	switch facet {
	case resolutionwalk.FacetResolution, connectoutcome.FacetReachability:
		return true
	default:
		return false
	}
}

func unionLeaves(have, more []string) []string {
	seen := make(map[string]bool, len(have))
	for _, l := range have {
		seen[l] = true
	}
	for _, l := range more {
		if !seen[l] {
			seen[l] = true
			have = append(have, l)
		}
	}
	sort.Strings(have)
	return have
}
