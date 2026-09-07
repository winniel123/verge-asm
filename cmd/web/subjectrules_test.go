package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/signal"
)

func subjectRuleFor(t *testing.T, rows []subjectRule, rule string) subjectRule {
	t.Helper()
	for _, row := range rows {
		if row.Rule == rule {
			return row
		}
	}
	t.Fatalf("rule %q is a census member but no row was built; rows: %+v", rule, rows)
	return subjectRule{}
}

func TestSubjectRulesKeepsNotEvaluableApartFromNotFired(t *testing.T) {
	censuses := []signal.Census{
		{
			Rule:         "fired-rule",
			Version:      signal.Version{Rule: "v3"},
			Fired:        []signal.Member{{Subject: "svc"}},
			NotEvaluable: []signal.Member{{Subject: "other"}},
		},
		{
			Rule:     "settled-rule",
			Version:  signal.Version{Rule: "v1"},
			NotFired: []signal.Member{{Subject: "svc"}},
		},
		{
			Rule:         "unread-rule",
			Version:      signal.Version{Rule: "v2"},
			NotEvaluable: []signal.Member{{Subject: "svc"}},
		},
		{
			Rule:    "stranger-rule",
			Version: signal.Version{Rule: "v1"},
			Fired:   []signal.Member{{Subject: "other"}},
		},
	}

	rows := subjectRulesFor(censuses, "svc")

	if len(rows) != 3 {
		t.Fatalf("a subject is a row exactly where it is a census member: got %d rows, want 3; rows: %+v", len(rows), rows)
	}
	if got := subjectRuleFor(t, rows, "fired-rule").Verdict; got != signal.Fired {
		t.Errorf("fired-rule verdict: got %q want %q", got, signal.Fired)
	}
	if got := subjectRuleFor(t, rows, "settled-rule").Verdict; got != signal.NotFired {
		t.Errorf("settled-rule verdict: got %q want %q", got, signal.NotFired)
	}
	if got := subjectRuleFor(t, rows, "unread-rule").Verdict; got != signal.NotEvaluable {
		t.Errorf("absent evidence yields not-evaluable, never did not fire (ADR-0004): got %q want %q", got, signal.NotEvaluable)
	}
}

func TestSubjectRulesTableNeverRendersNotEvaluableAsDidNotFire(t *testing.T) {
	rows := []subjectRule{
		{Rule: "unread-rule", Version: "2", Severity: "high", SevLabel: "High", Verdict: signal.NotEvaluable},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "subjectrules", rows); err != nil {
		t.Fatalf("execute subjectrules template: %v", err)
	}
	page := buf.String()

	if strings.Contains(page, "did not fire") {
		t.Errorf("ADR-0004 forbids rendering a not-evaluable member as \"did not fire\"; body: %s", page)
	}
	if !strings.Contains(page, "not evaluable") {
		t.Errorf("a not-evaluable member must read as its own verdict; body: %s", page)
	}
}

func TestSubjectRulesTableKeepsTheOtherTwoVerdicts(t *testing.T) {
	rows := []subjectRule{
		{Rule: "fired-rule", Version: "3", Severity: "critical", SevLabel: "Critical", Verdict: signal.Fired},
		{Rule: "settled-rule", Version: "1", Severity: "low", SevLabel: "Low", Verdict: signal.NotFired},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "subjectrules", rows); err != nil {
		t.Fatalf("execute subjectrules template: %v", err)
	}
	page := buf.String()

	if !strings.Contains(page, ">fired<") {
		t.Errorf("a fired member must still read as fired; body: %s", page)
	}
	if !strings.Contains(page, "did not fire") {
		t.Errorf("an evaluated-and-false member must still read as did not fire; body: %s", page)
	}
	if strings.Contains(page, "not evaluable") {
		t.Errorf("neither of these two members is not-evaluable; body: %s", page)
	}
}

func subjectRuleRow(t *testing.T, page, rule string) string {
	t.Helper()
	for _, row := range strings.Split(page, "</tr>") {
		if strings.Contains(row, ">"+rule+"<") {
			return row
		}
	}
	t.Fatalf("rule %q has no row in the rendered table; body: %s", rule, page)
	return ""
}

func TestSubjectDetailFixturePinsANotEvaluableVerdict(t *testing.T) {
	for _, c := range []struct {
		name string
		rows []subjectRule
	}{
		{"service", devServiceData().Rules},
		{"service_withdrawn", devServiceWithdrawnData().Rules},
		{"endpoint", devEndpointData().Rules},
	} {
		if len(c.rows) == 0 {
			t.Fatalf("%s: the fixture carries no rules row, so the table pins nothing", c.name)
		}
		for _, row := range c.rows {
			switch row.Verdict {
			case signal.Fired, signal.NotFired, signal.NotEvaluable:
			default:
				t.Errorf("%s: rule %q carries verdict %q, which is not a census member (ADR-0024)", c.name, row.Rule, row.Verdict)
			}
		}
	}

	rows := devServiceData().Rules
	var unread []subjectRule
	for _, row := range rows {
		if row.Verdict == signal.NotEvaluable {
			unread = append(unread, row)
		}
	}
	if len(unread) == 0 {
		t.Fatalf("the subject-detail fixture holds no not-evaluable rule, so the third verdict has no end-to-end fixture; rows: %+v", rows)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "subjectrules", rows); err != nil {
		t.Fatalf("execute subjectrules template: %v", err)
	}
	page := buf.String()

	for _, row := range unread {
		cell := subjectRuleRow(t, page, row.Rule)
		if strings.Contains(cell, "did not fire") {
			t.Errorf("ADR-0004 forbids rendering the not-evaluable rule %q as \"did not fire\"; row: %s", row.Rule, cell)
		}
		if !strings.Contains(cell, "not evaluable") {
			t.Errorf("the not-evaluable rule %q must read as its own verdict; row: %s", row.Rule, cell)
		}
	}
}
