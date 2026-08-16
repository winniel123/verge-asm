package message

import "testing"

// A membership message fires on a Name or an Address root, carrying the census
// of what entered beneath it.
func TestMembershipFiresAtNameOrAddressRoot(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "203.0.113.5:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "example.com|203.0.113.5:443/tcp"},
	)
	for _, kind := range []string{"name", "address"} {
		msg := Membership(EntryAppeared, kind, "root-key", "", census, t0)
		if msg == nil {
			t.Fatalf("a %s root must fire a membership message", kind)
		}
		if msg.CensusLen() != 2 {
			t.Errorf("the membership message carries the census of what entered beneath, got %d", msg.CensusLen())
		}
	}
}

// A Service or Endpoint entering the estate is never itself a message — it rides
// the root membership census, so no root message fires for it and it appears only
// in the census of the root above it.
func TestEnteringServiceOrEndpointRidesCensusNotOwnMessage(t *testing.T) {
	for _, kind := range []string{"service", "endpoint"} {
		if RootFires(kind) {
			t.Errorf("%s must never be a membership root", kind)
		}
		if got := Membership(EntryAppeared, kind, "203.0.113.5:443/tcp", "", NewCensus(), t0); got != nil {
			t.Errorf("a %s entering must not fire its own message, got %+v", kind, got)
		}
	}

	// It appears in the root's census instead.
	census := NewCensus(
		CensusEntry{Kind: "service", Key: "203.0.113.5:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "example.com|203.0.113.5:443/tcp"},
	)
	root := Membership(EntryAppeared, "name", "example.com", "", census, t0)
	if root == nil || root.CensusLen() != 2 {
		t.Fatal("the entering Service and Endpoint ride the root membership census")
	}
	kinds := map[string]bool{}
	for _, e := range root.Census.Entries {
		kinds[e.Kind] = true
	}
	if !kinds["service"] || !kinds["endpoint"] {
		t.Error("the census must enumerate the entering Service and Endpoint")
	}
}

// The Entry decides cause and class: appeared/returned are drift (the world
// moved), revealed is coverage (a widened aperture — a first run).
func TestMembershipEntryDecidesClass(t *testing.T) {
	cases := map[Entry]struct {
		cause Cause
		class Class
	}{
		EntryAppeared: {CauseDrift, ClassDrift},
		EntryReturned: {CauseDrift, ClassDrift},
		EntryRevealed: {CauseAperture, ClassCoverage},
	}
	for entry, want := range cases {
		msg := Membership(entry, "name", "example.com", "198.51.100.0/24", NewCensus(), t0)
		if msg == nil {
			t.Fatalf("entry %q must fire", entry)
		}
		if msg.Cause != want.cause || msg.Class != want.class {
			t.Errorf("entry %q => cause=%q class=%q, want %q/%q",
				entry, msg.Cause, msg.Class, want.cause, want.class)
		}
	}
}

// A revealed (aperture) membership fires at the Seed whose scope moved, not the
// entering subject: its row must link to that Seed (§5.3). The entering root
// still names the headline and rides the census; the fired-at is the Seed.
func TestRevealedMembershipFiresAtSeed(t *testing.T) {
	const seedKey = "198.51.100.0/24"
	msg := Membership(EntryRevealed, "name", "example.com", seedKey, NewCensus(), t0)
	if msg == nil {
		t.Fatal("a revealed membership must fire")
	}
	if msg.SubjectKind != "seed" || msg.FiredAt != seedKey {
		t.Errorf("revealed membership fired at %q/%q, want seed/%q", msg.SubjectKind, msg.FiredAt, seedKey)
	}
	if msg.LinkKind() != LinkSeed {
		t.Errorf("revealed membership links to %q, want %q", msg.LinkKind(), LinkSeed)
	}
	// appeared/returned still fire at the moved object itself.
	drift := Membership(EntryAppeared, "name", "example.com", "", NewCensus(), t0)
	if drift.SubjectKind != "name" || drift.FiredAt != "example.com" {
		t.Errorf("appeared membership fired at %q/%q, want name/example.com", drift.SubjectKind, drift.FiredAt)
	}
}
