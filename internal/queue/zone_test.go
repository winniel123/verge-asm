package queue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/scan"
)

func TestToZoneObservationParamsStampsSupplyInstantAndZoneSource(t *testing.T) {
	supply := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	recs := []scan.ZoneRecord{
		{Name: "example.com", Qtype: "A", Data: json.RawMessage(`{"rrs":["203.0.113.10"]}`), ObservedAt: supply},
		{Name: "www.example.com", Qtype: "CNAME", Data: json.RawMessage(`{"rrs":["example.com"]}`), ObservedAt: supply},
	}
	params := toZoneObservationParams(99, recs)
	if len(params) != 2 {
		t.Fatalf("got %d params, want 2", len(params))
	}
	for _, p := range params {
		if !p.ObservedAt.Time.Equal(supply) {
			t.Errorf("%s/%s observed_at = %s, want supply instant %s", p.SubjectKey, p.Discriminator, p.ObservedAt.Time, supply)
		}
		if p.Source != scan.ZoneSource {
			t.Errorf("source = %q, want %q", p.Source, scan.ZoneSource)
		}
		if p.VantageID.Valid {
			t.Errorf("zone observation carried a vantage, want none: %+v", p.VantageID)
		}
		if p.Facet != "dns-record" || p.SubjectKind != "name" {
			t.Errorf("facet/kind mis-set: %s/%s", p.Facet, p.SubjectKind)
		}
		if p.BatchID != 99 {
			t.Errorf("batch id = %d, want 99", p.BatchID)
		}
	}
	if params[0].Discriminator != "A" || params[1].Discriminator != "CNAME" {
		t.Errorf("qtype not carried onto the discriminator: %q, %q", params[0].Discriminator, params[1].Discriminator)
	}
}
