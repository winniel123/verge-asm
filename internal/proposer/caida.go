package proposer

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
	caidaSearchPageSize = 5000 // the search path caps first at 5000 (ADR-0227 §2)
	caidaSearchMaxPages = 8
	caidaSearchMaxBytes = 64 << 20
)

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
	OpaqueID string `json:"opaqueId"`
	OrgName  string `json:"orgName"`
	Source   string `json:"source"`
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
	seen := make(map[string]bool)
	var ids []string
	var named, read int

	for page := 0; page < caidaSearchMaxPages; page++ {
		p, err := c.searchPage(ctx, orgName, read)
		if err != nil {
			return nil, err
		}
		rows := *p.Data
		read += len(rows)
		for _, row := range rows {
			// The search is scored, so it answers with other RIRs and near names (ADR-0227 §2).
			if !strings.EqualFold(row.Source, rir) || !strings.Contains(strings.ToLower(row.OrgName), want) {
				continue
			}
			named++
			id := strings.TrimSuffix(row.OpaqueID, "_"+rir)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if read >= *p.TotalCount {
			if len(ids) == 0 && named > 0 {
				// CAIDA holds the org under no join key, which is a gap and not an absence (#50)
				return nil, fmt.Errorf("caida search matched %d %s records for %q and none carries an opaqueId", named, rir, orgName)
			}
			return ids, nil
		}
		if !p.PageInfo.HasNextPage {
			return nil, fmt.Errorf("caida search sent %d of %d rows for %q and reports no next page", read, *p.TotalCount, orgName)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("caida search sent an empty page at offset %d for %q", read, orgName)
		}
	}
	return nil, fmt.Errorf("caida search did not send every row for %q within %d pages", orgName, caidaSearchMaxPages)
}

func (c *CAIDA) searchPage(ctx context.Context, orgName string, offset int) (*caidaSearchPage, error) {
	q := url.Values{"name": {orgName}, "first": {strconv.Itoa(caidaSearchPageSize)}}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.caidaBase+"/search/?"+q.Encode(), nil)
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
		return nil, fmt.Errorf("caida search returned %d", resp.StatusCode)
	}
	var page caidaSearchPage
	if err := json.NewDecoder(io.LimitReader(resp.Body, caidaSearchMaxBytes)).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode caida search: %w", err)
	}
	if page.TotalCount == nil || page.PageInfo == nil || page.Data == nil {
		// encoding/json drops an unknown key, so a strange envelope reads as absence (ADR-0227 §3)
		return nil, fmt.Errorf("caida search returned no totalCount, pageInfo or data")
	}
	if e := bytes.TrimSpace(page.Errors); len(e) > 0 && !bytes.Equal(e, []byte("null")) {
		return nil, fmt.Errorf("caida search reported errors: %s", truncate(e))
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
