package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"sort"
	"strconv"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Coverage screen (screen 6, #551/#552) is served byte-for-byte from the frozen
// design-owned design-system/templates/coverage.tmpl (package v3.7.0, WORKFLOW v4),
// which replaces the repo-authored templates_coverage.go const (deleted). The tmpl
// renders inside the full app chrome ({{template "chrome" .}}) and declares the holes
// coveragePage shapes below: .Meters[{Label,Counted,Total(nullable),Unit,Pct,Detail}],
// .Messages[{Kind,Badge,Bound,Subject,Text,When,ISO}], .Gaps[{Subject,Gap,Expected,
// Since}], .Unevaluable[{ID,Version,Why}], .StaleZones[{Zone,Age}]. It styles against
// the design token vocabulary, so the render opts in with DesignTokens:true (the "head"
// block inlines tokens/*.css only then). coverage.tmpl auto-embeds through designfs's
// existing templates/*.tmpl glob, so no designfs.go change is needed.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/coverage.tmpl"))

// coveragePct is the meter fill percentage (0–100) an ADDRESS-scope meter renders
// (#19c): counted subjects over the enumerable addresses of the declared range,
// rounded to the nearest whole percent and clamped. render-goldens/main.go replicates
// this arithmetic byte-for-byte so the golden (which composes fixtures.json → coverage
// statically) and the seeded candidate compute the same fill. A name scope is a census
// (no denominator) and never calls this.
func coveragePct(counted, total int) int {
	if total <= 0 {
		return 0
	}
	p := int(math.Round(float64(counted) / float64(total) * 100))
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// coldScopeView is a declared Seed shaped for the cold-tier opt-in section: the
// scope, its kind, and whether it has opted into the full-range Scan.
type coldScopeView struct {
	ID        int64
	IsAddress bool
	Scope     string
	OptedIn   bool
}

// toColdScopeViews decorates each Seed with its cold-tier opt-in state, so the
// operator sees which scopes the full-range sweep covers. Every declared scope
// is shown — an un-opted scope as an invitation to opt in.
func toColdScopeViews(seeds []seedView, optedIn []int64) []coldScopeView {
	in := make(map[int64]bool, len(optedIn))
	for _, id := range optedIn {
		in[id] = true
	}
	out := make([]coldScopeView, 0, len(seeds))
	for _, s := range seeds {
		out = append(out, coldScopeView{
			ID: s.ID, IsAddress: s.IsAddress, Scope: s.Scope, OptedIn: in[s.ID],
		})
	}
	return out
}

// setColdScope opts a Seed scope into the full-range cold Scan, or back out (v1
// spec §3.4, ADR-0044). Enabling the cold tier is per-Seed, not global: this
// handler writes the scope opt-in and reconciles the Scan's enabled flag, and
// does NOTHING else — crucially it never dispatches a Scan. Adding a scope
// queues nothing; it only marks the tier enabled, and the cold Scan then fans
// out on its own monthly cadence, never on this config-save. It is reached only
// through requireAdmin, so a viewer can read the opt-in state but never move it.
func (s *server) setColdScope(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSeeds(w, r, acct, seedsForms{coldError: "That scope could not be found."})
		return
	}
	// The form carries the intended end state, not a blind flip: a stale page is
	// idempotent rather than a surprising reversal.
	optIn := r.FormValue("opt_in") == "true"
	if optIn {
		if err := s.store.OptInColdScope(r.Context(), db.OptInColdScopeParams{
			SeedID: id, CreatedBy: acct.ID,
		}); err != nil {
			s.serverError(w, "opt in cold scope", err)
			return
		}
	} else {
		if err := s.store.OptOutColdScope(r.Context(), id); err != nil {
			s.serverError(w, "opt out cold scope", err)
			return
		}
	}
	// Reconcile the tier's enabled flag with its scope. This is the whole of the
	// enablement — the tier is enabled exactly while a scope is opted in — and it
	// is the only thing that puts the cold Scan on the dispatcher's cadence.
	if err := s.store.SyncColdScanEnabled(r.Context()); err != nil {
		s.serverError(w, "sync cold scan enabled", err)
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// --- Coverage (#301, ADR-0110) ----------------------------------------------
//
// The V3 Coverage screen, ported from design-system/examples/console/Coverage.jsx:
// "what was measured, what was not, and why". Four regions, each wired to a real
// read of the same shape as the example's sample data, and each empty-stated where
// no honest read exists — no fabricated data (see templates_coverage.go).

// coverageMeterView is one CoverageMeter row of the aperture card. Two shapes
// (SPEC-CHANGE #19c, refining ADR-0095): an ADDRESS scope renders counted/total —
// Total = the enumerable addresses of the declared range (a /24's usable size),
// Counted = the subjects the batch walked, Pct = the precomputed fill — while a NAME
// scope stays a census (Total == nil → the striped census bar), because a name scope
// enumerates nothing on its own. ADR-0095's estate-proportion bar is untouched: a
// declared range is NOT the estate, so counted/total over a range is a census of that
// range, never a proportion of the operator's estate. Counted/Total are pre-formatted
// strings so the template prints them verbatim (Total is a *string: nil == census).
type coverageMeterView struct {
	Label   string
	Counted string
	Total   *string
	Unit    string
	Pct     int
	Detail  string
}

// coverageGapView is one row of the "expected, not observed" register: a subject
// we expected to observe and did not, the kind of absence, what was expected, and
// how long the gap has stood. Since is an em dash where no honest instant is read.
type coverageGapView struct {
	Subject  string
	Gap      string
	Expected string
	Since    string
}

// coverageMessageView is one currency message: a gap, a stale source, a position
// that went silent, or a batch whose conclusions were not evaluable. Kind drives the
// badge (a dotted GapBadge for "gap", a bronze staleness chip otherwise) — never the
// severity ramp. Bound is the staleness chip's trailing figure (e.g. "9d"), empty
// where the chip carries none. When is the relative-time column the design added
// (#19e) and ISO is its full RFC-3339 instant, rendered as the column's title tooltip;
// both empty omit the column.
type coverageMessageView struct {
	Kind    string
	Badge   string
	Bound   string
	Subject string
	Text    string
	When    string
	ISO     string
}

// unevaluableRuleView is one rule the batch could not evaluate: its id and
// version (rendered as a SignalRuleRef chip) and why it was blocked.
type unevaluableRuleView struct {
	ID      string
	Version string
	Why     string
}

// coverageStaleZoneView is one per-zone stale callout (#19e): the zone whose supplied
// zone file has aged past its re-supply window, and how old it is (e.g. "2 re-supply
// intervals"), so removal detection's suspension is named per scope rather than once.
type coverageStaleZoneView struct {
	Zone string
	Age  string
}

// coveragePage renders the Coverage screen (#301, §6.3, ADR-0110). It shapes four
// regions from real reads: the aperture meters (one census per declared scope),
// the coverage messages and the gaps register (both from the Gap'd-Service
// register #254 and the unavailable-vantage register ADR-0108), and the
// unevaluable rules (the rules whose census carries not-evaluable members this
// batch). It is viewer-readable (requireLogin) — no mutation lives here.
func (s *server) coveragePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// VERGE_DEV pixel-parity path (#551/#552). The frozen coverage.tmpl renders the
	// #19c address-scope counted/total meter, the four relative-time currency messages
	// and the per-zone stale callout — a curated corpus whose exact strings, the
	// counted/total figures (with the "16 skipped" breakdown) and the when/iso pair (the
	// two do not correlate through the fixture clock: When is the last-check age, ISO the
	// underlying event instant) are the design's, not a live-estate read. Reproducing
	// them from the live derivations would mean fabricating domain data, which
	// SPEC-CHANGE forbids — so, exactly as the SignIn/Setup screens pin their dev fixture
	// (login providers, recovery codes, setup token) and serve it under devMode with a
	// drift test, coverage serves the pinned fixtures.json → coverage slice here so the
	// seeded candidate renders byte-for-byte what the golden composes. A real deployment
	// (devMode == false) falls through to the honest live reads below.
	if s.devMode {
		s.render(w, "coverage", s.coverageFixtureData(acct))
		return
	}

	// Aperture meters — one census read per declared scope. Best-effort on the zone
	// read (it feeds only a name scope's declared-name count): a failure leaves the
	// meter's census at zero rather than 500ing the whole page.
	seeds, err := s.store.ListSeeds(ctx)
	if err != nil {
		s.serverError(w, "list seeds", err)
		return
	}
	var zones []db.ListZoneDeclarationsRow
	if z, zerr := s.store.ListZoneDeclarations(ctx); zerr == nil {
		zones = z
	}
	meters := apertureMeters(seeds, zones)

	// Gaps + coverage messages. A blanket responder answers on every port, so its
	// reach is a Gap, never reached (ADR-0104 §4): it surfaces both as a gap row and
	// as a currency message. Both reads are best-effort — a failure degrades to no
	// statement, never a fabricated absence.
	var gaps []coverageGapView
	var messages []coverageMessageView
	if svc, berr := s.store.ListBlanketedReachServices(ctx); berr == nil {
		gaps, messages = blanketGapsAndMessages(svc)
	}
	// A vantage we could not reach reads as "we could not look from here", not as an
	// empty measurement (ADR-0108, #249).
	if rows, uerr := s.store.ListUnavailableVantages(ctx); uerr == nil {
		messages = append(messages, unavailableVantageMessages(rows)...)
	}
	sortCoverageMessages(messages)

	// Unevaluable rules — the rules whose current census carries not-evaluable
	// members. Best-effort: a corpus failure degrades to no list rather than a 500,
	// exactly as the dashboard's signal regions do.
	var unevaluable []unevaluableRuleView
	if corpus, cerr := s.buildSignalCorpus(r); cerr == nil {
		unevaluable = unevaluableRules(corpus)
	}

	s.render(w, "coverage", map[string]any{
		"Title": "Coverage", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "coverage",
		// coverage.tmpl styles against the design token vocabulary; the "head" block
		// inlines tokens/*.css only when this datum is set (as Profile/ErrorPage do).
		"DesignTokens": true,
		"Meters":       meters,
		"Messages":     messages,
		"Gaps":         gaps,
		"Unevaluable":  unevaluable,
		// The per-zone stale callout is gated on a real stale-zone read; that currency
		// read lands later, so the block ports the structure without ever rendering
		// fabricated staleness.
		"StaleZones": []coverageStaleZoneView(nil),
	})
}

// apertureMeters shapes one census CoverageMeter per declared scope. An address
// scope is its own complete enumeration, so its census counts the addresses it
// covers (seed.AddressCount via humanCount). A name scope enumerates nothing on
// its own — its addresses arrive by resolution — so its census counts the owner
// names its supplied zone declares, and states that its addresses come by
// resolution. Neither claims a denominator or a proportion of the estate.
func apertureMeters(seeds []db.ListSeedsRow, zones []db.ListZoneDeclarationsRow) []coverageMeterView {
	declared := make(map[string]int, len(zones))
	for _, z := range zones {
		if z.NameDomain.Valid {
			declared[z.NameDomain.String] = len(signal.DeclaredNames(z.Content, z.NameDomain.String))
		}
	}
	out := make([]coverageMeterView, 0, len(seeds))
	for _, sd := range seeds {
		if sd.Kind == "address" && sd.AddressCidr != nil {
			// A live estate has no first-class "subjects the batch walked within this
			// declared range" numerator yet (see the ADR note refining ADR-0095), so the
			// live address meter renders the honest census of the addresses it enumerates
			// — Total nil. The #19c counted/total form is realized in the fixture-seeded
			// instance the design golden depicts (coverageFixtureData); wiring it to a live
			// range awaits that numerator rather than fabricating one.
			out = append(out, coverageMeterView{
				Label:   sd.AddressCidr.String(),
				Counted: humanCount(*sd.AddressCidr),
				Unit:    "addresses",
				Detail:  "address scope — a census of the addresses it enumerates, never a proportion of your estate",
			})
			continue
		}
		domain := sd.NameDomain.String
		out = append(out, coverageMeterView{
			Label:   domain,
			Counted: strconv.Itoa(declared[domain]),
			Unit:    "declared names",
			Detail:  "name scope — no denominator; its addresses arrive by resolution, and custody extension reaches what resolution reveals",
		})
	}
	return out
}

// blanketGapsAndMessages turns the Gap'd-Service register (#254) into the gaps
// rows and their matching currency messages. A blanket responder is an
// address-level fact — it answers on every port — so the keys collapse to their
// distinct addresses, in a stable order.
func blanketGapsAndMessages(keys []string) ([]coverageGapView, []coverageMessageView) {
	seen := map[string]bool{}
	var addrs []string
	for _, key := range keys {
		if addr, _, _ := splitServiceKey(key); addr != "" && !seen[addr] {
			seen[addr] = true
			addrs = append(addrs, addr)
		}
	}
	sort.Strings(addrs)

	var gaps []coverageGapView
	var msgs []coverageMessageView
	for _, addr := range addrs {
		gaps = append(gaps, coverageGapView{
			Subject:  addr,
			Gap:      "no origin",
			Expected: "origin behind the edge",
			Since:    "—",
		})
		msgs = append(msgs, coverageMessageView{
			Kind:    "gap",
			Badge:   "no origin",
			Subject: addr,
			Text:    "Answers TCP on all ports — a proxy edge, not your origin. Its reach is recorded as a Gap, never reached. To measure the real surface, declare your origin addresses as an address scope.",
		})
	}
	return gaps, msgs
}

// unavailableVantageMessages turns the unavailable-vantage register (ADR-0108)
// into silent currency messages: a position we could not look from, named, so an
// outage reads as blindness rather than emptiness.
func unavailableVantageMessages(rows []db.ListUnavailableVantagesRow) []coverageMessageView {
	out := make([]coverageMessageView, 0, len(rows))
	for _, v := range rows {
		resolver := v.Resolver
		if resolver == "" {
			resolver = "—"
		}
		out = append(out, coverageMessageView{
			Kind:    "silent",
			Badge:   "silent",
			Subject: v.Name,
			Text:    "We could not look from this position — its resolver " + resolver + " was unreachable, so its open spans are not evaluable.",
		})
	}
	return out
}

// sortCoverageMessages orders the messages deterministically — by kind, then
// subject — so the currency list is stable across loads.
func sortCoverageMessages(msgs []coverageMessageView) {
	sort.SliceStable(msgs, func(i, j int) bool {
		if msgs[i].Kind != msgs[j].Kind {
			return msgs[i].Kind < msgs[j].Kind
		}
		return msgs[i].Subject < msgs[j].Subject
	})
}

// unevaluableRules lists the rules whose current census carries not-evaluable
// members this batch — a subject in the rule's domain the rule could not read
// (signal.NotEvaluable). EvaluateCorpus already orders its output stably.
func unevaluableRules(corpus signal.Corpus) []unevaluableRuleView {
	var out []unevaluableRuleView
	for _, c := range signal.EvaluateCorpus(corpus) {
		n := len(c.NotEvaluable)
		if n == 0 {
			continue
		}
		suffix := "s"
		if n == 1 {
			suffix = ""
		}
		out = append(out, unevaluableRuleView{
			ID:      c.Rule,
			Version: c.Version.Rule,
			Why:     fmt.Sprintf("%d subject%s in its domain could not be read this batch", n, suffix),
		})
	}
	return out
}
