package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

// The sources sub-tab's view layer is the design-owned settings.tmpl (its
// "settings-sources" define, package v3.13.0). The repo authors no markup here; the
// handler below wires the source catalogue into the tmpl's declared holes.

// The three consent tiers a source runs under (v1 spec §3.1, ADR-0003, ADR-0023).
// consent names the door, never who walked through it: the value is authored by
// the project and ships in the release, so it is a constant of the catalogue
// below rather than a per-install fact.
const (
	consentUnencumbered = "unencumbered"
	consentAccepted     = "operator-accepted"
	consentCredentialed = "operator-credentialed"
)

// catalogSource is one authored row of the source catalogue. The catalogue is
// release data — identity, the three source properties, and the state a source
// *ships* in — the same for every install (ADR-0003). It is held in the binary,
// never the database: the only per-install fact is the operator's override of a
// shipped default, and that is what source_state holds.
//
// A proposer is deliberately kept in the same catalogue as a source even though
// ADR-0012 rules that a proposer is not a source: the enablement screen governs
// both (§6.4's source-enablement prompt covers the RIR registry paths, which
// propose), and the distinction is carried on the row rather than erased. A
// proposer admits nothing, so only consent applies to it — authority and
// completeness are left empty, never invented.
type catalogSource struct {
	Slug         string
	Name         string // the source's real name — a rendering, never the key
	IsProposer   bool   // ADR-0012: a proposer is not a source; only consent applies
	Authority    string // declared / measured / inferred; "" for a proposer
	Completeness string // enumerable / corroborative; "" for a proposer
	Consent      string // the tier it runs under; "" for a barred source
	DefaultOn    bool   // the state it ships in (§3.1)
	Barred       bool   // excluded on terms — no operator reading consents past it
	NoRunner     bool   // catalogued, but no execution path ships yet (#241) — nothing to enable
	ShipNote     string // project-worded, why it ships in this state; never the source's own terms

	// The two marked groups of the consent prompt (§6.4, ADR-0003 second
	// amendment, #47), rendered for an operator-accepted source at the moment
	// it is enabled. Stated in the project's own words about what is
	// unresolved, never the source's terms, and each group renders even when
	// empty — LACNIC's actionable group is empty by construction.
	MayResolve   []string // what you may be able to resolve — operator-varying questions
	Unresolvable []string // what nobody has been able to resolve — the constant questions
}

// sourceCatalog is the authored set the release ships. Defaults are §3.1's
// consent-bar ruling: the keyless RIR org→prefix paths (ARIN, AFRINIC, APNIC via
// CAIDA) on; HackerTarget and unauthenticated Cert Spotter excluded on terms. The
// four registry proposer paths (RIPEstat, RIPE Database, APNIC registry, LACNIC
// registry) are operator-accepted by tier but ship catalogued-not-executing: no
// proposer.Source runner emits for them yet, so they carry NoRunner (#241) — off
// for everyone, non-toggleable, no consent dialog offered — and return to the
// operator-accepted tier the moment a runner lands, the same reversal crt.sh made.
// crt.sh ships on and executing (§3.1, throttled): its runner is the ct Scan
// (ADR-0106), which polls certificate transparency and admits Names, so it is a
// live source again — reversing the not-yet-executing state #241 held it in until
// the runner landed.
var sourceCatalog = []catalogSource{
	{
		Slug: "crtsh", Name: "crt.sh",
		Authority: "inferred", Completeness: "corroborative", Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Certificate transparency logs. Admits the Names a certificate's SAN list carries — authority: inferred — never a wildcard, and observes nothing (ADR-0027). Queried on the ct Scan's daily cadence, throttled to 5 req/min; a failed fetch admits nothing and never an absence.",
	},
	{
		Slug: "ct-tail", Name: "CT drift tail (logs-direct)",
		Authority: "inferred", Completeness: "corroborative", Consent: consentUnencumbered,
		DefaultOn: false,
		ShipNote:  "Certificate transparency, read directly and forward-only (spec §4). Watches new issuance for names you already know, admitting the same way crt.sh does (authority: inferred, ADR-0027). Ships OFF: the tail downloads every new certificate across the CT logs to keep the few that match your estate, so it is heavier than the crt.sh poll — enable it when you want same-shard drift detection. A failed poll admits nothing and never an absence.",
	},
	{
		Slug: "arin", Name: "ARIN (entities?fn=)", IsProposer: true, Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Keyless org→prefix path. Covers North America.",
	},
	{
		Slug: "afrinic", Name: "AFRINIC (CAIDA ⋈ delegated-stats)", IsProposer: true, Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Keyless org→prefix path via CAIDA joined to delegated-stats.",
	},
	{
		Slug: "apnic-caida", Name: "APNIC (CAIDA ⋈ delegated-stats)", IsProposer: true, Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Keyless org→prefix path via CAIDA joined to delegated-stats.",
	},
	{
		Slug: "ripestat", Name: "RIPEstat", IsProposer: true, Consent: consentAccepted, NoRunner: true,
		ShipNote:     "Catalogued — no proposer runner ships for this path yet (#241), so it is off for everyone and offers no toggle. Its tier is operator-accepted: when a runner lands it returns off, enabled only by your own acceptance of the source's terms, and proposes address scopes that enter the estate only once you confirm a proposal into a seed.",
		MayResolve:   []string{"Whether you resell a service built on the source's data.", "Your own reading of whether writing prefixes to an inventory is re-packaging, and of the purpose list you are bound by."},
		Unresolvable: []string{"No reply has ever come, and no record of an approach exists."},
	},
	{
		Slug: "ripe-db", Name: "RIPE Database", IsProposer: true, Consent: consentAccepted, NoRunner: true,
		ShipNote:     "Catalogued — no proposer runner ships for this path yet (#241), so it is off for everyone and offers no toggle. Its tier is operator-accepted: when a runner lands it returns off, enabled only by your own acceptance of the source's terms, and proposes address scopes that enter the estate only once you confirm a proposal into a seed.",
		MayResolve:   []string{"Your own reading of whether inventorying your own estate is a permitted purpose."},
		Unresolvable: []string{"No reply has ever come, and no record of an approach exists."},
	},
	{
		Slug: "apnic-registry", Name: "APNIC registry", IsProposer: true, Consent: consentAccepted, NoRunner: true,
		ShipNote:     "Catalogued — no proposer runner ships for this path yet (#241), so it is off for everyone and offers no toggle. Its tier is operator-accepted: when a runner lands it returns off, enabled only by your own acceptance of the source's terms, and proposes address scopes that enter the estate only once you confirm a proposal into a seed.",
		MayResolve:   []string{"Whether you hold, or will seek, the registry's approval.", "Your own reading of the retrieval-system clause's carve-out."},
		Unresolvable: []string{"No reply has ever come, and no record of an approach exists."},
	},
	{
		Slug: "lacnic-registry", Name: "LACNIC registry", IsProposer: true, Consent: consentAccepted, NoRunner: true,
		ShipNote:     "Catalogued — no proposer runner ships for this path yet (#241), so it is off for everyone and offers no toggle. Its tier is operator-accepted, but its terms cannot be retrieved: when a runner lands, enabling it would accept a source whose terms nobody has been able to read.",
		MayResolve:   nil, // empty by construction — the actionable group renders empty here (#47)
		Unresolvable: []string{"Nobody has been able to retrieve these terms."},
	},
	{
		Slug: "hackertarget", Name: "HackerTarget",
		Authority: "measured", Completeness: "corroborative", Barred: true,
		ShipNote: "Excluded on terms. Its terms bar the software's inherent behaviour, which fails regardless of who the operator is — so no operator reading consents past it.",
	},
	{
		Slug: "certspotter", Name: "Cert Spotter (operator key)",
		Authority: "inferred", Completeness: "corroborative", Consent: consentCredentialed,
		DefaultOn: false,
		ShipNote:  "Certificate transparency, bulk-by-name — the operator-keyed primary (spec §2). Set VERGE_CERTSPOTTER_TOKEN on the worker to select it as the active ct source in place of crt.sh; absent the key, crt.sh runs. Admits the Names a certificate's SAN list carries — authority: inferred — never a wildcard, and observes nothing (ADR-0027), the same way crt.sh does. Its authenticated tier clears the consent bar (ADR-0003); the key is worker-only and web never reads it.",
	},
}

func catalogBySlug(slug string) (catalogSource, bool) {
	for _, c := range sourceCatalog {
		if c.Slug == slug {
			return c, true
		}
	}
	return catalogSource{}, false
}

// sourceView is a catalogue row merged with its per-install override, shaped for
// rendering. Kind and consent stay visible so a proposer never reads as a source.
type sourceView struct {
	Slug         string
	Name         string
	KindLabel    string // "source" or "proposer"
	Authority    string
	Completeness string
	Consent      string
	Enabled      bool
	Toggleable   bool
	NoRunner     bool // catalogued, no execution path yet (#241)
	ShipNote     string
	ShowGroups   bool
	MayResolve   []string
	Unresolvable []string
}

// dnsQtypeSet is the qtype set the dns Scan puts on the wire, one aperture input
// of the seven (§3.2). It is declared release data — authored by the project and
// shipped in the binary, the same for every install — so it is held here beside
// the source catalogue rather than read from Postgres. The authority is
// resolutionwalk.DefaultOffers().Qtypes; this mirror is asserted equal to it by
// TestDNSQtypeSetMatchesLeaf so the two never drift.
var dnsQtypeSet = []string{"A", "AAAA", "CNAME", "NS", "SOA", "MX", "TXT"}

// cadenceLabel renders a Scan's cadence in seconds as the phrase the aperture
// statement states. The common shipped cadences get a word; anything else is
// stated in its plain unit rather than invented into a near-fit.
func cadenceLabel(seconds int64) string {
	switch {
	case seconds <= 0:
		return "—"
	case seconds%86400 == 0:
		if seconds == 86400 {
			return "daily"
		}
		return fmt.Sprintf("every %d days", seconds/86400)
	case seconds%3600 == 0:
		if seconds == 3600 {
			return "hourly"
		}
		return fmt.Sprintf("every %d hours", seconds/3600)
	case seconds%60 == 0:
		return fmt.Sprintf("every %d minutes", seconds/60)
	default:
		return fmt.Sprintf("every %d seconds", seconds)
	}
}

// sourcesModal renders the source-enablement surface (§6.4) as the Settings
// sources sub-tab: the discovery-source catalogue split by the state each source
// ships in, with the two marked consent groups on every operator-accepted source.
// A viewer may read it; only an admin sees a toggle. A source is a discovery
// source — distinct from an Integration (a third-party install tile, on its own
// sub-tab, #308): #281 originally parked this catalogue under the "integrations"
// tab as a stopgap before the real Integrations screen existed; #308 gave
// integrations their own tab and returned this catalogue to its own "sources" tab.
func (s *server) sourcesModal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: "sources"})
}

// sourceTierRow is one source shaped for the spec tier cards (#26): its id, name,
// kind label (source/proposer), the project's one-line note, and its effective on
// state. The three tiers — unencumbered / operator-accepted / barred — are the
// release-authored consent tiers.
type sourceTierRow struct {
	ID   string
	Name string
	Kind string
	What string
	On   bool
}

// fillSourcesSection buckets the discovery-source catalogue into the three spec
// consent tiers for the sources sub-tab, and opens the consent dialog when
// ?consent=<id> names an operator-accepted source that is currently off.
func (s *server) fillSourcesSection(r *http.Request, f settingsForms, data map[string]any) error {
	views, err := s.sourceViews(r)
	if err != nil {
		return err
	}

	var unencumbered, operatorAccepted, barred []sourceTierRow
	for _, v := range views {
		row := sourceTierRow{ID: v.Slug, Name: v.Name, Kind: v.KindLabel, What: v.ShipNote, On: v.Enabled}
		// A catalogued source with no runner (#241) is off for everyone, non-toggleable,
		// and offers no consent — regardless of its consent tier. It is bucketed before
		// the operator-accepted case, which would otherwise claim a consent-accepted
		// proposer that has no runner (the four RIR registry proposers, ruling #30).
		switch {
		case v.NoRunner:
			barred = append(barred, row)
		case v.Consent == consentAccepted:
			operatorAccepted = append(operatorAccepted, row)
		case v.Toggleable: // unencumbered, runnable
			unencumbered = append(unencumbered, row)
		default: // barred — excluded on terms; also stays off for everyone
			barred = append(barred, row)
		}
	}

	data["Unencumbered"] = unencumbered
	data["OperatorAccepted"] = operatorAccepted
	data["Barred"] = barred
	data["SourceError"] = f.sourceError

	// The measured reliability bar for the bulk CT sources (spec §3, #879): the
	// pass/fail-per-limb and degraded state the CT-source card renders. Exposed here
	// for the UI; #880/#881 render the active-source hero and KPI tiles from it.
	rel, err := s.ctReliabilityViews(r.Context())
	if err != nil {
		return err
	}
	data["CTReliability"] = rel
	data["CTReliabilityBar"] = ctReliabilityBar{
		SuccessTarget: fmt.Sprintf("≥ %d%%", int(scan.CTSuccessRateBar*100)),
		LatencyTarget: fmt.Sprintf("≤ %d s", scan.CTP95LatencyBarMS/1000),
	}

	// The active-source hero (#880, spec §6): which bulk source is live, its reliability
	// against the bar, and the last run's readout. It reads the two reliability windows and
	// the last ct Batch's admitted-name count — never the worker token (spec §2.4).
	names, err := s.store.CTLastBatchAdmitCount(r.Context())
	if err != nil {
		return err
	}
	var crtshView, certView ctReliabilityView
	for _, v := range rel {
		switch v.Slug {
		case scan.CrtshSource:
			crtshView = v
		case scan.CertSpotterSource:
			certView = v
		}
	}
	data["CTHero"] = newCTSourceHero(crtshView, certView, names, s.now())

	// The consent dialog (#26): opened by ?consent=<id>, it renders that source's
	// terms and the acceptance checkbox. It renders only for an operator-accepted,
	// currently-off source; a stray param opens no dialog.
	if id := r.URL.Query().Get("consent"); id != "" {
		// A catalogued-not-executing source (#241) offers no consent dialog even
		// though its tier is operator-accepted — there is nothing to enable yet.
		if c, ok := catalogBySlug(id); ok && c.Consent == consentAccepted && !c.NoRunner {
			data["Consent"] = map[string]any{
				"ID": c.Slug, "Name": c.Name, "Terms": consentTerms(c),
			}
		}
	}
	return nil
}

// consentTerms flattens a source's unresolved-reading groups into the flat terms
// list the consent dialog renders (#26). The project states what is unresolved in
// its own words, never the source's terms verbatim.
func consentTerms(c catalogSource) []string {
	terms := make([]string, 0, len(c.MayResolve)+len(c.Unresolvable))
	terms = append(terms, c.MayResolve...)
	terms = append(terms, c.Unresolvable...)
	return terms
}

// sourceViews merges the authored catalogue with the operator's overrides. A
// source's effective state is its override where one exists and its shipped
// default otherwise.
func (s *server) sourceViews(r *http.Request) ([]sourceView, error) {
	states, err := s.store.ListSourceStates(r.Context())
	if err != nil {
		return nil, err
	}
	override := make(map[string]bool, len(states))
	for _, st := range states {
		override[st.Slug] = st.Enabled
	}

	out := make([]sourceView, 0, len(sourceCatalog))
	for _, c := range sourceCatalog {
		enabled := c.DefaultOn
		if o, ok := override[c.Slug]; ok {
			enabled = o
		}
		// A source with no runner has nothing to enable — its effective state is
		// never `on`, whatever an override or default might say — and it is not
		// toggleable, exactly as a barred source is not (#241).
		if c.NoRunner {
			enabled = false
		}
		kind := "source"
		if c.IsProposer {
			kind = "proposer"
		}
		out = append(out, sourceView{
			Slug: c.Slug, Name: c.Name, KindLabel: kind,
			Authority: c.Authority, Completeness: c.Completeness, Consent: c.Consent,
			Enabled: enabled, Toggleable: !c.Barred && !c.NoRunner, NoRunner: c.NoRunner,
			ShipNote:     c.ShipNote,
			ShowGroups:   c.Consent == consentAccepted,
			MayResolve:   c.MayResolve,
			Unresolvable: c.Unresolvable,
		})
	}
	return out, nil
}

// ctReliabilityView is one bulk CT source's reliability-bar card data for the
// sources tab (spec §3, §6.2/§6.4, #879). It carries the measured limbs display-ready
// and a pass/fail per limb — or, for the keyless fallback (crt.sh), the exempt
// marking so it renders muted, not failed. Degraded is the below-bar primary state
// the card surfaces without a silent swap to crt.sh (runtime failover is deferred,
// spec §7). HasData is false for a source with no recent samples, so the card reads
// "no recent data" rather than a false failure.
type ctReliabilityView struct {
	Slug     string
	Name     string
	Exempt   bool
	HasData  bool
	Degraded bool
	Samples  int64
	LastRun  time.Time // newest sample's instant; zero with no data. The active-source hero
	//                    (#880) reads it to tell which bulk source is live — only the
	//                    config-selected source keeps recording, so the freshest wins.

	SuccessPct  string // e.g. "99.5%", or "—" with no data
	SuccessPass bool

	P95Display  string // e.g. "3.2 s", or "—" with no data
	LatencyPass bool

	FalseEmpty     int64
	FalseEmptyPass bool
}

// ctReliabilityBar is the bar's targets, formatted for the KPI tiles (spec §3). It is
// release-authored, the same for every install, so it is derived from the scan-package
// constants rather than read from Postgres.
type ctReliabilityBar struct {
	SuccessTarget string // "≥ 99%"
	LatencyTarget string // "≤ 5 s"
}

// ctReliabilityViews reads and evaluates the reliability bar for the two bulk CT
// sources (spec §3, #879). The tail (ct-tail) is not bulk-by-name and carries no bar.
// The worker records a sample per bulk query; this reads each source's rolling window
// and evaluates it against the bar. crt.sh reports exempt, the operator-keyed primary
// reports pass/fail per limb and degraded when it misses one.
func (s *server) ctReliabilityViews(ctx context.Context) ([]ctReliabilityView, error) {
	slugs := []string{scan.CrtshSource, scan.CertSpotterSource}
	out := make([]ctReliabilityView, 0, len(slugs))
	for _, slug := range slugs {
		row, err := s.store.CTReliabilityWindow(ctx, db.CTReliabilityWindowParams{
			Source:     slug,
			SampleSize: scan.CTReliabilityWindowSize,
		})
		if err != nil {
			return nil, err
		}
		report := scan.EvaluateCTReliability(slug, scan.CTReliabilityWindow{
			Total:        row.Total,
			Successes:    row.Successes,
			Empties:      row.Empties,
			P95LatencyMS: row.P95LatencyMs,
		})
		name := slug
		if c, ok := catalogBySlug(slug); ok {
			name = c.Name
		}
		var lastRun time.Time
		if row.LastAt.Valid {
			lastRun = row.LastAt.Time
		}
		out = append(out, newCTReliabilityView(name, lastRun, report))
	}
	return out, nil
}

// newCTReliabilityView shapes one evaluated report for rendering, formatting the
// measured success rate as a percentage and the p95 latency in seconds. A source with
// no samples shows an em dash for both, never a fabricated zero.
func newCTReliabilityView(name string, lastRun time.Time, r scan.CTReliabilityReport) ctReliabilityView {
	v := ctReliabilityView{
		Slug: r.Source, Name: name, LastRun: lastRun,
		Exempt: r.Exempt, HasData: r.HasData, Degraded: r.Degraded, Samples: r.Samples,
		SuccessPass: r.SuccessPass, LatencyPass: r.LatencyPass,
		FalseEmpty: r.FalseEmpty, FalseEmptyPass: r.FalseEmptyPass,
		SuccessPct: "—", P95Display: "—",
	}
	if r.HasData {
		v.SuccessPct = fmt.Sprintf("%.1f%%", r.SuccessRate*100)
		v.P95Display = fmt.Sprintf("%.1f s", float64(r.P95LatencyMS)/1000)
	}
	return v
}

// ctSourceHero is the active-source hero the CT theme leads with (#880, spec §6.1). It
// names which bulk source is live, derived from the freshest reliability sample: web
// never reads the worker's VERGE_CERTSPOTTER_TOKEN (spec §2.4, ADR-0053), and only the
// config-selected source keeps recording samples, so the fresher window is the one this
// config runs. Key presence is inferred from that selection, never from the token —
// Cert Spotter live means the key is set (spec §2.3's exact key⇒source mapping), crt.sh
// live means it is not. The run readout and the KPI-tile source (Active) both read the
// live source. A below-bar primary sets Degraded, so the card draws the honest edge
// (§6.3): the Scan keeps running the primary, there is no silent swap to crt.sh.
type ctSourceHero struct {
	HasRun      bool              // at least one bulk source has recorded a sample
	IsPrimary   bool              // the operator-keyed primary (Cert Spotter) is live, not the crt.sh fallback
	StatusClass string            // the badge variant: accent (primary), danger (primary under bar), neutral (fallback)
	StatusLabel string            // "primary · Cert Spotter" / "fallback · crt.sh"
	DormantName string            // the source that would run under the other config
	DormantRole string            // "fallback" / "primary" — the role the dormant source would fill
	KeyDetected bool              // VERGE_CERTSPOTTER_TOKEN presence, inferred from the live source
	KeyLabel    string            // "detected" / "not set"
	LastRunRel  string            // "4m" — age of the last bulk run; "" with no run
	Names       int64             // Names the last ct Batch admitted
	Degraded    bool              // the live primary is under its bar (§6.3): no silent swap
	Active      ctReliabilityView // the live source's limbs, for the three KPI tiles
}

// newCTSourceHero derives the hero from the two bulk sources' reliability windows and the
// last ct Batch's admitted-name count. The live source is whichever still records samples;
// with samples from both, the fresher window wins. With no sample from either, no bulk ct
// scan has run under this deployment yet, so nothing is asserted live — the card names
// crt.sh as the keyless default and how to promote the primary, without claiming a run.
func newCTSourceHero(crtsh, certspotter ctReliabilityView, names int64, now time.Time) ctSourceHero {
	certName := strings.TrimSuffix(certspotter.Name, " (operator key)")
	crtHas := crtsh.HasData && !crtsh.LastRun.IsZero()
	certHas := certspotter.HasData && !certspotter.LastRun.IsZero()

	if !crtHas && !certHas {
		return ctSourceHero{
			StatusClass: "neutral",
			StatusLabel: "fallback · " + crtsh.Name,
			DormantName: certName,
			DormantRole: "primary",
			KeyLabel:    "not set",
		}
	}

	if certHas && (!crtHas || certspotter.LastRun.After(crtsh.LastRun)) {
		h := ctSourceHero{
			HasRun:      true,
			IsPrimary:   true,
			StatusClass: "accent",
			StatusLabel: "primary · " + certName,
			DormantName: crtsh.Name,
			DormantRole: "fallback",
			KeyDetected: true,
			KeyLabel:    "detected",
			LastRunRel:  profileRelTime(certspotter.LastRun, now),
			Names:       names,
			Degraded:    certspotter.Degraded,
			Active:      certspotter,
		}
		if h.Degraded {
			h.StatusClass = "danger" // a below-bar primary reads in danger (§6.3)
		}
		return h
	}

	return ctSourceHero{
		HasRun:      true,
		StatusClass: "neutral",
		StatusLabel: "fallback · " + crtsh.Name,
		DormantName: certName,
		DormantRole: "primary",
		KeyLabel:    "not set",
		LastRunRel:  profileRelTime(crtsh.LastRun, now),
		Names:       names,
		Active:      crtsh,
	}
}

// toggleSource records an admin's on/off choice for one source. Toggling is an
// authenticated admin act (it reaches here only through requireAdmin). A barred
// source has no consent instrument the modal operator can satisfy, so it cannot
// be toggled on or off; an unknown slug is refused rather than written.
func (s *server) toggleSource(w http.ResponseWriter, r *http.Request, acct db.Account) {
	slug := r.FormValue("slug")
	c, ok := catalogBySlug(slug)
	// A barred source has no consent instrument to satisfy, and a source with no
	// runner (#241) has nothing to run, so neither is toggleable; an unknown slug
	// is refused rather than written.
	if !ok || c.Barred || c.NoRunner {
		http.Error(w, "unknown source", http.StatusBadRequest)
		return
	}
	enabled, err := strconv.ParseBool(r.FormValue("enabled"))
	if err != nil {
		http.Error(w, "bad state", http.StatusBadRequest)
		return
	}
	// Enabling an operator-accepted source is gated on accepting its terms: the
	// project could not clear them on your behalf, so the enable act must carry your
	// acceptance. Without it, bounce back to the terms dialog rather than enabling —
	// a real gate, not only a UI affordance. Disabling and unencumbered sources are
	// never gated.
	if enabled && c.Consent == consentAccepted && r.FormValue("agreed") == "" {
		http.Redirect(w, r, "/sources?terms="+url.QueryEscape(slug), http.StatusSeeOther)
		return
	}
	if _, err := s.store.UpsertSourceState(r.Context(), db.UpsertSourceStateParams{
		Slug: slug, Enabled: enabled,
	}); err != nil {
		s.serverError(w, "upsert source state", err)
		return
	}
	http.Redirect(w, r, "/sources", http.StatusSeeOther)
}

// settingsSources records an admin's on/off choice from the spec sources tab (#26).
// It is the settings-tab twin of toggleSource: the form posts an id and an enable
// flag, and enabling an operator-accepted source carries accept_terms=true from the
// consent dialog. Without that acceptance, enabling an operator-accepted source
// bounces to the consent dialog (?consent=<id>) rather than enabling — a real gate,
// not only a UI affordance. It is reached only through requireAdmin.
func (s *server) settingsSources(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id := r.FormValue("id")
	c, ok := catalogBySlug(id)
	if !ok || c.Barred || c.NoRunner {
		s.renderSettings(w, r, acct, settingsForms{section: "sources", sourceError: "That source could not be found."})
		return
	}
	enable, err := strconv.ParseBool(r.FormValue("enable"))
	if err != nil {
		s.renderSettings(w, r, acct, settingsForms{section: "sources", sourceError: "That source state was not understood."})
		return
	}
	// Enabling an operator-accepted source is gated on accepting its terms.
	if enable && c.Consent == consentAccepted && r.FormValue("accept_terms") != "true" {
		http.Redirect(w, r, "/settings?tab=sources&consent="+url.QueryEscape(id), http.StatusSeeOther)
		return
	}
	if _, err := s.store.UpsertSourceState(r.Context(), db.UpsertSourceStateParams{
		Slug: id, Enabled: enable,
	}); err != nil {
		s.serverError(w, "upsert source state", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=sources", http.StatusSeeOther)
}
