package flaky

// RunOutcome is the minimal signal needed to compute flake statistics for a
// single run: whether it passed.
type RunOutcome struct {
	Passed bool
}

// FlakeReport summarizes the outcomes of a batch of reruns of the same
// command.
type FlakeReport struct {
	TotalRuns int
	PassCount int
	FailCount int

	// FlakeRatePercent is the percentage of runs that disagreed with the
	// majority outcome. It is 0 whenever every run agreed - all passed, or
	// all failed - even though FailCount may still be nonzero in the
	// all-fail case: a command that fails every single time isn't flaky,
	// it's just broken.
	FlakeRatePercent float64

	// Flaky is true only when the batch produced both passing and failing
	// runs. This is the signal flakyfinder's exit code is based on.
	Flaky bool
}

// ComputeFlakeReport tallies pass/fail counts across outcomes and derives
// the flake rate and flaky verdict for the batch. It is a pure function:
// given the same outcomes it always returns the same report.
func ComputeFlakeReport(outcomes []RunOutcome) FlakeReport {
	report := FlakeReport{TotalRuns: len(outcomes)}
	if report.TotalRuns == 0 {
		return report
	}

	for _, o := range outcomes {
		if o.Passed {
			report.PassCount++
		} else {
			report.FailCount++
		}
	}

	report.Flaky = report.PassCount > 0 && report.FailCount > 0
	if report.Flaky {
		minority := report.PassCount
		if report.FailCount < minority {
			minority = report.FailCount
		}
		report.FlakeRatePercent = float64(minority) / float64(report.TotalRuns) * 100
	}

	return report
}
