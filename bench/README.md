# JSON benchmarks

Run from the repository root with Python 3 and Go installed on Linux or macOS:

```sh
python3 bench/json_bench.py
```

The script generates deterministic fixtures, builds a test binary once, and runs
each parse/stringify fixture in a fresh process three times. Results default to a
printed temporary directory. Fixture generation and compilation are excluded from
the measurements. Generated fixtures remain in `testdata/json` for direct Go
benchmark runs.

```sh
# Quick check of all sizes and operations
python3 bench/json_bench.py --benchtime 1x --repeat 1

# Longer measurements and separate profiling runs
python3 bench/json_bench.py --fixtures medium-10x.json medium-100x.json \
  --benchtime 3s --repeat 5 --profiles --output /tmp/json-benchmark-results
```

`--output` must name a new directory. `--operations parse` or
`--operations stringify` limits the operations measured. Go environment settings
such as `GOMAXPROCS`, `GOGC`, `GOMEMLIMIT`, and `GOCACHE` are inherited.

`results.csv` contains one row per process:

- Go benchmark iterations, ns/op, MB/s, bytes allocated/op, and allocations/op.
- Process wall time, user/system CPU time, and CPU utilization (CPU time divided
  by wall time; multiple cores can produce values above 100%).
- Peak resident memory in MiB, page faults, and context switches.

The Go per-operation metrics exclude fixture setup. Process CPU and peak RSS
include startup, fixture loading, benchmark calibration, and garbage collection.
Stringify setup also parses the input. Peak RSS measures the entire process,
including the Go runtime and input; it is neither incremental operation memory
nor retained heap. Duration-based benchmarks may run different iteration counts,
so use a fixed count such as `--benchtime 10x` when comparing equal workloads.

Raw Go benchmark logs and `metadata.json` capture results and execution context.
With `--profiles`, extra runs produce CPU and sampled heap profiles without
affecting the CSV measurements. The saved binary supports analysis:

```sh
go tool pprof /tmp/json-benchmark-results/goja.test \
  /tmp/json-benchmark-results/parse-medium-100x.json.cpu.pprof
go tool pprof -alloc_space /tmp/json-benchmark-results/goja.test \
  /tmp/json-benchmark-results/parse-medium-100x.json.heap.pprof
```
