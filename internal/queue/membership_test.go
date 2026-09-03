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

// resolutionSpan builds one open `resolution` span row for a Name at a vantage, so
// a test can pose the cross-class composition the withdrawal decision reads.
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

// The withdrawal legend's three exit/entry grounds each fire from a representative
// estate transition. decideNameDeparture is the pure heart of the estate wiring
// (#637): a Name every available vantage suppresses leaves `measured-absent`
// (renders `withdrawn`); an operator narrowing leaves `descoped`; a still-declared
// or still-admitted Name stays. `returned` derives on read from a `measured-absent`
// closure and is proven in the drift feed classifier tests.
func TestDecideNameDeparture(t *testing.T) {
	tests := []struct {
		name        string
		open        []db.ListOpenSpansForSubjectRow
		seedCovered bool
		excluded    bool
		wantReason  drift.ClosureReason
		wantLeft    bool
	}{
		{
			// withdrawn: the shipped single-vantage position reads NameError — every
			// available vantage suppresses, so the Name leaves by measurement.
			name:       "measured-absent single vantage NameError",
			open:       []db.ListOpenSpansForSubjectRow{resolutionSpan(1, pgtype.Int8{}, estateOutcomeNameError)},
			wantReason: drift.ReasonMeasuredAbsent,
			wantLeft:   true,
		},
		{
			// withdrawn: two vantages, both suppress (NameError + Shadowed) — cross-class
			// unanimity, so the Name leaves.
			name: "measured-absent cross-vantage unanimous suppression",
			open: []db.ListOpenSpansForSubjectRow{
				resolutionSpan(1, pgtype.Int8{Int64: 10, Valid: true}, estateOutcomeNameError),
				resolutionSpan(2, pgtype.Int8{Int64: 20, Valid: true}, estateOutcomeShadowed),
			},
			wantReason: drift.ReasonMeasuredAbsent,
			wantLeft:   true,
		},
		{
			// stays: one vantage still admits the Name (Resolved) — presence is
			// existential across vantages, so no withdrawal.
			name: "one admitting vantage keeps the name",
			open: []db.ListOpenSpansForSubjectRow{
				resolutionSpan(1, pgtype.Int8{Int64: 10, Valid: true}, estateOutcomeNameError),
				resolutionSpan(2, pgtype.Int8{Int64: 20, Valid: true}, estateOutcomeResolved),
			},
			wantLeft: false,
		},
		{
			// stays: a Gap is not-evaluable and blocks withdrawal even alongside a
			// suppressing vantage.
			name: "gap blocks withdrawal",
			open: []db.ListOpenSpansForSubjectRow{
				resolutionSpan(1, pgtype.Int8{Int64: 10, Valid: true}, estateOutcomeNameError),
				resolutionSpan(2, pgtype.Int8{Int64: 20, Valid: true}, "Gap"),
			},
			wantLeft: false,
		},
		{
			// descoped: an operator exclusion draws the boundary inward — the Name leaves
			// `descoped` even though its resolution still admits it.
			name:       "descoped by exclusion beats a resolving name",
			open:       []db.ListOpenSpansForSubjectRow{resolutionSpan(1, pgtype.Int8{}, estateOutcomeResolved)},
			excluded:   true,
			wantReason: drift.ReasonDescoped,
			wantLeft:   true,
		},
		{
			// stays: declared input (a Seed) keeps the Name regardless of a suppressing
			// resolution — the estate does not withdraw a Name the operator still declares.
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

// Kept for readability of the table above — the estate outcome tags the composition
// reads, mirrored from internal/estate's constants.
const (
	estateOutcomeResolved  = "Resolved"
	estateOutcomeNameError = "NameError"
	estateOutcomeShadowed  = "Shadowed"
)

// openedByAperture is the entry half of the estate wiring: a Seed-declared subject's
// first timeline opens `revealed` (the aperture is why we looked), a subject no Seed
// declares opens `appeared`. Each subject kind fires from a representative Seed.
func TestOpenedByAperture(t *testing.T) {
	in := membershipInputs{seeds: []db.ListSeedsRow{nameSeed("example.com"), addressSeed("198.51.100.0/24")}}

	cases := []struct {
		kind, key string
		want      bool
	}{
		{"name", "api.example.com", true},       // revealed: beneath the declared name Seed
		{"name", "example.com", true},           // revealed: the declared apex itself
		{"name", "notexample.com", false},       // appeared: a look-alike outside the Seed
		{"name", "discovered.other.net", false}, // appeared: the world brought it, no Seed declares it
		{"address", "198.51.100.7", true},       // revealed: inside the declared address Seed
		{"address", "203.0.113.7", false},       // appeared: outside every address Seed
		{"service", "198.51.100.7:443", true},   // revealed: the Service rides a declared Address
		{"service", "203.0.113.7:443", false},   // appeared: the Service's Address is undeclared
	}
	for _, c := range cases {
		if got := openedByAperture(c.kind, c.key, in); got != c.want {
			t.Errorf("openedByAperture(%q, %q) = %v, want %v", c.kind, c.key, got, c.want)
		}
	}
}

// An Exclusion cuts the subject back OUT of the Declared aperture, so an excluded
// subject is unmarked even where a Seed scope still contains it (#1039). The case is
// ordinary: an exclusion cuts the `Seed` limb alone, so an excluded Address a custody
// extension reaches is still probed and still opens a span (ADR-0133 §3). The world's
// resolution is why we looked at that one, so it opens `appeared`, and a later
// re-entry across its `descoped` closure must not read `revealed` (drift.ReEntryKind).
func TestOpenedByApertureExcludedSubjectIsUnmarked(t *testing.T) {
	in := membershipInputs{
		seeds:      []db.ListSeedsRow{nameSeed("example.com"), addressSeed("198.51.100.0/24")},
		exclusions: []db.ListExclusionsRow{nameExclusion("api.example.com"), addressExclusion("198.51.100.0/28")},
	}

	cases := []struct {
		kind, key string
		want      bool
	}{
		{"name", "api.example.com", false},     // excluded by name, though the Seed declares it
		{"name", "other.example.com", true},    // the exclusion covers one name only
		{"address", "198.51.100.7", false},     // inside the excluded /28
		{"address", "198.51.100.90", true},     // inside the Seed, outside the exclusion
		{"service", "198.51.100.7:443", false}, // the Service rides an excluded Address
		{"service", "198.51.100.90:443", true},
	}
	for _, c := range cases {
		if got := openedByAperture(c.kind, c.key, in); got != c.want {
			t.Errorf("openedByAperture(%q, %q) = %v, want %v", c.kind, c.key, got, c.want)
		}
	}

	// A subtree exclusion cuts the whole branch back out, the same guard nameExcluded
	// applies on the way out.
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

// nameWithinDomain gates subtree coverage on a label boundary, so a look-alike name
// is not read as within the domain — the guard both Seed coverage and subtree
// exclusion share.
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

// nameExcluded distinguishes an exact `name` exclusion (the FQDN alone) from a
// `subtree` exclusion (the name and everything beneath).
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

// resolutionWitnesses groups a Name's open resolution spans into one witness per
// vantage class, ignoring non-resolution facets, so the withdrawal composition sees
// exactly the resolution evidence.
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

// observedResolutionNames is the distinct set of Names a batch's resolution
// observations carry, in first-seen order — the Names whose membership the fold
// re-composes; other facets do not re-decide a Name's membership.
func TestObservedResolutionNames(t *testing.T) {
	obs := []wire.Observation{
		{Facet: resolutionwalk.FacetResolution, Subject: "a.example.com"},
		{Facet: "dns-record", Subject: "a.example.com"},
		{Facet: resolutionwalk.FacetResolution, Subject: "b.example.com"},
		{Facet: resolutionwalk.FacetResolution, Subject: "a.example.com"}, // dup
		{Facet: resolutionwalk.FacetResolution, Subject: ""},              // no subject
	}
	got := observedResolutionNames(obs)
	want := []string{"a.example.com", "b.example.com"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("observedResolutionNames = %v, want %v", got, want)
	}
}
