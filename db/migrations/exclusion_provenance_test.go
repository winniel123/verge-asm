package migrations

import (
	"regexp"
	"strings"
	"testing"
)

func TestExclusionCarriesItsProposal(t *testing.T) {
	// created_by is the same account on both paths, so only a proposal id separates them (#1799).
	up := strings.ToLower(upMigrations(t))

	col := regexp.MustCompile(`(?s)add column proposal_id\s+bigint[^;]*`).FindString(up)
	if col == "" {
		t.Fatal("exclusion declares no proposal_id column; an undo cannot tell a decline's row " +
			"from a hand-declared one without it (#1799)")
	}
	if !strings.Contains(col, "references proposal (id)") {
		t.Errorf("proposal_id must reference proposal (id), got: %s", strings.TrimSpace(col))
	}
	if !strings.Contains(col, "on delete set null") {
		t.Errorf("the FK from exclusion.proposal_id to proposal(id) must be ON DELETE SET NULL, "+
			"so a removed proposal leaves the exclusion standing, got: %s", strings.TrimSpace(col))
	}
	if strings.Contains(col, "not null") {
		t.Errorf("proposal_id must stay nullable; a hand-declared exclusion answers no proposal, got: %s",
			strings.TrimSpace(col))
	}
	if !strings.Contains(up, "exclusion_provenance") {
		t.Error("no exclusion_provenance constraint; only an address exclusion can answer a proposal")
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
