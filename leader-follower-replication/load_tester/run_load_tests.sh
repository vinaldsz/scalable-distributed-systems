#!/usr/bin/env bash
# Runs all 16 load test scenarios (4 configs × 4 write percentages).
# Results land in analysis/results/.
#
# Usage:
#   ./load_tester/run_load_tests.sh                  # local (localhost)
#   HOST=1.2.3.4 ./load_tester/run_load_tests.sh     # against AWS EC2

set -euo pipefail

DURATION="${DURATION:-60s}"
CONCURRENCY="${CONCURRENCY:-20}"
RESULTS_DIR="analysis/results"
mkdir -p "$RESULTS_DIR"

# If HOST is set, override the base URLs in the load tester via env vars.
HOST_FLAG=""
if [[ -n "${HOST:-}" ]]; then
  export LT_HOST="$HOST"
  echo "Targeting remote host: $HOST"
fi

CONFIGS=("lf1" "lf2" "lf3" "ll")
WRITE_PCTS=(1 10 50 90)

for cfg in "${CONFIGS[@]}"; do
  for wpct in "${WRITE_PCTS[@]}"; do
    out="$RESULTS_DIR/${cfg}_w${wpct}.csv"
    echo "==> config=$cfg write%=$wpct -> $out"
    go run ./cmd/loadtester \
      -config "$cfg" \
      -write-pct "$wpct" \
      -duration "$DURATION" \
      -concurrency "$CONCURRENCY" \
      -out "$out"
  done
done

echo ""
echo "All scenarios complete. Run: python3 analysis/generate_graphs.py"
