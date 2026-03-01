# HW7: Cascading Failure & Bulkhead Recovery

## Overview

This project demonstrates cascading failure in distributed systems and the bulkhead isolation pattern to prevent full-service collapse when a dependency becomes slow.

## Architecture

### Search Service (`:8080`)

- `GET /products/search?q=<query>`
- `GET /health`
- `GET /metrics`

### Recommendation Service (`:8081`)

- Simulated slow dependency (~500ms)
- Limited concurrency

## Two Versions

### Broken (No Bulkhead)

- No concurrency guard around recommendation calls
- Slow downstream can block request processing and accumulate goroutines

### Fixed (Bulkhead)

- Semaphore-based limit (`bucketDepth=5`)
- Timeout-based graceful degradation
- Health stays responsive under load

## Load Test Plan

| Profile          | Users | Duration | Purpose                            |
| ---------------- | ----: | -------: | ---------------------------------- |
| Baseline         |    50 |       3m | Baseline behavior                  |
| Sustained Stress |    75 |      15m | Observe buildup/degradation        |
| Failure Push     |   100 |      20m | Force cascading behavior on broken |

## Current Artifacts

- Graphs currently available in `images/50users-3m/`
- `images/75users-15m/` is currently empty
- `images/100users-20m/` is currently empty

## Metrics Snapshot (from current CSV artifacts)

### Aggregated Results

| Run           | Requests | Failures | Avg (ms) | P95 (ms) | Max (ms) |    RPS |
| ------------- | -------: | -------: | -------: | -------: | -------: | -----: |
| Broken 50u/3m |   15,772 |        0 |   254.46 |      630 |      820 |  87.87 |
| Fixed 50u/3m  |   22,417 |        0 |    88.59 |      230 |      543 | 124.92 |
| Fixed 75u/15m |  172,722 |        0 |    85.64 |      230 |      820 | 192.08 |

### Quick Comparison (50u/3m)

- Avg latency improved from 254.46ms to 88.59ms (~2.87x better)
- Throughput improved from 87.87 RPS to 124.92 RPS (~42.16% higher)
- Both runs recorded 0 failures in the CSV snapshot

Use this section as the baseline until 75u broken and 100u runs are completed.

## 50 Users / 3 Minutes (Available Graphs)

![Latency Comparison](images/50users-3m/latency_comparison.png)
![Throughput Over Time](images/50users-3m/throughput_timeline.png)
![Aggregate Metrics](images/50users-3m/aggregate_metrics.png)
![Improvement Ratios](images/50users-3m/improvement_ratios.png)
![Failure Rate](images/50users-3m/failure_rate.png)

## Deployment + Test Commands (AWS)

Use one ALB URL at a time and verify it before each run.

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw7-crashing-and-recovering

# Always confirm the current ALB from terraform output
cd terraform
terraform output
cd ..

# Replace with your current active ALB from terraform output
export HOST="http://hw7-alb-776225006.us-west-2.elb.amazonaws.com"

# Sanity check before any locust run
curl -s "$HOST/health" | jq .
curl -s "$HOST/metrics" | jq .
curl -s "$HOST/products/search?q=test" | jq .
```

### Run Broken Version

```bash
cd terraform
terraform apply -lock=false -var="app_version=broken" -auto-approve
cd ..
sleep 60
curl -s "$HOST/health" | jq .

locust --host="$HOST" --users=50  --spawn-rate=5 --run-time=3m  --headless --html="metrics/50users-3m/broken_50u_3m_report.html"   --csv="metrics/50users-3m/broken_50u_3m"   --logfile="metrics/50users-3m/broken_50u_3m.log"
locust --host="$HOST" --users=75  --spawn-rate=5 --run-time=15m --headless --html="metrics/75users-15m/broken_75u_15m_report.html"  --csv="metrics/75users-15m/broken_75u_15m"  --logfile="metrics/75users-15m/broken_75u_15m.log"
locust --host="$HOST" --users=100 --spawn-rate=5 --run-time=20m --headless --html="metrics/100users-20m/broken_100u_20m_report.html" --csv="metrics/100users-20m/broken_100u_20m" --logfile="metrics/100users-20m/broken_100u_20m.log"
```

### Run Fixed Version

```bash
cd terraform
terraform apply -lock=false -var="app_version=fixed" -auto-approve
cd ..
sleep 60
curl -s "$HOST/health" | jq .

locust --host="$HOST" --users=50  --spawn-rate=5 --run-time=3m  --headless --html="metrics/50users-3m/fixed_50u_3m_report.html"   --csv="metrics/50users-3m/fixed_50u_3m"   --logfile="metrics/50users-3m/fixed_50u_3m.log"
locust --host="$HOST" --users=75  --spawn-rate=5 --run-time=15m --headless --html="metrics/75users-15m/fixed_75u_15m_report.html"  --csv="metrics/75users-15m/fixed_75u_15m"  --logfile="metrics/75users-15m/fixed_75u_15m.log"
locust --host="$HOST" --users=100 --spawn-rate=5 --run-time=20m --headless --html="metrics/100users-20m/fixed_100u_20m_report.html" --csv="metrics/100users-20m/fixed_100u_20m" --logfile="metrics/100users-20m/fixed_100u_20m.log"
```

## Generate Graphs

```bash
python3 generate_graphs.py
```

Move generated images into the target folder you want to document:

```bash
mkdir -p images/50users-3m
mv -f latency_comparison.png throughput_timeline.png aggregate_metrics.png improvement_ratios.png failure_rate.png images/50users-3m/
```

## Why You Might See 100% Locust Failures

If report shows all requests failed (often very low avg latency and `Average size = 0`), common causes are:

1. Wrong/old ALB host in `--host`
2. ECS service not healthy yet (target group still unhealthy)
3. Mixed output names (e.g., `broken` HTML + `fixed` CSV) causing confusion
4. ALB returning errors for all requests during rollout

Quick checks:

```bash
curl -i "$HOST/health"
curl -i "$HOST/metrics"
aws ecs describe-services --cluster hw7-cluster --services hw7-search-broken hw7-search-fixed --region us-west-2 --query 'services[].[serviceName,runningCount,desiredCount,status]' --output table
```

## CloudWatch Logs

```bash
aws logs tail /ecs/hw7-search-broken --follow --region us-west-2
aws logs tail /ecs/hw7-search-fixed --follow --region us-west-2
```

## Folder Structure

```text
hw7-crashing-and-recovering/
├── src/
├── terraform/
├── metrics/
│   ├── 50users-3m/
│   ├── 75users-15m/
│   └── 100users-20m/
├── images/
│   ├── 50users-3m/
│   ├── 75users-15m/
│   └── 100users-20m/
├── locustfile.py
├── generate_graphs.py
└── README.md
```
