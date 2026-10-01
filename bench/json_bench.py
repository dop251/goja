#!/usr/bin/env python3
"""Measure JSON benchmarks in separate processes (Linux/macOS, Python 3)."""

import argparse
import csv
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import tempfile
import time


ROOT = Path(__file__).resolve().parents[1]
FIXTURES = (
    "medium.json", "medium-10x.json", "medium-100x.json",
    "unicode.json", "numbers.json", "strings.json",
)


def run(command, **kwargs):
    return subprocess.run(command, cwd=ROOT, check=True, **kwargs)


def measure(command, log):
    start = time.monotonic()
    with log.open("w") as output:
        child = subprocess.Popen(command, cwd=ROOT, stdout=output, stderr=subprocess.STDOUT)
        try:
            _, status, usage = os.wait4(child.pid, 0)
        except BaseException:
            child.kill()
            child.wait()
            raise
        child.returncode = os.waitstatus_to_exitcode(status)
    elapsed = time.monotonic() - start
    if child.returncode:
        raise RuntimeError(f"Benchmark failed ({child.returncode}); see {log}")
    cpu = usage.ru_utime + usage.ru_stime
    return {
        "wall_seconds": elapsed,
        "user_cpu_seconds": usage.ru_utime,
        "system_cpu_seconds": usage.ru_stime,
        "cpu_percent": 100 * cpu / elapsed,
        "peak_rss_mib": usage.ru_maxrss / (1024**2 if platform.system() == "Darwin" else 1024),
        "minor_page_faults": usage.ru_minflt,
        "major_page_faults": usage.ru_majflt,
        "voluntary_context_switches": usage.ru_nvcsw,
        "involuntary_context_switches": usage.ru_nivcsw,
    }


def benchmark_metrics(log):
    for line in log.read_text().splitlines():
        if not line.startswith("BenchmarkJSON"):
            continue
        columns = line.split()
        metrics = dict(zip(columns[3::2], columns[2::2]))
        return {
            "iterations": int(columns[1]),
            "ns_per_op": float(metrics["ns/op"]),
            "mb_per_second": float(metrics["MB/s"]),
            "bytes_per_op": float(metrics["B/op"]),
            "allocs_per_op": float(metrics["allocs/op"]),
        }
    raise RuntimeError(f"No benchmark result found in {log}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--benchtime", default="1s", help="Go benchmark duration or iteration count (default: 1s)")
    parser.add_argument("--repeat", type=int, default=3, help="Fresh processes per case (default: 3)")
    parser.add_argument("--fixtures", nargs="+", choices=FIXTURES, default=FIXTURES)
    parser.add_argument("--operations", nargs="+", choices=("parse", "stringify"), default=("parse", "stringify"))
    parser.add_argument("--output", type=Path, help="New results directory (default: temporary directory)")
    parser.add_argument("--profiles", action="store_true", help="Also run each case separately with Go CPU and heap profiles")
    args = parser.parse_args()
    if args.repeat < 1:
        parser.error("--repeat must be positive")
    if not re.fullmatch(r"(?:[1-9][0-9]*x|(?:[0-9]+(?:\.[0-9]+)?)(?:ns|us|ms|s|m|h))", args.benchtime):
        parser.error("--benchtime must be a duration such as 1s or an iteration count such as 10x")
    if not hasattr(os, "wait4"):
        parser.error("Process resource measurements require Linux or macOS")
    if args.output:
        output = args.output.resolve()
        output.mkdir(parents=True, exist_ok=False)
    else:
        output = Path(tempfile.mkdtemp(prefix="goja-json-bench-"))
    print(f"Results: {output}", flush=True)
    metadata = {
        "platform": platform.platform(),
        "logical_cpus": os.cpu_count(),
        "go_version": run(["go", "version"], capture_output=True, text=True).stdout.strip(),
        "commit": run(["git", "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip(),
        "working_tree": run(["git", "status", "--short", "--untracked-files=all"], capture_output=True, text=True).stdout,
        "environment": {key: os.environ.get(key) for key in ("GOMAXPROCS", "GOGC", "GOMEMLIMIT", "GOFLAGS")},
        "benchtime": args.benchtime,
        "repeat": args.repeat,
        "fixtures": args.fixtures,
        "operations": args.operations,
    }
    (output / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n")
    run(["go", "run", "./testdata/json/gen.go"])
    binary = output / "goja.test"
    run(["go", "test", "-c", "-o", str(binary), "."])
    with (output / "results.csv").open("w", newline="") as csv_file:
        writer = None
        for operation in args.operations:
            for fixture in args.fixtures:
                pattern = f"^BenchmarkJSON{operation.capitalize()}Fixture$/^{re.escape(fixture)}$"
                command = [str(binary), "-test.run=^$", f"-test.bench={pattern}",
                           f"-test.benchtime={args.benchtime}", "-test.count=1", "-test.benchmem"]
                for repetition in range(1, args.repeat + 1):
                    label = f"{operation}-{fixture}-{repetition}"
                    log = output / f"{label}.log"
                    resources = measure(command, log)
                    row = {"operation": operation, "fixture": fixture, "repeat": repetition,
                           "input_bytes": (ROOT / "testdata/json" / fixture).stat().st_size,
                           **benchmark_metrics(log), **resources}
                    if writer is None:
                        writer = csv.DictWriter(csv_file, fieldnames=list(row))
                        writer.writeheader()
                    writer.writerow(row)
                    csv_file.flush()
                    print(f"{label}: {row['mb_per_second']:.2f} MB/s, "
                          f"{row['peak_rss_mib']:.1f} MiB peak RSS, "
                          f"{row['cpu_percent']:.1f}% CPU", flush=True)
                if args.profiles:
                    prefix = output / f"{operation}-{fixture}"
                    with Path(f"{prefix}-profile.log").open("w") as log:
                        run(command + [f"-test.cpuprofile={prefix}.cpu.pprof",
                                       f"-test.memprofile={prefix}.heap.pprof"],
                            stdout=log, stderr=subprocess.STDOUT)
    print(f"Report: {output / 'results.csv'}", flush=True)


if __name__ == "__main__":
    main()
