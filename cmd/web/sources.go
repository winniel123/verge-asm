package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

const (
	consentUnencumbered = "unencumbered"
	consentAccepted     = "operator-accepted"
	consentCredentialed = "operator-credentialed"
)

type catalogSource struct {
	Slug         string
	Name         string
	IsProposer   bool // a proposer is not a source, so only consent applies (ADR-0012)
	Authority    string
	Completeness string
	Consent      string
	DefaultOn    bool
	Barred       bool
	NoRunner     bool // no runner ships, so it stays off and untoggleable (#241)
	ShipNote     string

	// Each group renders even when empty (v1-spec §6.4, #47).

	MayResolve   []string
	Unresolvable []string
}

// The shipped on/off state is ADR-0003's consent-bar ruling, not a preference (v1-spec §3.1).

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
		MayResolve:   nil,
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

type sourceView struct {
	Slug         string
	Name         string
	KindLabel    string
	Authority    string
	Completeness string
	Consent      string
	Enabled      bool
	Toggleable   bool
	NoRunner     bool
	ShipNote     string
	ShowGroups   bool
	MayResolve   []string
	Unresolvable []string
}

var dnsQtypeSet = []string{"A", "AAAA", "CNAME", "NS", "SOA", "MX", "TXT"}

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

func (s *server) sourcesModal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, s.takeSettingsFlash(r, tabForSection("sources")))
}

type sourceTierRow struct {
	ID   string
	Name string
	Kind string
	What string
	On   bool
}

func (s *server) fillSourcesSection(r *http.Request, f settingsForms, data map[string]any) error {
	views, err := s.sourceViews(r)
	if err != nil {
		return err
	}

	var unencumbered, operatorAccepted, barred []sourceTierRow
	for _, v := range views {
		row := sourceTierRow{ID: v.Slug, Name: v.Name, Kind: v.KindLabel, What: v.ShipNote, On: v.Enabled}
		switch {
		case v.NoRunner:
			barred = append(barred, row)
		case v.Consent == consentAccepted:
			operatorAccepted = append(operatorAccepted, row)
		case v.Toggleable:
			unencumbered = append(unencumbered, row)
		default:
			barred = append(barred, row)
		}
	}

	data["Unencumbered"] = unencumbered
	data["OperatorAccepted"] = operatorAccepted
	data["Barred"] = barred
	data["SourceError"] = f.sourceError

	rel, err := s.ctReliabilityViews(r.Context())
	if err != nil {
		return err
	}
	data["CTReliability"] = rel
	data["CTReliabilityBar"] = ctReliabilityBar{
		SuccessTarget: fmt.Sprintf("≥ %d%%", int(scan.CTSuccessRateBar*100)),
		LatencyTarget: fmt.Sprintf("≤ %d s", scan.CTP95LatencyBarMS/1000),
	}

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

	tailEnabled := false
	for _, v := range views {
		if v.Slug == scan.CTTailSource {
			tailEnabled = v.Enabled
			break
		}
	}
	tail, err := s.store.CTTailLastBatch(r.Context())
	if err != nil {
		return err
	}
	captured, err := s.store.CountCertificateMaterial(r.Context())
	if err != nil {
		return err
	}
	data["CTCapabilities"] = newCTCapabilities(tailEnabled, tail, captured, s.now())

	if id := r.URL.Query().Get("consent"); id != "" {
		if c, ok := catalogBySlug(id); ok && c.Consent == consentAccepted && !c.NoRunner {
			data["Consent"] = map[string]any{
				"ID": c.Slug, "Name": c.Name, "Terms": consentTerms(c),
			}
		}
	}
	return nil
}

func consentTerms(c catalogSource) []string {
	// The project states what is unresolved in its own words, never the source's (ADR-0003).
	terms := make([]string, 0, len(c.MayResolve)+len(c.Unresolvable))
	terms = append(terms, c.MayResolve...)
	terms = append(terms, c.Unresolvable...)
	return terms
}

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

type ctReliabilityView struct {
	Slug     string
	Name     string
	Exempt   bool
	HasData  bool
	Degraded bool
	Samples  int64
	LastRun  time.Time

	SuccessPct  string
	SuccessPass bool

	P95Display  string
	LatencyPass bool

	FalseEmpty     int64
	FalseEmptyPass bool
}

type ctReliabilityBar struct {
	SuccessTarget string
	LatencyTarget string
}

func (s *server) ctReliabilityViews(ctx context.Context) ([]ctReliabilityView, error) {
	// The tail is not bulk-by-name, so the reliability bar excludes it (ct-source-replacement §3).
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

// A below-bar primary is never swapped: runtime failover is deferred (ct-source-replacement §6.3).

type ctSourceHero struct {
	HasRun      bool
	IsPrimary   bool
	StatusClass string
	StatusLabel string
	DormantName string
	DormantRole string
	KeyDetected bool
	KeyLabel    string
	LastRunRel  string
	Names       int64
	Degraded    bool
	Active      ctReliabilityView
}

func newCTSourceHero(crtsh, certspotter ctReliabilityView, names int64, now time.Time) ctSourceHero {
	// The key lives on the worker, so liveness is inferred from the freshest sample (ADR-0053).
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
			h.StatusClass = "danger"
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

// Verification keeps no durable result, so the pool is its readout (ct-source-replacement §5).

type ctCapabilities struct {
	TailEnabled bool
	TailHasRun  bool
	TailLastRel string
	TailNames   int64
	Captured    int64
}

func newCTCapabilities(tailEnabled bool, tail db.CTTailLastBatchRow, captured int64, now time.Time) ctCapabilities {
	c := ctCapabilities{
		TailEnabled: tailEnabled,
		TailNames:   tail.Names,
		Captured:    captured,
	}
	if tail.LastAt.Valid {
		c.TailHasRun = true
		c.TailLastRel = profileRelTime(tail.LastAt.Time, now)
	}
	return c
}

func (s *server) toggleSource(w http.ResponseWriter, r *http.Request, acct db.Account) {
	slug := r.FormValue("slug")
	c, ok := catalogBySlug(slug)
	if !ok || c.Barred || c.NoRunner {
		s.failSettings(w, r, settingsForms{section: "sources", sourceError: "That source could not be found."})
		return
	}
	enabled, err := strconv.ParseBool(r.FormValue("enabled"))
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sources", sourceError: "That source state was not understood."})
		return
	}
	if enabled && c.Consent == consentAccepted && r.FormValue("agreed") == "" {
		s.failSettings(w, r, settingsForms{
			section:     "sources",
			sourceError: "Accept the terms before you enable " + c.Name + ".",
		})
		return
	}
	if _, err := s.store.UpsertSourceState(r.Context(), db.UpsertSourceStateParams{
		Slug: slug, Enabled: enabled,
	}); err != nil {
		s.serverError(w, "upsert source state", err)
		return
	}
	s.redirectBack(w, r, "/sources")
}

func (s *server) settingsSources(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id := r.FormValue("id")
	c, ok := catalogBySlug(id)
	if !ok || c.Barred || c.NoRunner {
		s.failSettings(w, r, settingsForms{section: "sources", sourceError: "That source could not be found."})
		return
	}
	enable, err := strconv.ParseBool(r.FormValue("enable"))
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sources", sourceError: "That source state was not understood."})
		return
	}
	// The project could not clear these terms for you, so enabling carries your consent (ADR-0003).
	if enable && c.Consent == consentAccepted && r.FormValue("accept_terms") != "true" {
		s.failSettings(w, r, settingsForms{
			section:     "sources",
			sourceError: "Accept the terms before you enable " + c.Name + ".",
		})
		return
	}
	if _, err := s.store.UpsertSourceState(r.Context(), db.UpsertSourceStateParams{
		Slug: id, Enabled: enable,
	}); err != nil {
		s.serverError(w, "upsert source state", err)
		return
	}
	s.backToSection(w, r, "sources")
}
