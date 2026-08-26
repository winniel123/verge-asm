package main

import (
	"reflect"
	"strconv"
	"testing"
)

// segsToHi converts a fixture's authored segments to the handler's hiSeg shape so the
// two can be deep-compared.
func segsToHi(segs []searchSeg) []hiSeg {
	out := make([]hiSeg, 0, len(segs))
	for _, s := range segs {
		out = append(out, hiSeg{Text: s.Text, Hit: s.Hit})
	}
	return out
}

// TestBuildSearchMatchesDesignFixture is the byte-exactness gate BEFORE the pixel
// harness: it folds every matched field of the fixtures.json search slice through the
// live #25a builder (searchSegs, over the reconstructed raw text) and deep-asserts the
// result equals the authored segmentation. A drift between the FIRST-case-insensitive
// -match rule and the frozen fixture fails here — a real collision — rather than in a
// screenshot diff. It also asserts the group counts fold to the authored total and the
// empty variant is the honest zero.
func TestBuildSearchMatchesDesignFixture(t *testing.T) {
	fx := loadSearchFixture()
	if fx.Query == "" {
		t.Fatal("search fixture did not load (empty query) — fixtures.json search slice missing")
	}
	q := fx.Query

	// check re-segments a field through the builder and deep-asserts the authored segs.
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

	// The group counts fold to the authored total (assets + signals + batches + docs).
	got := len(fx.Assets) + len(fx.Signals) + len(fx.Batches) + len(fx.Docs)
	if got != fx.Total {
		t.Errorf("group counts fold to %d, want authored total %d", got, fx.Total)
	}

	// The empty variant is the honest zero-result state.
	if fx.EmptyVariant.Query == "" || fx.EmptyVariant.Total != 0 {
		t.Errorf("empty variant = %+v, want a non-empty query with total 0", fx.EmptyVariant)
	}
}
