package proposer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
)

type ARIN struct { // one endpoint and one permission, so both kinds are one proposer, not two
	doer Doer
	base string
}

func NewARIN(doer Doer, base string) *ARIN { return &ARIN{doer: doer, base: base} }

func (a *ARIN) Slug() string { return SlugARIN }

const maxARINBody = 8 << 20

const (
	classOrg      = "org"
	classCustomer = "customer"
	classPOC      = "poc"
)

var customerHandle = regexp.MustCompile(`^C\d+$`)

type arinSearch struct {
	Results []arinEntity `json:"entitySearchResults"`
}

type arinEntity struct {
	Handle     string          `json:"handle"`
	VCardArray json.RawMessage `json:"vcardArray"`
	Networks   []arinNetwork   `json:"networks"`
	Links      []arinLink      `json:"links"`
}

type arinLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

func (e arinEntity) class() string {
	for _, l := range e.Links {
		for _, c := range []string{classCustomer, classOrg, classPOC} {
			if strings.Contains(l.Href, "/rest/"+c+"/") {
				return c
			}
		}
	}
	// An org handle carries letters or hyphens (GOOGL-1), so only a SWIP customer matches.
	if customerHandle.MatchString(e.Handle) {
		return classCustomer
	}
	// A misread poc simply carries no networks, so an unknown handle defaults to org.
	return classOrg
}

type arinNetwork struct { // RFC 9083's cidr0 extension states exact aligned prefixes
	Cidrs []arinCidr `json:"cidr0_cidrs"`
}

type arinCidr struct {
	V4Prefix string `json:"v4prefix"`
	V6Prefix string `json:"v6prefix"`
	Length   int    `json:"length"`
}

func (c arinCidr) prefix() (netip.Prefix, bool) {
	addr := c.V4Prefix
	if addr == "" {
		addr = c.V6Prefix
	}
	if addr == "" {
		return netip.Prefix{}, false
	}
	p, err := netip.ParsePrefix(fmt.Sprintf("%s/%d", addr, c.Length))
	if err != nil {
		return netip.Prefix{}, false
	}
	return p, true
}

func (a *ARIN) Propose(ctx context.Context, orgName string) ([]Candidate, error) {
	// The Whois-RWS base answers this path with an HTML 404; only the RDAP base works (#611).
	searchURL := a.base + "/entities?fn=" + url.QueryEscape(orgName)
	body, status, err := a.get(ctx, searchURL)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		// RDAP reports a no-match as 404: the absence of a proposal, not an errored path.
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("arin rdap entity search returned %d: %s", status, truncate(body))
	}

	var search arinSearch
	if err := json.Unmarshal(body, &search); err != nil {
		return nil, fmt.Errorf("decode arin search response: %w", err)
	}

	var out []Candidate
	// Networks live on the full entity, so one serial fetch per holder — priced by ADR-0012.
	for _, stub := range search.Results {
		// A half-finished walk must never pass as the whole answer.
		if err := ctx.Err(); err != nil {
			return out, fmt.Errorf("arin org-name walk interrupted: %w", err)
		}
		if stub.Handle == "" {
			continue
		}
		class := stub.class()
		if class == classPOC {
			continue
		}
		kind := RecordRIRDelegation
		if class == classCustomer {
			kind = RecordCompelledReassignment
		}

		entBody, entStatus, err := a.get(ctx, a.base+"/entity/"+url.PathEscape(stub.Handle))
		// A source error is all-or-nothing, so one unfetchable record must not fail the path.
		if err != nil || entStatus != http.StatusOK {
			continue
		}
		var ent arinEntity
		if err := json.Unmarshal(entBody, &ent); err != nil {
			continue
		}

		// The customer's own name matched the search, so it is never inherited from the org.
		name := fnFromVCard(ent.VCardArray)
		if name == "" {
			name = fnFromVCard(stub.VCardArray)
		}
		if name == "" {
			// The handle is the identifier the operator judges a nameless network by.
			name = stub.Handle
		}
		for _, n := range ent.Networks {
			for _, c := range n.Cidrs {
				p, ok := c.prefix()
				if !ok {
					continue // a malformed range is skipped, never invented
				}
				out = append(out, Candidate{
					SourceSlug: SlugARIN, RecordKind: kind,
					Scope: p.Masked(), OrgName: name,
				})
			}
		}
	}
	return out, nil
}

func (a *ARIN) get(ctx context.Context, u string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/rdap+json")

	resp, err := a.doer.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	// A hostile or runaway response must not exhaust memory.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxARINBody))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func fnFromVCard(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var outer []json.RawMessage
	// RFC 7095 jCard: ["vcard", [[name, params, type, value], ...]].
	if err := json.Unmarshal(raw, &outer); err != nil || len(outer) < 2 {
		return ""
	}
	var props [][]json.RawMessage
	if err := json.Unmarshal(outer[1], &props); err != nil {
		return ""
	}
	for _, p := range props {
		if len(p) < 4 {
			continue
		}
		var name string
		if err := json.Unmarshal(p[0], &name); err != nil || name != "fn" {
			continue
		}
		var val string
		if err := json.Unmarshal(p[3], &val); err != nil {
			continue
		}
		return val
	}
	return ""
}

func truncate(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max])
	}
	return string(b)
}
