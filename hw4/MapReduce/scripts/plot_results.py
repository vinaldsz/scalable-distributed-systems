import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

# Read results
df = pd.read_csv('benchmark_results.csv')

# Create figure with subplots
fig, axes = plt.subplots(2, 2, figsize=(15, 12))
fig.suptitle('MapReduce Performance Analysis (Max 3 Mappers)', fontsize=16, fontweight='bold')

# Plot 1: Total Time vs Number of Mappers (10MB file)
exp1_data = df[df['File_Size_MB'] == 10].sort_values('Num_Mappers')
ax1 = axes[0, 0]
ax1.plot(exp1_data['Num_Mappers'], exp1_data['Total_Time_Seconds'], 
         marker='o', linewidth=2, markersize=10, color='#2E86AB')
ax1.set_xlabel('Number of Mappers', fontsize=12)
ax1.set_ylabel('Total Time (seconds)', fontsize=12)
ax1.set_title('Scaling with Number of Mappers\n(10MB Input File)', fontsize=13)
ax1.grid(True, alpha=0.3)
ax1.set_xticks([1, 2, 3])

# Add value labels on points
for idx, row in exp1_data.iterrows():
    ax1.annotate(f'{row["Total_Time_Seconds"]:.1f}s', 
                xy=(row['Num_Mappers'], row['Total_Time_Seconds']),
                xytext=(0, 10), textcoords='offset points',
                ha='center', fontsize=9)

# Plot 2: Speedup vs Number of Mappers
if len(exp1_data) > 0 and exp1_data['Num_Mappers'].min() == 1:
    baseline_time = exp1_data[exp1_data['Num_Mappers'] == 1]['Total_Time_Seconds'].values[0]
    speedup = baseline_time / exp1_data['Total_Time_Seconds']
    ideal_speedup = exp1_data['Num_Mappers']
    efficiency = (speedup / exp1_data['Num_Mappers']) * 100
    
    ax2 = axes[0, 1]
    ax2.plot(exp1_data['Num_Mappers'], speedup, 
             marker='o', linewidth=2, markersize=10, label='Actual Speedup', color='#A23B72')
    ax2.plot(exp1_data['Num_Mappers'], ideal_speedup, 
             linestyle='--', linewidth=2, label='Ideal Speedup', color='#F18F01', alpha=0.7)
    ax2.set_xlabel('Number of Mappers', fontsize=12)
    ax2.set_ylabel('Speedup Factor', fontsize=12)
    ax2.set_title('Speedup Analysis\n(Actual vs Ideal)', fontsize=13)
    ax2.legend(fontsize=10)
    ax2.grid(True, alpha=0.3)
    ax2.set_xticks([1, 2, 3])
    
    # Add efficiency annotations
    for idx, row in exp1_data.iterrows():
        if row['Num_Mappers'] > 1:
            eff = efficiency.loc[idx]
            sp = speedup.loc[idx]
            ax2.annotate(f'{sp:.2f}x\n({eff:.0f}%)', 
                        xy=(row['Num_Mappers'], sp),
                        xytext=(0, 10), textcoords='offset points',
                        ha='center', fontsize=8)

# Plot 3: Total Time vs File Size (3 mappers)
exp2_data = df[df['Num_Mappers'] == 3].sort_values('File_Size_MB')
ax3 = axes[1, 0]
ax3.plot(exp2_data['File_Size_MB'], exp2_data['Total_Time_Seconds'], 
         marker='s', linewidth=2, markersize=10, color='#6A994E')
ax3.set_xlabel('File Size (MB)', fontsize=12)
ax3.set_ylabel('Total Time (seconds)', fontsize=12)
ax3.set_title('Scaling with Input Size\n(3 Mappers)', fontsize=13)
ax3.grid(True, alpha=0.3)

# Add throughput labels
for idx, row in exp2_data.iterrows():
    throughput = row['File_Size_MB'] / row['Total_Time_Seconds']
    ax3.annotate(f'{throughput:.2f} MB/s', 
                xy=(row['File_Size_MB'], row['Total_Time_Seconds']),
                xytext=(0, -15), textcoords='offset points',
                ha='center', fontsize=8)

# Plot 4: Time Breakdown (Stacked Bar)
ax4 = axes[1, 1]
mappers = exp1_data['Num_Mappers']
split_times = exp1_data['Split_Time']
map_times = exp1_data['Map_Time']
reduce_times = exp1_data['Reduce_Time']

width = 0.5
x = np.arange(len(mappers))

p1 = ax4.bar(x, split_times, width, label='Split', color='#E63946')
p2 = ax4.bar(x, map_times, width, bottom=split_times, label='Map', color='#F1C453')
p3 = ax4.bar(x, reduce_times, width, 
             bottom=split_times+map_times, label='Reduce', color='#457B9D')

ax4.set_xlabel('Number of Mappers', fontsize=12)
ax4.set_ylabel('Time (seconds)', fontsize=12)
ax4.set_title('Time Breakdown by Phase\n(10MB Input)', fontsize=13)
ax4.set_xticks(x)
ax4.set_xticklabels([str(int(m)) for m in mappers])
ax4.legend(fontsize=10)
ax4.grid(True, alpha=0.3, axis='y')

# Add percentage labels on bars
for i, (s, m, r) in enumerate(zip(split_times, map_times, reduce_times)):
    total = s + m + r
    if s/total > 0.1:
        ax4.text(i, s/2, f'{s/total*100:.0f}%', ha='center', va='center', fontsize=9)
    if m/total > 0.1:
        ax4.text(i, s+m/2, f'{m/total*100:.0f}%', ha='center', va='center', fontsize=9)
    if r/total > 0.1:
        ax4.text(i, s+m+r/2, f'{r/total*100:.0f}%', ha='center', va='center', fontsize=9)

plt.tight_layout()
plt.savefig('mapreduce_performance.png', dpi=300, bbox_inches='tight')
print("✓ Plot saved as 'mapreduce_performance.png'")

# Print summary statistics
print("\n" + "="*50)
print("PERFORMANCE SUMMARY")
print("="*50)

if len(exp1_data) > 0 and exp1_data['Num_Mappers'].min() == 1:
    max_speedup = speedup.max()
    max_speedup_mappers = exp1_data.loc[speedup.idxmax(), 'Num_Mappers']
    max_efficiency = (max_speedup / max_speedup_mappers) * 100
    
    print(f"\nScalability with Mappers:")
    print(f"  Best speedup: {max_speedup:.2f}x with {int(max_speedup_mappers)} mappers")
    print(f"  Efficiency: {max_efficiency:.1f}%")
    print(f"  Baseline (1 mapper): {baseline_time:.2f}s")
    print(f"  Best time (3 mappers): {exp1_data['Total_Time_Seconds'].min():.2f}s")

print(f"\nThroughput (3 mappers):")
for _, row in exp2_data.iterrows():
    throughput = row['File_Size_MB'] / row['Total_Time_Seconds']
    print(f"  {int(row['File_Size_MB'])}MB: {throughput:.2f} MB/s ({row['Total_Time_Seconds']:.2f}s)")

print(f"\nPhase Distribution (average across all experiments):")
avg_split = (df['Split_Time'] / df['Total_Time_Seconds'] * 100).mean()
avg_map = (df['Map_Time'] / df['Total_Time_Seconds'] * 100).mean()
avg_reduce = (df['Reduce_Time'] / df['Total_Time_Seconds'] * 100).mean()
print(f"  Split: {avg_split:.1f}%")
print(f"  Map: {avg_map:.1f}%")
print(f"  Reduce: {avg_reduce:.1f}%")

print("="*50)