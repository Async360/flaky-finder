package flaky

import "sort"

// EnvSnapshot captures the seed-env variable values that were in effect for
// one run.
type EnvSnapshot map[string]string

// Result pairs an environment snapshot with whether that run passed. It is
// the input to AnalyzeCorrelation.
type Result struct {
	Env    EnvSnapshot
	Passed bool
}

// Correlation reports that a particular value of an env var showed a
// higher failure rate than some other value of that same env var across
// the batch of runs, suggesting it may explain the observed flakiness.
type Correlation struct {
	Key             string
	Value           string
	Occurrences     int
	Failures        int
	FailRatePercent float64
}

// AnalyzeCorrelation is a pure function that, given a set of run results
// (each an env snapshot plus a pass/fail outcome), determines which
// (key, value) pairs are associated with a higher failure rate than other
// values seen for the same key. A key is only considered if at least two
// distinct values were observed for it - if every run used the same value,
// there's nothing to compare it against, so it can't explain why some runs
// passed and others failed.
//
// The result is sorted by descending failure rate, so the most suspicious
// correlations come first.
func AnalyzeCorrelation(results []Result) []Correlation {
	type stat struct {
		occurrences int
		failures    int
	}

	byKey := map[string]map[string]*stat{}
	var keyOrder []string

	for _, r := range results {
		for k, v := range r.Env {
			values, ok := byKey[k]
			if !ok {
				values = map[string]*stat{}
				byKey[k] = values
				keyOrder = append(keyOrder, k)
			}
			s, ok := values[v]
			if !ok {
				s = &stat{}
				values[v] = s
			}
			s.occurrences++
			if !r.Passed {
				s.failures++
			}
		}
	}

	var out []Correlation
	for _, key := range keyOrder {
		values := byKey[key]
		if len(values) < 2 {
			continue
		}

		bestRate := -1.0
		for _, s := range values {
			rate := float64(s.failures) / float64(s.occurrences)
			if bestRate < 0 || rate < bestRate {
				bestRate = rate
			}
		}

		for value, s := range values {
			rate := float64(s.failures) / float64(s.occurrences)
			if rate > bestRate {
				out = append(out, Correlation{
					Key:             key,
					Value:           value,
					Occurrences:     s.occurrences,
					Failures:        s.failures,
					FailRatePercent: rate * 100,
				})
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].FailRatePercent != out[j].FailRatePercent {
			return out[i].FailRatePercent > out[j].FailRatePercent
		}
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		return out[i].Value < out[j].Value
	})

	return out
}
