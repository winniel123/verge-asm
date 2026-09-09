package queue

import (
	"context"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/vantageclass"
)

// The first vantage of a class opens the Exposure timelines it composes (ADR-0017 #58).

func vantageClassMessages(ctx context.Context, store messageStore, batchID int64, observedAt time.Time, changes []spanChange, legs batchLegs) ([]*message.Message, error) {
	if !openedReachLeg(changes) {
		return nil, nil
	}
	// The bounded prev read calls every class of a fresh Service new, so it only shortlists.
	candidates := widenedClasses(legs.cur, legs.prev)
	if len(candidates) == 0 {
		return nil, nil
	}
	vantages, err := store.ListVantagesForDispatch(ctx)
	if err != nil {
		return nil, err
	}
	byClass := map[string][]int64{}
	for _, v := range vantages {
		c := string(vantageclass.Derive(v.DialledAddr.String, v.Egress.String, legs.covered))
		byClass[c] = append(byClass[c], v.ID)
	}
	// The census is what this fold could see, so it stays on the batch's legs (spec §7).
	started := exposuresStarted(legs.cur, legs.prev)
	var msgs []*message.Message
	for _, class := range candidates {
		// A batch row is committed scope, so it orders what two span timestamps cannot (ADR-0014).
		folded, err := store.ReachFoldedBeforeAtVantages(ctx, db.ReachFoldedBeforeAtVantagesParams{
			Kinds:         []string{scan.HotKind, scan.ColdKind},
			VantageIds:    byClass[class],
			BeforeBatchID: batchID,
		})
		if err != nil {
			return nil, err
		}
		if folded {
			continue
		}
		if m := message.VantageClassWidened(class, started, observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func openedReachLeg(changes []spanChange) bool {
	for _, c := range changes {
		if c.Opened && c.Facet == connectoutcome.FacetReachability {
			return true
		}
	}
	return false
}

func widenedClasses(cur, prev []classLeg) []string {
	before := map[string]bool{}
	for _, l := range prev {
		before[l.class] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, l := range cur {
		// An unverified vantage presents no address, so it starts no leg (exposure.VerifyClass).
		if l.class != string(custody.ClassInternet) && l.class != string(custody.ClassInternal) {
			continue
		}
		if before[l.class] || seen[l.class] {
			continue
		}
		seen[l.class] = true
		out = append(out, l.class)
	}
	return out
}

func exposuresStarted(cur, prev []classLeg) []message.CensusEntry {
	now, before := legsBySubject(cur), legsBySubject(prev)
	var out []message.CensusEntry
	seen := map[string]bool{}
	for _, l := range cur {
		if seen[l.subject] {
			continue
		}
		seen[l.subject] = true
		after, ok := composeExposure(now[l.subject], l.subject)
		if !ok {
			continue
		}
		if _, had := composeExposure(before[l.subject], l.subject); had {
			continue
		}
		out = append(out, message.CensusEntry{Kind: message.KindService, Key: l.subject, Detail: string(after)})
	}
	return out
}

func legsBySubject(legs []classLeg) map[string][]classLeg {
	out := make(map[string][]classLeg, len(legs))
	for _, l := range legs {
		out[l.subject] = append(out[l.subject], l)
	}
	return out
}

func composeExposure(legs []classLeg, service string) (exposure.ExposureValue, bool) {
	internet, iok := composeLeg(legs, service, string(custody.ClassInternet))
	internal, nok := composeLeg(legs, service, string(custody.ClassInternal))
	if !iok || !nok {
		return "", false
	}
	return exposure.Project(
		exposure.Leg{Status: exposure.LegValued, Value: internet},
		exposure.Leg{Status: exposure.LegValued, Value: internal})
}
