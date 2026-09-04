package main

import (
	"fmt"
	"html/template"
	"log"
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

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/coverage.tmpl"))

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

type coldScopeView struct {
	ID        int64
	IsAddress bool
	Scope     string
	OptedIn   bool
}

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

func (s *server) setColdScope(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// Adding a scope queues nothing: the cold tier never runs unasked (v1-spec §3.4, ADR-0044).
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "scans", coldError: "That scope could not be found."})
		return
	}
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
	if err := s.store.SyncColdScanEnabled(r.Context()); err != nil {
		s.serverError(w, "sync cold scan enabled", err)
		return
	}
	s.backToSection(w, r, "scans")
}

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

type coverageGapView struct {
	Subject  string
	Gap      string
	Expected string
	Since    string
}

type coverageMessageView struct {
	Kind    string
	Badge   string
	Bound   string
	Subject string
	Text    string
	When    string
	ISO     string
}

type unevaluableRuleView struct {
	ID      string
	Version string
	Why     string
}

type coverageStaleZoneView struct {
	Zone string
	Age  string
}

func (s *server) coveragePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	if s.devMode {
		s.render(w, r, "coverage", s.coverageFixtureData(acct))
		return
	}

	seeds, err := s.store.ListSeeds(ctx)
	if err != nil {
		s.serverError(w, "list seeds", err)
		return
	}
	var zones []db.ListZoneDeclarationsRow
	if z, zerr := s.store.ListZoneDeclarations(ctx); zerr == nil {
		zones = z
	}
	var walked []walkedAddr
	if svcs, serr := s.store.ListCurrentServiceSubjects(ctx, db.ListCurrentServiceSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); serr == nil {
		walked = walkedAddresses(svcs)
	}
	var sharedEdges map[netip.Prefix]int
	// A hidden-evidence degrade logs; the empty-region degrades around it do not (#989).
	if m, ferr := addressScopeSharedEdges(ctx, s.store); ferr == nil {
		sharedEdges = m
	} else {
		log.Printf("web: coverage: address-scope shared edges: %v", ferr)
	}
	meters := apertureMeters(seeds, zones, walked, s.now(), sharedEdges)

	var gaps []coverageGapView
	var messages []coverageMessageView
	if svc, berr := s.store.ListBlanketedReachServices(ctx); berr == nil {
		gaps, messages = blanketGapsAndMessages(svc)
	}
	if rows, uerr := s.store.ListUnavailableVantages(ctx); uerr == nil {
		messages = append(messages, unavailableVantageMessages(rows)...)
	}
	sortCoverageMessages(messages)

	var unevaluable []unevaluableRuleView
	if corpus, cerr := s.buildSignalCorpus(r); cerr == nil {
		unevaluable = unevaluableRules(corpus)
	}

	var staleZonesView []coverageStaleZoneView
	if cadence, cerr := s.store.GetZoneCadenceSeconds(ctx); cerr == nil && cadence > 0 {
		if rows, zerr := s.store.ListZoneFileStatus(ctx); zerr == nil {
			staleZonesView = staleZones(rows, cadence, time.Now())
		}
	}

	s.render(w, r, "coverage", map[string]any{
		"Title": "Coverage", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":   "coverage",
		"Meters":      meters,
		"Messages":    messages,
		"Gaps":        gaps,
		"Unevaluable": unevaluable,
		"StaleZones":  staleZonesView,
	})
}

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
		// A name scope's shared edge belongs to the custody-extension census, never this meter (#987).
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

// The address-scope cap has no ceiling, so a range's count need not fit int (ADR-0127).

const maxMeterTotal = int64(^uint(0) >> 1)

func addressMeter(p netip.Prefix, walked []walkedAddr, now time.Time, sharedEdges int) coverageMeterView {
	// The contradiction row is display, never a gate: every counted address is still probed (#989).
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

type walkedAddr struct {
	Addr       netip.Addr
	ObservedAt time.Time
}

func coveredInRange(walked []walkedAddr, p netip.Prefix) int {
	seen := make(map[netip.Addr]struct{}, len(walked))
	for _, w := range walked {
		if p.Contains(w.Addr) {
			seen[w.Addr] = struct{}{}
		}
	}
	return len(seen)
}

func oldestCurrentInRange(walked []walkedAddr, p netip.Prefix) (time.Time, bool) {
	var oldest time.Time
	// The trailing-edge staleness Gap folds to this one figure, never a per-address message (#882).
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

// A zone past this bound ages into a Gap rather than being trusted (docs/guides/zone-files.md).

const staleZoneIntervals = 2

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

func sortCoverageMessages(msgs []coverageMessageView) {
	sort.SliceStable(msgs, func(i, j int) bool {
		if msgs[i].Kind != msgs[j].Kind {
			return msgs[i].Kind < msgs[j].Kind
		}
		return msgs[i].Subject < msgs[j].Subject
	})
}

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
