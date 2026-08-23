package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

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

// coverageMeterView is one CoverageMeter row of the aperture card. It renders in
// the component's *census* state only — a census claims no denominator and no
// percentage, which is this screen's standing rule (ADR-0095: state what the
// instrument looks at, never a proportion of the estate). Count is pre-formatted.
type coverageMeterView struct {
	Label  string
	Count  string
	Unit   string
	Detail string
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

// coverageMessageView is one currency message: a gap, a stale source, or a
// position that went silent. Kind drives the badge (a dotted GapBadge for "gap",
// a bronze staleness chip otherwise) — never the severity ramp.
type coverageMessageView struct {
	Kind    string
	Badge   string
	Subject string
	Text    string
}

// unevaluableRuleView is one rule the batch could not evaluate: its id and
// version (rendered as a SignalRuleRef chip) and why it was blocked.
type unevaluableRuleView struct {
	ID      string
	Version string
	Why     string
}

// coveragePage renders the Coverage screen (#301, §6.3, ADR-0110). It shapes four
// regions from real reads: the aperture meters (one census per declared scope),
// the coverage messages and the gaps register (both from the Gap'd-Service
// register #254 and the unavailable-vantage register ADR-0108), and the
// unevaluable rules (the rules whose census carries not-evaluable members this
// batch). It is viewer-readable (requireLogin) — no mutation lives here.
func (s *server) coveragePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

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
		"NavActive":   "coverage",
		"Meters":      meters,
		"Messages":    messages,
		"Gaps":        gaps,
		"Unevaluable": unevaluable,
		// The zone-stale callout is gated on a real stale-zone read; that currency
		// read lands later, so the block ports the structure without ever rendering
		// fabricated staleness.
		"StaleZones": []string(nil),
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
			out = append(out, coverageMeterView{
				Label:  sd.AddressCidr.String(),
				Count:  humanCount(*sd.AddressCidr),
				Unit:   "addresses",
				Detail: "address scope — a census of the addresses it enumerates, never a proportion of your estate",
			})
			continue
		}
		domain := sd.NameDomain.String
		out = append(out, coverageMeterView{
			Label:  domain,
			Count:  strconv.Itoa(declared[domain]),
			Unit:   "declared names",
			Detail: "name scope — no denominator; its addresses arrive by resolution, and custody extension reaches what resolution reveals",
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
