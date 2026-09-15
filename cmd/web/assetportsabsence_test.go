package main

import (
	"errors"
	"testing"
)

const (
	portsDidNotResolve = "The open-port read did not resolve on this load."
	portsEmptyState    = "No open port measured"

	headerLegChip = `<span class="as-hleg">`
	neverLooked   = "never looked"
)

func TestAssetPortsReadFailureRendersDidNotResolve(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.openSpansErr = errors.New("list all open spans failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, portsDidNotResolve, "failed open-spans read")
	wantNotIn(t, page, portsEmptyState, "failed open-spans read")
	wantIn(t, page, "Open ports", "failed open-spans read")
}

func TestAssetLegReadFailureRendersDidNotResolve(t *testing.T) {
	// The class-aware leg read is the ports read's second half, so it degrades the same region.
	f, base := assetRegionFixture(t, "api.example.com")
	f.reachSpansErr = errors.New("list service reachability spans by class failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, portsDidNotResolve, "failed leg read")
	wantNotIn(t, page, portsEmptyState, "failed leg read")
}

func TestAssetPortsReadFailureHidesTheHeaderInternetLegChip(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	f.openSpansErr = errors.New("list all open spans failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantNotIn(t, page, headerLegChip, "failed open-spans read")
	// A substituted leg claims the estate did not look, on a database fault (ADR-0168 §2).
	wantNotIn(t, page, neverLooked, "failed open-spans read")
}

func TestAssetPortsReadFailureKeepsEveryOtherRegion(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")
	f.openSpansErr = errors.New("list all open spans failed")

	page := assetRegionPage(t, f, base, "api.example.com")

	// A 500 removed all of these, and the ports read feeds none of them (ADR-2030 §4).
	for _, want := range []string{"How it got here", kvCellSeed, "Signals here", "DNS records", "Drift trail"} {
		wantIn(t, page, want, "failed open-spans read")
	}
}

func TestAssetWithNoOpenPortKeepsItsOriginalEmptyState(t *testing.T) {
	f, base := assetRegionFixture(t, "api.example.com")

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, portsEmptyState, "a Name that holds no open port")
	wantNotIn(t, page, portsDidNotResolve, "a Name that holds no open port")
}
