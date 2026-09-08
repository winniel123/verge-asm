package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

func (f *fakeStore) ListVantagesForDispatch(context.Context) ([]db.ListVantagesForDispatchRow, error) {
	rows := make([]db.ListVantagesForDispatchRow, 0, len(f.vantages))
	for _, v := range f.vantages {
		rows = append(rows, db.ListVantagesForDispatchRow{
			ID: v.ID, Name: v.Name, Class: v.Class, Resolver: v.Resolver,
			Egress: v.Egress, DialledAddr: v.DialledAddr, CreatedAt: v.CreatedAt,
		})
	}
	return rows, nil
}

func TestComposeResolutionCrossClass(t *testing.T) {
	both := []string{"internal", "internet"}
	res := func(outcome string, addrs ...string) resolutionValue {
		return resolutionValue{Outcome: outcome, Addresses: addrs}
	}
	cases := []struct {
		name    string
		classes map[string]resolutionValue
		running []string
		lame    bool
		want    composedResolution
	}{
		{
			name:    "split horizon disagrees",
			classes: map[string]resolutionValue{"internet": res(signal.NameError), "internal": res(signal.Resolved, "10.0.0.5")},
			running: both,
			want:    composedResolution{outcome: signal.ResolutionNotEvaluable, inEstate: true},
		},
		{
			name:    "a running class with no current value",
			classes: map[string]resolutionValue{"internet": res(signal.Resolved, "203.0.113.5")},
			running: both,
			want:    composedResolution{outcome: signal.ResolutionNotEvaluable, inEstate: true},
		},
		{
			name:    "agreement unions the address sets",
			classes: map[string]resolutionValue{"internet": res(signal.Resolved, "203.0.113.5"), "internal": res(signal.Resolved, "10.0.0.5")},
			running: both,
			want:    composedResolution{outcome: signal.Resolved, addresses: []string{"10.0.0.5", "203.0.113.5"}, inEstate: true},
		},
		{
			name:    "agreed NameError withdraws",
			classes: map[string]resolutionValue{"internet": res(signal.NameError), "internal": res(signal.NameError)},
			running: both,
			want:    composedResolution{outcome: signal.NameError, inEstate: false},
		},
		{
			name:    "single class, lame delegation over a gap",
			classes: map[string]resolutionValue{"internet": res(signal.Gap)},
			running: []string{"internet"},
			lame:    true,
			want:    composedResolution{outcome: signal.Lame, inEstate: true},
		},
		{
			name:    "single class shadowed",
			classes: map[string]resolutionValue{"internet": res(signal.Shadowed)},
			running: []string{"internet"},
			want:    composedResolution{outcome: signal.Shadowed, inEstate: false},
		},
		{
			name:    "never observed anywhere",
			classes: nil,
			running: both,
			want:    composedResolution{outcome: signal.ResolutionNotEvaluable, inEstate: false},
		},
		{
			name:    "no vantage at all",
			classes: nil,
			running: nil,
			want:    composedResolution{outcome: signal.ResolutionNotEvaluable, inEstate: false},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := composeResolution(c.classes, c.running, c.lame)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("composeResolution = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestSplitHorizonNameRulesReadNotEvaluable(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedZone(t, f, admin, "example.com", "$ORIGIN example.com.\n@ IN SOA ns1 admin 1 2 3 4 5\nwww IN A 203.0.113.20\n")

	const (
		absent   = "ghost.example.com"
		declared = "www.example.com"
		lonely   = "only.example.com"
		agreed   = "agree.example.com"
	)
	f.addClassResolution(t, absent, "internet", obsClock, `{"outcome":"NameError"}`)
	f.addClassResolution(t, absent, "internal", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)
	f.addClassResolution(t, declared, "internet", obsClock, `{"outcome":"NameError"}`)
	f.addClassResolution(t, declared, "internal", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.6"]}`)
	f.addClassResolution(t, lonely, "internal", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.7"]}`)
	f.addClassResolution(t, agreed, "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.8"]}`)
	f.addClassResolution(t, agreed, "internal", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.8"]}`)

	srv := &server{signalsStore: f, vantageClassStore: f, now: fixedClock()}
	req := httptest.NewRequest(http.MethodGet, "/signals", nil)
	facts := nameFactsByKey(mustNameFacts(t, srv, req))

	absentRule := nameRule(t, "resolved-name-absent-from-zone")
	declaredRule := nameRule(t, "zone-declared-name-returns-name-error")

	if got := absentRule.Eval(facts[absent]); got != signal.NotEvaluable {
		t.Errorf("%s on split horizon = %v, want not-evaluable (%+v)", absentRule.Name(), got, facts[absent])
	}
	if got := declaredRule.Eval(facts[declared]); got != signal.NotEvaluable {
		t.Errorf("%s on split horizon = %v, want not-evaluable (%+v)", declaredRule.Name(), got, facts[declared])
	}
	if got := absentRule.Eval(facts[lonely]); got != signal.NotEvaluable {
		t.Errorf("%s with a running class holding no value = %v, want not-evaluable (%+v)", absentRule.Name(), got, facts[lonely])
	}
	if got := absentRule.Eval(facts[agreed]); got != signal.Fired {
		t.Errorf("%s on cross-class agreement = %v, want fired (%+v)", absentRule.Name(), got, facts[agreed])
	}
	if !facts[absent].InEstate {
		t.Errorf("a split-horizon name is not withdrawn: %+v", facts[absent])
	}
}
