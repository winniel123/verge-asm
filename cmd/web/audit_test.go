package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

func recordActAt(t *testing.T, f *fakeStore, at time.Time, actor act.Actor, a act.Act) {
	t.Helper()
	kind, actorJSON, err := act.EncodeActor(actor)
	if err != nil {
		t.Fatalf("encode actor: %v", err)
	}
	action, subject, err := act.EncodeSubject(a)
	if err != nil {
		t.Fatalf("encode subject: %v", err)
	}
	f.actNow = at
	f.appendActRow(db.InsertActParams{
		ActorKind: kind, Actor: actorJSON, Action: action, Subject: subject,
	})
}

// The tab renders what the recorder wrote, so tickets 5 to 9 are demoable on it (#127 §4).

func TestAuditTabRendersTheFourColumns(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	recordActAt(t, f, now.Add(-4*time.Minute),
		act.Account{AccountID: 7, UsernameSnapshot: "alice"},
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "10.0.0.0/8"}})
	recordActAt(t, f, now.Add(-2*time.Hour),
		act.GrantHolder{Grant: act.Invite{InviteID: 12}},
		act.InviteAccepted{AccountAtRole: act.AccountAtRole{Username: "bob", Role: "viewer"}})

	base := startAt(t, f, now)
	page := settingsTabBody(t, login(t, base, "admin", "hunter2hunter2"), base, "audit")

	// A relative stamp with ISO 8601 on hover, the mark on both an Actor and a Subject
	// payload, the Action label the spec draws, and a grant as an st-tag because it is a kind.
	for _, want := range []string{
		">4m<",
		`title="2026-09-11T11:56:00Z"`,
		"@alice",
		"Seed declared",
		"10.0.0.0/8",
		`<span class="st-tag">invite 12</span>`,
		"@bob · viewer",
		"Invite accepted",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the audit tab is missing %q", want)
		}
	}
	if strings.Contains(page, ">alice<") {
		t.Error("the Actor cell rendered a bare username, which collides with a grant label")
	}
	// The four shipped columns, and no fifth (spec §4.4, §12).
	if strings.Contains(page, "Source IP") {
		t.Error("the audit table grew a Source IP column, which §12 rules out")
	}
}

// A long Subject is clipped, not scrolled, because .st-table td is nowrap (spec §4.4).

func TestAuditSubjectCarriesTheEllipsisTreatment(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	const long = "hooks.example.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
	recordActAt(t, f, now, act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.ChannelDeclared{ChannelRef: act.ChannelRef{Endpoint: long}})

	base := startAt(t, f, now)
	page := settingsTabBody(t, login(t, base, "admin", "hunter2hunter2"), base, "audit")

	if !strings.Contains(page, "text-overflow:ellipsis") {
		t.Error("the Subject cell carries no ellipsis treatment")
	}
	if !strings.Contains(page, `title="`+long+`"`) {
		t.Error("a clipped Subject is unrecoverable: the cell carries no full value on hover")
	}
}

// ADR-0158 limb 4: a client-side scope reaches only the rows already sent (spec §6.2).

func TestAuditPeriodScopesServerSide(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	recordActAt(t, f, now.Add(-2*time.Hour), act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "10.0.0.0/8"}})
	recordActAt(t, f, now.Add(-40*24*time.Hour), act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.ZoneDeclared{ZoneRef: act.ZoneRef{Zone: "old.example.com"}})

	base := startAt(t, f, now)
	c := login(t, base, "admin", "hunter2hunter2")

	// The default is the second preset, so the 40-day-old row is out of scope.
	page := settingsTabBody(t, c, base, "audit")
	if !strings.Contains(page, "10.0.0.0/8") {
		t.Error("the default period dropped a row it covers")
	}
	if strings.Contains(page, "old.example.com") {
		t.Error("the default period rendered a row 40 days old, so the scope is not applied")
	}
	if !strings.Contains(page, "Last 7d") {
		t.Error("the period control does not name the resolved window")
	}

	wide := getBody(t, c, base+"/settings?tab=audit&period=90d", http.StatusOK)
	if !strings.Contains(wide, "old.example.com") {
		t.Error("period=90d did not widen the server-side predicate")
	}
	// An unknown token falls back rather than 404ing, and it changes no gate.
	fallback := getBody(t, c, base+"/settings?tab=audit&period=nonsense", http.StatusOK)
	if strings.Contains(fallback, "old.example.com") || !strings.Contains(fallback, "Last 7d") {
		t.Error("an unknown period token did not fall back to the default")
	}
}

// A preset carries no upper bound, so a clock skew cannot hide the newest row (spec §5.1).

func TestAuditPresetPeriodHasNoUpperBound(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// The column defaults created_at from the database clock, not from this one.
	recordActAt(t, f, now.Add(3*time.Second), act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "10.0.0.0/8"}})

	base := startAt(t, f, now)
	page := settingsTabBody(t, login(t, base, "admin", "hunter2hunter2"), base, "audit")
	if !strings.Contains(page, "10.0.0.0/8") {
		t.Error("a row stamped ahead of this clock fell outside the preset window")
	}
}

// A custom range does carry both bounds, and the end date is inclusive (spec §6.2).

func TestAuditCustomRangeBoundsBothEnds(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	recordActAt(t, f, time.Date(2026, 9, 2, 23, 30, 0, 0, time.UTC),
		act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.ZoneDeclared{ZoneRef: act.ZoneRef{Zone: "inside.example.com"}})
	recordActAt(t, f, time.Date(2026, 9, 3, 0, 30, 0, 0, time.UTC),
		act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.ZoneDeclared{ZoneRef: act.ZoneRef{Zone: "outside.example.com"}})

	base := startAt(t, f, now)
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/settings?tab=audit&start=2026-09-01&end=2026-09-02", http.StatusOK)
	if !strings.Contains(page, "inside.example.com") {
		t.Error("the custom range dropped a row on its inclusive end date")
	}
	if strings.Contains(page, "outside.example.com") {
		t.Error("the custom range admitted a row past its end date")
	}
	if !strings.Contains(page, "2026-09-01 – 2026-09-02") {
		t.Error("the period control does not name the custom range")
	}
}

// E.3 claims the whole record is empty, so a narrowed period may not borrow it (§8 · E.3).

func TestAuditEmptyStatesAreTwoDifferentFacts(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startAt(t, f, now)
	c := login(t, base, "admin", "hunter2hunter2")

	empty := settingsTabBody(t, c, base, "audit")
	for _, want := range []string{
		"No acts recorded yet",
		"This instance has taken no auditable act since the record began. Your next seed, scan or team change lands here.",
	} {
		if !strings.Contains(empty, want) {
			t.Errorf("the empty corpus is missing §8 · E.3's %q", want)
		}
	}
	for _, gone := range []string{"No audit log", "the delivery record", "keeps no separate queryable log"} {
		if strings.Contains(empty, gone) {
			t.Errorf("the withdrawn string %q still ships", gone)
		}
	}

	recordActAt(t, f, now.Add(-40*24*time.Hour), act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "10.0.0.0/8"}})
	narrow := settingsTabBody(t, c, base, "audit")
	if strings.Contains(narrow, "since the record began") {
		t.Error("a narrowed period claims the whole record is empty, which is false")
	}
	if !strings.Contains(narrow, "No acts in this period") {
		t.Error("an empty period carries no empty state of its own")
	}
}

// The lede is E.2 whole, and the second clause is not repaired here (§8 · E.2).

func TestAuditLedeIsTheWholeReplacement(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	page := settingsTabBody(t, login(t, base, "admin", "hunter2hunter2"), base, "audit")

	if !strings.Contains(page, `<p class="st-lede">Who did what, when.</p>`) {
		t.Error("the lede is not §8 · E.2 verbatim")
	}
	if strings.Contains(page, "keeps no log line of its own") {
		t.Error("the lede still carries the clause the corpus makes false")
	}
}

// audit is one of the thirteen identifiers that refuse a viewer outright (spec §6.1).

func TestAuditTabRefusesAViewer(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	vc := login(t, base, "viewer", "hunter2hunter2")
	getBody(t, vc, base+"/settings?tab=audit", http.StatusForbidden)
	// ADR-0173's api carve-out is the single one, and it is untouched.
	getBody(t, vc, base+"/settings?tab=api", http.StatusOK)
	if len(f.actRows) != 0 {
		t.Fatalf("a refused viewer wrote %d acts", len(f.actRows))
	}
}

// A row this build cannot read fails the period loudly, never partially (ADR-0168 §4).

func TestAuditRefusesToRenderAPartialPeriod(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	recordActAt(t, f, now, act.Account{AccountID: 1, UsernameSnapshot: "admin"},
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "10.0.0.0/8"}})
	f.actRows = append(f.actRows, db.Act{
		ID:        99,
		CreatedAt: pgtype.Timestamptz{Time: now.Add(-time.Minute), Valid: true},
		ActorKind: "account", Actor: []byte(`{"account_id":1,"username_snapshot":"admin"}`),
		Action: "seed.evaporated", Subject: []byte(`{}`),
	})

	base := startAt(t, f, now)
	c := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, c, base+"/settings?tab=audit", http.StatusInternalServerError)
	if strings.Contains(page, "10.0.0.0/8") {
		t.Error("the tab rendered the readable rows and hid the one it could not read")
	}
}

// validTab reads Get("tab") only, so period changes no gate and ADR-0173 §1 is untouched.

func TestAuditPeriodChangesNoGate(t *testing.T) {
	for _, token := range []string{"24h", "90d", "custom_2026-09-01_2026-09-02", "nonsense"} {
		if got := validTab("audit"); got != "audit" {
			t.Fatalf("validTab(%q) = %q", "audit", got)
		}
		if got := resolveAuditPeriod(token).Token; got == "" {
			t.Errorf("period %q resolved to an empty token", token)
		}
	}
	if got := resolveAuditPeriod("").Token; got != auditDefaultPeriod {
		t.Errorf("the empty token resolved to %q, want %q", got, auditDefaultPeriod)
	}
}
