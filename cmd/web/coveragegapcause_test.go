package main

import (
	"strings"
	"testing"
)

func TestCoverageGapExpectedNamesTheProxyEdge(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "104.21.61.6:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "origin behind the proxy edge") {
		t.Errorf("the Coverage gap row must expect an origin behind the proxy edge; body: %s", page)
	}
	if strings.Contains(page, "origin behind the edge") {
		t.Errorf("bare \"edge\" is reserved for Exposure's edge-only; body: %s", page)
	}
}
