// Package flaky contains the pure, side-effect-free logic behind
// flakyfinder: generating per-run environment perturbations, computing
// jitter delays, tallying flake statistics, and correlating environment
// values with failures. None of the code in this package touches the
// filesystem, the clock's wall time, or a subprocess - it only computes
// values from its inputs, which makes it straightforward to unit test.
package flaky

// SeedEnv describes one environment variable whose value should cycle
// through a fixed list of values across successive runs. It corresponds to
// a single `--seed-env KEY=v1,v2,v3` flag on the command line.
type SeedEnv struct {
	Key    string
	Values []string
}

// GenerateEnv returns the environment variable overrides that should be
// applied to the run at the given zero-based index. For each SeedEnv it
// picks Values[runIndex % len(Values)], so values are cycled round-robin
// across runs. SeedEnv entries with no values are skipped.
//
// GenerateEnv is a pure function: it has no side effects, and the same
// seedEnvs and runIndex always produce the exact same result, which is what
// makes the env perturbation applied to any given run fully reproducible.
func GenerateEnv(seedEnvs []SeedEnv, runIndex int) map[string]string {
	env := make(map[string]string, len(seedEnvs))
	for _, se := range seedEnvs {
		if len(se.Values) == 0 {
			continue
		}
		idx := runIndex % len(se.Values)
		env[se.Key] = se.Values[idx]
	}
	return env
}
