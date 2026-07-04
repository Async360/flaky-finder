package flaky

import "testing"

func TestJitterDuration(t *testing.T) {
	tests := []struct {
		name     string
		seed     int64
		runIndex int
		maxMs    int
	}{
		{"zero max is always zero", 1, 0, 0},
		{"negative max is always zero", 1, 0, -5},
		{"typical case", 7, 3, 100},
		{"large run index", 7, 10000, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JitterDuration(tt.seed, tt.runIndex, tt.maxMs)
			if tt.maxMs <= 0 && got != 0 {
				t.Errorf("JitterDuration(%d, %d, %d) = %d, want 0", tt.seed, tt.runIndex, tt.maxMs, got)
			}
			if got < 0 || got > tt.maxMs {
				if tt.maxMs > 0 {
					t.Errorf("JitterDuration(%d, %d, %d) = %d, out of range [0, %d]", tt.seed, tt.runIndex, tt.maxMs, got, tt.maxMs)
				}
			}
		})
	}
}

func TestJitterDurationIsDeterministic(t *testing.T) {
	for i := 0; i < 50; i++ {
		first := JitterDuration(42, i, 250)
		second := JitterDuration(42, i, 250)
		if first != second {
			t.Fatalf("JitterDuration not deterministic at run %d: %d != %d", i, first, second)
		}
	}
}

func TestJitterDurationDiffersBySeed(t *testing.T) {
	// Not a strict requirement, but different seeds should generally
	// produce different sequences - guards against an accidental constant
	// implementation.
	a := make([]int, 10)
	b := make([]int, 10)
	for i := range a {
		a[i] = JitterDuration(1, i, 1000)
		b[i] = JitterDuration(2, i, 1000)
	}
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Errorf("expected different seeds to produce different jitter sequences, got identical: %v", a)
	}
}
