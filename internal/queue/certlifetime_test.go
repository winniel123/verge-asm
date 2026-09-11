package queue

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
)

func ninetyDayCert(notAfter time.Time) string {
	nb := notAfter.Add(-90 * oneDay)
	return `{"outcome":"presented","not_before":"` + nb.Format(time.RFC3339) + `","not_after":"` + notAfter.Format(time.RFC3339) + `",` +
		`"san_dns":["admin.example.com"],"chain_certs":[{"subject":"CN=admin","issuer":"CN=ca","self_sig_verifies":false,"key_alg":"RSA","key_bits":2048,"sig_digest":"SHA-256"}]}`
}

func batchWindow(prev, latest time.Time) db.FoldedBatchWindowRow {
	return db.FoldedBatchWindowRow{PrevAt: prevAt(prev), LatestAt: prevAt(latest)}
}

func certSpan(key string, vantage int64, value string, observed time.Time) db.ListOpenEndpointCertificateSpansRow {
	return db.ListOpenEndpointCertificateSpansRow{
		SubjectKey: key, VantageID: pgtype.Int8{Int64: vantage, Valid: true},
		Value: []byte(value), ObservedAt: prevAt(observed),
	}
}

func clockMessagesOf(store *fakeMessageStore) []db.InsertMessageParams {
	var out []db.InsertMessageParams
	for _, m := range store.inserted {
		if m.Cause == string(message.CauseThreshold) {
			out = append(out, m)
		}
	}
	return out
}

func runFold(t *testing.T, store *fakeMessageStore, observedAt time.Time, changes []spanChange, devMode bool) []routed {
	t.Helper()
	var log []routed
	if err := produceMessages(context.Background(), store, 40, observedAt, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), devMode, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	return log
}

const oneDay = 24 * time.Hour

var (
	lifetimeNotAfter = time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	lifetimeCrossing = lifetimeNotAfter.Add(-30 * oneDay)
)

func TestCertificateLifetimeFiresOnceWhenTheHorizonCrossesInsideTheWindow(t *testing.T) {
	cert := ninetyDayCert(lifetimeNotAfter)
	cases := []struct {
		name         string
		prev, latest time.Time
		want         int
	}{
		{"window ends before the crossing", lifetimeCrossing.Add(-36 * time.Hour), lifetimeCrossing.Add(-12 * time.Hour), 0},
		{"window holds the crossing", lifetimeCrossing.Add(-12 * time.Hour), lifetimeCrossing.Add(12 * time.Hour), 1},
		{"window opens after the crossing", lifetimeCrossing.Add(12 * time.Hour), lifetimeCrossing.Add(36 * time.Hour), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := &fakeMessageStore{
				window:    batchWindow(c.prev, c.latest),
				certSpans: []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, cert, c.latest)},
			}
			log := runFold(t, store, c.latest, nil, false)
			got := clockMessagesOf(store)
			if len(got) != c.want {
				t.Fatalf("want %d clock message(s), got %d: %+v", c.want, len(got), store.inserted)
			}
			if c.want == 0 {
				if store.annotationReads != 0 {
					t.Errorf("no edge, so the dial is never read; got %d reads", store.annotationReads)
				}
				return
			}
			m := got[0]
			if m.Class != string(message.ClassClock) || m.SubjectKind != "endpoint" || m.FiredAt != sensitiveEp {
				t.Errorf("an unchanged span's crossing is clock class at the endpoint; got %+v", m)
			}
			if !strings.Contains(m.Headline, "no measurement moved") || !strings.Contains(m.Headline, "certificate-expiring now fired") {
				t.Errorf("the headline states that no measurement moved and names the rule: %q", m.Headline)
			}
			if !m.Instant.Valid || !m.Instant.Time.Equal(c.latest) {
				t.Errorf("the instant is the fold's, got %v", m.Instant)
			}
			if len(log) != 1 || log[0].class != message.ClassClock {
				t.Errorf("routed once on the clock class, got %+v", log)
			}
		})
	}
}

func TestCertificateLifetimeFiresExpiredAndNotTheClearingOfExpiring(t *testing.T) {
	notAfter := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	store := &fakeMessageStore{
		window:    batchWindow(notAfter.Add(-12*time.Hour), notAfter.Add(12*time.Hour)),
		certSpans: []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, ninetyDayCert(notAfter), notAfter.Add(12*time.Hour))},
	}
	runFold(t, store, notAfter.Add(12*time.Hour), nil, false)
	got := clockMessagesOf(store)
	if len(got) != 1 {
		t.Fatalf("expiring clears silently and expired fires: want 1, got %d: %+v", len(got), store.inserted)
	}
	if !strings.Contains(got[0].Headline, "certificate-expired now fired") || !strings.Contains(got[0].Headline, "not_after 2026-08-30 has passed") {
		t.Errorf("headline: %q", got[0].Headline)
	}
}

func TestCertificateLifetimeNeedsAPreviousFoldedBatch(t *testing.T) {
	store := &fakeMessageStore{
		window:    db.FoldedBatchWindowRow{LatestAt: prevAt(lifetimeCrossing.Add(12 * time.Hour))},
		certSpans: []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, ninetyDayCert(lifetimeNotAfter), lifetimeCrossing)},
	}
	runFold(t, store, lifetimeCrossing.Add(12*time.Hour), nil, false)
	if len(store.inserted) != 0 {
		t.Errorf("a first batch has no decided before, got %+v", store.inserted)
	}
	if store.certReads != 0 {
		t.Errorf("no window, so no certificate read; got %d", store.certReads)
	}
	if want := []string{scan.ZoneKind, scan.CTKind, scan.CTTailKind}; !reflect.DeepEqual(store.unfoldedKinds, want) {
		t.Errorf("the window skips the kinds that never fold a message; got %v", store.unfoldedKinds)
	}
}

func TestCertificateLifetimeDeclinesAnObservationOlderThanItsHorizon(t *testing.T) {
	latest := lifetimeCrossing.Add(12 * time.Hour)
	store := &fakeMessageStore{
		window:    batchWindow(lifetimeCrossing.Add(-12*time.Hour), latest),
		certSpans: []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, ninetyDayCert(lifetimeNotAfter), latest.Add(-60*oneDay))},
	}
	runFold(t, store, latest, nil, false)
	if len(store.inserted) != 0 {
		t.Errorf("an observation past its horizon is not-evaluable (ADR-0043): %+v", store.inserted)
	}
}

func TestCertificateLifetimeReadsTheFirstVantageOnly(t *testing.T) {
	latest := lifetimeCrossing.Add(12 * time.Hour)
	cert := ninetyDayCert(lifetimeNotAfter)
	store := &fakeMessageStore{
		window: batchWindow(lifetimeCrossing.Add(-12*time.Hour), latest),
		certSpans: []db.ListOpenEndpointCertificateSpansRow{
			certSpan(sensitiveEp, 1, cert, latest),
			certSpan(sensitiveEp, 2, cert, latest),
		},
	}
	runFold(t, store, latest, nil, false)
	if got := clockMessagesOf(store); len(got) != 1 {
		t.Errorf("one endpoint, one crossing, one message (ADR-0080): %+v", got)
	}
}

func TestCertificateLifetimeReadsAMovedSpanAsDrift(t *testing.T) {
	latest := produceT0
	changes := []spanChange{
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certValid)},
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certValid)},
	}
	store := &fakeMessageStore{
		window:    batchWindow(latest.Add(-oneDay), latest),
		certSpans: []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, certExpired, latest)},
	}
	log := runFold(t, store, latest, changes, false)
	if got := clockMessagesOf(store); len(got) != 0 {
		t.Errorf("the certificate span moved, so no firing is clock class: %+v", got)
	}
	var expired []db.InsertMessageParams
	for _, m := range store.inserted {
		if strings.Contains(m.Headline, "certificate-expired now fired") {
			expired = append(expired, m)
		}
	}
	if len(expired) != 1 {
		t.Fatalf("one drift firing for certificate-expired, got %d: %+v", len(expired), store.inserted)
	}
	m := expired[0]
	if m.Cause != string(message.CauseDrift) || m.Class != string(message.ClassDrift) {
		t.Errorf("the world moved, so the firing is drift; got %s %s", m.Cause, m.Class)
	}
	if !strings.Contains(m.Headline, "certificate moved") {
		t.Errorf("a drift firing names the moved facet: %q", m.Headline)
	}
	if len(store.inserted) != 2 || len(log) != 2 {
		t.Errorf("certificate-self-signed crossed on the same move, so two messages: %+v", store.inserted)
	}
}

func TestCertificateLifetimeLeavesAnOpeningToTheFacetMoveCensus(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Opened: true, Value: []byte(certExpired)},
	}
	store := &fakeMessageStore{window: batchWindow(produceT0.Add(-oneDay), produceT0)}
	runFold(t, store, produceT0, changes, false)
	for _, m := range store.inserted {
		if strings.Contains(m.Headline, "now fired") {
			t.Errorf("an opening rides the move's census (ADR-0033 §3): %q", m.Headline)
		}
	}
}

func TestCertificateLifetimeMutesAnAnnotatedPair(t *testing.T) {
	latest := lifetimeCrossing.Add(12 * time.Hour)
	store := &fakeMessageStore{
		window:      batchWindow(lifetimeCrossing.Add(-12*time.Hour), latest),
		certSpans:   []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, ninetyDayCert(lifetimeNotAfter), latest)},
		annotations: []db.Annotation{{SubjectKey: sensitiveEp, SignalName: "certificate-expiring"}},
	}
	log := runFold(t, store, latest, nil, false)
	if len(store.inserted) != 0 || len(log) != 0 {
		t.Errorf("an annotated pair's edge is recorded and is not a message (ADR-0016): %+v", store.inserted)
	}
	if store.annotationReads != 1 {
		t.Errorf("the dial is read once, when an edge exists; got %d", store.annotationReads)
	}
}

func TestCertificateLifetimeNeverFiresNotYetValidFromTheClock(t *testing.T) {
	notBefore := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	cert := ninetyDayCert(notBefore.Add(90 * oneDay))
	for _, w := range []struct {
		name         string
		prev, latest time.Time
	}{
		{"fired at both instants", notBefore.Add(-2 * oneDay), notBefore.Add(-oneDay)},
		{"clears across not_before", notBefore.Add(-oneDay), notBefore.Add(oneDay)},
	} {
		t.Run(w.name, func(t *testing.T) {
			store := &fakeMessageStore{
				window:    batchWindow(w.prev, w.latest),
				certSpans: []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, cert, w.latest)},
			}
			runFold(t, store, w.latest, nil, false)
			if len(store.inserted) != 0 {
				t.Errorf("the clock only clears not-yet-valid, never fires it: %+v", store.inserted)
			}
		})
	}
}

func TestCertificateLifetimeIsSilentInDevMode(t *testing.T) {
	latest := lifetimeCrossing.Add(12 * time.Hour)
	store := &fakeMessageStore{
		window:    batchWindow(lifetimeCrossing.Add(-12*time.Hour), latest),
		certSpans: []db.ListOpenEndpointCertificateSpansRow{certSpan(sensitiveEp, 1, ninetyDayCert(lifetimeNotAfter), latest)},
	}
	runFold(t, store, latest, nil, true)
	if len(store.inserted) != 0 || store.certReads != 0 {
		t.Errorf("a devMode worker produces nothing and reads nothing (ADR-0197 §1)")
	}
}

func TestCertificateLifetimeReadsSpansOncePerFold(t *testing.T) {
	store := &fakeMessageStore{window: batchWindow(produceT0.Add(-oneDay), produceT0)}
	runFold(t, store, produceT0, nil, false)
	if store.certReads != 1 {
		t.Errorf("one span read per fold, got %d", store.certReads)
	}
}
