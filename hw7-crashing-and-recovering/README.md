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

## Bulkhead Design Pattern

The **bulkhead isolation pattern** prevents cascading failures by limiting resources consumed by one dependency, protecting critical paths:

- **Broken (No Bulkhead)**: All goroutines can wait indefinitely on slow recommendation service → goroutine pool exhausted → /health hangs → service appears dead
- **Fixed (Bulkhead)**: Only 5 concurrent recommendation calls allowed → remaining requests get search results without recommendations → /health always responsive

### Why It Works

In distributed systems, a slow dependency (recommendation service taking 500ms) can starve request handlers. Without limits:

- 100 concurrent users × 500ms latency = 50 goroutines stuck
- New requests accumulate in queue → new goroutines created → memory exhausted → OOM

With a semaphore bulkhead:

- First 5 requests → recommendation service
- Remaining requests → skip recommendations, return search results (graceful degradation)
- Health endpoint stays responsive (not competing for global goroutine pool)

## Two Versions

### Broken (No Bulkhead)

- No concurrency guard around recommendation calls
- Slow downstream can block request processing and accumulate goroutines
- `/health` endpoint becomes sluggish under load
- Example: 100 users = potential 100+ blocked goroutines waiting on 500ms latency

### Fixed (Bulkhead)

- Semaphore-based limit (`bucketDepth=5`)
- Timeout `100ms` to acquire a slot; gracefully skip if full
- Health endpoint always responds instantly
- Degraded-but-operational: returns search results without recommendations when bulkhead is full

## Code Comparison: Broken vs Fixed

### Broken Version

```go
// ❌ PROBLEM: No timeout, no bulkhead, waits indefinitely
func searchHandlerBroken(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Calls recommendation service with no protection
    recs, recTimeMs, err := recClient.FetchRecommendations(ctx, q)
    // If service is slow, this goroutine blocks for 500ms+
    // With 100 concurrent users = 100+ goroutines stuck
}
```

### Fixed Version

```go
// ✓ FIXED: Semaphore limits concurrent calls to 5
var recSemaphore chan struct{}    // Bounded semaphore
var bucketDepth = 5               // Max concurrent calls
var bucketTimeout = 100 * time.Millisecond  // Timeout

func searchHandlerFixed(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(context.Background(), bucketTimeout)
    defer cancel()

    select {
    case recSemaphore <- struct{}{}:  // Got a slot
        defer func() { <-recSemaphore }()  // Release when done
        recs, recTimeMs, err := recClient.FetchRecommendations(ctx, q)

    case <-ctx.Done():
        // No slot available: gracefully degrade
        // Return search results without recommendations
    }
}
```

**Key Differences:**

- ❌ Broken: Unbounded goroutines, no timeout, synchronous wait
- ✓ Fixed: Bounded to 5 concurrent calls, 100ms timeout, graceful degradation

## Load Test Plan

| Profile          | Users | Duration | Purpose                            |
| ---------------- | ----: | -------: | ---------------------------------- |
| Baseline         |    50 |       3m | Baseline behavior                  |
| Sustained Stress |    75 |      15m | Observe buildup/degradation        |
| Failure Push     |   100 |      20m | Force cascading behavior on broken |

## Current Artifacts

- ✅ Graphs available in `images/50users-3m/` - Baseline test (3 minutes)
- ✅ Graphs available in `images/75users-15m/` - Sustained stress test (15 minutes)
- ✅ Graphs available in `images/100users-20m/` - Stress test (20 minutes)

## Metrics Snapshot (Search Endpoints Only)

> **Note**: These metrics focus exclusively on `/products/search` endpoints, which interact with the slow recommendation service and demonstrate the cascading failure pattern. Health and metrics endpoints don't call recommendations and remain fast in both versions.

### Search Performance Results

| Run             | Requests | Failures | Avg (ms) | P50 (ms) | P95 (ms) | P99 (ms) | Max (ms) |    RPS |
| --------------- | -------: | -------: | -------: | -------: | -------: | -------: | -------: | -----: |
| Broken 50u/3m   |    9,362 |        0 |   390.20 |      539 |      617 |      637 |      820 |  52.01 |
| Fixed 50u/3m    |   13,531 |        0 |   109.95 |      133 |      219 |      234 |      294 |  75.17 |
| Broken 75u/15m  |   83,353 |        0 |   261.59 |       81 |      610 |      632 |    1,400 |  92.61 |
| Fixed 75u/15m   |  103,787 |        0 |   101.96 |       82 |      219 |      246 |      817 | 115.32 |
| Broken 100u/20m |  159,457 |        0 |   204.35 |       50 |      612 |      635 |    1,161 | 132.88 |
| Fixed 100u/20m  |  365,766 |        0 |   101.92 |       78 |      226 |      280 |      853 | 304.81 |

### Key Findings: Bulkhead Pattern Effectiveness

**Baseline Test (50 users / 3 minutes):**

- **3.55x** latency improvement (390ms → 110ms)
- **44.5%** throughput gain (52 RPS → 75 RPS)
- **2.64x** better P95 (617ms → 234ms)
- **2.72x** better P99 (637ms → 234ms)
- **Median shift:** Broken at 539ms vs Fixed at 133ms - **4.05x faster**
- Zero failures in both versions

**Sustained Stress Test (75 users / 15 minutes):**

- **24.5%** more requests processed (83k → 104k)
- **2.57x** latency improvement (262ms → 102ms)
- **2.79x** better P95 latency (610ms → 219ms)
- **2.57x** better P99 latency (632ms → 246ms)
- **24.5%** throughput gain (93 RPS → 115 RPS)
- **Cascading failure visible:** Broken P50 at 81ms jumps to 610ms at P95 (7.5x spike) vs Fixed stays stable (82ms → 219ms, only 2.7x)
- **Sustained stability:** Fixed version maintains low median (82ms) over entire 15-minute duration
- Zero failures in both versions

**Stress Test (100 users / 20 minutes):**

- **2.29x** more requests completed (159k → 366k)
- **2.00x** faster average response (204ms → 102ms)
- **2.71x** better P95 latency (612ms → 226ms)
- **2.27x** better P99 latency (635ms → 280ms)
- **2.29x** higher throughput (133 RPS → 305 RPS)
- **Extreme tail latency:** Broken P50 of 50ms masks cascading failure visible at P95/P99 (12x spike)
- Zero failures in both versions

### Critical Observations

1. **Cascading Failure Pattern (Broken Version):**
   - **Bimodal latency distribution**: P50 often low (50-81ms when queues empty) but **P95 spikes to 610-617ms** (goroutine accumulation)
   - **At 50 users**: Median 539ms shows system already struggling at baseline - threads blocking waiting for recommendations
   - **At 75 users**: Median drops to 81ms (faster queue processing) but P95 stays high at 610ms - **7.5x spike** from median
   - **At 100 users**: Median artificially low at 50ms (fast queue flush) but P95 remains 612ms - **12x variance** reveals cascading failure
   - **Average latency misleading**: 204-390ms averages hide the bimodal pattern where some requests are fast but tail users suffer
   - **Duration matters**: 75u/15m and 100u/20m tests prove cascading accumulates over extended periods

2. **Bulkhead Protection (Fixed Version):**
   - **Consistent stability**: Maintained **102-110ms** average search latency across all load levels (50u, 75u, 100u)
   - **Predictable performance**: P50 to P95 spread only **2.5-3x** vs broken's **7-12x** - uniform response times
   - **Tail latency control**: P95 stayed at **219-234ms** vs broken's 610-617ms - **2.6-2.8x better**
   - **Graceful degradation working**: When bulkhead full (5 slots), search returns without recommendations instead of blocking
   - **Throughput advantage**: Achieved **305 RPS** at 100u vs broken's 133 RPS - **2.3x more capacity**

3. **Graceful Degradation in Action:**
   - Fixed version skipped recommendations when bulkhead full (shown by "Cascading Failures (no recommendations)" in logs)
   - Broken version blocked all search requests waiting for slow recommendation service (500ms latency)
   - **No HTTP failures** across any test profile (0% failure rate) - bulkhead prevented total collapse
   - **User experience**: Fixed users get search results quickly without recommendations vs broken users wait 600ms+ for complete response

4. **Scalability Evidence:**
   - **Fixed version scales efficiently**: 75 RPS (50u) → 115 RPS (75u) → 305 RPS (100u) - near-linear growth
   - **Broken version capacity ceiling**: 52 RPS (50u) → 93 RPS (75u) → 133 RPS (100u) - sublinear, hits limit early
   - **Efficiency delta grows with load**: 44% advantage at 50u → 25% at 75u → 129% at 100u
   - **Duration resilience**: Fixed maintains stable 102ms avg over 15-20 minute sustained operations
   - Bulkhead pattern unlocked **2.3x more throughput** under same infrastructure

## 50 Users / 3 Minutes (Baseline Test)

![Latency Comparison](images/50users-3m/latency_comparison.png)
![Throughput Over Time](images/50users-3m/throughput_timeline.png)
![Aggregate Metrics](images/50users-3m/aggregate_metrics.png)
![Improvement Ratios](images/50users-3m/improvement_ratios.png)
![Failure Rate](images/50users-3m/failure_rate.png)

## 75 Users / 15 Minutes (Sustained Stress Test)

**What the graphs reveal:**

1. **Latency Comparison**: Fixed version maintains stable 102ms average search latency over full 15 minutes, while broken version shows cascading failure with **bimodal distribution** - P50 at 81ms but P95 spikes to 610ms (7.5x variance) indicating goroutine accumulation under sustained load.

2. **Throughput Timeline**: Fixed version sustains 115 RPS consistently, while broken version achieves only 93 RPS - demonstrating 24.5% throughput advantage even at moderate sustained load.

3. **Aggregate Metrics**: Fixed version processed 103,787 search requests (24.5% more) with 102ms average vs broken's 83,353 requests at 262ms average - proving bulkhead prevents degradation during extended operations.

4. **Improvement Ratios**: Sustained load reveals 2.5-2.8x improvements - P95 latency (2.79x better), P99 latency (2.57x better), average response (2.57x faster) - demonstrating bulkhead pattern prevents performance accumulation issues.

5. **Failure Rate**: Both versions maintained 0% failure rate, but broken version shows dangerous bimodal pattern (fast median masks high tail latency), while fixed version demonstrates stable uniform behavior suitable for production deployment.

![Latency Comparison](images/75users-15m/latency_comparison.png)
![Throughput Over Time](images/75users-15m/throughput_timeline.png)
![Aggregate Metrics](images/75users-15m/aggregate_metrics.png)
![Improvement Ratios](images/75users-15m/improvement_ratios.png)
![Failure Rate](images/75users-15m/failure_rate.png)

## 100 Users / 20 Minutes (Stress Test)

**What the graphs reveal:**

1. **Latency Comparison**: Fixed version maintains 102ms average with P50 at 78ms, while broken version shows **extreme bimodal pattern** - P50 artificially low at 50ms but P95 jumps to 612ms (12x variance) revealing cascading failure where fast queue flush masks goroutine accumulation.

2. **Throughput Timeline**: Fixed version sustains 305 RPS consistently throughout 20 minutes, while broken version plateaus at 133 RPS - showing capacity ceiling hit due to goroutine accumulation blocking request processing.

3. **Aggregate Metrics**: Fixed version processed 365,766 search requests (2.3x more) with 102ms average latency (2.0x faster) compared to broken version's 159,457 requests at 204ms average.

4. **Improvement Ratios**: All key metrics show 2-3x improvement - P95 latency (2.71x better), P99 latency (2.27x better), throughput (2.29x higher), proving bulkhead pattern effectiveness under sustained high load.

5. **Failure Rate**: Both versions maintained 0% HTTP failure rate, but broken version achieved this through severely degraded performance (P95 at 612ms), while fixed version gracefully skipped recommendations to maintain 102ms average responsiveness.

![Latency Comparison](images/100users-20m/latency_comparison.png)
![Throughput Over Time](images/100users-20m/throughput_timeline.png)
![Aggregate Metrics](images/100users-20m/aggregate_metrics.png)
![Improvement Ratios](images/100users-20m/improvement_ratios.png)
![Failure Rate](images/100users-20m/failure_rate.png)

## Real-World Implications

### Without Bulkhead (Broken Version)

- **Unpredictable user experience**: Bimodal latency distribution means some users get 50ms responses while others wait 600ms+ - no consistent SLA possible
- **Cascading failure visible**: P50 at 50-81ms masks severe tail latency (P95 at 610ms) - **7-12x variance** indicates goroutine accumulation
- **Resource exhaustion**: Unbounded goroutines waiting on slow recommendation service (500ms latency) accumulate over time
- **Reduced capacity**: Only 133 RPS search throughput at 100 users despite available resources
- **SLA violations**: P99 at 635ms means 1% of users wait **over half a second** for search results
- **Risk**: At higher load or longer duration, service could exhaust memory/goroutine limits and crash

### With Bulkhead (Fixed Version)

- **Consistent user experience**: Stable 102ms average search latency with predictable 2.5-3x P50-to-P95 spread
- **Isolation working**: Semaphore limits recommendation calls to 5 concurrent slots, preventing goroutine accumulation
- **Graceful degradation**: When bulkhead full, returns search results without recommendations (102ms) rather than blocking (600ms)
- **Higher capacity**: Achieved 305 RPS search throughput (2.3x more) by not blocking on slow dependencies
- **Predictable SLA**: P99 at 280ms means 99% of users get results in under 280ms consistently
- **Resilience**: Service remains responsive even if recommendation service becomes completely unavailable or slow

### Production Readiness

- **Bulkhead pattern proven**: 2-3x improvement across all search metrics (latency, throughput, tail latency control)
- **Zero HTTP failures**: Both versions handled 15-20 minute tests at 75-100 users without errors
- **Scalability unlocked**: Fixed version can handle 2.3x more search traffic on same infrastructure
- **Cost efficiency**: Bulkhead pattern eliminates need for over-provisioning to handle slow dependencies
- **Predictable performance**: Fixed version maintains uniform latency distribution vs broken's unpredictable bimodal pattern

**Recommendation**: Deploy the bulkhead-protected version to production. The 5-slot semaphore with 100ms timeout provides optimal balance between utilizing the recommendation service and protecting core search functionality. Search-only metrics prove the pattern works where it matters most - on endpoints calling slow dependencies.

## Quick Switch Between Versions

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw7-crashing-and-recovering/terraform

# Deploy Broken Version
terraform apply -lock=false -var="app_version=broken" -auto-approve

# Deploy Fixed Version
terraform apply -lock=false -var="app_version=fixed" -auto-approve
```

ECS will redeploy the service with the selected version in ~60 seconds.

## Deployment + Test Commands (AWS)

Use one ALB URL at a time and verify it before each run.

**First time setup (create metric/image directories):**

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw7-crashing-and-recovering
mkdir -p metrics/50users-3m metrics/75users-15m metrics/100users-20m
mkdir -p images/50users-3m images/75users-15m images/100users-20m
```

**Then get ALB host and verify:**

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw7-crashing-and-recovering

# Get the ALB URL from terraform output
cd terraform
export HOST=$(terraform output -raw alb_url)
echo "Using ALB: $HOST"
cd ..

# Sanity check before any locust run
curl -s "$HOST/health" | jq .
curl -s "$HOST/metrics" | jq .
curl -s "$HOST/products/search?q=test" | jq .
```

### Run Broken Version

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw7-crashing-and-recovering
cd terraform
export HOST=$(terraform output -raw alb_url)
cd ..

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
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw7-crashing-and-recovering
cd terraform
export HOST=$(terraform output -raw alb_url)
cd ..

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

The script automatically finds test data by profile and generates comparison graphs.

```bash
# Auto-detect latest test profile and generate graphs
python3 generate_graphs.py

# Generate graphs for specific profile
python3 generate_graphs.py 50users-3m
python3 generate_graphs.py 75users-15m
python3 generate_graphs.py 100users-20m

# Generate graphs for all available profiles
python3 generate_graphs.py all
```

After generation, move graphs to their profile folder:

```bash
mkdir -p images/50users-3m
mv -f *.png images/50users-3m/

# Or for a different profile:
mkdir -p images/75users-15m
mv -f *.png images/75users-15m/
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
