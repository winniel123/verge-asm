package queue

import (
	"net/netip"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

func resolutionSpan(id int64, vantage pgtype.Int8, outcome string) db.ListOpenSpansForSubjectRow {
	return db.ListOpenSpansForSubjectRow{
		ID:          id,
		SubjectKind: "name",
		Facet:       resolutionwalk.FacetResolution,
		VantageID:   vantage,
		Value:       []byte(`{"outcome":"` + outcome + `"}`),
	}
}

func nameSeed(domain string) db.ListSeedsRow {
	return db.ListSeedsRow{Kind: "name", NameDomain: pgtype.Text{String: domain, Valid: true}}
}

func addressSeed(cidr string) db.ListSeedsRow {
	p := netip.MustParsePrefix(cidr)
	return db.ListSeedsRow{Kind: "address", AddressCidr: &p}
}

func subtreeExclusion(name string) db.ListExclusionsRow {
	return db.ListExclusionsRow{Kind: "subtree", Name: pgtype.Text{String: name, Valid: true}}
}

func TestDecideNameDeparture(t *testing.T) {
	// `returned` is absent because drift derives it on read from a measured-absent closure (#637).
	tests := []struct {
		name        string
		open        []db.ListOpenSpansForSubjectRow
		seedCovered bool
		excluded    bool
		wantReason  drift.ClosureReason
		wantLeft    bool
	}{
		{
			name:       "measured-absent single vantage NameError",
			open:       []db.ListOpenSpansForSubjectRow{resolutionSpan(1, pgtype.Int8{}, estateOutcomeNameError)},
			wantReason: drift.ReasonMeasuredAbsent,
			wantLeft:   true,
		},
		{
			name: "measured-absent cross-vantage unanimous suppression",
			open: []db.ListOpenSpansForSubjectRow{
				resolutionSpan(1, pgtype.Int8{Int64: 10, Valid: true}, estateOutcomeNameError),
				resolutionSpan(2, pgtype.Int8{Int64: 20, Valid: true}, estateOutcomeShadowed),
			},
			wantReason: drift.ReasonMeasuredAbsent,
			wantLeft:   true,
		},
		{
			name: "one admitting vantage keeps the name",
			open: []db.ListOpenSpansForSubjectRow{
				resolutionSpan(1, pgtype.Int8{Int64: 10, Valid: true}, estateOutcomeNameError),
				resolutionSpan(2, pgtype.Int8{Int64: 20, Valid: true}, estateOutcomeResolved),
			},
			wantLeft: false,
		},
		{
			name: "gap blocks withdrawal",
			open: []db.ListOpenSpansForSubjectRow{
				resolutionSpan(1, pgtype.Int8{Int64: 10, Valid: true}, estateOutcomeNameError),
				resolutionSpan(2, pgtype.Int8{Int64: 20, Valid: true}, "Gap"),
			},
			wantLeft: false,
		},
		{
			name:       "descoped by exclusion beats a resolving name",
			open:       []db.ListOpenSpansForSubjectRow{resolutionSpan(1, pgtype.Int8{}, estateOutcomeResolved)},
			excluded:   true,
			wantReason: drift.ReasonDescoped,
			wantLeft:   true,
		},
		{
			name:        "seed-covered name stays despite NameError",
			open:        []db.ListOpenSpansForSubjectRow{resolutionSpan(1, pgtype.Int8{}, estateOutcomeNameError)},
			seedCovered: true,
			wantLeft:    false,
		},
		{
			name:     "no open spans cannot leave",
			open:     nil,
			wantLeft: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reason, left := decideNameDeparture(tc.open, tc.seedCovered, tc.excluded)
			if left != tc.wantLeft || reason != tc.wantReason {
				t.Fatalf("decideNameDeparture = (%q, %v), want (%q, %v)", reason, left, tc.wantReason, tc.wantLeft)
			}
		})
	}
}

// These mirror internal/estate's own constants, so a tag renamed there must be renamed here.

const (
	estateOutcomeResolved  = "Resolved"
	estateOutcomeNameError = "NameError"
	estateOutcomeShadowed  = "Shadowed"
)

func TestOpenedByAperture(t *testing.T) {
	in := membershipInputs{seeds: []db.ListSeedsRow{nameSeed("example.com"), addressSeed("198.51.100.0/24")}}

	cases := []struct {
		kind, key string
		want      bool
	}{
		{"name", "api.example.com", true},
		{"name", "example.com", true},
		{"name", "notexample.com", false},
		{"name", "discovered.other.net", false},
		{"address", "198.51.100.7", true},
		{"address", "203.0.113.7", false},
		{"service", "198.51.100.7:443", true},
		{"service", "203.0.113.7:443", false},
	}
	for _, c := range cases {
		if got := openedByAperture(c.kind, c.key, in); got != c.want {
			t.Errorf("openedByAperture(%q, %q) = %v, want %v", c.kind, c.key, got, c.want)
		}
	}
}

func TestOpenedByApertureExcludedSubjectIsUnmarked(t *testing.T) {
	in := membershipInputs{
		seeds:      []db.ListSeedsRow{nameSeed("example.com"), addressSeed("198.51.100.0/24")},
		exclusions: []db.ListExclusionsRow{nameExclusion("api.example.com"), addressExclusion("198.51.100.0/28")},
	}

	cases := []struct {
		kind, key string
		want      bool
	}{
		{"name", "api.example.com", false},
		{"name", "other.example.com", true},
		{"address", "198.51.100.7", false},
		{"address", "198.51.100.90", true},
		{"service", "198.51.100.7:443", false},
		{"service", "198.51.100.90:443", true},
	}
	for _, c := range cases {
		if got := openedByAperture(c.kind, c.key, in); got != c.want {
			t.Errorf("openedByAperture(%q, %q) = %v, want %v", c.kind, c.key, got, c.want)
		}
	}

	sub := membershipInputs{
		seeds:      []db.ListSeedsRow{nameSeed("example.com")},
		exclusions: []db.ListExclusionsRow{subtreeExclusion("internal.example.com")},
	}
	if openedByAperture("name", "db.internal.example.com", sub) {
		t.Error("a name beneath a subtree exclusion is outside the Declared aperture")
	}
	if !openedByAperture("name", "www.example.com", sub) {
		t.Error("a name the subtree exclusion does not cover stays aperture-declared")
	}
}

func TestNameWithinDomain(t *testing.T) {
	cases := []struct {
		name, domain string
		want         bool
	}{
		{"example.com", "example.com", true},
		{"api.example.com", "example.com", true},
		{"deep.api.example.com", "example.com", true},
		{"notexample.com", "example.com", false},
		{"example.com.evil.net", "example.com", false},
		{"EXAMPLE.com", "example.com", true},
		{"api.example.com.", "example.com", true},
		{"", "example.com", false},
		{"example.com", "", false},
	}
	for _, c := range cases {
		if got := nameWithinDomain(c.name, c.domain); got != c.want {
			t.Errorf("nameWithinDomain(%q, %q) = %v, want %v", c.name, c.domain, got, c.want)
		}
	}
}

func TestNameExcluded(t *testing.T) {
	exact := []db.ListExclusionsRow{{Kind: "name", Name: pgtype.Text{String: "api.example.com", Valid: true}}}
	if !nameExcluded("api.example.com", exact) {
		t.Error("exact exclusion should cover the exact name")
	}
	if nameExcluded("other.api.example.com", exact) {
		t.Error("exact exclusion must not cover a child")
	}
	sub := []db.ListExclusionsRow{subtreeExclusion("example.com")}
	if !nameExcluded("deep.example.com", sub) {
		t.Error("subtree exclusion should cover a child")
	}
	if nameExcluded("example.org", sub) {
		t.Error("subtree exclusion must not cover an unrelated name")
	}
}

func TestResolutionWitnesses(t *testing.T) {
	open := []db.ListOpenSpansForSubjectRow{
		resolutionSpan(1, pgtype.Int8{Int64: 10, Valid: true}, estateOutcomeNameError),
		resolutionSpan(2, pgtype.Int8{Int64: 20, Valid: true}, estateOutcomeResolved),
		{ID: 3, Facet: "reachability", VantageID: pgtype.Int8{Int64: 10, Valid: true}, Value: []byte(`{"outcome":"reached"}`)},
	}
	w := resolutionWitnesses(open)
	if len(w) != 2 {
		t.Fatalf("want 2 vantage witnesses (reachability ignored), got %d: %+v", len(w), w)
	}
	for _, cw := range w {
		if len(cw.Outcomes) != 1 {
			t.Errorf("class %q: want one outcome, got %v", cw.Class, cw.Outcomes)
		}
	}
}

func TestObservedResolutionNames(t *testing.T) {
	obs := []wire.Observation{
		{Facet: resolutionwalk.FacetResolution, Subject: "a.example.com"},
		{Facet: "dns-record", Subject: "a.example.com"},
		{Facet: resolutionwalk.FacetResolution, Subject: "b.example.com"},
		{Facet: resolutionwalk.FacetResolution, Subject: "a.example.com"},
		{Facet: resolutionwalk.FacetResolution, Subject: ""},
	}
	got := observedResolutionNames(obs)
	want := []string{"a.example.com", "b.example.com"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("observedResolutionNames = %v, want %v", got, want)
	}
}
