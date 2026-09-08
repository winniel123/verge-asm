package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) addNamedVantage(name, class string) int64 {
	v := db.Vantage{ID: f.vantageNextID, Name: name, Class: class, DialledAddr: classPresentedDialled(class)}
	f.vantages = append(f.vantages, v)
	f.vantageNextID++
	return v.ID
}

func TestTimelineLabelsNameEachVantage(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	eu := f.addNamedVantage("eu-west", "internet")
	us := f.addNamedVantage("us-east", "internet")
	const key = "198.51.100.1:443/tcp"
	f.observeFromVantage(t, eu, key, obsClock, `{"outcome":"reached","result":"open"}`)
	f.observeFromVantage(t, us, key, obsClock, `{"outcome":"not-reached","result":"refused"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)
	for _, want := range []string{"reachability · eu-west · prober", "reachability · us-east · prober"} {
		if !strings.Contains(drill, want) {
			t.Errorf("timeline label %q missing; body: %s", want, drill)
		}
	}
}

func TestTimelineLabelsDistinguishSources(t *testing.T) {
	row := func(source string) db.ListSpansForSubjectRow {
		return db.ListSpansForSubjectRow{
			Facet: "dns-record", Discriminator: "A", Source: source,
			VantageID: pgtype.Int8{Int64: 1, Valid: true}, VantageName: pgtype.Text{String: "local", Valid: true},
			Value: []byte(`{"rrs":["203.0.113.7"]}`), OpenedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
		}
	}
	resolver := buildTimeline("dns-record", "A", []db.ListSpansForSubjectRow{row("resolver")})
	zone := buildTimeline("dns-record", "A", []db.ListSpansForSubjectRow{row("zone")})
	if resolver.Label == zone.Label {
		t.Fatalf("two sources on one facet render identically as %q", resolver.Label)
	}
	if resolver.Label != "dns-record · A · local · resolver" || zone.Label != "dns-record · A · local · zone" {
		t.Errorf("labels = %q, %q; want each to name its source", resolver.Label, zone.Label)
	}
	if resolver.VantageID != 1 || resolver.Vantage != "local" || resolver.Source != "resolver" {
		t.Errorf("timeline view carries %d/%q/%q; want 1/local/resolver", resolver.VantageID, resolver.Vantage, resolver.Source)
	}

	unnamed := buildTimeline("dns-record", "A", []db.ListSpansForSubjectRow{{
		Facet: "dns-record", Discriminator: "A", Source: "resolver",
		VantageID: pgtype.Int8{Int64: 7, Valid: true},
		Value:     []byte(`{}`), OpenedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
	}})
	if unnamed.Label != "dns-record · A · vantage 7 · resolver" {
		t.Errorf("an unnamed vantage renders as %q; want its id", unnamed.Label)
	}
}

func TestSignalDrawerDriftTitleNamesVantage(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	eu := f.addNamedVantage("eu-west", "internet")
	us := f.addNamedVantage("us-east", "internet")
	const key = "198.51.100.1:443/tcp"
	f.observeFromVantage(t, eu, key, obsClock, `{"outcome":"not-reached","result":"refused"}`)
	f.observeFromVantage(t, us, key, obsClock, `{"outcome":"not-reached","result":"refused"}`)
	f.observeFromVantage(t, eu, key, obsClock.Add(24*time.Hour), `{"outcome":"reached","result":"open"}`)
	f.observeFromVantage(t, us, key, obsClock.Add(48*time.Hour), `{"outcome":"reached","result":"open"}`)
	srv := newServer(f, testKey, "", fixedClock())

	diff := srv.signalDrift(httptest.NewRequest(http.MethodGet, "/signals", nil), "service", key)
	if diff == nil {
		t.Fatal("no drift diff for a service that transitioned")
	}
	if diff.Title != "reachability · us-east · prober · drift" {
		t.Errorf("drift title = %q; want the latest transition's vantage and source named", diff.Title)
	}
}

func TestAssetDriftKeepsTheFacetSubject(t *testing.T) {
	tl := buildTimeline("dns-record", "A", []db.ListSpansForSubjectRow{{
		Facet: "dns-record", Discriminator: "A", Source: "resolver",
		VantageID: pgtype.Int8{Int64: 1, Valid: true}, VantageName: pgtype.Text{String: "local", Valid: true},
		Value: []byte(`{"rrs":["203.0.113.7"]}`), OpenedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
	}})
	events := assetDrift([]timelineView{tl})
	if len(events) != 1 || events[0].Subject != "dns-record · A" {
		t.Fatalf("asset drift events = %+v; want one event with subject dns-record · A", events)
	}
}
