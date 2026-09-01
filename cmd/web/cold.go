package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"time"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/seed"
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
		s.failSettings(w, r, settingsForms{section: "scans", coldError: "That scope could not be found."})
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
	// #21d: the cold-tier region relocated to Settings → Scans. The 303 goes back to the
	// URL the opt-in was submitted from (ADR-0130 §3), which is /settings?tab=scans or
	// the folded /scans read surface that renders the same region — so a long scope list
	// keeps its scroll offset instead of jumping to the top of the default tab.
	s.backToSection(w, r, "scans")
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
//
// AsOf/AsOfISO carry #890's oldest-current as-of (#847, #882): the earliest instant among
// the still-current subjects the batch walked in the range. Counted is those current
// subjects and Total the declared count, so a shortfall (Counted < Total) is the honest lag
// of a scope the cadence cannot finish — never a reached-ever count, and it folds the
// trailing-edge staleness Gaps into one figure rather than minting a per-address message.
// The as-of states how stale that current frontier is. Currency stays nominal — the
// numerator reads through the k × declared-cadence window, never the effective cadence.
// AsOf is the relative phrase ("3d ago"), AsOfISO its RFC-3339 tooltip; both empty omit the
// line, since no honest instant exists when nothing current sits in the range.
//
// Covers/SharedEdges are ADR-0129's #956 contradiction row (#989), and they are set
// on an ADDRESS scope alone. Covers is the addresses the scope covers and SharedEdges
// how many of them fan-out measured as a shared edge; both are pre-formatted strings,
// and both are EMPTY where the evidence does not exist — an unmeasured scope, a scope
// with nothing above the boundary, or a `Scan` out of force. The pair carries no
// threshold and no verdict: the boundary they were compared against stays inside the
// versioned `Custody` derivation, and the row states the two counts and names the
// exclusion remedy. It is display and never a gate — every address counted is still a
// declared subject and is still probed.
type coverageMeterView struct {
	Label       string
	Counted     string
	Total       *string
	Unit        string
	Pct         int
	Detail      string
	AsOf        string
	AsOfISO     string
	Covers      string
	SharedEdges string
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

// coveragePage renders the Coverage screen (#301, §6.3, ADR-0110). It shapes the
// regions from real reads: the aperture meters (an address scope's #19c counted/total
// over its range, a name scope's census), the coverage messages and the gaps register
// (both from the Gap'd-Service register #254 and the unavailable-vantage register
// ADR-0108), the unevaluable rules (the rules whose census carries not-evaluable
// members this batch) and the per-zone stale callout (#19e — a name-scope zone aged
// past two re-supply intervals). It is viewer-readable (requireLogin) — no mutation.
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
		s.render(w, r, "coverage", s.coverageFixtureData(acct))
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
	// Covered subjects for the #19c address-scope meters: the distinct addresses the
	// batch walked, drawn from the current Service subjects (each a triple sitting on
	// an address). Best-effort and live-tier gated — an unavailable read leaves the
	// address meters at a zero numerator rather than 500ing the page, never a
	// fabricated count.
	var walked []walkedAddr
	if svcs, serr := s.store.ListCurrentServiceSubjects(ctx, db.ListCurrentServiceSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); serr == nil {
		walked = walkedAddresses(svcs)
	}
	// #989's contradiction row: the declared scopes holding addresses fan-out measured
	// as shared edges. Best-effort — a failed read leaves the meters with no row, which
	// is the same shape an unmeasured scope renders, rather than 500ing the page or
	// naming evidence the read did not return.
	var sharedEdges map[netip.Prefix]int
	if m, ferr := addressScopeSharedEdges(ctx, s.store); ferr == nil {
		sharedEdges = m
	}
	meters := apertureMeters(seeds, zones, walked, s.now(), sharedEdges)

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

	// Per-zone stale callout (#19e): a name-scope zone whose latest supplied file has
	// aged past two re-supply intervals is stale — removal detection is suspended for
	// that scope until a fresh upload. Both reads are best-effort; either failing (or
	// no declared re-supply interval) leaves no callout rather than a fabricated one.
	var staleZonesView []coverageStaleZoneView
	if cadence, cerr := s.store.GetZoneCadenceSeconds(ctx); cerr == nil && cadence > 0 {
		if rows, zerr := s.store.ListZoneFileStatus(ctx); zerr == nil {
			staleZonesView = staleZones(rows, cadence, time.Now())
		}
	}

	s.render(w, r, "coverage", map[string]any{
		"Title": "Coverage", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "coverage",
		// coverage.tmpl styles against the design token vocabulary; the "head" block
		// inlines tokens/*.css only when this datum is set (as Profile/ErrorPage do).
		"DesignTokens": true,
		"Meters":       meters,
		"Messages":     messages,
		"Gaps":         gaps,
		"Unevaluable":  unevaluable,
		"StaleZones":   staleZonesView,
	})
}

// apertureMeters shapes one CoverageMeter per declared scope. An address scope
// renders the #19c counted/total form — the covered subjects the batch walked over
// the enumerable addresses of its declared range (addressMeter). A name scope
// enumerates nothing on its own — its addresses arrive by resolution — so it stays
// a census counting the owner names its supplied zone declares, and states that its
// addresses come by resolution. Neither claims a proportion of the estate.
//
// sharedEdges carries #989's contradiction row per declared scope, keyed by the masked
// prefix (addressScopeSharedEdges). A NIL map is the honest empty — no scope holds a
// measured shared edge, or the read degraded — and it renders no row anywhere. A name
// scope never takes one: its addresses arrive by resolution, so a shared edge it fronts
// belongs to the custody-extension census (#987) and not to this meter.
func apertureMeters(seeds []db.ListSeedsRow, zones []db.ListZoneDeclarationsRow, walked []walkedAddr, now time.Time, sharedEdges map[netip.Prefix]int) []coverageMeterView {
	declared := make(map[string]int, len(zones))
	for _, z := range zones {
		if z.NameDomain.Valid {
			declared[z.NameDomain.String] = len(signal.DeclaredNames(z.Content, z.NameDomain.String))
		}
	}
	out := make([]coverageMeterView, 0, len(seeds))
	for _, sd := range seeds {
		if sd.Kind == "address" && sd.AddressCidr != nil {
			p := *sd.AddressCidr
			out = append(out, addressMeter(p, walked, now, sharedEdges[p.Masked()]))
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

// maxMeterTotal is the largest denominator the address meter renders through the
// coveragePct int contract. An address Seed is bounded at declaration to the operator
// address-scope cap (seed.WithinCap, server.addressCap), which ADR-0127 leaves with no
// upper ceiling — so a range's enumerable count is no longer guaranteed to fit int.
// This guard degrades an out-of-range denominator to the honest census rather than
// overflowing the fill arithmetic.
const maxMeterTotal = int64(^uint(0) >> 1)

// addressMeter shapes the #19c address-scope meter: the covered subjects the batch
// walked within the declared range (the numerator) over the enumerable addresses of
// that range (the denominator = seed.AddressCount, NOT the estate). The fill is the
// ruled coveragePct(counted, total). A range whose enumerable count exceeds the meter
// arithmetic degrades to the honest census (Total nil) rather than fabricating a fill.
//
// sharedEdges is how many addresses inside p fan-out measured as a shared edge (#989).
// ZERO renders no contradiction row, which is both acceptance criteria at once: a scope
// with nothing above the boundary is silent, and so is an unmeasured one, because the
// declaration limb's absence rule is open-then-label. Above zero the row states the two
// counts on BOTH meter shapes — the evidence does not depend on whether the denominator
// fits the fill arithmetic.
func addressMeter(p netip.Prefix, walked []walkedAddr, now time.Time, sharedEdges int) coverageMeterView {
	covers, shared := "", ""
	if sharedEdges > 0 {
		covers, shared = humanCount(p), strconv.Itoa(sharedEdges)
	}
	total := seed.AddressCount(p)
	if total.IsInt64() {
		if t := total.Int64(); t > 0 && t <= maxMeterTotal {
			counted := coveredInRange(walked, p)
			totalStr := humanCount(p)
			m := coverageMeterView{
				Label:       p.String(),
				Counted:     strconv.Itoa(counted),
				Total:       &totalStr,
				Unit:        "subjects",
				Pct:         coveragePct(counted, int(t)),
				Detail:      "address scope — the subjects still current over the enumerable addresses of the declared range; a shortfall is the honest lag of a scope the cadence cannot finish, never hidden",
				Covers:      covers,
				SharedEdges: shared,
			}
			// #890: the oldest-current as-of — how stale the current numerator is. Present
			// only where a current, in-range subject carries a real instant (the honest
			// empty otherwise). The nominal k × declared-cadence window is the currency
			// gate on the walked set, so this never stretches to the effective cadence.
			if asOf, ok := oldestCurrentInRange(walked, p); ok {
				m.AsOf = agoLabel(asOf, now)
				m.AsOfISO = asOf.UTC().Format(time.RFC3339)
			}
			return m
		}
	}
	return coverageMeterView{
		Label:       p.String(),
		Counted:     humanCount(p),
		Unit:        "addresses",
		Detail:      "address scope — a census of the addresses it enumerates, never a proportion of your estate",
		Covers:      covers,
		SharedEdges: shared,
	}
}

// walkedAddr is one current, IP-hosted Service subject the batch reached: the address
// it sits on and when that reading was observed. The observed instant carries the
// currency frontier — #890's oldest-current as-of is the minimum of these over a range.
type walkedAddr struct {
	Addr       netip.Addr
	ObservedAt time.Time
}

// coveredInRange counts the distinct walked addresses that fall within p — the
// covered subjects an address-scope meter's numerator reports (#19c). Distinctness
// guards against a range being credited twice for two Services on one address.
func coveredInRange(walked []walkedAddr, p netip.Prefix) int {
	seen := make(map[netip.Addr]struct{}, len(walked))
	for _, w := range walked {
		if p.Contains(w.Addr) {
			seen[w.Addr] = struct{}{}
		}
	}
	return len(seen)
}

// oldestCurrentInRange is #890's oldest-current as-of: the earliest observed instant
// among the still-current subjects the batch walked inside p. It is the currency
// frontier — for a scope too large to finish inside one cadence, the oldest current
// address sits near the nominal k × declared-cadence bound, so this states how stale
// the current numerator is without minting a per-address message (#882: the trailing-
// edge Gap folds to declared/current, and this is its staleness). ok is false where no
// current, IP-hosted subject with a real instant sits in the range — the honest empty,
// never a fabricated zero instant.
func oldestCurrentInRange(walked []walkedAddr, p netip.Prefix) (time.Time, bool) {
	var oldest time.Time
	ok := false
	for _, w := range walked {
		if !p.Contains(w.Addr) || w.ObservedAt.IsZero() {
			continue
		}
		if !ok || w.ObservedAt.Before(oldest) {
			oldest = w.ObservedAt
			ok = true
		}
	}
	return oldest, ok
}

// walkedAddresses draws the batch-walked addresses from the current Service subjects
// — a Service is an (Address, port, transport) triple, so its key carries the address
// the batch reached and its observed_at the instant of the reading. Keys on a name host
// (name@service form) carry no address and are skipped; a range's numerator counts only
// the addresses actually resolved to an IP, and each carries the observed instant that
// feeds the oldest-current as-of.
func walkedAddresses(svcs []db.ListCurrentServiceSubjectsRow) []walkedAddr {
	out := make([]walkedAddr, 0, len(svcs))
	for _, sv := range svcs {
		addr, _, _ := splitServiceKey(sv.SubjectKey)
		if a, err := netip.ParseAddr(addr); err == nil {
			out = append(out, walkedAddr{Addr: a, ObservedAt: sv.ObservedAt.Time})
		}
	}
	return out
}

// staleZoneIntervals is the design's staleness bound (#19e, fixtures.json): a zone
// aged past two re-supply intervals has gone stale — "the source went stale".
const staleZoneIntervals = 2

// staleZones derives the per-zone stale callout (#19e): each name-scope zone whose
// latest supplied file has aged past staleZoneIntervals re-supply intervals, with its
// age rendered in the fixtures' own "N re-supply intervals" form. A zone with no
// supplied file (or no domain) contributes nothing — the design's empty pattern
// (Acceptance §7), never a fabricated zero. Ordered by zone for a stable callout.
func staleZones(rows []db.ListZoneFileStatusRow, cadenceSeconds int64, now time.Time) []coverageStaleZoneView {
	interval := time.Duration(cadenceSeconds) * time.Second
	if interval <= 0 {
		return nil
	}
	var out []coverageStaleZoneView
	for _, z := range rows {
		if !z.SuppliedAt.Valid || !z.NameDomain.Valid || z.NameDomain.String == "" {
			continue
		}
		intervals := int(now.Sub(z.SuppliedAt.Time) / interval)
		if intervals < staleZoneIntervals {
			continue
		}
		out = append(out, coverageStaleZoneView{
			Zone: z.NameDomain.String,
			Age:  fmt.Sprintf("%d re-supply intervals", intervals),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Zone < out[j].Zone })
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
