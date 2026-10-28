"""Summarize repeated XDB/API results, pool gains, and optional baseline changes."""
import argparse
import glob
import json
import math
from pathlib import Path
from statistics import median


METRICS = ("requests_per_second", "ips_per_second", "p50_us", "p95_us",
           "error_rate", "server_cpu_percent", "server_rss_max_sampled_bytes",
           "bytes_per_request", "allocs_per_request", "pool_heap_bytes")
ENVIRONMENT = ("go", "os", "arch", "gomaxprocs", "clock")
KEYS = ("kind", "version", "endpoint", "pool", "workers", "batch")


def key(row):
    return tuple(row[name] for name in KEYS)


def summarize(reports, min_samples=1):
    groups, environment, datasets = {}, None, {}
    for report in reports:
        current = {name: report.get(name) for name in ENVIRONMENT}
        if environment is not None and environment != current:
            raise ValueError("mixed benchmark environments; summarize each environment separately")
        environment = current
        kind = report.get("kind", "xdb")
        if kind not in ("api", "xdb"):
            raise ValueError(f"unknown benchmark kind: {kind}")
        version = report.get("version", 0)
        identity = report.get("dataset", {"name": Path(report.get("database", "")).name,
                                           "bytes": report.get("database_bytes")})
        dataset_key = f"{kind}:{version}"
        if dataset_key in datasets and datasets[dataset_key] != identity:
            raise ValueError("mixed database snapshots")
        datasets[dataset_key] = identity
        if not report.get("results"):
            raise ValueError("empty benchmark results")
        for source in report["results"]:
            row = dict(source, kind=kind, version=version, endpoint=source.get("endpoint", "-"))
            if any(row.get(name, 0) < 1 for name in ("pool", "workers", "batch", "requests")):
                raise ValueError("invalid scenario counts")
            for name in METRICS:
                if name in row and (not isinstance(row[name], (int, float)) or not math.isfinite(row[name])):
                    raise ValueError(f"invalid metric {name}")
            groups.setdefault(key(row), []).append(row)
    if not groups:
        raise ValueError("no benchmark data")
    rows = []
    for identity, samples in sorted(groups.items()):
        if len(samples) < min_samples:
            raise ValueError(f"scenario {identity} needs {min_samples} samples, found {len(samples)}")
        if len({sample["requests"] for sample in samples}) != 1:
            raise ValueError("mixed request counts for the same scenario")
        metrics = {}
        for name in METRICS:
            values = [sample[name] for sample in samples if name in sample]
            if values:
                if len(values) != len(samples):
                    raise ValueError(f"missing metric {name} in some samples")
                metrics[name] = {"median": median(values), "min": min(values), "max": max(values)}
        rows.append(dict(zip(KEYS, identity), samples=len(samples), requests=samples[0]["requests"],
                         failed_samples=sum(bool(s.get("errors", 0) or s.get("misses", 0)) for s in samples),
                         metrics=metrics))
    result = {"schema": 1, "environment": environment, "datasets": datasets, "rows": rows}
    lookup = {key(row): row for row in rows}
    comparisons = []
    for row in rows:
        if row["pool"] == 1:
            continue
        baseline_key = tuple(1 if name == "pool" else row[name] for name in KEYS)
        if baseline_key in lookup:
            comparisons.append(compare(lookup[baseline_key], row))
    result["pool_comparisons"] = comparisons
    return result


def compare(before, after):
    metric = "ips_per_second" if after["kind"] == "xdb" else "requests_per_second"
    a, b = before["metrics"][metric]["median"], after["metrics"][metric]["median"]
    p, q = before["metrics"]["p95_us"]["median"], after["metrics"]["p95_us"]["median"]
    return dict(zip(KEYS, key(after)), before_pool=before["pool"],
                throughput_ratio=b / a if a else None,
                p95_change_percent=(q / p - 1) * 100 if p else None)


def add_baseline(summary, baseline):
    if summary["environment"] != baseline["environment"] or summary["datasets"] != baseline["datasets"]:
        raise ValueError("baseline environment/database differs; comparisons would be misleading")
    previous = {key(row): row for row in baseline["rows"]}
    rows = []
    for row in summary["rows"]:
        if key(row) in previous:
            old = previous[key(row)]
            if old["requests"] != row["requests"]:
                raise ValueError("baseline request counts differ")
            rows.append(compare(old, row))
    if not rows:
        raise ValueError("no matching baseline scenarios")
    summary["baseline_comparisons"] = rows


def render(summary):
    lines = ["Benchmark summary (median [min, max]); latency in us, RSS in MiB",
             json.dumps(summary["environment"], ensure_ascii=False)]
    for row in summary["rows"]:
        metrics = row["metrics"]
        metric = "ips_per_second" if row["kind"] == "xdb" else "requests_per_second"
        throughput, p95 = metrics[metric], metrics["p95_us"]
        prefix = f"{row['kind']} v{row['version']} {row['endpoint']} pool={row['pool']} workers={row['workers']} batch={row['batch']} n={row['samples']}"
        line = (f"{prefix}: {'IP/s' if row['kind']=='xdb' else 'req/s'}={throughput['median']:.1f} "
                f"[{throughput['min']:.1f}, {throughput['max']:.1f}] "
                f"P95={p95['median']:.1f} [{p95['min']:.1f}, {p95['max']:.1f}] failed_samples={row['failed_samples']}")
        if "server_cpu_percent" in metrics:
            line += f" CPU%={metrics['server_cpu_percent']['median']:.1f} RSS={metrics['server_rss_max_sampled_bytes']['median']/1024**2:.1f}"
        lines.append(line)
    for name in ("pool_comparisons", "baseline_comparisons"):
        if name not in summary:
            continue
        lines.append(name)
        for row in summary[name]:
            lines.append(f"  {row['kind']} v{row['version']} {row['endpoint']} pool={row['pool']} workers={row['workers']} batch={row['batch']}: throughput_ratio={row['throughput_ratio']} P95_change%={row['p95_change_percent']}")
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("inputs", nargs="+", help="raw report paths/globs")
    parser.add_argument("--min-samples", type=int, default=3)
    parser.add_argument("--baseline", type=Path, help="previous summary JSON")
    parser.add_argument("--out", type=Path, help="summary JSON path")
    parser.add_argument("--text", type=Path, help="human-readable summary path")
    args = parser.parse_args()
    if args.min_samples < 1:
        parser.error("min-samples must be positive")
    paths = sorted({path for pattern in args.inputs for path in glob.glob(pattern)})
    if not paths:
        parser.error("no report files matched")
    try:
        result = summarize([json.loads(Path(p).read_text(encoding="utf-8")) for p in paths], args.min_samples)
        if args.baseline:
            add_baseline(result, json.loads(args.baseline.read_text(encoding="utf-8")))
        text = render(result)
        if args.out:
            args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2, allow_nan=False) + "\n", encoding="utf-8")
        if args.text:
            args.text.write_text(text, encoding="utf-8")
        print(text, end="")
        if any(row["failed_samples"] for row in result["rows"]):
            raise SystemExit("benchmark contains failed requests/lookups")
    except (ValueError, KeyError, OSError, TypeError) as error:
        parser.error(str(error))


if __name__ == "__main__":
    main()
