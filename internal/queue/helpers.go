package queue

import (
	"context"

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
