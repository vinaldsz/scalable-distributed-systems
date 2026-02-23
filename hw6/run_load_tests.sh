#!/bin/bash

# Load testing script to identify the breaking point of the product search service
# This script runs progressive load tests and captures metrics

set -e

RESULTS_DIR="load_test_results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"

echo "==================================="
echo "Product Search Service Load Tests"
echo "==================================="
echo "Results will be saved to: $RESULTS_DIR"
echo

# Check if locust is installed
if ! command -v locust &> /dev/null; then
    echo "Installing Locust..."
    pip install locust requests
fi

# Build and start the service locally
echo "Building and starting the service..."
docker-compose up --build -d
sleep 15

echo "Service started. Testing the breaking point..."
echo

# Test phases with increasing user counts
declare -a USER_COUNTS=(10 20 50 100 200)
declare -a SPAWN_RATES=(1 2 5 10 20)

for i in "${!USER_COUNTS[@]}"; do
    USERS=${USER_COUNTS[$i]}
    SPAWN_RATE=${SPAWN_RATES[$i]}
    DURATION="300"  # 5 minutes per test
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test Phase $((i+1)): $USERS users, spawn rate $SPAWN_RATE/sec"
    echo "Duration: ${DURATION}s"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Run locust test
    locust \
        --host=http://localhost:8080 \
        --users=$USERS \
        --spawn-rate=$SPAWN_RATE \
        --run-time=${DURATION}s \
        --headless \
        --csv="$RESULTS_DIR/test_phase_${i}_${USERS}users" \
        2>&1 | tee "$RESULTS_DIR/test_phase_${i}_${USERS}users.log"
    
    # Get current metrics
    echo ""
    echo "Collecting system metrics..."
    
    # Get container stats
    docker stats --no-stream product-search-src-1 >> "$RESULTS_DIR/container_stats_phase_${i}.txt" 2>/dev/null || true
    
    # Wait before next phase
    if [ $i -lt $((${#USER_COUNTS[@]} - 1)) ]; then
        echo "Waiting 30 seconds before next test phase..."
        sleep 30
    fi
    
    echo ""
done

echo ""
echo "==================================="
echo "Load testing complete!"
echo "Results saved to: $RESULTS_DIR"
echo ""
echo "Analysis:"
echo "- Check CSV files for detailed metrics (requests/sec, response times)"
echo "- Check logs for any errors or failures"
echo "- Container stats show CPU and memory usage at each phase"
echo "==================================="

# Stop the service
docker-compose down

# Generate summary report
echo ""
echo "Generating summary report..."
python3 - "$RESULTS_DIR" << 'EOF'
import sys
import os
import json
import csv
from pathlib import Path

results_dir = sys.argv[1]
print("\n=== LOAD TEST SUMMARY ===\n")

# Process each phase
phases = sorted([f for f in os.listdir(results_dir) if f.startswith("test_phase")])
for phase in phases:
    if phase.endswith(".csv"):
        filepath = os.path.join(results_dir, phase)
        with open(filepath) as f:
            reader = csv.DictReader(f)
            rows = list(reader)
            if rows:
                print(f"\n📊 {phase.replace('.csv', '')}:")
                total_requests = len(rows)
                failed = sum(1 for r in rows if r.get('HttpMethod') == 'None')
                avg_response = sum(float(r.get('ResponseTime', 0)) for r in rows) / len(rows) if rows else 0
                print(f"   Total Requests: {total_requests}")
                print(f"   Failed: {failed}")
                print(f"   Avg Response Time: {avg_response:.2f}ms")
EOF

echo "✅ Load testing complete!"
