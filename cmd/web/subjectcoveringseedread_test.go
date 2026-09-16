package main

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	seedChipDidNotResolve  = "covered since — the Seed read did not resolve"
	seedChainDidNotResolve = "The covering-Seed read did not resolve on this load."
	seedRowDidNotResolve   = `<span class="sd-micro">Seed</span><span class="v">the Seed read did not resolve</span>`
	chainReachesNoSeed     = "The chain does not reach a declared Seed."
)

func TestSubjectDetailNamesAFailedCoveringSeedReadOnACitedAddress(t *testing.T) {
	for _, c := range subjectSpanReadCases() {
		t.Run(c.name, func(t *testing.T) {
			// Both fixtures carry a resolution citing the address, so that read resolves (#2061).
			f := c.store(t)
			f.coveringAddressSeedErr = errors.New("find covering address seed failed")

			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")
			page := getBody(t, ac, base+c.path, http.StatusOK)

			wantIn(t, page, seedChipDidNotResolve, "failed covering-seed read")
			wantIn(t, page, seedRowDidNotResolve, "failed covering-seed read")
			wantIn(t, page, seedChainDidNotResolve, "failed covering-seed read")
			// The chain line is a claim about the chain, drawn from a read that never resolved.
			wantNotIn(t, page, chainReachesNoSeed, "failed covering-seed read")
			// A citing resolution settles membership on its own, so the page stays live (#2050).
			wantNotIn(t, page, membershipDidNotResolve, "failed covering-seed read")
			wantNotIn(t, page, withdrawnHeadline, "failed covering-seed read")
			wantIn(t, page, rescanOffered, "failed covering-seed read")
		})
	}
}

func TestSubjectDetailKeepsNoCoveringSeedWhenThatReadResolves(t *testing.T) {
	for _, empty := range []struct {
		name string
		set  func(*fakeStore)
	}{
		{"no seed declared", func(*fakeStore) {}},
		// No row is the honest no-covering-Seed answer, never a failed read (ADR-0168 §4).
		{"no rows error", func(f *fakeStore) { f.coveringAddressSeedErr = pgx.ErrNoRows }},
	} {
		for _, c := range subjectSpanReadCases() {
			t.Run(empty.name+"/"+c.name, func(t *testing.T) {
				f := c.store(t)
				empty.set(f)

				base := start(t, f, "")
				ac := login(t, base, "admin", "hunter2hunter2")
				page := getBody(t, ac, base+c.path, http.StatusOK)

				wantIn(t, page, chainReachesNoSeed, "an address no Seed covers")
				for _, banned := range []string{seedChipDidNotResolve, seedRowDidNotResolve, seedChainDidNotResolve} {
					wantNotIn(t, page, banned, "an address no Seed covers")
				}
			})
		}
	}
}
