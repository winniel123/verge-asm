package proposer

import (
	"bufio"
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

// CAIDA is a keyless org->prefix path for a registry that publishes no org-name
// search of its own — AFRINIC and APNIC (ADR-0012). It is built by joining two
// keyless datasets: CAIDA's org->opaque-id mapping resolves the searched name to
// the registry's opaque holder ids, and the RIR's delegated-extended-stats file
// turns those ids into the ranges delegated under them. Every range here is an
// RIR delegation, so each Candidate carries that record kind.
type CAIDA struct {
	doer          Doer
	slug          string // the catalogue slug (afrinic, apnic-caida)
	rir           string // the RIR name in the delegated-stats rows (afrinic, apnic)
	caidaBase     string
	delegatedBase string
}

func NewCAIDA(doer Doer, slug, rir, caidaBase, delegatedBase string) *CAIDA {
	return &CAIDA{doer: doer, slug: slug, rir: rir, caidaBase: caidaBase, delegatedBase: delegatedBase}
}

func (c *CAIDA) Slug() string { return c.slug }

type caidaOrgIDs struct {
	OpaqueIDs []string `json:"opaque_ids"`
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
	u := c.caidaBase + "/org2ids?org=" + url.QueryEscape(orgName)
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
		return nil, fmt.Errorf("caida org2ids returned %d", resp.StatusCode)
	}
	var parsed caidaOrgIDs
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode caida org2ids: %w", err)
	}
	return parsed.OpaqueIDs, nil
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
			continue // header, summary, or non-extended row
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
