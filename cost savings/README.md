# Cost Savings Assignment

## Phase 1: Synchronous Baseline

This phase tests a synchronous order API where each order waits for payment verification before response.

Service under test:

- POST /orders/sync
- GET /health

Load tool:

- Locust headless mode

## Test Configurations

Normal operations run:

- Users: 5
- Spawn rate: 1 per second
- Duration: 30 seconds

Flash sale run:

- Users: 20
- Spawn rate: 10 per second
- Duration: 60 seconds

## Result Files

Normal:

- metrics/phase1/phase1_normal_stats.csv
- metrics/phase1/phase1_normal_failures.csv

Flash:

- metrics/phase1/phase1_flash_stats.csv
- metrics/phase1/phase1_flash_failures.csv

## Findings Summary

### POST /orders/sync

Normal operations:

- Request count: 38
- Failures: 0
- Median response time: 3002.63 ms
- Average response time: 3012.66 ms
- Max response time: 3082.61 ms
- Throughput: 1.35 requests/sec

Flash sale:

- Request count: 95
- Failures: 0
- Median response time: 11916.24 ms
- Average response time: 10708.90 ms
- Max response time: 11916.24 ms
- Throughput: 1.66 requests/sec

Change from normal to flash:

- Median latency increased from about 3.0s to about 11.9s
- Average latency increased from about 3.0s to about 10.7s
- Throughput increased only slightly despite much higher demand

### Aggregated

Normal operations:

- Requests: 42
- Failures: 0
- Median response time: 3000.00 ms
- Average response time: 2727.73 ms

Flash sale:

- Requests: 104
- Failures: 0
- Median response time: 11916.24 ms
- Average response time: 9783.29 ms

## Interpretation

Even with zero failures in this short test window, user experience degrades severely during flash load:

- Customers wait around 10 to 12 seconds for order responses
- System does not scale throughput proportionally with demand
- This validates the synchronous bottleneck problem and motivates async processing in Phase 3

## Repro Commands

Run from the cost savings folder.

Create output folder:
mkdir -p "$(pwd)/metrics/phase1"

Normal operations:
docker compose run --rm \
 -v "$(pwd)/metrics/phase1:/results" \
 locust -f /home/locust/locustfile.py \
 --host=http://order-receiver:8080 \
 --users 5 --spawn-rate 1 --run-time 30s --headless \
 --csv=/results/phase1_normal \
 --html=/results/phase1_normal.html

Flash sale:
docker compose run --rm \
 -v "$(pwd)/metrics/phase1:/results" \
 locust -f /home/locust/locustfile.py \
 --host=http://order-receiver:8080 \
 --users 20 --spawn-rate 10 --run-time 60s --headless \
 --csv=/results/phase1_flash \
 --html=/results/phase1_flash.html
