package runner

import (
	"testing"
	"time"

	"github.com/Async360/flaky-finder/internal/flaky"
)

func TestRunCapturesExitCodeAndOutput(t *testing.T) {
	records := Run(Options{
		Command: "echo hello; echo world 1>&2; exit 0",
		Times:   3,
	})

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	for i, r := range records {
		if !r.Passed || r.ExitCode != 0 {
			t.Errorf("record %d: expected pass/exit=0, got passed=%v exit=%d", i, r.Passed, r.ExitCode)
		}
		if r.StdoutSize == 0 {
			t.Errorf("record %d: expected nonzero stdout size", i)
		}
		if r.StderrSize == 0 {
			t.Errorf("record %d: expected nonzero stderr size", i)
		}
		if r.Index != i {
			t.Errorf("record %d: expected Index=%d, got %d", i, i, r.Index)
		}
	}
}

func TestRunCapturesNonZeroExitCode(t *testing.T) {
	records := Run(Options{
		Command: "exit 7",
		Times:   1,
	})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Passed {
		t.Errorf("expected failing run, got Passed=true")
	}
	if records[0].ExitCode != 7 {
		t.Errorf("expected ExitCode=7, got %d", records[0].ExitCode)
	}
}

func TestRunAppliesSeedEnvPerturbation(t *testing.T) {
	records := Run(Options{
		Command:  `[ "$MODE" = "bad" ] && exit 1; exit 0`,
		Times:    4,
		SeedEnvs: []flaky.SeedEnv{{Key: "MODE", Values: []string{"good", "bad"}}},
	})

	if len(records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(records))
	}
	want := []bool{true, false, true, false}
	for i, r := range records {
		if r.Passed != want[i] {
			t.Errorf("record %d: MODE=%s, expected passed=%v, got %v", i, r.Env["MODE"], want[i], r.Passed)
		}
	}
}

func TestRunAppliesJitterBeforeEachRun(t *testing.T) {
	var slept []time.Duration
	records := Run(Options{
		Command:  "exit 0",
		Times:    3,
		JitterMs: 50,
		Seed:     1,
		Sleep: func(d time.Duration) {
			slept = append(slept, d)
		},
	})

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	if len(slept) == 0 {
		t.Fatalf("expected the injected Sleep to be called at least once")
	}
	for _, d := range slept {
		if d < 0 || d > 50*time.Millisecond {
			t.Errorf("jitter sleep %s out of range [0, 50ms]", d)
		}
	}
}

func TestRunZeroTimesReturnsNoRecords(t *testing.T) {
	records := Run(Options{Command: "exit 0", Times: 0})
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}
