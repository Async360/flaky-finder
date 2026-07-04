// Command flakyfinder reruns an arbitrary test command multiple times to
// help reproduce flaky tests locally, before they intermittently fail in
// CI. See README.md for full usage.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Async360/flaky-finder/internal/flaky"
	"github.com/Async360/flaky-finder/internal/runner"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run parses arguments, executes the requested command, prints a report,
// and returns the process exit code. It's kept separate from main so it
// can be exercised without literally calling os.Exit.
func run(args []string) int {
	if len(args) == 0 || args[0] != "run" {
		printUsage(os.Stderr)
		return 2
	}

	fs := flag.NewFlagSet("flakyfinder run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	times := fs.Int("times", 20, "number of times to rerun the command")
	jitterMs := fs.Int("jitter-ms", 0, "sleep a random 0..N ms before each run")
	seed := fs.Int64("seed", 1, "seed for the jitter RNG, for reproducibility")
	var seedEnvFlags stringSliceFlag
	fs.Var(&seedEnvFlags, "seed-env", "KEY=v1,v2,v3 - cycle KEY through the given values across runs (repeatable)")

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cmdArgs := fs.Args()
	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "flakyfinder: no command given, expected: flakyfinder run [flags] -- <command...>")
		return 2
	}
	if *times <= 0 {
		fmt.Fprintln(os.Stderr, "flakyfinder: --times must be greater than 0")
		return 2
	}

	seedEnvs, err := parseSeedEnvs(seedEnvFlags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "flakyfinder: %v\n", err)
		return 2
	}

	command := flaky.JoinShellCommand(cmdArgs)

	records := runner.Run(runner.Options{
		Command:  command,
		Times:    *times,
		SeedEnvs: seedEnvs,
		JitterMs: *jitterMs,
		Seed:     *seed,
	})

	return report(os.Stdout, command, records)
}

// parseSeedEnvs turns repeated --seed-env KEY=v1,v2,v3 flag values into
// flaky.SeedEnv entries.
func parseSeedEnvs(raw []string) ([]flaky.SeedEnv, error) {
	seedEnvs := make([]flaky.SeedEnv, 0, len(raw))
	for _, entry := range raw {
		key, valuesRaw, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --seed-env %q, expected KEY=v1,v2,v3", entry)
		}
		values := strings.Split(valuesRaw, ",")
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		if len(values) == 0 || (len(values) == 1 && values[0] == "") {
			return nil, fmt.Errorf("invalid --seed-env %q, need at least one value", entry)
		}
		seedEnvs = append(seedEnvs, flaky.SeedEnv{Key: key, Values: values})
	}
	return seedEnvs, nil
}

// report prints the per-run detail, summary, and any suspected
// correlations to w, and returns the process exit code: 0 if no flakiness
// was detected, 1 if it was.
func report(w *os.File, command string, records []runner.Record) int {
	fmt.Fprintf(w, "flakyfinder: rerunning `%s` %d time(s)\n\n", command, len(records))

	outcomes := make([]flaky.RunOutcome, len(records))
	results := make([]flaky.Result, len(records))
	for i, rec := range records {
		status := "PASS"
		if !rec.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(w, "run %-3d %s  exit=%-4d dur=%-10s stdout=%-8s stderr=%-8s env=%s\n",
			rec.Index+1, status, rec.ExitCode,
			rec.Duration.Round(time.Millisecond), sizeStr(rec.StdoutSize), sizeStr(rec.StderrSize),
			envStr(rec.Env))

		outcomes[i] = flaky.RunOutcome{Passed: rec.Passed}
		results[i] = flaky.Result{Env: flaky.EnvSnapshot(rec.Env), Passed: rec.Passed}
	}

	flakeReport := flaky.ComputeFlakeReport(outcomes)

	fmt.Fprintln(w, "\n--- summary ---")
	fmt.Fprintf(w, "total runs:  %d\n", flakeReport.TotalRuns)
	fmt.Fprintf(w, "passed:      %d\n", flakeReport.PassCount)
	fmt.Fprintf(w, "failed:      %d\n", flakeReport.FailCount)
	fmt.Fprintf(w, "flake rate:  %.1f%%\n", flakeReport.FlakeRatePercent)
	fmt.Fprintf(w, "flaky:       %s\n", yesNo(flakeReport.Flaky))

	correlations := flaky.AnalyzeCorrelation(results)
	if len(correlations) > 0 {
		fmt.Fprintln(w, "\n--- suspected env correlations ---")
		for _, c := range correlations {
			fmt.Fprintf(w, "%s=%s  ->  %d/%d runs failed (%.1f%%)\n",
				c.Key, c.Value, c.Failures, c.Occurrences, c.FailRatePercent)
		}
	}

	if flakeReport.Flaky {
		return 1
	}
	return 0
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func sizeStr(n int) string {
	return strconv.Itoa(n) + "B"
}

func envStr(env map[string]string) string {
	if len(env) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + env[k]
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `flakyfinder - reproduce flaky tests locally by rerunning a command N times

Usage:
  flakyfinder run [flags] -- <command...>

Flags:
  --times int        number of times to rerun the command (default 20)
  --seed-env KEY=v1,v2,v3
                      cycle env var KEY through the given values across runs
                      (repeatable for multiple vars)
  --jitter-ms int     sleep a random 0..N ms before each run
  --seed int          seed for the jitter RNG, for reproducibility (default 1)

Exit codes:
  0   no flakiness detected (all runs passed, or all runs failed identically)
  1   flakiness detected (mixed pass/fail results)
  2   usage error
`)
}

// stringSliceFlag implements flag.Value, accumulating each occurrence of a
// repeated flag into a slice instead of overwriting a single value.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}
