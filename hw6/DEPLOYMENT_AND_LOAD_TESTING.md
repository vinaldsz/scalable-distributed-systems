# Product Search Service - Deployment & Load Testing Guide

## Prerequisites

- AWS Account with credentials configured (`aws configure`)
- Terraform installed
- Docker installed
- Locust installed (`pip install locust`)
- jq installed (for JSON formatting)

---

## STEP-BY-STEP DEPLOYMENT

### Step 1: Stop Local Service (if running)

```bash
# Kill any existing process on port 8080
killall main           # or whatever process is running
```

**Screenshot**: None needed - just verify terminal shows "killed" or "no matching processes"

---

### Step 2: Verify Docker Image Builds Successfully

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw6

# Build the Docker image locally
docker build -f src/Dockerfile -t product-search:latest src/

echo "Build complete! Testing image..."
docker run --rm product-search:latest ./product-search --help 2>&1 | head -5
```

![Docker Build Success](screenshots/docker_build_success.png)

---

### Step 3: Initialize Terraform

```bash
cd terraform/

# Initialize Terraform (downloads AWS provider)
terraform init

# Should see: "Terraform has been successfully configured!"
```

---

### Step 4: Validate Terraform Configuration

```bash
terraform validate

# Should output: "Success! The configuration is valid."
```

---

### Step 5: Plan the Deployment

```bash
terraform plan

# Review the output - should show:
# - ECR repository creation
# - ECS Cluster creation
# - ECS Service creation
# - Security Group creation
# - etc.
```

- `aws_ecs_cluster.this will be created`
- `aws_ecs_service.this will be created`
- `aws_security_group.this will be created`
- etc.

---

### Step 6: Apply Terraform Configuration

```bash
terraform apply

# Wait 3-5 minutes for resources to be created
# Once complete you should see outputs like:
# ecs_cluster_name = "product-search-cluster"
# ecs_service_name = "product-search"
```

![Terraform Apply Success](screenshots/terraform_apply_success.png)

---

### Step 7: Verify ECS Service is Running

```bash
TASK_ARN=$(aws ecs list-tasks --cluster product-search-cluster --region us-west-2 --query 'taskArns[0]' --output text)
TASK_IP=$(aws ecs describe-tasks --cluster product-search-cluster --tasks "$TASK_ARN" --region us-west-2 --query 'tasks[0].attachments[0].details[?name==`publicIPv4Address`].value' --output text)

curl "http://$TASK_IP:8080/health"
# Expected: {"status":"ok"}
```

---

### Step 8: Test Search Functionality

```bash

# Test search endpoint
curl "http://$TASK_IP:8080/products/search?q=Electronics"

# Expected: Returns a match count and response time (should be <20ms)
```

![Search Endpoint Test](screenshots/curl_electronics_output.png)

### Step 9: Open CloudWatch Dashboard

Navigate to AWS Console:

1. Go to **CloudWatch** → **ECS** → **Cluster** → **Service** → **Metrics**

![CloudWatch Metrics Before Load Test](screenshots/cloudwatch_metrics_before_load_test.png)

---

## STEP-BY-STEP LOAD TESTING

### Locust Test Setup (FastHttpUser, minimal wait)

The Locust file uses `FastHttpUser` with a short wait time to keep pressure high.

Run from the `hw6` folder so paths are correct:

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw6
```

Grab the current task public IP (no ALB in this deployment):

```bash
TASK_ARN=$(aws ecs list-tasks --cluster product-search-cluster --region us-west-2 --query 'taskArns[0]' --output text)
ENI_ID=$(aws ecs describe-tasks --cluster product-search-cluster --tasks "$TASK_ARN" --region us-west-2 --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' --output text)
TASK_IP=$(aws ec2 describe-network-interfaces --network-interface-ids "$ENI_ID" --region us-west-2 --query 'NetworkInterfaces[0].Association.PublicIp' --output text)
echo "$TASK_IP"
```

### Test 1: Baseline (5 users, 2 minutes)

```bash
# Start Locust with 5 users
locust \
  --host=http://$TASK_IP:8080 \
  --users=5 \
  --spawn-rate=1 \
  --run-time=120s \
  --headless \
  -f ../locustfile.py 2>&1 | tee baseline_test.log
```

To run Web version,

```bash
locust -f locustfile.py --host=http://$TASK_IP:8080
```

**What to monitor during this test**:

1. Keep CloudWatch open showing ECS metrics
2. Watch CPU Utilization (will be very low ~1-2%)
3. Watch Response Time (should be <10ms)

**Baseline Metrics**:
![Locust Baseline Test - 5 Users](screenshots/locust_baseline_5users.png)

![CloudWatch Metrics - Baseline Test](screenshots/cloudwatch_baseline_5users.png)

---

### Test 2: Breaking Point (20 users, 3 minutes)

```bash
# Kill the local Locust if still running
killall locust 2>/dev/null

sleep 30

# Start Locust with 20 users
locust \
  --host=http://$TASK_IP:8080 \
  --users=20 \
  --spawn-rate=2 \
  --run-time=180s \
  --headless \
  -f locustfile.py 2>&1 | tee breaking_point_test.log
```

**Breaking Point Metrics**:
![Locust Breaking Point Test - 20 Users](screenshots/locust_breaking_point_20users.png)

- CloudWatch metrics during peak load
  ![CloudWatch Metrics - Breaking Point Test](screenshots/cloudwatch_breaking_point_20users.png)

---

## ANALYSIS SECTION

After both tests complete, answer these questions with screenshots:

### Question 1: Which resource hits the limit first?

```bash
# Review the metrics screenshots and note:
# - Is CPU at 100% while Memory is <70%?  → CPU-BOUND (need to SCALE)
# - Is Memory high while CPU is moderate?  → MEMORY-BOUND (need to OPTIMIZE)
```

**Answer**: **NEITHER CPU nor Memory hit their limits** - this is a **lock contention bottleneck**, NOT a resource shortage!

**From CloudWatch Metrics**:

- CPU Utilization: **Only ~5.48% maximum** (nowhere near 100%!)
- CPU remained extremely low throughout both tests
- Memory: Also well below capacity

![CloudWatch CPU Metrics](screenshots/cloudwatch_breaking_point_20users.png)

**From Performance Pattern**:

- 10x response time degradation despite 95% idle CPU
- 4x users → only 3x throughput (19→60 req/s)
- 0% error rate (stable but slow)

**Root Cause**: With CPU at only 5% but terrible performance, the bottleneck is **sync.Map lock contention**. At 20 concurrent users, goroutines spend time **waiting for lock access** (not executing on CPU). This is a classic concurrency bottleneck.

**Solution**: Horizontal scaling (multiple instances = less contention per instance) or code optimization (sharding, lock-free structures, Redis cache).

### Question 2: How much did response times degrade?

**Answer**: Response times degraded significantly between baseline and breaking point tests:

**Baseline Test (5 users, 2 minutes)**:

- Average Response Time: 2-7ms
- Median Response Time: 2ms
- Max Response Time: 60-98ms
- Throughput: ~19 requests/second
- Failure Rate: 0%

**Breaking Point Test (20 users, 3 minutes)**:

- Average Response Time: 49-55ms
- Median Response Time: 37-39ms
- Max Response Time: 158-222ms
- Throughput: ~60 requests/second
- Failure Rate: 0%

**Degradation Ratio**: Response time increased by approximately **10x** (from ~5ms average to ~53ms average)

Despite the significant response time degradation, the service maintained 0% error rate, indicating it handled the load without failing but with significantly reduced performance.

### Question 3: Could you solve this by doubling CPU (256 → 512)?

**To test this**:

```bash
cd terraform

# Update ECS CPU default in the module (256 -> 512)
sed -i '' 's/default     = "256"/default     = "512"/' modules/ecs/variables.tf

# Apply the change (this will update the current task definition)
terraform apply -auto-approve

sleep 60  # Wait for service to restart with new CPU

# Re-run the baseline test
locust \
  --host=http://$TASK_IP:8080 \
  --users=5 \
  --spawn-rate=1 \
  --run-time=120s \
  --headless \
  -f ../locustfile.py
```

**Answer**: **NO** - Doubling CPU would NOT help because CPU is already 95% idle (only 5.48% utilized)!

**Why CPU Increase Won't Work**:

- Current: 256 CPU units (0.25 vCPU), using only ~5%
- Doubling to 512 units would still leave you with 95% idle CPU
- The goroutines are **waiting** (blocked on locks), not **working** (using CPU)

**What WILL Work**:

1. **Horizontal scaling**: 2-4 task instances spreads load across independent sync.Map instances (less contention each)
2. **Code optimization**: Shard product catalog, use lock-free structures, or external cache (Redis)
3. **Async processing**: Reduce lock hold times in search logic

---

## OBSERVED RESULTS (with current config)

### Baseline (5 users, 256 CPU units = 0.25 vCPU):

**From Locust Logs**:

- ✅ Response Time: 2-7ms average, 2ms median
- ✅ Throughput: ~19 req/sec
- ✅ Errors: 0%

**From CloudWatch**:

- ✅ CPU: Very low (~1-2%)
- ✅ Status: **Healthy with ample spare capacity**

### Breaking Point (20 users, 256 CPU units = 0.25 vCPU):

**From Locust Logs**:

- ⚠️ Response Time: 49-55ms average, 37-39ms median (10x degradation)
- ⚠️ Throughput: ~60 req/sec (only 3x despite 4x users = sub-linear scaling)
- ✅ Errors: 0% (stable but slow)

**From CloudWatch**:

- ✅ CPU: **Only 5.48% maximum** (95% idle!)
- ✅ Memory: Well below capacity
- ⚠️ Status: **Lock contention bottleneck, NOT resource-bound**

**Critical Insights**:

1. **NOT CPU-bound**: CPU at 5% with terrible performance = goroutines waiting, not executing
2. **Lock contention**: sync.Map under high concurrency causes threads to block waiting for access
3. **Sub-linear scaling**: 4x users → only 3x throughput confirms concurrency bottleneck
4. **Stable architecture**: 0% errors despite degradation shows good error handling
5. **Solution**: Horizontal scaling distributes concurrent requests across multiple independent sync.Map instances, OR code optimization (sharding/Redis) eliminates shared lock

---

## CLEANUP (When Done)

```bash
cd terraform

# Destroy all AWS resources
terraform destroy

# Confirm by typing "yes" when prompted
# This will delete ECS resources, security group, and ECR repository.
```

---
