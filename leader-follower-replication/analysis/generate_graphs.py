#!/usr/bin/env python3
"""
Generate all required graphs from load test CSVs.

Input:  analysis/results/{config}_w{pct}.csv
Output: analysis/graphs/*.png

Run after load tests:
    python3 analysis/generate_graphs.py
"""

import os
import glob
import numpy as np
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker

RESULTS_DIR = "analysis/results"
GRAPHS_DIR = "analysis/graphs"
os.makedirs(GRAPHS_DIR, exist_ok=True)

CONFIGS = ["lf1", "lf2", "lf3", "ll"]
CONFIG_LABELS = {
    "lf1": "LF W=5,R=1",
    "lf2": "LF W=1,R=5",
    "lf3": "LF W=3,R=3",
    "ll":  "Leaderless W=5,R=1",
}
WRITE_PCTS = [1, 10, 50, 90]


def load_all() -> dict[str, pd.DataFrame]:
    """Load every CSV into a dict keyed by 'config_w{pct}'."""
    data = {}
    for cfg in CONFIGS:
        for wpct in WRITE_PCTS:
            path = os.path.join(RESULTS_DIR, f"{cfg}_w{wpct}.csv")
            if os.path.exists(path):
                df = pd.read_csv(path)
                data[f"{cfg}_w{wpct}"] = df
    return data


# ─── Graph 1: Latency CDF (reads vs writes) per config ────────────────────────

def graph_latency_cdf(data: dict[str, pd.DataFrame]):
    fig, axes = plt.subplots(2, 2, figsize=(14, 10))
    axes = axes.flatten()
    fig.suptitle("Latency CDF — Reads vs Writes", fontsize=14)

    for i, cfg in enumerate(CONFIGS):
        ax = axes[i]
        # Combine all write-% runs for this config to show full distribution.
        dfs = [data[k] for k in data if k.startswith(cfg + "_")]
        if not dfs:
            ax.set_title(f"{CONFIG_LABELS[cfg]} (no data)")
            continue
        combined = pd.concat(dfs)

        for op, color in [("read", "steelblue"), ("write", "tomato")]:
            sub = combined[combined["op"] == op]["latency_ms"].dropna()
            if sub.empty:
                continue
            sorted_vals = np.sort(sub)
            cdf = np.arange(1, len(sorted_vals) + 1) / len(sorted_vals)
            ax.plot(sorted_vals, cdf, label=op, color=color)

        ax.set_title(CONFIG_LABELS[cfg])
        ax.set_xlabel("Latency (ms)")
        ax.set_ylabel("Cumulative fraction")
        ax.legend()
        ax.grid(True, alpha=0.3)
        ax.set_xlim(left=0)

    plt.tight_layout()
    out = os.path.join(GRAPHS_DIR, "1_latency_cdf.png")
    plt.savefig(out, dpi=150)
    plt.close()
    print(f"Saved {out}")


# ─── Graph 2: P50/P95/P99 bar chart per config ────────────────────────────────

def graph_percentile_bars(data: dict[str, pd.DataFrame]):
    percentiles = [50, 95, 99]
    ops = ["read", "write"]
    x = np.arange(len(CONFIGS))
    width = 0.12

    fig, axes = plt.subplots(1, 2, figsize=(14, 6))
    fig.suptitle("Latency Percentiles by Config", fontsize=14)

    for oi, op in enumerate(ops):
        ax = axes[oi]
        for pi, pct in enumerate(percentiles):
            vals = []
            for cfg in CONFIGS:
                dfs = [data[k] for k in data if k.startswith(cfg + "_")]
                if not dfs:
                    vals.append(0)
                    continue
                combined = pd.concat(dfs)
                sub = combined[combined["op"] == op]["latency_ms"].dropna()
                vals.append(np.percentile(sub, pct) if not sub.empty else 0)

            offset = (pi - 1) * width
            bars = ax.bar(x + offset, vals, width, label=f"P{pct}")
            ax.bar_label(bars, fmt="%.0f", fontsize=7, padding=2)

        ax.set_title(f"{op.capitalize()} Latency")
        ax.set_xlabel("Config")
        ax.set_ylabel("Latency (ms)")
        ax.set_xticks(x)
        ax.set_xticklabels([CONFIG_LABELS[c] for c in CONFIGS], rotation=15, ha="right")
        ax.legend()
        ax.grid(True, axis="y", alpha=0.3)

    plt.tight_layout()
    out = os.path.join(GRAPHS_DIR, "2_percentile_bars.png")
    plt.savefig(out, dpi=150)
    plt.close()
    print(f"Saved {out}")


# ─── Graph 3: Stale read rate by config × write% ──────────────────────────────

def graph_stale_reads(data: dict[str, pd.DataFrame]):
    x = np.arange(len(CONFIGS))
    width = 0.18
    colors = ["#4c72b0", "#dd8452", "#55a868", "#c44e52"]

    fig, ax = plt.subplots(figsize=(12, 6))
    ax.set_title("Stale Read Rate by Config and Write%", fontsize=14)

    for pi, wpct in enumerate(WRITE_PCTS):
        rates = []
        for cfg in CONFIGS:
            key = f"{cfg}_w{wpct}"
            if key not in data:
                rates.append(0)
                continue
            df = data[key]
            reads = df[df["op"] == "read"]
            if reads.empty:
                rates.append(0)
            else:
                rates.append(reads["stale"].mean() * 100)

        offset = (pi - 1.5) * width
        bars = ax.bar(x + offset, rates, width, label=f"write%={wpct}%", color=colors[pi])
        ax.bar_label(bars, fmt="%.1f%%", fontsize=7, padding=2)

    ax.set_xlabel("Config")
    ax.set_ylabel("Stale reads (%)")
    ax.set_xticks(x)
    ax.set_xticklabels([CONFIG_LABELS[c] for c in CONFIGS], rotation=15, ha="right")
    ax.legend()
    ax.grid(True, axis="y", alpha=0.3)
    ax.yaxis.set_major_formatter(mticker.PercentFormatter())

    plt.tight_layout()
    out = os.path.join(GRAPHS_DIR, "3_stale_reads.png")
    plt.savefig(out, dpi=150)
    plt.close()
    print(f"Saved {out}")


# ─── Graph 4: Write-to-read interval histogram ────────────────────────────────

def graph_write_to_read_intervals(data: dict[str, pd.DataFrame]):
    fig, axes = plt.subplots(2, 2, figsize=(14, 10))
    axes = axes.flatten()
    fig.suptitle("Time Between Write and Read of Same Key (ms)", fontsize=14)

    for i, cfg in enumerate(CONFIGS):
        ax = axes[i]
        dfs = [data[k] for k in data if k.startswith(cfg + "_")]
        if not dfs:
            ax.set_title(f"{CONFIG_LABELS[cfg]} (no data)")
            continue
        combined = pd.concat(dfs)
        intervals = combined[
            (combined["op"] == "read") & (combined["write_to_read_ms"] >= 0)
        ]["write_to_read_ms"].dropna()

        if intervals.empty:
            ax.set_title(f"{CONFIG_LABELS[cfg]} (no intervals)")
            continue

        ax.hist(intervals, bins=50, color="steelblue", edgecolor="white", alpha=0.8)
        ax.axvline(intervals.median(), color="red", linestyle="--", label=f"median {intervals.median():.0f}ms")
        ax.set_title(CONFIG_LABELS[cfg])
        ax.set_xlabel("Write-to-read interval (ms)")
        ax.set_ylabel("Count")
        ax.legend()
        ax.grid(True, alpha=0.3)

    plt.tight_layout()
    out = os.path.join(GRAPHS_DIR, "4_write_to_read_intervals.png")
    plt.savefig(out, dpi=150)
    plt.close()
    print(f"Saved {out}")


# ─── Graph 5: Throughput vs write% ───────────────────────────────────────────

def graph_throughput(data: dict[str, pd.DataFrame]):
    fig, ax = plt.subplots(figsize=(10, 6))
    ax.set_title("Throughput (req/s) vs Write%", fontsize=14)

    DURATION_S = 60  # assumed test duration in seconds

    for cfg in CONFIGS:
        tputs = []
        for wpct in WRITE_PCTS:
            key = f"{cfg}_w{wpct}"
            if key not in data:
                tputs.append(0)
            else:
                tputs.append(len(data[key]) / DURATION_S)

        ax.plot(WRITE_PCTS, tputs, marker="o", label=CONFIG_LABELS[cfg])

    ax.set_xlabel("Write percentage (%)")
    ax.set_ylabel("Requests per second")
    ax.set_xticks(WRITE_PCTS)
    ax.set_xticklabels([f"{p}%" for p in WRITE_PCTS])
    ax.legend()
    ax.grid(True, alpha=0.3)

    plt.tight_layout()
    out = os.path.join(GRAPHS_DIR, "5_throughput.png")
    plt.savefig(out, dpi=150)
    plt.close()
    print(f"Saved {out}")


if __name__ == "__main__":
    data = load_all()
    if not data:
        print(f"No CSV files found in {RESULTS_DIR}. Run load tests first.")
        raise SystemExit(1)

    graph_latency_cdf(data)
    graph_percentile_bars(data)
    graph_stale_reads(data)
    graph_write_to_read_intervals(data)
    graph_throughput(data)

    print(f"\nAll graphs written to {GRAPHS_DIR}/")
