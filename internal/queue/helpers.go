package queue

import (
	"context"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/scan"
)

func nameSeedDomains(ctx context.Context, q *db.Queries) ([]string, error) {
	// The dns Scan resolves these unconditionally, independent of Custody (ADR-0084).
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

func admittedNames(ctx context.Context, q *db.Queries) ([]string, error) {
	// Consent was spent at admission, so a source toggle never un-admits a Name (ADR-0003, ADR-0006).
	return q.ListAdmittedNames(ctx)
}

func mergeResolutionNames(seedDomains, admitted []string) []string {
	// This widens the control-probe population; the probing gate stays at the Seeds (ADR-0066).
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

// A crt.sh value may carry whitespace, and the admitted-name rows were stored trimmed.

func resolutionNameKey(name string) string {
	// Folding inline once dropped a Name the resolver keys distinctly, unmeasured (ADR-0055, #256).
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
