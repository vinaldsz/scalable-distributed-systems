import argparse
import re
from pathlib import Path
from typing import Dict, List

import matplotlib.pyplot as plt

AGG_RE = re.compile(
    r"^\s*Aggregated\s+(\d+)\s+(\d+)\(([-\d.]+)%\)\s+\|\s+"
    r"(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+\|\s+([\d.]+)\s+([\d.]+)"
)


def parse_aggregated_rows(log_text: str) -> List[Dict[str, float]]:
    rows = []
    for line in log_text.splitlines():
        match = AGG_RE.match(line)
        if not match:
            continue
        rows.append(
            {
                "reqs": int(match.group(1)),
                "fails": int(match.group(2)),
                "fail_pct": float(match.group(3)),
                "avg_ms": int(match.group(4)),
                "min_ms": int(match.group(5)),
                "max_ms": int(match.group(6)),
                "med_ms": int(match.group(7)),
                "rps": float(match.group(8)),
                "fps": float(match.group(9)),
            }
        )
    return rows


def plot_series(rows: List[Dict[str, float]], title: str, outdir: Path, interval_sec: float) -> None:
    if not rows:
        return

    x_vals = [i * interval_sec for i in range(len(rows))]
    rps = [row["rps"] for row in rows]
    avg = [row["avg_ms"] for row in rows]
    fail_pct = [row["fail_pct"] for row in rows]

    fig, axes = plt.subplots(3, 1, figsize=(10, 10), sharex=True)

    axes[0].plot(x_vals, rps, color="#2c7fb8")
    axes[0].set_ylabel("Req/s")

    axes[1].plot(x_vals, avg, color="#f03b20")
    axes[1].set_ylabel("Avg latency (ms)")

    axes[2].plot(x_vals, fail_pct, color="#31a354")
    axes[2].set_ylabel("Fail %")
    axes[2].set_xlabel("Time (s)")

    fig.suptitle(title)
    fig.tight_layout(rect=[0, 0, 1, 0.96])
    fig.savefig(outdir / f"{title}_series.png", dpi=150)
    plt.close(fig)


def plot_summary(files_data: Dict[str, List[Dict[str, float]]], outdir: Path) -> None:
    names = []
    final_rps = []
    final_avg = []
    final_fail = []

    for name, rows in files_data.items():
        if not rows:
            continue
        last = rows[-1]
        names.append(name)
        final_rps.append(last["rps"])
        final_avg.append(last["avg_ms"])
        final_fail.append(last["fail_pct"])

    if not names:
        return

    fig, axes = plt.subplots(1, 3, figsize=(14, 4))

    axes[0].bar(names, final_rps, color="#2c7fb8")
    axes[0].set_title("Final Req/s")

    axes[1].bar(names, final_avg, color="#f03b20")
    axes[1].set_title("Final Avg Latency (ms)")

    axes[2].bar(names, final_fail, color="#31a354")
    axes[2].set_title("Final Fail %")

    for axis in axes:
        axis.tick_params(axis="x", rotation=35, labelsize=8)

    fig.tight_layout()
    fig.savefig(outdir / "summary_comparison.png", dpi=150)
    plt.close(fig)


def main() -> None:
    parser = argparse.ArgumentParser(description="Plot Locust log metrics")
    parser.add_argument(
        "--dir",
        default=".",
        help="Directory containing log files",
    )
    parser.add_argument(
        "--files",
        nargs="*",
        default=[
            "baseline_test.log",
            "breaking_point_test.log",
            "part3_baseline_test.log",
            "part3_breaking_point_test.log",
            "part3_extreme_breaking_point_test.log",
        ],
        help="Specific log files to parse",
    )
    parser.add_argument(
        "--interval",
        type=float,
        default=2.0,
        help="Assumed stats print interval (seconds)",
    )
    parser.add_argument(
        "--out",
        default="graphs",
        help="Output directory for graphs",
    )
    args = parser.parse_args()

    log_dir = Path(args.dir)
    outdir = Path(args.out)
    outdir.mkdir(parents=True, exist_ok=True)

    files_data: Dict[str, List[Dict[str, float]]] = {}
    for fname in args.files:
        fpath = log_dir / fname
        if not fpath.exists():
            print(f"Skipping missing: {fpath}")
            continue
        rows = parse_aggregated_rows(fpath.read_text())
        title = fpath.stem
        files_data[title] = rows
        plot_series(rows, title, outdir, args.interval)

    plot_summary(files_data, outdir)
    print(f"Graphs saved to: {outdir.resolve()}")


if __name__ == "__main__":
    main()
