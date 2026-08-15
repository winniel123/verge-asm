package proposer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
)

// ARIN is the keyless entities?fn= org-name path (ADR-0012). One search returns
// both the org's own delegated networks and the SWIP customer (C-handle)
// objects an upstream provider was compelled to reassign — they arrive in a
// single response under a single permission, so this is one proposer, not two.
// The two record kinds carry different caveats, so each Candidate records which
// kind produced it rather than the source being split.
type ARIN struct {
	doer Doer
	base string // e.g. https://whois.arin.net/rest
}

// NewARIN builds the ARIN proposer over an injected Doer and endpoint base.
func NewARIN(doer Doer, base string) *ARIN { return &ARIN{doer: doer, base: base} }

func (a *ARIN) Slug() string { return SlugARIN }

// arinResponse is the normalised shape verge-asm reads the entities?fn= result
// into: the org entities the search matched, each carrying its delegated
// networks and any reassigned customer blocks. Field names mirror the upstream
// Whois-RWS org/net/customer objects (name, CIDR) once flattened.
type arinResponse struct {
	Entities []struct {
		Handle   string `json:"handle"`
		Name     string `json:"name"`
		Networks []struct {
			CIDR string `json:"cidr"`
		} `json:"networks"`
		Customers []struct {
			Name string `json:"name"`
			CIDR string `json:"cidr"`
		} `json:"customers"`
	} `json:"entities"`
}

// Propose runs one org-name search and returns every candidate scope it found.
// An org network becomes an rir-delegation Candidate; a customer block becomes a
// compelled-reassignment Candidate under the customer's own name.
func (a *ARIN) Propose(ctx context.Context, orgName string) ([]Candidate, error) {
	u := a.base + "/entities?fn=" + url.QueryEscape(orgName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// A non-200 yields no proposal, never a proposal of absence: a
		// proposer's silence licenses nothing.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("arin entities?fn= returned %d: %s", resp.StatusCode, body)
	}

	var parsed arinResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode arin response: %w", err)
	}

	var out []Candidate
	for _, e := range parsed.Entities {
		for _, n := range e.Networks {
			p, err := netip.ParsePrefix(n.CIDR)
			if err != nil {
				continue // a malformed range is skipped, never invented
			}
			out = append(out, Candidate{
				SourceSlug: SlugARIN, RecordKind: RecordRIRDelegation,
				Scope: p.Masked(), OrgName: e.Name,
			})
		}
		for _, c := range e.Customers {
			p, err := netip.ParsePrefix(c.CIDR)
			if err != nil {
				continue
			}
			out = append(out, Candidate{
				SourceSlug: SlugARIN, RecordKind: RecordCompelledReassignment,
				Scope: p.Masked(), OrgName: c.Name,
			})
		}
	}
	return out, nil
}
