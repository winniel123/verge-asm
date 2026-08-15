package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Subjects screen (v1 spec §6.6, ADR-0072). At wave-0 only `Name` subjects
// exist — they come from the `resolution-walk` leaf's `resolution` facet (#188).
// The listing is the estate alone: every current Name, searchable, with **no
// denominator** — estate completeness is unmeasurable and refusing to state it
// is the model's closest analogue to honesty. A withdrawn Name is not a row; it
// is reached by its own key on the drill-down, marked as naming a population of
// no current member. Address/Service/Endpoint arrive with later measurement
// tickets and have no surface here yet.

// nameOutcomeNameError is the resolution outcome that withdraws a Name: our own
// resolver measuring a Name Error is the one route a Name leaves the estate
// (ADR-0006). It mirrors resolutionwalk.OutcomeNameError, kept as a local
// constant so the web binary reads the stored value without importing the leaf.
const nameOutcomeNameError = "NameError"

// resolutionValue is the JSON payload of a resolution observation, the shape the
// resolution-walk leaf emits. The web layer reads only the fields it renders.
type resolutionValue struct {
	Outcome   string   `json:"outcome"`
	Addresses []string `json:"addresses"`
}

func decodeResolution(raw []byte) resolutionValue {
	var v resolutionValue
	_ = json.Unmarshal(raw, &v)
	return v
}

// subjectRow is one Name in the estate listing: its rendered key (also the link
// target) and its current resolution value. No count and no membership badge —
// a row's key is a link that carries no state (ADR-0072).
type subjectRow struct {
	Name       string
	Resolution string
}

// citationHop is one link in a subject's "why is this here" chain, rendered
// top-to-bottom from the subject down to the Seed the chain terminates at.
type citationHop struct {
	Label  string // the micro-label: what kind of hop this is
	Value  string // the load-bearing value, rendered mono
	Detail string // optional muted qualifier
}

// subjectPageData is the drill-down view for one Name.
type subjectPageData struct {
	Name       string
	Withdrawn  bool
	Resolution string
	Addresses  []string
	Citation   []citationHop
	// CitationTerminated reports whether the chain reached a Seed. It always
	// should for a measured Name; a false here is an integrity gap worth showing
	// rather than hiding.
	CitationTerminated bool
}

func (s *server) subjectsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	search := strings.TrimSpace(r.FormValue("q"))
	rows, err := s.store.ListCurrentNameSubjects(r.Context(), search)
	if err != nil {
		s.serverError(w, "list current name subjects", err)
		return
	}
	views := make([]subjectRow, 0, len(rows))
	for _, row := range rows {
		views = append(views, subjectRow{
			Name:       row.SubjectKey,
			Resolution: decodeResolution(row.Value).Outcome,
		})
	}
	s.render(w, "subjects", map[string]any{
		"Title": "Subjects", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Subjects": views, "Search": search,
	})
}

func (s *server) subjectPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	key := r.PathValue("key")
	subject, err := s.store.GetNameSubject(r.Context(), key)
	if errors.Is(err, pgx.ErrNoRows) {
		// A Name nothing has ever measured is genuinely not a subject — not a
		// withdrawn one. Refusing it here is not the false absence ADR-0072
		// guards against: that guard is about a Name we measured *gone*, which
		// GetNameSubject still returns.
		s.renderStatus(w, http.StatusNotFound, "subject-missing", map[string]any{
			"Title": "No such subject", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"Name": key,
		})
		return
	}
	if err != nil {
		s.serverError(w, "get name subject", err)
		return
	}

	res := decodeResolution(subject.Value)
	data := subjectPageData{
		Name:       subject.SubjectKey,
		Withdrawn:  res.Outcome == nameOutcomeNameError,
		Resolution: res.Outcome,
		Addresses:  res.Addresses,
	}
	data.Citation, data.CitationTerminated = s.buildCitation(r, subject.SubjectKey)

	s.render(w, "subject", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Subject": data,
	})
}

// buildCitation assembles the "why is this here" chain for a Name: the subject
// itself, the observation that introduced it, and the Seed the chain terminates
// at. Every hop is best-effort — a missing hop degrades to a shorter chain
// rather than a 500, since the card is diagnostic and a partial answer still
// helps. It reports whether the chain reached a Seed.
func (s *server) buildCitation(r *http.Request, key string) ([]citationHop, bool) {
	hops := []citationHop{{
		Label: "Subject · Name", Value: key,
	}}

	terminated := false
	cit, err := s.store.GetNameCitation(r.Context(), key)
	if err == nil {
		detail := "source " + cit.Source
		if cit.ObservedAt.Valid {
			detail = "first measured " + cit.ObservedAt.Time.UTC().Format("2006-01-02 15:04 UTC") + " · " + detail
		}
		hops = append(hops, citationHop{
			Label:  "Introduced by · observation",
			Value:  "resolution-walk · " + cit.ScanKind + " Scan · batch #" + strconv.FormatInt(cit.BatchID, 10),
			Detail: detail,
		})
	}

	seed, err := s.store.FindCoveringNameSeed(r.Context(), key)
	if err == nil {
		detail := ""
		if seed.CreatedByUsername != "" {
			detail = "declared by " + seed.CreatedByUsername
		}
		if seed.CreatedAt.Valid {
			if detail != "" {
				detail += " · "
			}
			detail += seed.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		hops = append(hops, citationHop{
			Label:  "Declared · Seed",
			Value:  "name scope " + seed.NameDomain.String,
			Detail: detail,
		})
		terminated = true
	}

	return hops, terminated
}
