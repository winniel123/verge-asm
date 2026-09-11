package message

import (
	"encoding/json"
	"time"
)

// Revealed is aperture, not membership, and rides here as an entry (CONTEXT.md Transition).

type Entry string

const (
	EntryAppeared Entry = "appeared"
	EntryReturned Entry = "returned"
	EntryRevealed Entry = "revealed"
)

func RootFires(subjectKind string) bool {
	// A Service or Endpoint brings no ground the model was not already accounting for (ADR-0031).
	switch subjectKind {
	case "name", "address":
		return true
	default:
		return false
	}
}

func Membership(entry Entry, rootKind, rootKey, seedKey string, census Census, instant time.Time) *Message {
	if !RootFires(rootKind) {
		return nil
	}
	cause := CauseDrift
	subjectKind, firedAt := rootKind, rootKey
	if entry == EntryRevealed {
		// A widened aperture makes a first run one coverage-class message, with no special case.
		cause = CauseAperture
		// A revealed firing is about the Seed whose scope moved, not the entering subject (§5.3).
		subjectKind, firedAt = "seed", seedKey
	}
	c := census
	return &Message{
		Cause:       cause,
		Class:       ClassForCause(cause),
		SubjectKind: subjectKind,
		FiredAt:     firedAt,
		Instant:     instant,
		Census:      &c,
		Headline:    membershipHeadline(entry, rootKey, census),
	}
}

// A fold's own census is empty, so the count waits for the admitting tier (ADR-1806 §2, #1774).

func HeldMembership(entry Entry, rootKind, rootKey, seedKey string, pending CensusPending, instant time.Time) *Message {
	m := Membership(entry, rootKind, rootKey, seedKey, NewCensus(), instant)
	if m == nil {
		return nil
	}
	// The census and the clause it renders are both computed at release (ADR-1806 §2).
	m.Census = nil
	m.Headline = membershipCauseClause(entry, rootKey)
	p := pending
	m.CensusPending = &p
	return m
}

// The release poll reads what the fold saw, never live resolution (ADR-1806 §4).

type CensusPending struct {
	AfterBatch int64
	Basis      CensusBasis
}

// A revealed firing fires at the Seed, so the root is named here and read nowhere else (§5.3).

type CensusBasis struct {
	RootKind string `json:"root_kind"`
	RootKey  string `json:"root_key"`

	// The root span's own value at that batch answers "beneath the root" (ADR-1806 §4).

	RootValue json.RawMessage `json:"root_value,omitempty"`
}

func (b CensusBasis) Marshal() ([]byte, error) { return json.Marshal(b) }

func ParseCensusBasis(b []byte) (CensusBasis, error) {
	if len(b) == 0 {
		return CensusBasis{}, nil
	}
	var out CensusBasis
	if err := json.Unmarshal(b, &out); err != nil {
		return CensusBasis{}, err
	}
	return out, nil
}

// The cause clause is the fold's own bytes, so release appends and recomputes none (ADR-1806 §2).

func ReleasedMembershipHeadline(causeClause string, census Census) string {
	return causeClause + membershipCensusClause(census)
}
