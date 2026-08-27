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
	Kind     string           `json:"kind"`
	Label    string           `json:"label"`
	Subjects []fixtureSubject `json:"subjects"`
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

// TestBuildInventoryMatchesDesignFixture is the byte-exactness gate: buildInventory
// over the loader's 16 synthesized spans must reproduce
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
