package queue

import (
	"context"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
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
// ADR-0027. Dedup is by the lowercase/trailing-dot key both the Seed domains and
// the admitted-name rows are stored under (internal/scan/crtsh.go normaliseName),
// so a discovered name equal to a Seed domain does not double it. The result feeds
// the job's Names, never its seeds: the control-probe population widens with the
// resolution scope, but the probing gate stays bounded at the Seeds (ADR-0066).
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

// resolutionNameKey is the dedup key for the resolution set — the same
// lowercase/trailing-dot normalisation the Seed domains and admitted-name rows are
// already stored under. It is a merge key only, not the ADR-0055 subject key the
// resolver assigns when it measures the name.
func resolutionNameKey(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}

// vantages is the configured Vantage set, with a lookup from id to resolver.
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
		out = append(out, scan.Vantage{ID: r.ID, Name: r.Name, Resolver: r.Resolver})
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
