package flaky

import "math/rand"

// JitterDuration returns the number of milliseconds to sleep before the run
// at the given zero-based index, chosen uniformly from [0, maxMs]. If maxMs
// is zero or negative, it always returns 0 (no jitter).
//
// JitterDuration is a pure function of its inputs: the same seed, runIndex
// and maxMs always produce the same delay. That determinism is what lets
// --jitter-ms be reproduced exactly across invocations via --seed - the RNG
// for each run is seeded from (seed + runIndex), rather than from wall-clock
// time.
func JitterDuration(seed int64, runIndex int, maxMs int) int {
	if maxMs <= 0 {
		return 0
	}
	src := rand.NewSource(seed + int64(runIndex))
	r := rand.New(src)
	return r.Intn(maxMs + 1)
}
