package main

import (
	"reflect"
	"strconv"
	"testing"
)

func segsToHi(segs []searchSeg) []hiSeg {
	out := make([]hiSeg, 0, len(segs))
	for _, s := range segs {
		out = append(out, hiSeg{Text: s.Text, Hit: s.Hit})
	}
	return out
}

func TestBuildSearchMatchesDesignFixture(t *testing.T) {
	fx := loadSearchFixture()
	if fx.Query == "" {
		t.Fatal("search fixture did not load (empty query) — fixtures.json search slice missing")
	}
	q := fx.Query

	check := func(what string, want []searchSeg) {
		t.Helper()
		got := searchSegs(joinFixtureSegs(want), q)
		if !reflect.DeepEqual(got, segsToHi(want)) {
			t.Errorf("%s: searchSegs(%q, %q) = %#v, want %#v (the first-match rule does not reproduce the authored segmentation)",
				what, joinFixtureSegs(want), q, got, segsToHi(want))
		}
	}

	for i, a := range fx.Assets {
		check("assets["+strconv.Itoa(i)+"].name_segs", a.NameSegs)
	}
	for i, s := range fx.Signals {
		check("signals["+strconv.Itoa(i)+"].rule_segs", s.RuleSegs)
		check("signals["+strconv.Itoa(i)+"].subject_segs", s.SubjectSegs)
	}
	for i, b := range fx.Batches {
		check("batches["+strconv.Itoa(i)+"].label_segs", b.LabelSegs)
	}
	for i, d := range fx.Docs {
		check("docs["+strconv.Itoa(i)+"].title_segs", d.TitleSegs)
		check("docs["+strconv.Itoa(i)+"].snip_segs", d.SnipSegs)
	}

	got := len(fx.Assets) + len(fx.Signals) + len(fx.Batches) + len(fx.Docs)
	if got != fx.Total {
		t.Errorf("group counts fold to %d, want authored total %d", got, fx.Total)
	}

	if fx.EmptyVariant.Query == "" || fx.EmptyVariant.Total != 0 {
		t.Errorf("empty variant = %+v, want a non-empty query with total 0", fx.EmptyVariant)
	}
}
