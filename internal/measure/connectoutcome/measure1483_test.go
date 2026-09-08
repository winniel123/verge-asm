package connectoutcome

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
)

// Ticket #1483 (map #1084): measure 162.222.48.0/22. Does the dynamic band RST
// or DROP? This runs the real control-port path (probeControlPort ->
// controlResultOf -> blanketdiscrim.Decide) that Inventory reaches, plus a
// service-band probe. Gated behind MEASURE_1483 so it is inert in CI.

type portResult struct {
	Port    uint16                       `json:"port"`
	Conn    ConnResult                   `json:"conn_result"`
	Control blanketdiscrim.ControlResult `json:"control_result,omitempty"`
	Outcome Outcome                      `json:"outcome,omitempty"`
}

type addrReport struct {
	Addr         string                 `json:"addr"`
	ControlPorts []portResult           `json:"control_ports"`
	Verdict      blanketdiscrim.Verdict `json:"verdict"`
	ServicePorts []portResult           `json:"service_ports"`
}

type measurement struct {
	Ticket        string        `json:"ticket"`
	Map           string        `json:"map"`
	Scope         string        `json:"scope"`
	VantageEgress string        `json:"vantage_egress_ip"`
	MeasuredAt    string        `json:"measured_at_utc"`
	ControlBand   [2]uint16     `json:"control_band"`
	DrawnOnce     string        `json:"control_ports_drawn"`
	ServiceProbed []uint16      `json:"service_ports_probed"`
	Profile       SafetyProfile `json:"safety_profile"`
	Addresses     []addrReport  `json:"addresses"`
}

func TestMeasure1483(t *testing.T) {
	if os.Getenv("MEASURE_1483") == "" {
		t.Skip("set MEASURE_1483=1 to run the live network measurement")
	}

	// Spread across the four /24s of the /22, low and high host bytes.
	targets := []string{
		"162.222.48.1", "162.222.48.7", "162.222.48.200",
		"162.222.49.10", "162.222.50.100", "162.222.51.254",
	}
	// A representative service band. IANA/common listeners a WatchGuard would front.
	servicePorts := []uint16{22, 80, 443, 8080, 8443, 3389, 25, 53}

	profile := DefaultProfile()
	timeout := time.Duration(profile.ConnectTimeoutMillis) * time.Millisecond
	c := NetConnector{Timeout: timeout} // zero realm: a globally-reachable target passes EgressGuard

	// Drawn ONCE per batch and reused across every address, as production does (ADR-0069).
	ports := blanketdiscrim.CryptoPorts{}.Ports()

	dp := blanketdiscrim.DefaultParams()
	ctx := context.Background()
	m := measurement{
		Ticket:        "1483",
		Map:           "1084",
		Scope:         "162.222.48.0/22",
		VantageEgress: "104.222.18.57",
		MeasuredAt:    time.Now().UTC().Format(time.RFC3339),
		ControlBand:   [2]uint16{dp.PortBandLow, dp.PortBandHigh},
		DrawnOnce:     "batch, reused across addresses (ADR-0069)",
		ServiceProbed: servicePorts,
		Profile:       profile,
	}

	// Probe concurrently under a bounded pool. A DROP costs the full 3s timeout,
	// so sequential over 6 addresses exceeds the test budget. The cap holds the
	// vantage packet rate down while the wall clock stays short.
	sem := make(chan struct{}, 32)
	var wg sync.WaitGroup
	reps := make([]addrReport, len(targets))
	for i, a := range targets {
		addr := netip.MustParseAddr(a)
		rep := &reps[i]
		rep.Addr = a
		rep.ControlPorts = make([]portResult, len(ports))
		rep.ServicePorts = make([]portResult, len(servicePorts))
		for j, p := range ports {
			wg.Add(1)
			go func(j int, p uint16) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				raw := probeControlPort(ctx, c, profile, netip.AddrPortFrom(addr, p))
				rep.ControlPorts[j] = portResult{Port: p, Conn: raw, Control: controlResultOf(raw)}
			}(j, p)
		}
		for j, p := range servicePorts {
			wg.Add(1)
			go func(j int, p uint16) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				outcome, raw := Probe(ctx, c, profile, netip.AddrPortFrom(addr, p))
				rep.ServicePorts[j] = portResult{Port: p, Conn: raw, Outcome: outcome}
			}(j, p)
		}
	}
	wg.Wait()
	for i := range reps {
		results := make([]blanketdiscrim.ControlResult, len(reps[i].ControlPorts))
		for j, pr := range reps[i].ControlPorts {
			results[j] = pr.Control
		}
		reps[i].Verdict = blanketdiscrim.Decide(results)
		m.Addresses = append(m.Addresses, reps[i])
		t.Logf("%s -> verdict=%s", reps[i].Addr, reps[i].Verdict)
	}

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("MEASURE_1483_OUT")
	if out == "" {
		out = "measure1483.json"
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d bytes)", out, len(b))
}
