// The `http-identity` Scan issues one `GET /` over every reached `Endpoint` (ADR-0011, ADR-0024).
// It rides the daily hot-Scan reachability: the facet is decided only after
// `connect-outcome` has reported `reached` (v1 spec §3.3).
package scan

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/vantageclass"
	"github.com/winniel123/verge-asm/internal/wire"
)

// A Scan names the facet it decides; the leaf it dispatches names the exchange it runs.

const HTTPIdentityKind = "http-identity"

type HTTPIdentityJob struct {
	ScanID       int64
	VantageID    int64
	Vantage      string
	VantageClass string
	Kind         string
	Targets      []httpexchange.Target
	Params       httpexchange.Params
}

func BuildHTTPIdentityJobs(scanID int64, estate custody.Estate, services []ReachedService, vantages []Vantage) []HTTPIdentityJob {
	if len(services) == 0 || len(vantages) == 0 {
		return nil
	}
	byID := make(map[int64]Vantage, len(vantages))
	order := make([]int64, 0, len(vantages))
	for _, v := range vantages {
		if _, ok := byID[v.ID]; !ok {
			order = append(order, v.ID)
		}
		byID[v.ID] = v
	}

	covered := estate.CoversAddressScope
	vcByID := make(map[int64]custody.VantageClass, len(vantages))
	for _, v := range vantages {
		vcByID[v.ID] = vantageclass.Derive(v.Dialled, v.Egress, covered)
	}

	grouped := make(map[int64][]httpexchange.Target)
	for _, s := range services {
		if _, ok := byID[s.VantageID]; !ok {
			continue
		}
		a, err := netip.ParseAddr(s.Address)
		if err != nil {
			continue
		}
		if !estate.MayProbe(a, vcByID[s.VantageID]) {
			continue
		}
		grouped[s.VantageID] = append(grouped[s.VantageID], httpexchange.Target{
			Name:    "", // the distinguished nameless Endpoint: an identity no Name cited
			Address: s.Address,
			Port:    s.Port,
			Scheme:  schemeForPort(s.Port),
		})
	}

	var jobs []HTTPIdentityJob
	for _, id := range order {
		targets := grouped[id]
		if len(targets) == 0 {
			continue
		}
		v := byID[id]
		jobs = append(jobs, HTTPIdentityJob{
			ScanID:       scanID,
			VantageID:    id,
			Vantage:      v.Name,
			VantageClass: v.Class,
			Kind:         httpexchange.Kind,
			Targets:      targets,
			Params:       httpexchange.DefaultParams(),
		})
	}
	return jobs
}

func schemeForPort(port uint16) string {
	// Choosing a scheme widens nothing: a port speaking neither folds to no-http-response (ADR-0011).
	switch port {
	case 443, 8443, 6443:
		return "https"
	default:
		return "http"
	}
}

func (j HTTPIdentityJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(httpexchange.Scope{
		Vantage:      j.Vantage,
		VantageClass: j.VantageClass,
		Targets:      j.Targets,
		Params:       j.Params,
	})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal http-identity scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: j.Kind, Scope: raw}, nil
}

func (j HTTPIdentityJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(httpIdentityScopeRecord{
		Vantage: j.Vantage,
		Targets: j.Targets,
		Params:  j.Params,
	})
}

func (j HTTPIdentityJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Params) }

func EmptyHTTPIdentityScope(vantage string) ([]byte, error) {
	return json.Marshal(httpIdentityScopeRecord{Vantage: vantage, Targets: []httpexchange.Target{}})
}

type httpIdentityScopeRecord struct {
	Vantage string                `json:"vantage"`
	Targets []httpexchange.Target `json:"targets"`
	Params  httpexchange.Params   `json:"params,omitempty"`
}
