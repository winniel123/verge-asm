// Package corpus is the golden corpus for the tls-acceptance leaf (v1 spec §3.4,
// golden-corpus.md §4). It holds a checked-in enumeration of the cells the leaf
// owes a pinning row, each row being a (scope, scripted enumerator, expected NDJSON)
// triple run hermetically against an in-process enumerator — no network, no
// containers. A row protects the tls-acceptance leaf and nothing else, so it is a
// sibling of the connect-outcome / http-exchange / resolution corpora and never
// pooled with them.
package corpus

import (
	"context"
	"net/netip"

	ta "github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

// listener models one TLS listener's acceptance behaviour: whether it speaks TLS at
// all, and — per accepted version — the suites it accepts in its own selection
// preference order. A version key that is PRESENT means the version is accepted (an
// empty-but-present slice is TLS 1.3, accepted with no suites recorded); an absent
// key means the version is refused. This is the whole in-band model the accept-fold
// consumes, so a row scripts a real listener rather than a canned output.
type listener struct {
	spoke   bool
	accepts map[string][]string
}

// scriptEnumerator answers each handshake from a fixed map of listeners keyed by
// Service. An unscripted Service reads as a peer that never spoke TLS — a `no-tls`
// value — so a row that forgets a Service renders a legible negative rather than
// panicking.
type scriptEnumerator struct {
	byService map[string]listener
	calls     int
}

func newScript(byService map[string]listener) *scriptEnumerator {
	return &scriptEnumerator{byService: byService}
}

// Handshake implements tlsacceptance.Enumerator: it selects the listener for the
// target Service and answers the pinned version / offered suites from its model.
func (s *scriptEnumerator) Handshake(_ context.Context, target netip.AddrPort, version string, offered []string) ta.Attempt {
	s.calls++
	l, ok := s.byService[ta.ServiceKey(target, "tcp")]
	if !ok || !l.spoke {
		return ta.Attempt{} // nothing here spoke TLS
	}
	prefs, present := l.accepts[version]
	if !present {
		return ta.Attempt{Spoke: true} // it spoke TLS but refuses this version
	}
	if version == ta.TLS13 {
		return ta.Attempt{Spoke: true, Accepted: true}
	}
	offeredSet := make(map[string]bool, len(offered))
	for _, c := range offered {
		offeredSet[c] = true
	}
	for _, p := range prefs {
		if offeredSet[p] {
			return ta.Attempt{Spoke: true, Accepted: true, SelectedCipher: p}
		}
	}
	return ta.Attempt{Spoke: true} // all its suites already peeled — the refusing round
}
