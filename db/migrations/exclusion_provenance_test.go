package migrations

import (
	"strings"
	"testing"
)

func TestExclusionCarriesItsProposal(t *testing.T) {
	// created_by is the same account on both paths, so only a proposal id separates them (#1799).
	col := tableColumn(t, "exclusion", "proposal_id")

	if !strings.Contains(col, "bigint") {
		t.Errorf("proposal_id must be a BIGINT, got: %s", col)
	}
	if !strings.Contains(col, "references proposal (id)") {
		t.Errorf("proposal_id must reference proposal (id), got: %s", col)
	}
	if !strings.Contains(col, "on delete set null") {
		t.Errorf("the FK from exclusion.proposal_id to proposal(id) must be ON DELETE SET NULL, "+
			"so a removed proposal leaves the exclusion standing, got: %s", col)
	}
	if strings.Contains(col, "not null") {
		t.Errorf("proposal_id must stay nullable; a hand-declared exclusion answers no proposal, got: %s", col)
	}
	if _, ok := tableConstraints(t, "exclusion")["exclusion_provenance"]; !ok {
		t.Error("no exclusion_provenance constraint stands; only an address exclusion can answer a proposal")
	}
}

func TestExclusionProvenanceIsNotBackfilled(t *testing.T) {
	// Matching on address_cidr would claim the operator's own declaration (#1799).
	up := strings.ToLower(upMigrations(t))

	for _, s := range strings.Split(up, ";") {
		if !strings.Contains(s, "proposal_id") {
			continue
		}
		if strings.Contains(s, "update exclusion") {
			t.Errorf("a backfill of proposal_id reproduces #1799 on every CIDR the operator "+
				"declared and a proposal also named: %s", strings.TrimSpace(s))
		}
	}
}
