package proposer

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type CAIDA struct {
	doer          Doer
	slug          string
	rir           string // the RIR name the stats rows and the search's source carry, not the slug
	caidaBase     string
	delegatedBase string // CAIDA yields ids and no prefix, so this file holds the scopes (ADR-0227)
}

func NewCAIDA(doer Doer, slug, rir, caidaBase, delegatedBase string) *CAIDA {
	return &CAIDA{doer: doer, slug: slug, rir: rir, caidaBase: caidaBase, delegatedBase: delegatedBase}
}

func (c *CAIDA) Slug() string { return c.slug }

const (
	caidaSearchPageSize   = 5000 // the search path caps first at 5000 (ADR-0227 §2)
	caidaSearchMaxPages   = 8
	caidaSearchMaxBytes   = 64 << 20
	caidaASNLegMaxLookups = 32 // one ASN per asns/ call; measured need is 0 to 2 (ADR-0227, #1634)
)

// CAIDA holds the org but no opaqueId reaches it, even via asns/: a gap, not a failure (#1634).

var ErrNoJoinKey = errors.New("caida publishes no opaqueId for the organisation")

type caidaSearchPage struct {
	TotalCount *int              `json:"totalCount"`
	PageInfo   *caidaPageInfo    `json:"pageInfo"`
	Errors     json.RawMessage   `json:"errors"`
	Data       *[]caidaSearchRow `json:"data"`
}

type caidaPageInfo struct {
	HasNextPage bool `json:"hasNextPage"`
}

type caidaSearchRow struct {
	OpaqueID string   `json:"opaqueId"`
	OrgName  string   `json:"orgName"`
	Source   string   `json:"source"`
	ASN      string   `json:"asn"`
	Members  []string `json:"members"` // org records name ASNs here and carry no opaqueId (#1634)
}

func (c *CAIDA) Propose(ctx context.Context, orgName string) ([]Candidate, error) {
	ids, err := c.orgIDs(ctx, orgName)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil // no holder matched — no proposal, not a proposal of absence
	}
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	return c.delegations(ctx, orgName, idSet)
}

func (c *CAIDA) orgIDs(ctx context.Context, orgName string) ([]string, error) {
	rir := strings.ToUpper(c.rir)
	want := strings.ToLower(orgName)
	// The search is scored, so it answers with other RIRs and near names (ADR-0227 §2).
	matches := func(row caidaSearchRow) bool {
		return strings.EqualFold(row.Source, rir) && strings.Contains(strings.ToLower(row.OrgName), want)
	}
	seen := make(map[string]bool)
	var ids []string
	add := func(row caidaSearchRow) bool {
		id := strings.TrimSuffix(row.OpaqueID, "_"+rir)
		if id == "" {
			return false
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
		return true
	}

	rows, err := c.searchRows(ctx, orgName)
	if err != nil {
		return nil, err
	}
	var named int
	keyed := make(map[string]bool)
	memberSeen := make(map[string]bool)
	var members []string
	for _, row := range rows {
		if !matches(row) {
			continue
		}
		named++
		if add(row) {
			keyed[row.ASN] = true
			continue
		}
		for _, asn := range row.Members {
			if !memberSeen[asn] {
				memberSeen[asn] = true
				members = append(members, asn)
			}
		}
	}

	var pending []string
	for _, asn := range members {
		if !keyed[asn] {
			pending = append(pending, asn)
		}
	}
	if len(pending) > caidaASNLegMaxLookups {
		return nil, fmt.Errorf("caida search named %d unkeyed member ASNs for %q, over the asns cap of %d", len(pending), orgName, caidaASNLegMaxLookups)
	}
	for _, asn := range pending {
		asnRows, err := c.asnRecords(ctx, asn)
		if err != nil {
			return nil, err
		}
		for _, row := range asnRows {
			if matches(row) {
				add(row)
			}
		}
	}
	if len(ids) == 0 && named > 0 {
		// CAIDA holds the org under no join key, which is a gap and not an absence (#50)
		return nil, fmt.Errorf("caida search matched %d %s records for %q and none carries an opaqueId, after %d asns lookups: %w", named, rir, orgName, len(pending), ErrNoJoinKey)
	}
	return ids, nil
}

func (c *CAIDA) searchRows(ctx context.Context, orgName string) ([]caidaSearchRow, error) {
	var rows []caidaSearchRow
	for page := 0; page < caidaSearchMaxPages; page++ {
		p, err := c.searchPage(ctx, orgName, len(rows))
		if err != nil {
			return nil, err
		}
		got := *p.Data
		rows = append(rows, got...)
		if len(rows) >= *p.TotalCount {
			return rows, nil
		}
		if !p.PageInfo.HasNextPage {
			return nil, fmt.Errorf("caida search sent %d of %d rows for %q and reports no next page", len(rows), *p.TotalCount, orgName)
		}
		if len(got) == 0 {
			return nil, fmt.Errorf("caida search sent an empty page at offset %d for %q", len(rows), orgName)
		}
	}
	return nil, fmt.Errorf("caida search did not send every row for %q within %d pages", orgName, caidaSearchMaxPages)
}

func (c *CAIDA) searchPage(ctx context.Context, orgName string, offset int) (*caidaSearchPage, error) {
	q := url.Values{"name": {orgName}, "first": {strconv.Itoa(caidaSearchPageSize)}}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.fetchPage(ctx, "caida search", c.caidaBase+"/search/?"+q.Encode())
}

func (c *CAIDA) asnRecords(ctx context.Context, asn string) ([]caidaSearchRow, error) {
	what := "caida asns/" + asn
	p, err := c.fetchPage(ctx, what, c.caidaBase+"/asns/"+url.PathEscape(asn))
	if err != nil {
		return nil, err
	}
	rows := *p.Data
	if len(rows) < *p.TotalCount {
		return nil, fmt.Errorf("%s sent %d of %d rows", what, len(rows), *p.TotalCount)
	}
	return rows, nil
}

func (c *CAIDA) fetchPage(ctx context.Context, what, u string) (*caidaSearchPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", what, resp.StatusCode)
	}
	var page caidaSearchPage
	if err := json.NewDecoder(io.LimitReader(resp.Body, caidaSearchMaxBytes)).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode %s: %w", what, err)
	}
	if page.TotalCount == nil || page.PageInfo == nil || page.Data == nil {
		// encoding/json drops an unknown key, so a strange envelope reads as absence (ADR-0227 §3)
		return nil, fmt.Errorf("%s returned no totalCount, pageInfo or data", what)
	}
	if e := bytes.TrimSpace(page.Errors); len(e) > 0 && !bytes.Equal(e, []byte("null")) {
		return nil, fmt.Errorf("%s reported errors: %s", what, truncate(e))
	}
	return &page, nil
}

func (c *CAIDA) delegations(ctx context.Context, orgName string, ids map[string]bool) ([]Candidate, error) {
	u := c.delegatedBase + "/delegated-" + c.rir + "-extended-latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("delegated-stats returned %d", resp.StatusCode)
	}

	var out []Candidate
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 32<<20))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 8 {
			continue // a header, summary, or non-extended row carries no delegation
		}
		typ, start, value, opaque := fields[2], fields[3], fields[4], fields[7]
		if !ids[opaque] {
			continue
		}
		prefixes, err := rowPrefixes(typ, start, value)
		if err != nil {
			continue // a malformed row is skipped, never invented
		}
		for _, p := range prefixes {
			out = append(out, Candidate{
				SourceSlug: c.slug, RecordKind: RecordRIRDelegation,
				Scope: p, OrgName: orgName,
			})
		}
	}
	if err := sc.Err(); err != nil {
		return out, fmt.Errorf("scan delegated-stats: %w", err)
	}
	return out, nil
}

func rowPrefixes(typ, start, value string) ([]netip.Prefix, error) {
	addr, err := netip.ParseAddr(start)
	if err != nil {
		return nil, err
	}
	switch typ {
	case "ipv4":
		count, ok := new(big.Int).SetString(value, 10)
		if !ok || count.Sign() <= 0 {
			return nil, fmt.Errorf("bad ipv4 count %q", value)
		}
		return rangeToPrefixes(addr, count)
	case "ipv6":
		bits, err := strconv.Atoi(value)
		if err != nil || bits < 0 || bits > addr.BitLen() {
			return nil, fmt.Errorf("bad ipv6 prefix length %q", value)
		}
		return []netip.Prefix{netip.PrefixFrom(addr, bits).Masked()}, nil
	default:
		return nil, fmt.Errorf("non-address row type %q", typ)
	}
}
