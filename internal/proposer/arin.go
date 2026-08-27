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

// ARIN is the keyless RDAP org-name path (ADR-0012). One org-name search returns
// both the org's own delegated networks and the SWIP customer (C-handle) objects
// an upstream provider was compelled to reassign — they arrive under a single
// endpoint and a single permission, so this is one proposer, not two. The two
// record kinds carry different caveats, so each Candidate records which kind
// produced it rather than the source being split.
//
// The grammar is RDAP's `entities?fn=` formatted-name search, served keyless and
// public at ARIN's RDAP base `https://rdap.arin.net/registry` (NOT the Whois-RWS
// base `https://whois.arin.net/rest`, whose collection grammar is different and
// which answers this path with an HTML 404 — issue #611). RDAP search results
// are entity *stubs*: the networks live on the full entity, so a matched handle
// is fetched once to read its networks. That per-handle fetch is the volume
// constraint ADR-0012 already priced.
type ARIN struct {
	doer Doer
	base string // e.g. https://rdap.arin.net/registry
}

// NewARIN builds the ARIN proposer over an injected Doer and RDAP endpoint base.
func NewARIN(doer Doer, base string) *ARIN { return &ARIN{doer: doer, base: base} }

func (a *ARIN) Slug() string { return SlugARIN }

// maxARINBody caps a single RDAP body read. A busy org's entity runs to tens of
// kilobytes; this leaves generous headroom while refusing an unbounded read.
const maxARINBody = 8 << 20

// The three ARIN object classes an org-name search can match. Only an org and a
// customer hold address scope; a point of contact holds none and is skipped.
const (
	classOrg      = "org"
	classCustomer = "customer"
	classPOC      = "poc"
)

// customerHandle matches an ARIN SWIP customer handle — a `C` followed only by
// digits (e.g. C01839743). It is the fallback classifier when an entity carries
// no `alternate` link to read the class from; org handles carry letters or
// hyphens (GOOGL-1, HURRIC-1) and never match.
var customerHandle = regexp.MustCompile(`^C\d+$`)

// arinSearch is the RDAP entity-search envelope: a list of matched entity stubs.
// A stub carries the handle and formatted name but no networks — those are read
// from the full entity fetched by handle.
type arinSearch struct {
	Results []arinEntity `json:"entitySearchResults"`
}

// arinEntity is one RDAP entity, as a search stub or a full fetch. The vCard
// (jCard) array carries the formatted name; the links name the object class; a
// full fetch also carries networks.
type arinEntity struct {
	Handle     string          `json:"handle"`
	VCardArray json.RawMessage `json:"vcardArray"`
	Networks   []arinNetwork   `json:"networks"`
	Links      []arinLink      `json:"links"`
}

// arinLink is one RDAP link. The `alternate` link's href points at the Whois-RWS
// object and names its class in the path (/rest/org/, /rest/customer/, /rest/poc/).
type arinLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// class reports the entity's ARIN object class from its links — the sturdy
// signal, since the href path names the class outright — and falls back to the
// C-digit handle convention when no link classifies. An org is a delegation
// holder, a customer is a compelled reassignment, a poc holds no scope.
func (e arinEntity) class() string {
	for _, l := range e.Links {
		for _, c := range []string{classCustomer, classOrg, classPOC} {
			if strings.Contains(l.Href, "/rest/"+c+"/") {
				return c
			}
		}
	}
	if customerHandle.MatchString(e.Handle) {
		return classCustomer
	}
	// An unclassifiable, non-customer handle is treated as an org: if it is in
	// fact a poc it simply carries no networks and yields nothing.
	return classOrg
}

// arinNetwork is one IP network on an entity. cidr0_cidrs (RFC 9083's cidr0
// extension) states the network's exact aligned prefixes, which is what a
// Candidate scope is built from.
type arinNetwork struct {
	Cidrs []arinCidr `json:"cidr0_cidrs"`
}

// arinCidr is one aligned prefix from a network's cidr0_cidrs. Exactly one of
// v4prefix / v6prefix is set alongside the length.
type arinCidr struct {
	V4Prefix string `json:"v4prefix"`
	V6Prefix string `json:"v6prefix"`
	Length   int    `json:"length"`
}

// prefix builds the netip.Prefix a cidr0 entry names, reporting false on a shape
// ARIN did not serve cleanly so the caller can skip it rather than invent one.
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

// Propose runs one org-name search and returns every candidate scope it found.
// A matched org's network becomes an rir-delegation Candidate; a matched SWIP
// customer's network becomes a compelled-reassignment Candidate under the
// customer's own name — which the RDAP `fn` search returns because it matched on
// that name, so it is read from the customer entity itself, never inherited from
// the org.
//
// A search ARIN answers with 404 is a clean no-match — no candidates, not an
// error — so a genuinely-unknown org name does not present to the operator as a
// backend failure. If our own context is cancelled or times out mid-walk the
// call stops and reports it, rather than pass a half-finished walk off as the
// whole answer. A single matched record we cannot fetch (a transient 5xx, a
// rate-limit, or a record that vanished since the search) costs only its own
// candidates: the Registry treats any source error as all-or-nothing, so failing
// the whole path for one unreachable record would discard every candidate we did
// retrieve. Nothing is ever invented in a skipped record's place.
func (a *ARIN) Propose(ctx context.Context, orgName string) ([]Candidate, error) {
	searchURL := a.base + "/entities?fn=" + url.QueryEscape(orgName)
	body, status, err := a.get(ctx, searchURL)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		// RDAP reports a no-match as 404. It is the absence of a proposal, not
		// an errored path.
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("arin rdap entity search returned %d: %s", status, truncate(body))
	}

	var search arinSearch
	if err := json.Unmarshal(body, &search); err != nil {
		return nil, fmt.Errorf("decode arin search response: %w", err)
	}

	// Each matched entity's networks live on the full entity, not the search
	// stub, so a classifiable holder is fetched once by handle. That is one fetch
	// per matched holder run in series — the volume ADR-0012 already priced — so
	// a busy org's search is a burst of sequential round-trips.
	var out []Candidate
	for _, stub := range search.Results {
		if err := ctx.Err(); err != nil {
			return out, fmt.Errorf("arin org-name walk interrupted: %w", err)
		}
		if stub.Handle == "" {
			continue
		}
		class := stub.class()
		if class == classPOC {
			continue // a point of contact holds no address scope — no fetch needed
		}
		kind := RecordRIRDelegation
		if class == classCustomer {
			kind = RecordCompelledReassignment
		}

		entBody, entStatus, err := a.get(ctx, a.base+"/entity/"+url.PathEscape(stub.Handle))
		if err != nil || entStatus != http.StatusOK {
			continue
		}
		var ent arinEntity
		if err := json.Unmarshal(entBody, &ent); err != nil {
			continue
		}

		name := fnFromVCard(ent.VCardArray)
		if name == "" {
			name = fnFromVCard(stub.VCardArray)
		}
		if name == "" {
			// A network is worth proposing even when the name field is missing;
			// the handle is the identifier the operator judges it by instead.
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

// get issues one keyless RDAP GET and returns the body and status. The body is
// bounded so a hostile or runaway response cannot exhaust memory.
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxARINBody))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// fnFromVCard extracts the formatted-name (`fn`) value from an RDAP jCard array
// (RFC 7095): ["vcard", [ [name, params, type, value], ... ]]. It returns "" for
// any shape it does not recognise rather than failing the whole decode.
func fnFromVCard(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var outer []json.RawMessage
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

// truncate bounds an error body so a non-200's HTML or JSON detail stays a log
// line, not a page.
func truncate(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max])
	}
	return string(b)
}
