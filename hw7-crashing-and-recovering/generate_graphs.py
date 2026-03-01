#!/usr/bin/env python3
"""Generate visualization graphs from Locust load test metrics.

Usage:
  python3 generate_graphs.py                # Auto-detect and use latest profile
  python3 generate_graphs.py 50users-3m    # Use specific profile
  python3 generate_graphs.py all            # Generate for all available profiles
"""

import matplotlib
matplotlib.use('Agg')  # Use non-interactive backend for headless environments
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
from pathlib import Path
import numpy as np
import sys
import glob

def find_profile_csvs(profile_dir):
    """Search for broken and fixed stats CSVs in the profile directory."""
    broken_csv = None
    fixed_csv = None
    broken_history = None
    fixed_history = None
    
    stats_files = glob.glob(str(profile_dir / '*_stats.csv'))
    history_files = glob.glob(str(profile_dir / '*_stats_history.csv'))
    
    for f in stats_files:
        if 'broken' in f:
            broken_csv = f
        elif 'fixed' in f:
            fixed_csv = f
    
    for f in history_files:
        if 'broken' in f:
            broken_history = f
        elif 'fixed' in f:
            fixed_history = f
    
    return broken_csv, fixed_csv, broken_history, fixed_history

def validate_files(broken_stats, fixed_stats, broken_history, fixed_history):
    """Validate that all required files exist."""
    required = [broken_stats, fixed_stats, broken_history, fixed_history]
    missing = [f for f in required if not f or not Path(f).exists()]
    
    if missing:
        print(f"❌ Error: Missing files:")
        for f in missing:
            print(f"   - {f}")
        return False
    return True

def aggregate_search_metrics(stats_df):
    """Aggregate metrics for search endpoints only (exclude health/metrics)."""
    # Filter for search endpoints
    search_rows = stats_df[stats_df['Name'].str.contains('/products/search', na=False)]
    
    if len(search_rows) == 0:
        print("⚠️  Warning: No search endpoints found, using aggregated row")
        # Fallback to aggregated row (last row with empty Type)
        agg_row = stats_df[stats_df['Type'].isna() | (stats_df['Type'] == '')]
        if len(agg_row) > 0:
            return agg_row.iloc[0]
        else:
            print("❌ Error: No aggregated row found either")
            return stats_df.iloc[0]  # Last resort: use first row
    
    # Aggregate search metrics
    total_requests = search_rows['Request Count'].sum()
    total_failures = search_rows['Failure Count'].sum()
    
    # Weighted average for response times (by request count)
    weighted_avg = (search_rows['Average Response Time'] * search_rows['Request Count']).sum() / total_requests
    
    # For percentiles, we'll use the median across all search endpoints
    # (This is an approximation since we can't recalculate true percentiles from aggregated data)
    median_p50 = search_rows['50%'].median()
    median_p95 = search_rows['95%'].median()
    median_p99 = search_rows['99%'].median()
    
    # Min/Max are straightforward
    min_time = search_rows['Min Response Time'].min()
    max_time = search_rows['Max Response Time'].max()
    
    # Total RPS
    total_rps = search_rows['Requests/s'].sum()
    
    # Create aggregated series
    agg_metrics = pd.Series({
        'Request Count': total_requests,
        'Failure Count': total_failures,
        'Average Response Time': weighted_avg,
        '50%': median_p50,
        '95%': median_p95,
        '99%': median_p99,
        'Min Response Time': min_time,
        'Max Response Time': max_time,
        'Requests/s': total_rps
    })
    
    return agg_metrics

def generate_graphs(profile_name, metrics_dir):
    """Generate all comparison graphs for a given profile."""
    profile_dir = metrics_dir / profile_name
    
    if not profile_dir.exists():
        print(f"❌ Profile directory not found: {profile_dir}")
        return False
    
    # Find CSV files
    broken_stats_file, fixed_stats_file, broken_history_file, fixed_history_file = find_profile_csvs(profile_dir)
    
    if not validate_files(broken_stats_file, fixed_stats_file, broken_history_file, fixed_history_file):
        return False
    
    print(f"\n📊 Generating graphs for profile: {profile_name}")
    print(f"   Broken:  {Path(broken_stats_file).name}")
    print(f"   Fixed:   {Path(fixed_stats_file).name}")
    
    # Read metrics
    broken_stats = pd.read_csv(broken_stats_file)
    fixed_stats = pd.read_csv(fixed_stats_file)
    broken_history = pd.read_csv(broken_history_file)
    fixed_history = pd.read_csv(fixed_history_file)
    
    # Aggregate search-only metrics
    print(f"   Aggregating search endpoint metrics...")
    broken_metrics = aggregate_search_metrics(broken_stats)
    fixed_metrics = aggregate_search_metrics(fixed_stats)
    
    # Set up style
    plt.style.use('seaborn-v0_8-darkgrid')
    colors = {'broken': '#d62728', 'fixed': '#2ca02c'}
    
    # --- GRAPH 1: Latency Percentiles Comparison ---
    fig, ax = plt.subplots(figsize=(10, 6))
    percentiles = ['50%ile', '95%ile', '99%ile', 'Max']
    broken_latencies = [
        broken_metrics['50%'],
        broken_metrics['95%'],
        broken_metrics['99%'],
        broken_metrics['Max Response Time']
    ]
    fixed_latencies = [
        fixed_metrics['50%'],
        fixed_metrics['95%'],
        fixed_metrics['99%'],
        fixed_metrics['Max Response Time']
    ]
    
    x = np.arange(len(percentiles))
    width = 0.35
    
    bars1 = ax.bar(x - width/2, broken_latencies, width, label='Broken', color=colors['broken'], alpha=0.8)
    bars2 = ax.bar(x + width/2, fixed_latencies, width, label='Fixed', color=colors['fixed'], alpha=0.8)
    
    ax.set_xlabel('Percentile', fontsize=12, fontweight='bold')
    ax.set_ylabel('Response Time (ms)', fontsize=12, fontweight='bold')
    ax.set_title('Search Response Latency Comparison: Broken vs Fixed', fontsize=14, fontweight='bold')
    ax.set_xticks(x)
    ax.set_xticklabels(percentiles)
    ax.legend(fontsize=11)
    ax.grid(axis='y', alpha=0.3)
    
    # Add value labels on bars
    for bars in [bars1, bars2]:
        for bar in bars:
            height = bar.get_height()
            ax.text(bar.get_x() + bar.get_width()/2., height,
                    f'{int(height)}ms',
                    ha='center', va='bottom', fontsize=9)
    
    plt.tight_layout()
    plt.savefig('latency_comparison.png', dpi=150, bbox_inches='tight')
    print("✓ Saved: latency_comparison.png")
    plt.close()
    
    # --- GRAPH 2: Throughput Over Time ---
    fig, ax = plt.subplots(figsize=(12, 6))
    
    broken_history['TimeDate'] = pd.to_datetime(broken_history['Timestamp'], unit='s')
    fixed_history['TimeDate'] = pd.to_datetime(fixed_history['Timestamp'], unit='s')
    
    broken_history['Time_offset'] = (broken_history['TimeDate'] - broken_history['TimeDate'].min()).dt.total_seconds()
    fixed_history['Time_offset'] = (fixed_history['TimeDate'] - fixed_history['TimeDate'].min()).dt.total_seconds()
    
    ax.plot(broken_history['Time_offset'], broken_history['Requests/s'], 
            marker='o', linestyle='-', linewidth=2, markersize=4, label='Broken', color=colors['broken'], alpha=0.8)
    ax.plot(fixed_history['Time_offset'], fixed_history['Requests/s'], 
            marker='s', linestyle='-', linewidth=2, markersize=4, label='Fixed', color=colors['fixed'], alpha=0.8)
    
    ax.set_xlabel('Time (seconds)', fontsize=12, fontweight='bold')
    ax.set_ylabel('Requests per Second', fontsize=12, fontweight='bold')
    ax.set_title('Throughput Over Time: Broken vs Fixed', fontsize=14, fontweight='bold')
    ax.legend(fontsize=11)
    ax.grid(True, alpha=0.3)
    
    plt.tight_layout()
    plt.savefig('throughput_timeline.png', dpi=150, bbox_inches='tight')
    print("✓ Saved: throughput_timeline.png")
    plt.close()
    
    # --- GRAPH 3: Aggregate Metrics Comparison ---
    fig, ((ax1, ax2), (ax3, ax4)) = plt.subplots(2, 2, figsize=(14, 10))
    
    # Total Requests
    requests = [broken_metrics['Request Count'], fixed_metrics['Request Count']]
    ax1.bar(['Broken', 'Fixed'], requests, color=[colors['broken'], colors['fixed']], alpha=0.8)
    ax1.set_ylabel('Total Requests', fontsize=11, fontweight='bold')
    ax1.set_title('Total Search Requests', fontsize=12, fontweight='bold')
    ax1.grid(axis='y', alpha=0.3)
    for i, v in enumerate(requests):
        ax1.text(i, v, f'{int(v):,}', ha='center', va='bottom', fontsize=10, fontweight='bold')
    
    # Average Response Time
    avg_times = [broken_metrics['Average Response Time'], fixed_metrics['Average Response Time']]
    ax2.bar(['Broken', 'Fixed'], avg_times, color=[colors['broken'], colors['fixed']], alpha=0.8)
    ax2.set_ylabel('Avg Response Time (ms)', fontsize=11, fontweight='bold')
    ax2.set_title('Average Search Response Time', fontsize=12, fontweight='bold')
    ax2.grid(axis='y', alpha=0.3)
    for i, v in enumerate(avg_times):
        ax2.text(i, v, f'{int(v)}ms', ha='center', va='bottom', fontsize=10, fontweight='bold')
    
    # Min Response Time
    min_times = [broken_metrics['Min Response Time'], fixed_metrics['Min Response Time']]
    ax3.bar(['Broken', 'Fixed'], min_times, color=[colors['broken'], colors['fixed']], alpha=0.8)
    ax3.set_ylabel('Min Response Time (ms)', fontsize=11, fontweight='bold')
    ax3.set_title('Min Search Response Time', fontsize=12, fontweight='bold')
    ax3.grid(axis='y', alpha=0.3)
    for i, v in enumerate(min_times):
        ax3.text(i, v, f'{int(v)}ms', ha='center', va='bottom', fontsize=10, fontweight='bold')
    
    # Request Rate
    rates = [broken_metrics['Requests/s'], fixed_metrics['Requests/s']]
    ax4.bar(['Broken', 'Fixed'], rates, color=[colors['broken'], colors['fixed']], alpha=0.8)
    ax4.set_ylabel('Requests per Second', fontsize=11, fontweight='bold')
    ax4.set_title('Average Search Request Rate', fontsize=12, fontweight='bold')
    ax4.grid(axis='y', alpha=0.3)
    for i, v in enumerate(rates):
        ax4.text(i, v, f'{v:.1f}', ha='center', va='bottom', fontsize=10, fontweight='bold')
    
    plt.tight_layout()
    plt.savefig('aggregate_metrics.png', dpi=150, bbox_inches='tight')
    print("✓ Saved: aggregate_metrics.png")
    plt.close()
    
    # --- GRAPH 4: Improvement Ratio Summary ---
    fig, ax = plt.subplots(figsize=(10, 6))
    
    improvements = {
        'P95 Latency': (broken_metrics['95%'] / fixed_metrics['95%']),
        'P99 Latency': (broken_metrics['99%'] / fixed_metrics['99%']),
        'Max Latency': (broken_metrics['Max Response Time'] / fixed_metrics['Max Response Time']),
        'Throughput': (fixed_metrics['Requests/s'] / broken_metrics['Requests/s'])
    }
    
    bars = ax.barh(list(improvements.keys()), list(improvements.values()), color='#1f77b4', alpha=0.8)
    ax.set_xlabel('Improvement Factor (↑ Better)', fontsize=12, fontweight='bold')
    ax.set_title('Bulkhead Pattern Effectiveness', fontsize=14, fontweight='bold')
    ax.axvline(x=1, color='red', linestyle='--', linewidth=2, alpha=0.5, label='No Improvement')
    ax.grid(axis='x', alpha=0.3)
    
    # Add value labels
    for i, (bar, val) in enumerate(zip(bars, improvements.values())):
        ax.text(val + 0.1, bar.get_y() + bar.get_height()/2., f'{val:.2f}x',
                va='center', fontsize=11, fontweight='bold')
    
    ax.legend(fontsize=10)
    plt.tight_layout()
    plt.savefig('improvement_ratios.png', dpi=150, bbox_inches='tight')
    print("✓ Saved: improvement_ratios.png")
    plt.close()
    
    # --- GRAPH 5: Failure Rate Comparison ---
    fig, ax = plt.subplots(figsize=(10, 6))
    
    broken_failure_rate = (broken_metrics['Failure Count'] / broken_metrics['Request Count']) * 100
    fixed_failure_rate = (fixed_metrics['Failure Count'] / fixed_metrics['Request Count']) * 100
    
    failure_rates = [broken_failure_rate, fixed_failure_rate]
    bars = ax.bar(['Broken', 'Fixed'], failure_rates, color=[colors['broken'], colors['fixed']], alpha=0.8)
    
    ax.set_ylabel('Failure Rate (%)', fontsize=12, fontweight='bold')
    ax.set_title('Request Failure Rate Comparison', fontsize=14, fontweight='bold')
    ax.set_ylim(0, max(failure_rates) * 1.2)
    ax.grid(axis='y', alpha=0.3)
    
    # Add value labels
    for bar, val in zip(bars, failure_rates):
        ax.text(bar.get_x() + bar.get_width()/2., val,
                f'{val:.3f}%',
                ha='center', va='bottom', fontsize=10, fontweight='bold')
    
    plt.tight_layout()
    plt.savefig('failure_rate.png', dpi=150, bbox_inches='tight')
    print("✓ Saved: failure_rate.png")
    plt.close()
    
    print(f"✓ Graph generation complete for {profile_name}\n")
    return True

def list_available_profiles(metrics_dir):
    """List all available test profiles."""
    if not metrics_dir.exists():
        print(f"❌ Metrics directory not found: {metrics_dir}")
        return []
    
    profiles = [d.name for d in metrics_dir.iterdir() if d.is_dir()]
    return sorted(profiles)

def main():
    metrics_dir = Path('metrics')
    
    # Determine which profile(s) to generate
    if len(sys.argv) > 1:
        arg = sys.argv[1]
        if arg.lower() == 'all':
            profiles = list_available_profiles(metrics_dir)
            if not profiles:
                print("❌ No profiles found in metrics/")
                return 1
            print(f"Found {len(profiles)} profile(s): {', '.join(profiles)}")
        else:
            profiles = [arg]
    else:
        # Auto-detect latest profile
        available = list_available_profiles(metrics_dir)
        if not available:
            print("❌ No profiles found in metrics/")
            return 1
        profiles = [available[-1]]  # Use latest alphabetically
        print(f"Auto-detected profile: {profiles[0]}")
    
    # Generate graphs for each profile
    success_count = 0
    for profile in profiles:
        if generate_graphs(profile, metrics_dir):
            success_count += 1
    
    print("="*50)
    print(f"📊 Successfully generated graphs for {success_count}/{len(profiles)} profile(s)")
    print("="*50)
    print("\n💡 Move graphs to their profile folder:")
    for profile in profiles:
        print(f"   mkdir -p images/{profile}")
        print(f"   mv -f *.png images/{profile}/")
    
    return 0 if success_count == len(profiles) else 1

if __name__ == '__main__':
    sys.exit(main())

