package queue

import (
	"context"
	"net/netip"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

const dialledEdited = "192.0.2.7"

func editedReachRow(svc, dialled, outcome string, vantage int64) db.ListServiceReachabilitySpansByClassForServicesRow {
	return db.ListServiceReachabilitySpansByClassForServicesRow{
		SubjectKey: svc, VantageID: pgtype.Int8{Int64: vantage, Valid: true},
		Value: reachValue(outcome), DialledAddr: pgtype.Text{String: dialled, Valid: true},
	}
}

func scopeEditStore(svc string, scopes []*netip.Prefix) *fakeMessageStore {
	f := &fakeMessageStore{
		prev:          pgtype.Timestamptz{Time: produceT0.Add(-time.Hour), Valid: true},
		addressScopes: scopes,
		vantages: []db.ListVantagesForDispatchRow{
			{ID: 1, DialledAddr: pgtype.Text{String: dialledInternet, Valid: true}},
			{ID: 2, DialledAddr: pgtype.Text{String: dialledEdited, Valid: true}},
		},
	}
	rows := []db.ListServiceReachabilitySpansByClassForServicesRow{
		editedReachRow(svc, dialledInternet, "not-reached", 1),
		editedReachRow(svc, dialledEdited, "reached", 2),
	}
	for _, r := range rows {
		// The same rows on both legs, so any disagreement is the classifier's (ADR-1895 §4).
		f.current = append(f.current, r)
		f.at = append(f.at, db.ListServiceReachabilitySpansByClassAtForServicesRow(r))
	}
	return f
}

func classesOf(legs []classLeg) []string {
	out := make([]string, 0, len(legs))
	for _, l := range legs {
		out = append(out, l.subject+"/"+l.outcome+"="+l.class)
	}
	sort.Strings(out)
	return out
}

func TestReadBatchLegsClassifiesBothLegsUnderOneBinding(t *testing.T) {
	const svc = "203.0.113.10:443/tcp"
	scope := netip.MustParsePrefix("192.0.2.0/24")
	wide := netip.MustParsePrefix("10.0.0.0/8")

	cases := []struct {
		name   string
		scopes []*netip.Prefix
		want   []string
	}{
		{"before the edit", nil, []string{
			svc + "/not-reached=internet", svc + "/reached=internet",
		}},
		{"after the edit", []*netip.Prefix{&wide, &scope}, []string{
			svc + "/not-reached=internet", svc + "/reached=internal",
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := scopeEditStore(svc, c.scopes)
			legs, err := readBatchLegs(context.Background(), store, []spanChange{reachOpening(svc)})
			if err != nil {
				t.Fatalf("readBatchLegs: %v", err)
			}
			cur, prev := classesOf(legs.cur), classesOf(legs.prev)
			if len(cur) != len(c.want) {
				t.Fatalf("cur legs = %v, want %v", cur, c.want)
			}
			for i := range cur {
				if cur[i] != c.want[i] {
					t.Errorf("cur leg %d = %q, want %q", i, cur[i], c.want[i])
				}
				if prev[i] != cur[i] {
					t.Errorf("prev leg %d = %q but cur reads %q; one binding serves both",
						i, prev[i], cur[i])
				}
			}
		})
	}
}

func TestAScopeEditAloneFiresNoFlagship(t *testing.T) {
	const svc = "203.0.113.10:443/tcp"
	scope := netip.MustParsePrefix("192.0.2.0/24")
	wide := netip.MustParsePrefix("10.0.0.0/8")

	for _, c := range []struct {
		name   string
		scopes []*netip.Prefix
	}{
		{"before the edit", nil},
		{"after the edit", []*netip.Prefix{&wide, &scope}},
	} {
		t.Run(c.name, func(t *testing.T) {
			store := scopeEditStore(svc, c.scopes)
			changes := []spanChange{reachOpening(svc)}
			err := produceMessages(context.Background(), store, 7, produceT0, changes, nil, nil,
				membershipInputs{}, fakeEnqueuer(1, &[]routed{}), false, true)
			if err != nil {
				t.Fatalf("produce: %v", err)
			}
			for _, m := range store.inserted {
				if m.SubjectKind == "service" {
					t.Errorf("a scope edit carried no measurement, so no flagship fires; got %+v", m)
				}
			}
		})
	}
}
