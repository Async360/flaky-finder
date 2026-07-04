package flaky

import "testing"

func TestComputeFlakeReport(t *testing.T) {
	tests := []struct {
		name     string
		outcomes []RunOutcome
		want     FlakeReport
	}{
		{
			name:     "zero runs",
			outcomes: nil,
			want:     FlakeReport{TotalRuns: 0, PassCount: 0, FailCount: 0, FlakeRatePercent: 0, Flaky: false},
		},
		{
			name: "all pass",
			outcomes: []RunOutcome{
				{Passed: true}, {Passed: true}, {Passed: true},
			},
			want: FlakeReport{TotalRuns: 3, PassCount: 3, FailCount: 0, FlakeRatePercent: 0, Flaky: false},
		},
		{
			name: "all fail",
			outcomes: []RunOutcome{
				{Passed: false}, {Passed: false}, {Passed: false}, {Passed: false},
			},
			want: FlakeReport{TotalRuns: 4, PassCount: 0, FailCount: 4, FlakeRatePercent: 0, Flaky: false},
		},
		{
			name: "single run passing",
			outcomes: []RunOutcome{
				{Passed: true},
			},
			want: FlakeReport{TotalRuns: 1, PassCount: 1, FailCount: 0, FlakeRatePercent: 0, Flaky: false},
		},
		{
			name: "single run failing",
			outcomes: []RunOutcome{
				{Passed: false},
			},
			want: FlakeReport{TotalRuns: 1, PassCount: 0, FailCount: 1, FlakeRatePercent: 0, Flaky: false},
		},
		{
			name:     "mixed, 1 fail out of 20",
			outcomes: repeatOutcomes(19, true, 1, false),
			want:     FlakeReport{TotalRuns: 20, PassCount: 19, FailCount: 1, FlakeRatePercent: 5, Flaky: true},
		},
		{
			name:     "mixed, evenly split",
			outcomes: repeatOutcomes(10, true, 10, false),
			want:     FlakeReport{TotalRuns: 20, PassCount: 10, FailCount: 10, FlakeRatePercent: 50, Flaky: true},
		},
		{
			name:     "mixed, majority failing",
			outcomes: repeatOutcomes(3, true, 17, false),
			want:     FlakeReport{TotalRuns: 20, PassCount: 3, FailCount: 17, FlakeRatePercent: 15, Flaky: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeFlakeReport(tt.outcomes)
			if got != tt.want {
				t.Errorf("ComputeFlakeReport(%d outcomes) = %+v, want %+v", len(tt.outcomes), got, tt.want)
			}
		})
	}
}

func repeatOutcomes(nPass int, passVal bool, nFail int, failVal bool) []RunOutcome {
	out := make([]RunOutcome, 0, nPass+nFail)
	for i := 0; i < nPass; i++ {
		out = append(out, RunOutcome{Passed: passVal})
	}
	for i := 0; i < nFail; i++ {
		out = append(out, RunOutcome{Passed: failVal})
	}
	return out
}
