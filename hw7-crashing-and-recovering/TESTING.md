# HW7 Testing Guide

## Quick Test: Broken vs Fixed

### Option 1: Local Docker (Easiest)

```bash
# Terminal 1: Start broken version with recommendation service
docker-compose up search-broken

# Terminal 2: Run load test (will run for 5 minutes)
docker-compose --profile test up locust

# Then view Locust at http://localhost:8089
```

### Option 2: Local Go (Direct)

```bash
# Prerequisites: Go 1.25+

# Terminal 1: Start broken version
cd src && go run . broken

# Terminal 2: Start load test
pip install locust
locust -f ../locustfile.py --headless -u 20 -r 2 -t 2m --host=http://localhost:8080

# Terminal 3: Monitor metrics
while true; do
  curl -s http://localhost:8080/metrics | jq .active_goroutines
  sleep 1
done
```

### Option 3: Kubernetes (AWS ECS - Later)

```bash
cd terraform
terraform init
terraform apply -var="version=broken"
# Monitor CloudWatch metrics
```

---

## What to Observe

### Broken Version Metrics

When load test runs (20 users, 2/sec ramp):

1. **Response Time** - Watch it climb over time
   - Start: ~50ms
   - After 30s: ~500ms (queueing)
   - After 60s: ~2000-3000ms (goroutine accumulation)

2. **Goroutines** - Check `/metrics` endpoint
   - Start: ~5
   - Peak: 20+ (one per concurrent request)

3. **Health Check** - Try hitting `/health`
   - May timeout after ~60 seconds of load

### Fixed Version Metrics

Same load test, fixed version:

1. **Response Time** - Stays stable
   - Consistent: ~100-150ms
   - No queueing effect

2. **Goroutines** - Bounded
   - Range: 5-10 always
   - Bulkhead limits this

3. **Health Check** - Always responds fast
   - Always <5ms response time

---

## Expected Output Example

### Broken Version

```
$ curl -s http://localhost:8080/products/search?q=laptop | jq .
{
  "products": [
    {
      "id": 1,
      "name": "Product Alpha 1",
      "category": "Electronics",
      "description": "Pro Electronics item #1",
      "brand": "Alpha"
    }
  ],
  "total_found": 45,
  "recommendations": null,            ← ❌ EMPTY (timeout)
  "search_time_ms": 10,
  "recommendation_time_ms": 3500,     ← ❌ SLOW
  "total_time_ms": 3510,              ← ❌ SLOW
  "active_goroutines": 18             ← ❌ ACCUMULATING
}
```

### Fixed Version

```
$ curl -s http://localhost:8082/products/search?q=laptop | jq .
{
  "products": [
    {
      "id": 1,
      "name": "Product Alpha 1",
      "category": "Electronics",
      "description": "Pro Electronics item #1",
      "brand": "Alpha"
    }
  ],
  "total_found": 45,
  "recommendations": [...],           ← ✓ RETURNED
  "search_time_ms": 10,
  "recommendation_time_ms": 120,      ← ✓ FAST
  "total_time_ms": 130,               ← ✓ FAST
  "active_goroutines": 8              ← ✓ BOUNDED
}
```

---

## Locust Testing Commands

### Run broken version test

```bash
locust -f locustfile.py \
  --headless \
  -u 50 \
  -r 5 \
  -t 5m \
  --csv=test_broken \
  --host=http://localhost:8080
```

### Run fixed version test

```bash
locust -f locustfile.py \
  --headless \
  -u 50 \
  -r 5 \
  -t 5m \
  --csv=test_fixed \
  --host=http://localhost:8082
```

### Parameters Explained

- `-u 50`: 50 concurrent users
- `-r 5`: Ramp up 5 users per second
- `-t 5m`: Run for 5 minutes
- `--csv=test_results`: Export metrics to CSV

---

## Cleanup

```bash
# Stop Docker containers
docker-compose down

# Stop Locust
kill %1  # or Ctrl+C

# Clean local build artifacts
rm -rf src/search-service
```

---

## Troubleshooting

### Error: "Connection refused"

- Ensure recommendation service started before search service
- Check: `curl http://localhost:8081/rec_health`

### Error: "No module named locust"

- Install: `pip install locust==2.43.1`

### Build fails with "undefined reference"

- Ensure both files are in same `src/` directory
- Run: `cd src && go build . -v`

### Docker build fails

- Ensure `go.mod` exists in root directory
- Check: `file go.mod` (should be a text file)

---

## Next Steps

After testing locally:

1. **Capture metrics** from Locust output
2. **Fix deployment** - Update terraform/ for ECS deployment
3. **Deploy to AWS** - terraform apply
4. **Compare metrics** - Run same load test in cloud
5. **Document findings** - Write FAILURE_ANALYSIS.md

---

## Quick Metrics Export

```bash
# After Locust test, convert CSV to markdown table for report
python3 << 'EOF'
import pandas as pd
df = pd.read_csv('test_broken_stats.csv')
print(df[['Name', 'Requests', 'Failures', 'Median', '95%']].to_markdown(index=False))
EOF
```
