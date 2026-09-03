package queue

import (
	"context"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/scan"
)

// nameSeedDomains reads the name-scope Seed domains the dns Scan resolves over —
// unconditionally, independent of Custody (ADR-0084).
func nameSeedDomains(ctx context.Context, q *db.Queries) ([]string, error) {
	rows, err := q.ListNameSeedDomains(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Valid {
			names = append(names, r.String)
		}
	}
	return names, nil
}

// admittedNames reads the distinct CT-admitted names the dns Scan also resolves
// (ADR-0107, wave-1). It is unconditional of the crtsh source's current
// enablement: resolution is the dns Scan's act, not the source's; ADR-0003
// consent was spent at admission, and a Name already admitted leaves only by
// measurement (ADR-0006), never by a toggle.
func admittedNames(ctx context.Context, q *db.Queries) ([]string, error) {
	return q.ListAdmittedNames(ctx)
}

// mergeResolutionNames is the dns Scan's resolution set (ADR-0107): the name-scope
// Seed domains, then the CT-admitted names not already among them. The Seeds lead
// and keep their exact string — what the resolver already resolves them as — so
// this only *adds* the discovered names the estate was written against since
// ADR-0027. Dedup is by resolutionNameKey — the resolver's own CanonicalName key
// (#256), not a parallel fold — so two names collapse here exactly when the resolver
// would key them to one subject_key, and stay separate exactly when it would key
// them to two. A discovered name equal to a Seed domain does not double it; a name
// the resolver keys distinctly is not folded away and left unmeasured. The result
// feeds the job's Names, never its seeds: the control-probe population widens with
// the resolution scope, but the probing gate stays bounded at the Seeds (ADR-0066).
func mergeResolutionNames(seedDomains, admitted []string) []string {
	seen := make(map[string]struct{}, len(seedDomains)+len(admitted))
	out := make([]string, 0, len(seedDomains)+len(admitted))
	for _, s := range seedDomains {
		out = append(out, s)
		seen[resolutionNameKey(s)] = struct{}{}
	}
	for _, n := range admitted {
		key := resolutionNameKey(n)
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, n)
	}
	return out
}

// resolutionNameKey is the dedup key for the resolution set. It is exactly the
// subject_key the resolver assigns when it measures the name — resolutionwalk.
// CanonicalName, the ADR-0055 key — so the merge dedups on the same identity the
// resolver keys its observations under. Re-implementing the fold inline (as it once
// did, with strings.ToLower) folded a non-ASCII-uppercase name onto a Seed the
// resolver keys distinctly — CanonicalName lowercases ASCII only, so "Ä.example.com"
// and "ä.example.com" are two subjects, not one — and silently DROPPED one from the
// resolution set, so the resolver never measured it (#256). Routing through
// CanonicalName keeps dedup, storage, and the citation match consistent. Whitespace
// a crt.sh value may carry is trimmed first, matching how the admitted-name rows
// were stored (internal/scan/crtsh.go normaliseName).
func resolutionNameKey(name string) string {
	return resolutionwalk.CanonicalName(strings.TrimSpace(name))
}

type vantages struct {
	rows []db.ListVantagesForDispatchRow
}

func vantageList(ctx context.Context, q *db.Queries) (vantages, error) {
	rows, err := q.ListVantagesForDispatch(ctx)
	if err != nil {
		return vantages{}, err
	}
	return vantages{rows: rows}, nil
}

func (v vantages) scanVantages() []scan.Vantage {
	out := make([]scan.Vantage, 0, len(v.rows))
	for _, r := range v.rows {
		// Carry the presented-address facts so the hot/cold Scans derive each vantage's
		// class per batch for the Custody gate (#709), never the vestigial column.
		out = append(out, scan.Vantage{
			ID: r.ID, Name: r.Name, Resolver: r.Resolver,
			Dialled: r.DialledAddr.String, Egress: r.Egress.String,
		})
	}
	return out
}

func (v vantages) resolver(id int64) string {
	for _, r := range v.rows {
		if r.ID == id {
			return r.Resolver
		}
	}
	return ""
}
