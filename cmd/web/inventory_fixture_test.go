package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// fixtureInventory mirrors design-system/fixtures/fixtures.json → inventory. It is
// the design-owned contract the seeded /inventory MUST render byte-for-byte; this
// test folds the loader's exact spans (inventoryFixtureSpans) through buildInventory
// and asserts deep equality against it. It is the byte-exactness gate BEFORE the
// pixel harness — a drift between the loader corpus, the inventory derivations, and
// the frozen fixture fails here rather than in a screenshot diff.
type fixtureInventory struct {
	Inventory struct {
		Groups []fixtureGroup `json:"groups"`
	} `json:"inventory"`
}

type fixtureGroup struct {
	Kind        string           `json:"kind"`
	Label       string           `json:"label"`
	Total       int              `json:"total"`
	More        int              `json:"more"`
	ShowAllHref string           `json:"show_all_href"`
	Subjects    []fixtureSubject `json:"subjects"`
}

type fixtureSubject struct {
	Key    string         `json:"key"`
	Type   string         `json:"type"`
	Link   string         `json:"link"`
	Facets []fixtureFacet `json:"facets"`
}

type fixtureFacet struct {
	Label   string          `json:"label"`
	Summary string          `json:"summary"`
	IsGap   bool            `json:"is_gap"`
	Since   string          `json:"since"`
	Details []fixtureDetail `json:"details"`
}

type fixtureDetail struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// fixtureSpanRows builds the exact ListAllOpenSpansRow inputs the loader inserts,
// from the single-source-of-truth inventoryFixtureSpans slice.
func fixtureSpanRows(t *testing.T) []db.ListAllOpenSpansRow {
	t.Helper()
	rows := make([]db.ListAllOpenSpansRow, 0, len(inventoryFixtureSpans))
	for _, fs := range inventoryFixtureSpans {
		openedAt, err := fs.openedAt()
		if err != nil {
			t.Fatalf("fixture span %s/%s: %v", fs.kind, fs.key, err)
		}
		rows = append(rows, db.ListAllOpenSpansRow{
			SubjectKind:   fs.kind,
			SubjectKey:    fs.key,
			Facet:         fs.facet,
			Discriminator: fs.discriminator,
			Source:        "resolver",
			Value:         []byte(fs.value),
			IsGap:         fs.isGap,
			OpenedAt:      pgtype.Timestamptz{Time: openedAt, Valid: true},
		})
	}
	return rows
}

// TestInventoryFixtureCountsMatchPackage keeps the VERGE_DEV windowing override
// (devInventoryGroupTotals, applyInventoryFixtureCounts) honest against the frozen
// package: every group whose total the dev capture pins must equal fixtures.json's
// declared total, and — folded over the fixture's own subjects through the same
// windowing the handler runs — reproduce the fixture's declared `more`. This is the
// byte-exactness gate for the #756 count badge + "Show all N — M more" expander,
// before the pixel harness.
func TestInventoryFixtureCountsMatchPackage(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var want fixtureInventory
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}

	byKind := map[string]fixtureGroup{}
	for _, g := range want.Inventory.Groups {
		byKind[g.Kind] = g
	}

	for kind, total := range devInventoryGroupTotals {
		wg, ok := byKind[kind]
		if !ok {
			t.Fatalf("devInventoryGroupTotals pins kind %q not present in fixtures.json", kind)
		}
		if total != wg.Total {
			t.Errorf("group %q total drift: devInventoryGroupTotals = %d, fixtures.json = %d", kind, total, wg.Total)
		}
	}

	// Fold the fixture's declared groups through the real windowing + dev override and
	// assert the resulting Total/More match fixtures.json for the pinned group(s).
	got := buildInventory(fixtureSpanRows(t))
	windowInventoryGroups(got, "")
	applyInventoryFixtureCounts(got, "")
	for i := range got {
		g := got[i]
		if _, pinned := devInventoryGroupTotals[g.Kind]; !pinned {
			continue
		}
		wg := byKind[g.Kind]
		if g.Total != wg.Total || g.More != wg.More {
			t.Errorf("group %q windowed counts = Total %d / More %d, want fixtures.json Total %d / More %d",
				g.Kind, g.Total, g.More, wg.Total, wg.More)
		}
		if g.ShowAllHref != wg.ShowAllHref {
			t.Errorf("group %q show_all_href = %q, want %q", g.Kind, g.ShowAllHref, wg.ShowAllHref)
		}
	}
}

// TestBuildInventoryMatchesDesignFixture is the byte-exactness gate: buildInventory
// over the loader's 17 synthesized spans must reproduce
// design-system/fixtures/fixtures.json → inventory.groups exactly — every group,
// subject, and facet string AND their order.
func TestBuildInventoryMatchesDesignFixture(t *testing.T) {
	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures.json: %v", err)
	}
	var want fixtureInventory
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("parse fixtures.json: %v", err)
	}

	got := buildInventory(fixtureSpanRows(t))

	if len(got) != len(want.Inventory.Groups) {
		t.Fatalf("group count = %d, want %d", len(got), len(want.Inventory.Groups))
	}
	for gi, wg := range want.Inventory.Groups {
		g := got[gi]
		if g.Kind != wg.Kind || g.Label != wg.Label {
			t.Fatalf("group[%d] = %q/%q, want %q/%q", gi, g.Kind, g.Label, wg.Kind, wg.Label)
		}
		if len(g.Subjects) != len(wg.Subjects) {
			t.Fatalf("group[%d] %q subject count = %d, want %d", gi, g.Kind, len(g.Subjects), len(wg.Subjects))
		}
		for si, ws := range wg.Subjects {
			s := g.Subjects[si]
			if s.Key != ws.Key {
				t.Errorf("group[%d] subject[%d] key = %q, want %q", gi, si, s.Key, ws.Key)
			}
			if s.Type != ws.Type {
				t.Errorf("subject %q type = %q, want %q", ws.Key, s.Type, ws.Type)
			}
			if s.Link != ws.Link {
				t.Errorf("subject %q link = %q, want %q", ws.Key, s.Link, ws.Link)
			}
			if len(s.Facets) != len(ws.Facets) {
				t.Fatalf("subject %q facet count = %d, want %d", ws.Key, len(s.Facets), len(ws.Facets))
			}
			for fi, wf := range ws.Facets {
				f := s.Facets[fi]
				if f.Label != wf.Label {
					t.Errorf("subject %q facet[%d] label = %q, want %q", ws.Key, fi, f.Label, wf.Label)
				}
				if f.Summary != wf.Summary {
					t.Errorf("subject %q facet %q summary = %q, want %q", ws.Key, wf.Label, f.Summary, wf.Summary)
				}
				if f.IsGap != wf.IsGap {
					t.Errorf("subject %q facet %q is_gap = %v, want %v", ws.Key, wf.Label, f.IsGap, wf.IsGap)
				}
				if f.Since != wf.Since {
					t.Errorf("subject %q facet %q since = %q, want %q", ws.Key, wf.Label, f.Since, wf.Since)
				}
				if len(f.Details) != len(wf.Details) {
					t.Fatalf("subject %q facet %q detail count = %d, want %d (%#v)", ws.Key, wf.Label, len(f.Details), len(wf.Details), f.Details)
				}
				for di, wd := range wf.Details {
					d := f.Details[di]
					if d.Type != wd.Type || d.Data != wd.Data {
						t.Errorf("subject %q facet %q detail[%d] = %q/%q, want %q/%q", ws.Key, wf.Label, di, d.Type, d.Data, wd.Type, wd.Data)
					}
				}
			}
		}
	}
}
