"""
Generate graphs from load test results.
Usage: python generate_graphs.py
"""

import os
import csv
import matplotlib.pyplot as plt
import numpy as np

RESULTS_DIR = "results"
GRAPHS_DIR = "graphs"
os.makedirs(GRAPHS_DIR, exist_ok=True)

CONFIGS = [
    ("leader_w5r1", "Leader W=5 R=1"),
    ("leader_w1r5", "Leader W=1 R=5"),
    ("leader_w3r3", "Leader W=3 R=3"),
    ("leaderless_w5r1", "Leaderless W=5 R=1"),
]

RATIOS = [
    ("1", "1%w/99%r"),
    ("10", "10%w/90%r"),
    ("50", "50%w/50%r"),
    ("90", "90%w/10%r"),
]


def read_stats_csv(filepath):
    """Read locust stats CSV and return dict keyed by request name."""
    data = {}
    if not os.path.exists(filepath):
        return data
    with open(filepath, newline="") as f:
        reader = csv.DictReader(f)
        for row in reader:
            name = row.get("Name", "")
            if name and name != "Aggregated":
                data[name] = row
    return data


def read_stats_history_csv(filepath):
    """Read locust stats history CSV for time-series data."""
    rows = []
    if not os.path.exists(filepath):
        return rows
    with open(filepath, newline="") as f:
        reader = csv.DictReader(f)
        for row in reader:
            rows.append(row)
    return rows


def read_stale_count(filepath):
    """Read stale read count from file."""
    if not os.path.exists(filepath):
        return 0
    with open(filepath) as f:
        try:
            return int(f.read().strip())
        except ValueError:
            return 0


def get_percentiles_from_stats(stats_row):
    """Extract percentile data from stats CSV row."""
    percentiles = {}
    for key in stats_row:
        if "%" in key or key in ["Min Response Time", "Max Response Time",
                                   "Average Response Time", "Median Response Time"]:
            try:
                percentiles[key] = float(stats_row[key])
            except (ValueError, TypeError):
                pass
    return percentiles


# ---- Graph 1 & 2: Read and Write latency distributions per config ----

def plot_latency_charts():
    """For each config, plot read and write latency percentiles across all 4 ratios."""
    percentile_keys = ["50%", "66%", "75%", "80%", "90%", "95%", "98%", "99%", "99.9%", "100%"]

    for config_key, config_label in CONFIGS:
        fig, axes = plt.subplots(1, 2, figsize=(14, 5))
        fig.suptitle(f"Latency Distribution — {config_label}", fontsize=14)

        for ax, req_name, title in [
            (axes[0], "/get/{key}", "Read Latency"),
            (axes[1], "/set", "Write Latency"),
        ]:
            for ratio_key, ratio_label in RATIOS:
                run_name = f"{config_key}_{ratio_key}write"
                csv_path = os.path.join(RESULTS_DIR, f"{run_name}_stats.csv")
                stats = read_stats_csv(csv_path)

                if req_name not in stats:
                    continue

                row = stats[req_name]
                vals = []
                labels = []
                for pk in percentile_keys:
                    if pk in row:
                        try:
                            vals.append(float(row[pk]))
                            labels.append(pk)
                        except ValueError:
                            pass

                if vals:
                    ax.plot(labels, vals, marker="o", label=ratio_label, linewidth=2)

            ax.set_title(title)
            ax.set_xlabel("Percentile")
            ax.set_ylabel("Response Time (ms)")
            ax.legend()
            ax.grid(True, alpha=0.3)
            ax.tick_params(axis="x", rotation=45)

        plt.tight_layout()
        plt.savefig(os.path.join(GRAPHS_DIR, f"{config_key}_latency.png"), dpi=150)
        plt.close()
        print(f"  Saved {config_key}_latency.png")


# ---- Graph 3: Stale reads bar chart ----

def plot_stale_reads():
    """Bar chart of stale read counts across all 16 combinations."""
    labels = []
    counts = []

    for config_key, config_label in CONFIGS:
        for ratio_key, ratio_label in RATIOS:
            run_name = f"{config_key}_{ratio_key}write"
            stale_path = os.path.join(RESULTS_DIR, f"{run_name}_stale.txt")
            count = read_stale_count(stale_path)
            labels.append(f"{config_label}\n{ratio_label}")
            counts.append(count)

    fig, ax = plt.subplots(figsize=(16, 6))
    colors = []
    config_colors = ["#2196F3", "#FF9800", "#4CAF50", "#E91E63"]
    for i, _ in enumerate(CONFIGS):
        colors.extend([config_colors[i]] * len(RATIOS))

    bars = ax.bar(range(len(counts)), counts, color=colors, edgecolor="white")
    ax.set_xticks(range(len(labels)))
    ax.set_xticklabels(labels, fontsize=7, ha="center")
    ax.set_ylabel("Stale Read Count")
    ax.set_title("Stale Reads Across All Configurations")
    ax.grid(True, alpha=0.3, axis="y")

    # Add count labels on bars
    for bar, count in zip(bars, counts):
        if count > 0:
            ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                    str(count), ha="center", va="bottom", fontsize=8)

    plt.tight_layout()
    plt.savefig(os.path.join(GRAPHS_DIR, "stale_reads.png"), dpi=150)
    plt.close()
    print("  Saved stale_reads.png")


# ---- Graph 4: Average latency comparison (summary) ----

def plot_avg_latency_summary():
    """Grouped bar chart comparing avg read and write latency across configs and ratios."""
    fig, axes = plt.subplots(1, 2, figsize=(16, 6))
    fig.suptitle("Average Latency Comparison", fontsize=14)

    x = np.arange(len(RATIOS))
    width = 0.18

    for ax, req_name, title in [
        (axes[0], "/get/{key}", "Avg Read Latency (ms)"),
        (axes[1], "/set", "Avg Write Latency (ms)"),
    ]:
        for i, (config_key, config_label) in enumerate(CONFIGS):
            avgs = []
            for ratio_key, _ in RATIOS:
                run_name = f"{config_key}_{ratio_key}write"
                csv_path = os.path.join(RESULTS_DIR, f"{run_name}_stats.csv")
                stats = read_stats_csv(csv_path)
                if req_name in stats:
                    try:
                        avgs.append(float(stats[req_name].get("Average Response Time", 0)))
                    except ValueError:
                        avgs.append(0)
                else:
                    avgs.append(0)

            ax.bar(x + i * width, avgs, width, label=config_label)

        ax.set_xlabel("Write/Read Ratio")
        ax.set_ylabel("Avg Response Time (ms)")
        ax.set_title(title)
        ax.set_xticks(x + width * 1.5)
        ax.set_xticklabels([r[1] for r in RATIOS])
        ax.legend(fontsize=8)
        ax.grid(True, alpha=0.3, axis="y")

    plt.tight_layout()
    plt.savefig(os.path.join(GRAPHS_DIR, "avg_latency_summary.png"), dpi=150)
    plt.close()
    print("  Saved avg_latency_summary.png")


# ---- Graph 5: Throughput comparison ----

def plot_throughput():
    """Show requests/sec for reads and writes across all configs."""
    fig, ax = plt.subplots(figsize=(16, 6))

    labels = []
    read_rps = []
    write_rps = []

    for config_key, config_label in CONFIGS:
        for ratio_key, ratio_label in RATIOS:
            run_name = f"{config_key}_{ratio_key}write"
            csv_path = os.path.join(RESULTS_DIR, f"{run_name}_stats.csv")
            stats = read_stats_csv(csv_path)

            labels.append(f"{config_label}\n{ratio_label}")
            try:
                read_rps.append(float(stats.get("/get/{key}", {}).get("Requests/s", 0)))
            except (ValueError, TypeError):
                read_rps.append(0)
            try:
                write_rps.append(float(stats.get("/set", {}).get("Requests/s", 0)))
            except (ValueError, TypeError):
                write_rps.append(0)

    x = np.arange(len(labels))
    ax.bar(x - 0.2, read_rps, 0.4, label="Reads/s", color="#2196F3")
    ax.bar(x + 0.2, write_rps, 0.4, label="Writes/s", color="#FF9800")
    ax.set_xticks(x)
    ax.set_xticklabels(labels, fontsize=7, ha="center")
    ax.set_ylabel("Requests/sec")
    ax.set_title("Throughput: Reads vs Writes")
    ax.legend()
    ax.grid(True, alpha=0.3, axis="y")

    plt.tight_layout()
    plt.savefig(os.path.join(GRAPHS_DIR, "throughput.png"), dpi=150)
    plt.close()
    print("  Saved throughput.png")


def plot_rw_intervals():
    """Histogram of time between a write and the next read on the same key."""
    fig, axes = plt.subplots(2, 2, figsize=(14, 8))
    fig.suptitle("Read-Write Interval Distribution (time between write and next read on same key)", fontsize=13)

    for ax, (config_key, config_label) in zip(axes.flatten(), CONFIGS):
        all_intervals = []
        for ratio_key, ratio_label in RATIOS:
            run_name = f"{config_key}_{ratio_key}write"
            path = os.path.join(RESULTS_DIR, f"{run_name}_intervals.csv")
            if not os.path.exists(path):
                continue
            with open(path) as f:
                next(f)  # skip header
                intervals = [float(line.strip()) for line in f if line.strip()]
            if intervals:
                ax.hist(intervals, bins=50, alpha=0.6, label=ratio_label)
                all_intervals.extend(intervals)

        ax.set_title(config_label)
        ax.set_xlabel("Interval (ms)")
        ax.set_ylabel("Count")
        ax.legend(fontsize=8)
        ax.grid(True, alpha=0.3)

    plt.tight_layout()
    plt.savefig(os.path.join(GRAPHS_DIR, "rw_intervals.png"), dpi=150)
    plt.close()
    print("  Saved rw_intervals.png")


if __name__ == "__main__":
    print("Generating graphs...")
    plot_latency_charts()
    plot_stale_reads()
    plot_avg_latency_summary()
    plot_throughput()
    plot_rw_intervals()
    print("Done! Graphs saved to graphs/")
