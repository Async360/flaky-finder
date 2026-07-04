// Package runner does the impure work of actually executing the target
// test command multiple times: it shells out, applies environment
// perturbations and jitter delays computed by the pure internal/flaky
// package, and records what happened. It deliberately contains no decision
// logic of its own about flakiness - that all lives in internal/flaky where
// it can be unit tested without spawning processes.
package runner

import (
	"bytes"
	"os"
	"os/exec"
	"time"

	"github.com/Async360/flaky-finder/internal/flaky"
)

// Record captures the observable outcome of a single execution of the
// target command.
type Record struct {
	Index      int
	Env        flaky.EnvSnapshot
	Passed     bool
	ExitCode   int
	Duration   time.Duration
	StdoutSize int
	StderrSize int
}

// Options configures a batch of reruns of a shell command.
type Options struct {
	// Command is a shell command line, executed via `sh -c`.
	Command string
	// Times is how many times to run Command.
	Times int
	// SeedEnvs are the env vars to cycle per run; may be empty.
	SeedEnvs []flaky.SeedEnv
	// JitterMs is the maximum jitter delay in milliseconds; 0 disables it.
	JitterMs int
	// Seed drives the jitter RNG so delays are reproducible.
	Seed int64

	// Sleep defaults to time.Sleep; it exists so tests can override it
	// without actually pausing.
	Sleep func(time.Duration)
}

// Run executes opts.Command opts.Times times, applying the env perturbation
// and jitter described by opts, and returns one Record per execution in
// order.
func Run(opts Options) []Record {
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	records := make([]Record, 0, opts.Times)
	for i := 0; i < opts.Times; i++ {
		jitterMs := flaky.JitterDuration(opts.Seed, i, opts.JitterMs)
		if jitterMs > 0 {
			sleep(time.Duration(jitterMs) * time.Millisecond)
		}

		env := flaky.GenerateEnv(opts.SeedEnvs, i)

		// Shelling out via `sh -c` is intentional, not an oversight: this
		// tool's whole job is to rerun a caller-supplied test command
		// exactly as the caller's own shell would run it, including any
		// pipes/&&/redirection they included. opts.Command is meant to be
		// supplied by whoever invokes flakyfinder from their own terminal
		// or CI job - the same trust boundary as running the command
		// directly - not attacker-controlled input from an untrusted
		// remote source. Don't wire opts.Command up to anything but a
		// local, trusted caller.
		cmd := exec.Command("sh", "-c", opts.Command)
		cmd.Env = append(os.Environ(), envToSlice(env)...)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		start := time.Now()
		err := cmd.Run()
		elapsed := time.Since(start)

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				// Command couldn't even be started (e.g. missing shell).
				exitCode = -1
			}
		}

		records = append(records, Record{
			Index:      i,
			Env:        env,
			Passed:     exitCode == 0,
			ExitCode:   exitCode,
			Duration:   elapsed,
			StdoutSize: stdout.Len(),
			StderrSize: stderr.Len(),
		})
	}

	return records
}

func envToSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
