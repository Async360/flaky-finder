# flaky-finder

`flakyfinder` is a small CLI that reruns any test command N times,
perturbing its environment and timing, and reports on whether it's flaky -
before it flakes out in CI.

## The problem

Flaky tests are a well-known tax on engineering time: a test that
intermittently fails for reasons unrelated to the code under test forces
developers to re-run CI, second-guess real regressions, and eventually just
start ignoring red builds. Once a team stops trusting its test suite, CI
stops doing its job.

The tooling to *find* flakiness is scattered and framework-specific: Jest has
`--testSequencer` and seed-based ordering flags, RSpec has `--seed`, Minitest
has `--seed`, ExUnit has `--seed`, pytest has `pytest-randomly`. Each of these
lives inside its own language's test runner and only understands that
runner's process model. None of them help you if your CI flakiness comes
from environment differences (a var set in the CI runner but not locally,
locale/timezone drift, a `NODE_ENV` that differs by pipeline stage) or from
timing (a race that only shows up under a certain scheduling jitter) - and
none of them work across languages.

`flakyfinder` doesn't replace any of those tools. It sits a layer above all
of them: it treats your test command as an opaque shell invocation, reruns it
however many times you like, perturbs the environment and the timing between
runs, and tells you whether the result was consistent or flaky - regardless
of whether that command is `npm test`, `bundle exec rspec`, `mix test`,
`pytest`, or `go test ./...`.

## Install

```
go install github.com/Async360/flaky-finder@latest
```

This installs a binary named `flaky-finder` (matching the module path) into
your `$GOBIN`. Or build from source, which is how the examples below name
the binary:

```
git clone https://github.com/Async360/flaky-finder.git
cd flaky-finder
go build -o flakyfinder .
```

## Usage

```
flakyfinder run [flags] -- <command...>
```

Pass the command the same way you'd normally type it - unquoted, letting your
own shell split it into words. Don't wrap the whole thing in an extra layer
of quotes; `flakyfinder run -- npm test` is correct, `flakyfinder run --
'npm test'` is not (the latter hands flakyfinder a single string "npm test"
as one argument, which it will try to run as a program literally named
"npm test").

### Basic: rerun a command 20 times (the default)

```
flakyfinder run -- go test ./...
```

### Rerun a specific number of times

```
flakyfinder run --times 50 -- npm test
```

### Probe env-dependent flakiness

`--seed-env KEY=v1,v2,v3` cycles the environment variable `KEY` through the
given values, one per run (repeat the flag for multiple variables):

```
flakyfinder run --times 12 \
  --seed-env NODE_ENV=test,staging \
  --seed-env TZ=UTC,America/New_York \
  -- npm test
```

If the failures line up with one particular value, flakyfinder's report will
call it out:

```
--- suspected env correlations ---
NODE_ENV=staging  ->  6/6 runs failed (100.0%)
```

### Probe timing-dependent flakiness

`--jitter-ms N` sleeps a random `0..N` milliseconds before each run. The
delay sequence is derived from `--seed` (default `1`), so a given
`--seed`/`--jitter-ms`/`--times` combination always reproduces the exact same
sequence of delays:

```
flakyfinder run --times 30 --jitter-ms 250 --seed 7 -- pytest -k test_worker_pool
```

### Combine both

```
flakyfinder run --times 30 \
  --seed-env RACE_MODE=on,off \
  --jitter-ms 100 --seed 42 \
  -- bundle exec rspec spec/queue_spec.rb
```

## What the report shows

For every run: its pass/fail status, exit code, wall-clock duration, and the
size in bytes of stdout/stderr it produced (useful for spotting runs that
silently produced no output, or ones that dumped an unusual stack trace).

At the end, a summary:

```
--- summary ---
total runs:  20
passed:      17
failed:      3
flake rate:  15.0%
flaky:       yes
```

`flake rate` is the percentage of runs that disagreed with the majority
outcome. It's always `0%` when every run agreed - including when every run
failed identically, since a command that's simply broken every time isn't
flaky, it's just broken.

If any `--seed-env` variables were supplied, flakyfinder also runs a
correlation pass: for each variable, it compares the failure rate of each
value against the best (lowest-failure) value seen for that same variable,
and surfaces any value whose failure rate is higher - i.e. any value that
looks like a plausible cause of the flakiness.

## Exit codes

- `0` - no flakiness detected: every run passed, or every run failed the same way.
- `1` - flakiness detected: the runs produced a mix of passes and failures.
  This makes flakyfinder itself CI-gateable - wire it into a job as a
  pre-merge check on a suspect test and let a nonzero exit block the merge.
- `2` - usage error (bad flags, no command given after `--`, etc).

## Limitations

- flakyfinder can only reproduce flakiness that's a function of the things it
  controls: your process's environment variables and the timing of when each
  run starts. It cannot reproduce flakiness caused by genuinely external
  factors - a flaky network call, a shared database or queue with real
  concurrent state, a third-party API's rate limiting, clock skew between
  machines, or a dependent service that's itself unreliable. If your test's
  flakiness comes from one of those, flakyfinder will just show you "flaky"
  without being able to tell you why.
- The env and jitter perturbations only affect what flakyfinder itself
  controls before invoking your command; they can't reach into a test
  framework's internal PRNG (e.g. property-based testing seeds) unless that
  framework already reads its seed from an environment variable you can pass
  via `--seed-env`.
- Runs are executed sequentially, not in parallel, so a full `--times 50` run
  takes roughly 50x as long as running the command once. This is
  intentional: parallel execution would itself introduce a new source of
  nondeterminism (resource contention) that would muddy the results.
- The target command is executed via `sh -c`, exactly as if you'd typed it
  into your own shell - which means it's meant to be pointed at a command
  you already trust to run yourself, not at arbitrary untrusted input.

## Development

```
go build ./...
go vet ./...
gofmt -l .      # should print nothing
go test ./...
```

The pure decision logic (env-perturbation cycling, jitter calculation, flake
rate math, and failure-correlation analysis) lives in `internal/flaky` and is
covered by table-driven unit tests with no subprocesses involved. The
subprocess execution itself lives in `internal/runner` and is exercised with
a handful of tests against real `sh` invocations.

## License

MIT - see [LICENSE](LICENSE). Copyright (c) 2026 Async360.
