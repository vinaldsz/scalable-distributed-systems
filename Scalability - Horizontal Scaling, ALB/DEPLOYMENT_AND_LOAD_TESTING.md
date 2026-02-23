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

![Docker Build Success](images/DockerRunning.png)

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

![Terraform Apply Success](images/TerraformApply.png)

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

![Search Endpoint Test](images/CurlElectronicsOuput.png)

### Step 9: Open CloudWatch Dashboard

Navigate to AWS Console:

1. Go to **CloudWatch** → **ECS** → **Cluster** → **Service** → **Metrics**

![CloudWatch Metrics Before Load Test](images/CloudWatchMetrics-BeforeLoadTest.png)

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
![Locust Baseline Test - 5 Users](images/LocustScenario1.png)

![CloudWatch Metrics - Baseline Test](images/CloudWatch-Locust5_2.png)

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
![Locust Breaking Point Test - 20 Users](images/LocustScenario2.png)

- CloudWatch metrics during peak load
  ![CloudWatch Metrics - Breaking Point Test](images/CloudWatch-Locust20_2.png)

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

![CloudWatch CPU Metrics](images/CloudWatch-Locust20_2.png)

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

# Part 3: Horizontal Scaling with ALB and Auto-Scaling

This guide walks through deploying your product search service with Application Load Balancer (ALB) and auto-scaling to handle the load that broke your system in Part 2.

## Architecture Overview

**Part 3 (Horizontal Scaling)**:

- 2-4 auto-scaled ECS tasks behind ALB
- ALB distributes load across healthy instances
- Each task has independent sync.Map (reduces contention)
- Auto-scaling based on 70% CPU target

---

## Step 1: Deploy Infrastructure

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw6/terraform-alb/

# Initialize Terraform
terraform init

# Validate configuration
terraform validate

# Preview changes
terraform plan

# Deploy (builds Docker image, creates ALB, ECS with 2 tasks, auto-scaling)
terraform apply -auto-approve
```

**Expected Duration**: 3-5 minutes

**What Gets Created**:

- ✅ ALB (Application Load Balancer) on port 80
- ✅ Target Group with /health checks (30s interval, 2 healthy threshold)
- ✅ 2 ECS tasks (256 CPU, 512 MB each)
- ✅ Auto-scaling: min 2, max 4, target 70% CPU
- ✅ Security groups (ALB → ECS on port 8080)
- ✅ CloudWatch logs and metrics

---

## Step 2: Verify ALB Setup

```bash
# Get ALB DNS name
ALB_DNS=$(terraform output -raw alb_dns_name)
echo "ALB URL: http://$ALB_DNS"

# Test health endpoint
curl "http://$ALB_DNS/health"
# Expected: {"status":"ok"}

# Test search endpoint
curl "http://$ALB_DNS/products/search?q=Electronics"
# Expected: JSON with match count and response time

# Check target group health (both tasks should be healthy)
aws elbv2 describe-target-health \
  --target-group-arn $(terraform output -raw target_group_arn) \
  --region us-west-2
# Expected: State: "healthy" for 2 targets
```

Outputs:

```bash
(.venv) vinaldsouza@Vinals-MacBook-Air terraform-alb % curl "http://product-search-alb-302286753.us-west-2.elb.amazonaws.com/products/search?q=Electronics"
{"products":[{"id":8,"name":"Product Alpha 8","category":"Electronics","description":"Pro Electronics item #8 — durable, reliable, and versatile.","brand":"Alpha"},{"id":16,"name":"Product Alpha 16","category":"Electronics","description":"Pro Electronics item #16 — durable, reliable, and versatile.","brand":"Alpha"},{"id":24,"name":"Product Alpha 24","category":"Electronics","description":"Pro Electronics item #24 — durable, reliable, and versatile.","brand":"Alpha"},{"id":32,"name":"Product Alpha 32","category":"Electronics","description":"Pro Electronics item #32 — durable, reliable, and versatile.","brand":"Alpha"},{"id":40,"name":"Product Alpha 40","category":"Electronics","description":"Pro Electronics item #40 — durable, reliable, and versatile.","brand":"Alpha"},{"id":48,"name":"Product Alpha 48","category":"Electronics","description":"Pro Electronics item #48 — durable, reliable, and versatile.","brand":"Alpha"},{"id":56,"name":"Product Alpha 56","category":"Electronics","description":"Pro Electronics item #56 — durable, reliable, and versatile.","brand":"Alpha"},{"id":64,"name":"Product Alpha 64","category":"Electronics","description":"Pro Electronics item #64 — durable, reliable, and versatile.","brand":"Alpha"},{"id":72,"name":"Product Alpha 72","category":"Electronics","description":"Pro Electronics item #72 — durable, reliable, and versatile.","brand":"Alpha"},{"id":80,"name":"Product Alpha 80","category":"Electronics","description":"Pro Electronics item #80 — durable, reliable, and versatile.","brand":"Alpha"},{"id":88,"name":"Product Alpha 88","category":"Electronics","description":"Pro Electronics item #88 — durable, reliable, and versatile.","brand":"Alpha"},{"id":96,"name":"Product Alpha 96","category":"Electronics","description":"Pro Electronics item #96 — durable, reliable, and versatile.","brand":"Alpha"}],"total_found":12,"search_time":"101.552µs"}
(.venv) vinaldsouza@Vinals-MacBook-Air terraform-alb %
(.venv) vinaldsouza@Vinals-MacBook-Air terraform-alb % curl "http://product-search-alb-302286753.us-west-2.elb.amazonaws.com/health"
{"status":"ok"}%
```

---

## Step 3: Open CloudWatch Dashboard

Before running load tests, open CloudWatch to monitor:

1. Go to AWS Console → CloudWatch → Metrics
2. Navigate to: ECS → Cluster → Service
3. Watch these metrics:
   - **CPUUtilization** (per task and average)
   - **MemoryUtilization**

   ![alt text](CloudWatchECSALB-before.png)

4. Navigate to: Application ELB → Target Group
   - **RequestCount**
   - **TargetResponseTime**
   - **HealthyHostCount**
     ![alt text](CloudWatchAppELB-before.png)

---

## Step 4: Run Baseline Test (5 users, 2 minutes)

```bash
cd /Users/vinaldsouza/Desktop/Scalable-Systems/scalable-distributed-systems/hw6/

# Run baseline test with ALB DNS
locust \
  --host=http://$ALB_DNS \
  --users=5 \
  --spawn-rate=1 \
  --run-time=120s \
  --headless \
  -f locustfile.py 2>&1 | tee part3_baseline_test.log
```

**Expected Results**:

- ✅ Response Time: ~80-120ms avg for `/products/search` (max spike ~17s on cold start)
- ✅ Throughput: ~14-16 req/s for `/products/search`
- ✅ CPU per task: Low-to-moderate (see CloudWatch)
- ✅ Errors: 0%
- ✅ Task count: 2 (no scale-out expected)

---

## Step 5: Run Breaking Point Test (20 users, 3 minutes)

```bash
sleep 30  # Let metrics stabilize

locust \
  --host=http://$ALB_DNS \
  --users=20 \
  --spawn-rate=2 \
  --run-time=180s \
  --headless \
  -f locustfile.py 2>&1 | tee part3_breaking_point_test.log
```

![alt text](CloudWatchMemoryCPU-After.png)

**Observed Results (20 users)**:

- ✅ Response Time: ~52-56ms avg for `/products/search`
- ✅ Throughput: ~60-63 req/s for `/products/search`
- ✅ Errors: 0%
- ✅ Auto-scaling: No scale-out observed at 20 users (CPU average stayed below target)

## Step 5: Run Extreme Breaking Point Test (1000 users, 50 spawn 6 minutes)

Initial Tasks
![alt text](ExtremeInitialTasks.png)

Tasks Increased
![alt text](ExtremeAfterTasks.png)

CPU and Memory Utilisation
![alt text](ExtremeAfter-CPUMemory.png)

**Observed Results (1000 users)**:

- ✅ Response Time: ~70-85ms avg for `/products/search`
- ✅ Throughput: ~2,600-2,750 req/s for `/products/search`
- ✅ Errors: 0%
- ✅ Auto-scaling: Scaled from 2 to 4 tasks (see screenshots)

---

## Step 6: Resilience Testing (Optional)

**While the 20-user test is running**, stop one task:

```bash
# In another terminal
CLUSTER_NAME=$(cd terraform-alb && terraform output -raw ecs_cluster_name)
TASK_ARN=$(aws ecs list-tasks --cluster $CLUSTER_NAME --region us-west-2 --query 'taskArns[0]' --output text)

# Stop the task
aws ecs stop-task --cluster $CLUSTER_NAME --task $TASK_ARN --region us-west-2

# Watch target health
watch -n 5 "aws elbv2 describe-target-health \
  --target-group-arn $(cd terraform-alb && terraform output -raw target_group_arn) \
  --region us-west-2 \
  --query 'TargetHealthDescriptions[].{Target:Target.Id,Health:TargetHealth.State}' \
  --output table"
```

**Observe**:

- ⏱️ Within 60s, ALB marks stopped task as "unhealthy"
- ✅ Load test continues successfully (ALB routes to healthy targets)
- 🔄 Auto-scaling + ECS launch replacement task to meet desired count
- ✅ New task becomes healthy and joins rotation

This demonstrates **horizontal scaling resilience**: individual instance failures don't impact service availability.

---

## Step 7: Compare Results with Part 2

| Metric                        | Part 2 (1 instance) | Part 3 (2-4 instances) | Improvement         |
| ----------------------------- | ------------------- | ---------------------- | ------------------- |
| **Baseline (5 users)**        |                     |                        |                     |
| Response Time                 | 2-7ms               | ~80-120ms              | Higher (cold start) |
| Throughput                    | ~19 req/s           | ~14-16 req/s           | Slightly lower      |
| CPU                           | 5% on 1 task        | Low-moderate (2 tasks) | Distributed         |
| **Breaking Point (20 users)** |                     |                        |                     |
| Response Time                 | 49-55ms             | ~52-56ms               | Similar             |
| Throughput                    | ~60 req/s           | ~60-63 req/s           | Similar             |
| CPU                           | 5.48% on 1 task     | Below target (2 tasks) | Not saturated       |
| Task Count                    | 1 (fixed)           | 2 (no scale-out)       | Stable              |
| Errors                        | 0%                  | 0%                     | Maintained          |
| **Extreme (1000 users)**      |                     |                        |                     |
| Response Time                 | N/A                 | ~70-85ms               | N/A                 |
| Throughput                    | N/A                 | ~2,600-2,750 req/s     | N/A                 |
| CPU                           | N/A                 | Above target           | Scaled out          |
| Task Count                    | N/A                 | 2 -> 4                 | Elastic             |
| Errors                        | N/A                 | 0%                     | Maintained          |

**Key Insight**:

- Part 2: Single instance with 20 concurrent users → severe lock contention on sync.Map → terrible performance despite idle CPU
- Part 3: Load distributed across 3-4 instances → each handles 5-7 concurrent users → minimal lock contention → good performance

---

## Step 8: Experiment

### Test Different Scaling Policies

Edit `terraform-alb/main.tf` and change auto-scaling parameters:

```terraform
module "autoscaling" {
  # ... existing ...
  min_capacity = 1  # Try 1 instead of 2
  max_capacity = 6  # Try 6 instead of 4
  cpu_target   = 50.0  # Try 50% instead of 70%
}
```

Then `terraform apply -auto-approve` and re-run tests to see the difference.

### Test Gradual Load Increase

Instead of jumping 0→20 users, try ramping gradually:

```bash
locust \
  --host=http://$ALB_DNS \
  --users=20 \
  --spawn-rate=0.5 \
  --run-time=300s \
  --headless \
  -f locustfile.py
```

Watch auto-scaling respond more smoothly to gradual load increase.

---

## Cleanup

```bash
cd terraform-alb/

# Destroy all resources
terraform destroy -auto-approve

# Confirm deletion:
# - ALB deleted
# - Target group deleted
# - ECS service and tasks stopped
# - Auto-scaling policies removed
# - CloudWatch logs (kept for retention period)
```

---

## Key Learnings

1. **Horizontal scaling solves concurrency bottlenecks**: Distributing load across multiple instances reduces lock contention per instance

2. **Not all bottlenecks are CPU/memory**: Part 2 had 95% idle CPU but terrible performance (lock contention)

3. **ALB provides resilience**: Individual task failures don't impact service availability

4. **Auto-scaling is reactive**: There's a delay (cooldown periods) between metric changes and scaling actions

5. **Right-sizing matters**: Starting with 2 tasks (baseline capacity) prevents initial performance hit during cold starts

---
