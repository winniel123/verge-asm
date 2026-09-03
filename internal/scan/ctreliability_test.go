package scan

import "testing"

func TestEvaluateCTReliability(t *testing.T) {
	// crt.sh is the keyless fallback, so it is exempt from every limb (ct-source-replacement.md §3).
	cases := []struct {
		name   string
		source string
		win    CTReliabilityWindow
		want   CTReliabilityReport
	}{
		{
			name:   "primary clears every limb",
			source: CertSpotterSource,
			win:    CTReliabilityWindow{Total: 200, Successes: 199, Empties: 0, P95LatencyMS: 3200},
			want: CTReliabilityReport{
				Source: CertSpotterSource, Exempt: false, HasData: true, Samples: 200,
				SuccessRate: 199.0 / 200.0, SuccessPass: true,
				P95LatencyMS: 3200, LatencyPass: true,
				FalseEmpty: 0, FalseEmptyPass: true,
				Degraded: false,
			},
		},
		{
			name:   "primary below the success bar is degraded",
			source: CertSpotterSource,
			win:    CTReliabilityWindow{Total: 200, Successes: 196, Empties: 0, P95LatencyMS: 3200},
			want: CTReliabilityReport{
				Source: CertSpotterSource, Exempt: false, HasData: true, Samples: 200,
				SuccessRate: 196.0 / 200.0, SuccessPass: false,
				P95LatencyMS: 3200, LatencyPass: true,
				FalseEmpty: 0, FalseEmptyPass: true,
				Degraded: true,
			},
		},
		{
			name:   "primary over the latency bar is degraded",
			source: CertSpotterSource,
			win:    CTReliabilityWindow{Total: 200, Successes: 200, Empties: 0, P95LatencyMS: 5001},
			want: CTReliabilityReport{
				Source: CertSpotterSource, Exempt: false, HasData: true, Samples: 200,
				SuccessRate: 1, SuccessPass: true,
				P95LatencyMS: 5001, LatencyPass: false,
				FalseEmpty: 0, FalseEmptyPass: true,
				Degraded: true,
			},
		},
		{
			name:   "one false-empty degrades the primary",
			source: CertSpotterSource,
			win:    CTReliabilityWindow{Total: 200, Successes: 200, Empties: 1, P95LatencyMS: 1000},
			want: CTReliabilityReport{
				Source: CertSpotterSource, Exempt: false, HasData: true, Samples: 200,
				SuccessRate: 1, SuccessPass: true,
				P95LatencyMS: 1000, LatencyPass: true,
				FalseEmpty: 1, FalseEmptyPass: false,
				Degraded: true,
			},
		},
		{
			name:   "crt.sh is exempt and never degraded, even below every limb",
			source: CrtshSource,
			win:    CTReliabilityWindow{Total: 8, Successes: 4, Empties: 2, P95LatencyMS: 59600},
			want: CTReliabilityReport{
				Source: CrtshSource, Exempt: true, HasData: true, Samples: 8,
				SuccessRate: 0.5, SuccessPass: false,
				P95LatencyMS: 59600, LatencyPass: false,
				FalseEmpty: 2, FalseEmptyPass: false,
				Degraded: false,
			},
		},
		{
			name:   "no samples is not judged",
			source: CertSpotterSource,
			win:    CTReliabilityWindow{},
			want: CTReliabilityReport{
				Source: CertSpotterSource, Exempt: false, HasData: false, Samples: 0,
				SuccessRate: 0, SuccessPass: false,
				P95LatencyMS: 0, LatencyPass: true,
				FalseEmpty: 0, FalseEmptyPass: true,
				Degraded: false,
			},
		},
		{
			name:   "the success bar is exactly 99 percent, inclusive",
			source: CertSpotterSource,
			win:    CTReliabilityWindow{Total: 100, Successes: 99, Empties: 0, P95LatencyMS: 10},
			want: CTReliabilityReport{
				Source: CertSpotterSource, Exempt: false, HasData: true, Samples: 100,
				SuccessRate: 0.99, SuccessPass: true,
				P95LatencyMS: 10, LatencyPass: true,
				FalseEmpty: 0, FalseEmptyPass: true,
				Degraded: false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateCTReliability(tc.source, tc.win)
			if got != tc.want {
				t.Errorf("EvaluateCTReliability(%q, %+v)\n got %+v\nwant %+v", tc.source, tc.win, got, tc.want)
			}
		})
	}
}
