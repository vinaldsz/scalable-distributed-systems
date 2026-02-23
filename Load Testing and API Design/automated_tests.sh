#!/bin/bash

# Automated Load Testing Script with Results Capture
# Runs all three scenarios and exports CSV results automatically

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Create results directory with timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_DIR="test_results_${TIMESTAMP}"
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}"
echo "╔══════════════════════════════════════════════╗"
echo "║  Automated Load Testing - All Scenarios      ║"
echo "╚══════════════════════════════════════════════╝"
echo -e "${NC}"
echo "Results will be saved to: $RESULTS_DIR"
echo ""

# Function to run a single test
run_test() {
    local scenario=$1
    local users=$2
    local spawn_rate=$3
    local duration=$4
    local http_class=$5
    local fast_class=$6
    
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Testing: $scenario (Users: $users, Duration: ${duration}s)${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    local test_dir="$RESULTS_DIR/$scenario"
    mkdir -p "$test_dir"
    
    # Run locust headless mode with CSV output
    docker-compose exec -T locust locust \
        -f /home/locust/locustfile.py \
        --host=http://product-api:8080 \
        $http_class $fast_class \
        --users=$users \
        --spawn-rate=$spawn_rate \
        --run-time=${duration}s \
        --headless \
        --csv=$test_dir/results \
        2>&1 | tee "$test_dir/output.log"
    
    echo -e "${GREEN}✓ Test completed: $scenario${NC}"
    echo ""
}

# Test 1: Read-Heavy
run_test "read-heavy" 100 5 180 "ReadHeavyHttpUser" "ReadHeavyFastHttpUser"

# Test 2: Balanced
run_test "balanced" 60 5 180 "BalancedHttpUser" "BalancedFastHttpUser"

# Test 3: Write-Heavy
run_test "write-heavy" 40 3 240 "WriteHeavyHttpUser" "WriteHeavyFastHttpUser"

# Summary
echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         ALL TESTS COMPLETED                  ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Results saved to: $RESULTS_DIR${NC}"
echo ""
echo "Files generated:"
echo "  • results_stats.csv - Statistics for each endpoint"
echo "  • results_stats_history.csv - Stats over time"
echo "  • results_failures.csv - Error details"
echo "  • output.log - Full test output"
echo ""
echo "Next: Copy these results into STRESS_TEST_RESULTS.md"
echo ""

# Extract and display key metrics
echo -e "${BLUE}════════════════════════════════════════════════"
echo "QUICK SUMMARY"
echo "════════════════════════════════════════════════${NC}"
echo ""

for scenario in read-heavy balanced write-heavy; do
    if [ -f "$RESULTS_DIR/$scenario/results_stats.csv" ]; then
        echo -e "${YELLOW}$scenario:${NC}"
        
        # Extract key metrics (adjust column numbers based on CSV format)
        awk -F',' '
            NR==2 {
                print "  Requests: " $3
                print "  Failures: " $4
                print "  Median: " $8 "ms"
                print "  95th %ile: " $10 "ms"
                print "  99th %ile: " $11 "ms"
                print "  RPS: " $15
            }
        ' "$RESULTS_DIR/$scenario/results_stats.csv"
        echo ""
    fi
done

echo -e "${GREEN}Done! Open $RESULTS_DIR to see all results.${NC}"
