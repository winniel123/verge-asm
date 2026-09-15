package exposure

import "testing"

func TestCensusAccountsForEveryService(t *testing.T) {
	reached := Leg{Status: LegValued, Value: Reached}
	notReached := Leg{Status: LegValued, Value: NotReached}
	gap := Leg{Status: LegGap}
	never := Leg{Status: LegNeverConfigured}

	var c Census
	for _, svc := range []struct{ internet, internal Leg }{
		{reached, reached},
		{reached, notReached},
		{notReached, reached},
		{notReached, notReached},
		{reached, never},
		{gap, reached},
	} {
		c.Count(svc.internet, svc.internal)
	}

	want := Census{Exposed: 1, EdgeOnly: 1, Firewalled: 1, Unreachable: 1, OneLegged: 2}
	if c != want {
		t.Errorf("census = %+v, want %+v", c, want)
	}
	if got := c.Total(); got != 6 {
		t.Errorf("census total = %d, want 6; a row in no cell is a row the band cannot account for", got)
	}
}
