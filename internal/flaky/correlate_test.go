package flaky

import "testing"

func TestAnalyzeCorrelation(t *testing.T) {
	t.Run("no results", func(t *testing.T) {
		got := AnalyzeCorrelation(nil)
		if len(got) != 0 {
			t.Errorf("expected no correlations, got %+v", got)
		}
	})

	t.Run("no env vars at all", func(t *testing.T) {
		results := []Result{
			{Env: EnvSnapshot{}, Passed: true},
			{Env: EnvSnapshot{}, Passed: false},
		}
		got := AnalyzeCorrelation(results)
		if len(got) != 0 {
			t.Errorf("expected no correlations, got %+v", got)
		}
	})

	t.Run("constant value across all runs is never reported", func(t *testing.T) {
		// Every run used the same env value, even though some failed - so
		// this value can't be blamed for the mixed outcome.
		results := []Result{
			{Env: EnvSnapshot{"NODE_ENV": "test"}, Passed: true},
			{Env: EnvSnapshot{"NODE_ENV": "test"}, Passed: false},
			{Env: EnvSnapshot{"NODE_ENV": "test"}, Passed: true},
		}
		got := AnalyzeCorrelation(results)
		if len(got) != 0 {
			t.Errorf("expected no correlations for a constant value, got %+v", got)
		}
	})

	t.Run("identifies the failing value clearly", func(t *testing.T) {
		// NODE_ENV=staging always fails, NODE_ENV=test always passes.
		results := []Result{
			{Env: EnvSnapshot{"NODE_ENV": "test"}, Passed: true},
			{Env: EnvSnapshot{"NODE_ENV": "test"}, Passed: true},
			{Env: EnvSnapshot{"NODE_ENV": "test"}, Passed: true},
			{Env: EnvSnapshot{"NODE_ENV": "staging"}, Passed: false},
			{Env: EnvSnapshot{"NODE_ENV": "staging"}, Passed: false},
		}
		got := AnalyzeCorrelation(results)
		if len(got) != 1 {
			t.Fatalf("expected exactly 1 correlation, got %+v", got)
		}
		c := got[0]
		if c.Key != "NODE_ENV" || c.Value != "staging" {
			t.Errorf("expected correlation on NODE_ENV=staging, got %s=%s", c.Key, c.Value)
		}
		if c.Occurrences != 2 || c.Failures != 2 {
			t.Errorf("expected 2/2 occurrences failing, got %d/%d", c.Failures, c.Occurrences)
		}
		if c.FailRatePercent != 100 {
			t.Errorf("expected 100%% fail rate, got %.1f", c.FailRatePercent)
		}
	})

	t.Run("ignores keys unrelated to failure and reports only the guilty one", func(t *testing.T) {
		results := []Result{
			{Env: EnvSnapshot{"NODE_ENV": "test", "REGION": "us"}, Passed: true},
			{Env: EnvSnapshot{"NODE_ENV": "test", "REGION": "eu"}, Passed: true},
			{Env: EnvSnapshot{"NODE_ENV": "staging", "REGION": "us"}, Passed: false},
			{Env: EnvSnapshot{"NODE_ENV": "staging", "REGION": "eu"}, Passed: false},
		}
		got := AnalyzeCorrelation(results)
		if len(got) != 1 {
			t.Fatalf("expected exactly 1 correlation (NODE_ENV=staging), got %+v", got)
		}
		if got[0].Key != "NODE_ENV" || got[0].Value != "staging" {
			t.Errorf("expected NODE_ENV=staging, got %s=%s", got[0].Key, got[0].Value)
		}
	})

	t.Run("partial correlation still surfaces the higher-failure value", func(t *testing.T) {
		// TZ=UTC fails 1/4 of the time, TZ=America/New_York fails 3/4 of
		// the time - the latter should be flagged, the former should not.
		results := []Result{
			{Env: EnvSnapshot{"TZ": "UTC"}, Passed: true},
			{Env: EnvSnapshot{"TZ": "UTC"}, Passed: true},
			{Env: EnvSnapshot{"TZ": "UTC"}, Passed: true},
			{Env: EnvSnapshot{"TZ": "UTC"}, Passed: false},
			{Env: EnvSnapshot{"TZ": "America/New_York"}, Passed: false},
			{Env: EnvSnapshot{"TZ": "America/New_York"}, Passed: false},
			{Env: EnvSnapshot{"TZ": "America/New_York"}, Passed: false},
			{Env: EnvSnapshot{"TZ": "America/New_York"}, Passed: true},
		}
		got := AnalyzeCorrelation(results)
		if len(got) != 1 {
			t.Fatalf("expected exactly 1 correlation, got %+v", got)
		}
		if got[0].Value != "America/New_York" {
			t.Errorf("expected America/New_York to be flagged, got %s", got[0].Value)
		}
		if got[0].Failures != 3 || got[0].Occurrences != 4 {
			t.Errorf("expected 3/4, got %d/%d", got[0].Failures, got[0].Occurrences)
		}
	})

	t.Run("sorted by descending fail rate", func(t *testing.T) {
		results := []Result{
			{Env: EnvSnapshot{"A": "good", "B": "good"}, Passed: true},
			{Env: EnvSnapshot{"A": "bad", "B": "good"}, Passed: false},
			{Env: EnvSnapshot{"A": "bad", "B": "good"}, Passed: false},
			{Env: EnvSnapshot{"A": "good", "B": "worse"}, Passed: false},
		}
		got := AnalyzeCorrelation(results)
		if len(got) < 2 {
			t.Fatalf("expected at least 2 correlations, got %+v", got)
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].FailRatePercent < got[i].FailRatePercent {
				t.Errorf("results not sorted by descending fail rate: %+v", got)
			}
		}
	})
}
