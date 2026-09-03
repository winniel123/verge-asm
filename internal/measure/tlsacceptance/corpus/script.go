// Package corpus is the golden corpus for the tls-acceptance leaf (v1 spec §3.4,
// golden-corpus.md §4). Every row renders hermetically against an in-process
// enumerator, and protects this leaf alone — never pooled with a sibling corpus.
package corpus

import (
	"context"
	"net/netip"

	ta "github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

type listener struct {
	spoke   bool
	accepts map[string][]string
}

type scriptEnumerator struct {
	byService map[string]listener
	calls     int
}

func newScript(byService map[string]listener) *scriptEnumerator {
	return &scriptEnumerator{byService: byService}
}

func (s *scriptEnumerator) Handshake(_ context.Context, target netip.AddrPort, version string, offered []string) ta.Attempt {
	// A row scripts a real listener, never a canned output, so the accept-fold sees the whole model.
	s.calls++
	l, ok := s.byService[ta.ServiceKey(target, "tcp")]
	// An unscripted Service reads as no-tls, so a forgotten row stays legible, never a panic.
	if !ok || !l.spoke {
		return ta.Attempt{}
	}
	prefs, present := l.accepts[version]
	if !present {
		return ta.Attempt{Spoke: true}
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
	return ta.Attempt{Spoke: true}
}
