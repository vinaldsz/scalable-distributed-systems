# HW7: Load Testing Commands Reference

## Quick Setup

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw7-crashing-and-recovering
mkdir -p metrics
```

## Configuration

- **ALB URL**: http://hw7-alb-776225006.us-west-2.elb.amazonaws.com
- **Users**: 50 concurrent users
- **Spawn Rate**: 5 users/second
- **Test Duration**: 3 minutes

---

## Verify Service Health (Use Before Load Testing)

Before running any load tests, always verify the service is healthy:

```bash
# Basic health check
curl -s http://hw7-alb-776225006.us-west-2.elb.amazonaws.com/health | jq .

# Expected output:
# {
#   "status": "ok",
#   "goroutines": 0
# }

# Test product search
curl -s "http://hw7-alb-776225006.us-west-2.elb.amazonaws.com/products/search?q=laptop" | jq .

# Check which version is deployed
curl -s "http://hw7-alb-776225006.us-west-2.elb.amazonaws.com/products/search?q=test" | jq -r '.version'

# Check metrics endpoint
curl -s http://hw7-alb-776225006.us-west-2.elb.amazonaws.com/metrics | jq .
```

---

## STEP 1: Test Broken Version

### 1.1 Deploy Broken Version

```bash
terraform init
terraform validate
terraform plan
terraform apply -var="app_version=broken"
```

Output:
No error should pop up in any of the command outputs on the terminal
`Apply complete! Resources: 17 added, 0 changed, 0 destroyed. ` -> Should look similar to this.

### 1.2 Check the health using curl

```bash
# Wait for ECS tasks to start (60-90 seconds)
sleep 60

# Check health endpoint
curl -s http://hw7-alb-776225006.us-west-2.elb.amazonaws.com/health | jq .

# Test product search endpoint
curl -s "http://hw7-alb-776225006.us-west-2.elb.amazonaws.com/products/search?q=laptop" | jq .

# Verify version is "broken"
curl -s "http://hw7-alb-776225006.us-west-2.elb.amazonaws.com/products/search?q=test" | jq -r '.version'
```

Expected output:

```json
{
  "status": "ok",
  "goroutines": 0
}
```

### 1.3 Run Load Test for Broken Version

```bash
locust \
    --host="http://hw7-alb-2132149712.us-west-2.elb.amazonaws.com" \
    --users=50 \
    --spawn-rate=5 \
    --run-time=3m \
    --headless \
    --html="metrics/broken_run_report.html" \
    --csv="metrics/broken_run" \
    --logfile="metrics/broken_run.log"
```

### 1.4 View Broken Version Results

```bash
# Open HTML report
open metrics/broken_run_report.html

# View summary from logs
tail -30 metrics/broken_run.log | grep -A 15 "LOAD TEST SUMMARY"

# View CSV stats
cat metrics/broken_run_stats.csv
```

---

## STEP 2: Deploy Fixed Version

### 2.2 Deploy Fixed Version

```bash
cd terraform
terraform apply -var="app_version=fixed" -auto-approve
cd ..
```

### 2.3 Wait for Deployment to Stabilize

````bash
# Wait 60 seconds for ECS tasks to start
sleep 60

# Verify fixed version is running
curl -s http://hw7-alb-1506796314.us-west-2.elb.amazonaws.com/health | jq .
```
expected output:
```bash
{
  "status": "ok",
  "goroutines": 0
}
````

```bash
curl -s "http://hw7-alb-1506796314.us-west-2.elb.amazonaws.com/products/search?q=test" | jq .
```

Expected output:

```bash
{
  "products": null,
  "total_found": 0,
  "recommendations": [],
  "search_time_ms": 0,
  "recommendation_time_ms": 0,
  "total_time_ms": 100,
  "active_goroutines": 1,
  "version": "fixed"
}
```

# Verify version is "fixed"

curl -s "http://hw7-alb-1506796314.us-west-2.elb.amazonaws.com/products/search?q=test" | jq -r '.version'

```

Expected output for version check:

```

fixed

````

### 2.4 Check ECS Service Status

```bash
aws ecs describe-services \
    --cluster hw7-cluster \
    --services hw7-search-fixed \
    --region us-west-2 \
    --query 'services[0].[status,runningCount,desiredCount]' \
    --output table
````

---

|DescribeServices|
+----------------+
| ACTIVE |
| 2 |
| 2 |
+----------------+

---

## STEP 3: Test Fixed Version

### 3.1 Run Load Test for Fixed Version

```bash
cd ..
locust \
    --host="http://hw7-alb-1506796314.us-west-2.elb.amazonaws.com" \
    --users=50 \
    --spawn-rate=5 \
    --run-time=3m \
    --headless \
    --html="metrics/fixed_run_report.html" \
    --csv="metrics/fixed_run" \
    --logfile="metrics/fixed_run.log"
```

### 3.2 View Fixed Version Results

```bash
# Open HTML report
open metrics/fixed_run_report.html # This will open the report in the default browser


# View summary from logs
tail -30 metrics/fixed_run.log | grep -A 15 "LOAD TEST SUMMARY"

# View CSV stats
cat metrics/fixed_run_stats.csv
```

---

## STEP 4: Generate Comparison Graphs

### 4.1 Run Graph Generation Script

```bash
python3 generate_graphs.py
```

### 4.2 View Generated Graphs

```bash
# List all generated graphs
ls -lh metrics/*.png

# Open all graphs
open metrics/*.png
```

---

## STEP 5: Compare Results

### 5.1 Side-by-Side CSV Comparison

```bash
echo "=== BROKEN VERSION ==="
cat metrics/broken_run_stats.csv | column -t -s','

echo ""
echo "=== FIXED VERSION ==="
cat metrics/fixed_run_stats.csv | column -t -s','
```

### 5.2 Quick Performance Comparison

```bash
echo "Average Response Time:"
echo -n "  Broken: "
tail -1 metrics/broken_run_stats.csv | cut -d',' -f7
echo -n "  Fixed: "
tail -1 metrics/fixed_run_stats.csv | cut -d',' -f7

echo ""
echo "Request Rate:"
echo -n "  Broken: "
tail -1 metrics/broken_run_stats.csv | cut -d',' -f11
echo -n "  Fixed: "
tail -1 metrics/fixed_run_stats.csv | cut -d',' -f11

echo ""
echo "Failure Rate:"
echo -n "  Broken: "
tail -1 metrics/broken_run_stats.csv | cut -d',' -f3
echo -n "  Fixed: "
tail -1 metrics/fixed_run_stats.csv | cut -d',' -f3
```

### 5.3 View CloudWatch Logs

```bash
# View ECS task logs
aws logs tail /ecs/hw7-search-broken --follow --region us-west-2
aws logs tail /ecs/hw7-search-fixed --follow --region us-west-2
```

---

## TROUBLESHOOTING

### Understanding Health Check Responses

**Healthy Response:**

```json
{
  "status": "ok",
  "goroutines": 0
}
```

**During Load (Broken Version):**

- Goroutine count will increase significantly (50+)
- Response times will degrade
- Some requests may timeout

**During Load (Fixed Version):**

- Goroutine count stays bounded (max ~10-12)
- Response times remain consistent
- System stays responsive

### If ECS Tasks Aren't Starting

```bash
# Check task status
aws ecs list-tasks --cluster hw7-cluster --service-name hw7-search-broken --region us-west-2

# Describe task to see why it stopped
TASK_ARN=$(aws ecs list-tasks --cluster hw7-cluster --service-name hw7-search-broken --region us-west-2 --query 'taskArns[0]' --output text)
aws ecs describe-tasks --cluster hw7-cluster --tasks $TASK_ARN --region us-west-2

# View CloudWatch logs for the failed container
aws logs tail /ecs/hw7-search-broken --since 30m --region us-west-2

# Follow logs in real-time
aws logs tail /ecs/hw7-search-broken --follow --region us-west-2

# Check for specific errors
aws logs tail /ecs/hw7-search-broken --since 1h --region us-west-2 | grep -i "error\|fail\|exit"
```

**Common Issues:**

- **"Unknown version" error**: The command parameter in ECS isn't being passed correctly
- **Port binding errors**: Another service is using the same port
- **Image pull errors**: ECR permissions or image doesn't exist
- **Out of memory**: Task memory configuration too low

### If ALB Returns 503

```bash
# Check target group health
aws elbv2 describe-target-health \
    --target-group-arn $(terraform output -raw target_group_arn) \
    --region us-west-2

# Wait longer for tasks to become healthy
for i in {1..20}; do
    echo "Health check attempt $i/20:"
    HEALTH=$(curl -s http://hw7-alb-2132149712.us-west-2.elb.amazonaws.com/health)
    if [ $? -eq 0 ]; then
        echo "$HEALTH" | jq .
        break
    else
        echo "Service not ready, waiting 10 seconds..."
        sleep 10
    fi
done

# Check ECS task health
aws ecs describe-services \
    --cluster hw7-cluster \
    --services hw7-search-broken hw7-search-fixed \
    --region us-west-2 \
    --query 'services[].[serviceName,status,runningCount,desiredCount]' \
    --output table
```

### Re-run Just One Test

```bash
# Re-test broken (if still deployed)
terraform apply -lock=false -var="app_version=broken" -auto-approve
sleep 60
locust --host="http://hw7-alb-2132149712.us-west-2.elb.amazonaws.com" --users=50 --spawn-rate=5 --run-time=3m --headless --html="metrics/broken_run_report.html" --csv="metrics/broken_run"

# Re-test fixed
terraform apply -lock=false -var="app_version=fixed" -auto-approve
sleep 60
locust --host="http://hw7-alb-2132149712.us-west-2.elb.amazonaws.com" --users=50 --spawn-rate=5 --run-time=3m --headless --html="metrics/fixed_run_report.html" --csv="metrics/fixed_run"
```

---

## CLEANUP

### Destroy All Infrastructure

```bash
cd terraform
terraform destroy -var="app_version=fixed" -auto-approve
```

---

## FILES GENERATED

After running tests, you'll have:

```
metrics/
├── broken_run_report.html       # Interactive Locust report (broken)
├── broken_run_stats.csv         # Aggregated statistics (broken)
├── broken_run_stats_history.csv # Time-series data (broken)
├── broken_run_exceptions.csv    # Exceptions logged (broken)
├── broken_run_failures.csv      # Failed requests (broken)
├── broken_run.log               # Locust execution log (broken)
├── fixed_run_report.html        # Interactive Locust report (fixed)
├── fixed_run_stats.csv          # Aggregated statistics (fixed)
├── fixed_run_stats_history.csv  # Time-series data (fixed)
├── fixed_run_exceptions.csv     # Exceptions logged (fixed)
├── fixed_run_failures.csv       # Failed requests (fixed)
├── fixed_run.log                # Locust execution log (fixed)
├── latency_comparison.png       # Response time comparison graph
├── throughput_timeline.png      # Throughput over time graph
├── aggregate_metrics.png        # Overall metrics comparison
├── improvement_ratios.png       # Performance improvement ratios
└── failure_rate.png             # Failure rate comparison
```

---

## AUTOMATED SCRIPT

For fully automated testing, use:

```bash
chmod +x run_complete_test.sh
./run_complete_test.sh
```
