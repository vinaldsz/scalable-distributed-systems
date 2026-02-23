# Product Search Service - Load Testing & Bottleneck Analysis

## Objective

Deploy a product search service and use load testing to discover its breaking point. The key question: **When your service slows down, how do you know if you need better code or just more servers?**

## Architecture

### Service Specification

- **Platform**: Go service running on ECS Fargate
- **Initial Configuration**: 1 instance with 256 CPU units (0.25 vCPU) and 512 MB memory
- **Data**: 100,000 products stored in memory using `sync.Map`
- **Search Logic**: Checks exactly 100 products per request (simulates fixed computation time)

### Product Data Structure

```json
{
  "id": 1,
  "name": "Product Alpha 1",
  "category": "Electronics",
  "description": "High-quality product",
  "brand": "Alpha"
}
```

### API Endpoints

- `GET /products/search?q={query}` - Search products by name or category
- `GET /health` - Health check endpoint

### Response Format

```json
{
  "products": [...],       // Max 20 results
  "total_found": 12,       // Total matches found
  "search_time": "1.2ms"   // Search duration
}
```

## Key Implementation Details

### Search Limitations

- **Critical requirement**: Each search checks exactly 100 products then stops
- This simulates real-world scenarios where:
  - Processing fixed data volume (e.g., first 100 product embeddings)
  - Running computationally expensive operations (AI inference, video processing)
  - Can't optimize the algorithm further - need more compute power

### Thread Safety

- Uses `sync.Map` for concurrent access
- No locks needed for reads - excellent for high-concurrency scenarios
- Queries can run in parallel without serialization

### Data Generation

- 100,000 products generated at startup
- Uses modulo operator for consistent, predictable data:
  - Names: "Product {Brand} {ID}"
  - Categories: Rotate through 8 categories
  - Brands: Rotate through 8 brands
- Ensures consistent behavior for repeatable load tests

## Quick Start

### Prerequisites

```bash
# macOS with Homebrew
brew install docker terraform go python3

# Python dependencies for load testing
pip install locust requests
```

### Local Testing without docker

1. **Initialise and run the go file** :
   `go mod init`
   ` go run main.go`

Output:

```
2026/02/22 15:14:12 Generating product catalog...
2026/02/22 15:14:12 Generated 100000 products
2026/02/22 15:14:12 Listening on :8080
```

2. Test the response using curl

```bash
curl "http://localhost:8080/products/search?q=Electronics"
{"products":[{"id":8,"name":"Product Alpha 8","category":"Electronics","description":"Pro Electronics item #8 — durable, reliable, and versatile.","brand":"Alpha"},{"id":16,"name":"Product Alpha 16","category":"Electronics","description":"Pro Electronics item #16 — durable, reliable, and versatile.","brand":"Alpha"},{"id":24,"name":"Product Alpha 24","category":"Electronics","description":"Pro Electronics item #24 — durable, reliable, and versatile.","brand":"Alpha"},{"id":32,"name":"Product Alpha 32","category":"Electronics","description":"Pro Electronics item #32 — durable, reliable, and versatile.","brand":"Alpha"},{"id":40,"name":"Product Alpha 40","category":"Electronics","description":"Pro Electronics item #40 — durable, reliable, and versatile.","brand":"Alpha"},{"id":48,"name":"Product Alpha 48","category":"Electronics","description":"Pro Electronics item #48 — durable, reliable, and versatile.","brand":"Alpha"},{"id":56,"name":"Product Alpha 56","category":"Electronics","description":"Pro Electronics item #56 — durable, reliable, and versatile.","brand":"Alpha"},{"id":64,"name":"Product Alpha 64","category":"Electronics","description":"Pro Electronics item #64 — durable, reliable, and versatile.","brand":"Alpha"},{"id":72,"name":"Product Alpha 72","category":"Electronics","description":"Pro Electronics item #72 — durable, reliable, and versatile.","brand":"Alpha"},{"id":80,"name":"Product Alpha 80","category":"Electronics","description":"Pro Electronics item #80 — durable, reliable, and versatile.","brand":"Alpha"},{"id":88,"name":"Product Alpha 88","category":"Electronics","description":"Pro Electronics item #88 — durable, reliable, and versatile.","brand":"Alpha"},{"id":96,"name":"Product Alpha 96","category":"Electronics","description":"Pro Electronics item #96 — durable, reliable, and versatile.","brand":"Alpha"}],"total_found":12,"search_time":"461.792µs"}
```

3. Check the health

```bash
curl "http://localhost:8080/health"
{"status":"ok"}%
```

### Local Testing with docker

1. **Start the service locally**:

```bash
docker-compose up --build
```

2. **Test the service**:

```bash
# Health check
curl http://localhost:8080/health

# Search for products
curl "http://localhost:8080/products/search?q=Electronics"
```

3. **Run basic performance tests**:

```bash
chmod +x test_local.sh
./test_local.sh
```

4. **Run comprehensive load testing**:

```bash
chmod +x run_load_tests.sh
./run_load_tests.sh
```

### AWS Deployment

1. **Build and push Docker image**:

```bash
cd src
docker build -t product-search:latest .
# Tag and push to ECR (your account ID and region)
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin [YOUR_ACCOUNT_ID].dkr.ecr.us-east-1.amazonaws.com
docker tag product-search:latest [YOUR_ACCOUNT_ID].dkr.ecr.us-east-1.amazonaws.com/product-search:latest
docker push [YOUR_ACCOUNT_ID].dkr.ecr.us-east-1.amazonaws.com/product-search:latest
```

2. **Update Terraform variables**:

```bash
cd terraform
cat > terraform.tfvars << EOF
aws_region      = "us-east-1"
container_image = "[YOUR_ACCOUNT_ID].dkr.ecr.us-east-1.amazonaws.com/product-search:latest"
desired_count   = 1
min_capacity    = 1
max_capacity    = 5
task_cpu        = "256"
task_memory     = "512"
EOF
```

3. **Deploy with Terraform**:

```bash
terraform init
terraform plan
terraform apply
```

4. **Get the load balancer DNS**:

```bash
terraform output alb_dns_name
```

## Load Testing Strategy

### Test Phases

Progressive load testing to identify the breaking point:

| Phase | Users | Spawn Rate | Duration | Expected Result           |
| ----- | ----- | ---------- | -------- | ------------------------- |
| 1     | 10    | 1/sec      | 5 min    | Service handles easily    |
| 2     | 20    | 2/sec      | 5 min    | Still healthy             |
| 3     | 50    | 5/sec      | 5 min    | CPU starts rising         |
| 4     | 100   | 10/sec     | 5 min    | CPU at limit              |
| 5     | 200   | 20/sec     | 5 min    | Breaking point identified |

### Metrics Collected

**From Locust (load generator)**:

- Requests per second
- Response time statistics (min, max, avg, median, p95, p99)
- Error rate
- Failure types

**From CloudWatch (ECS)**:

- CPU Utilization (%)
- Memory Utilization (%)
- Active connection count
- Target response time

**From Container Stats**:

- Docker resource usage
- Process-level CPU/memory
- Network I/O

## Interpreting Results: Code vs Infrastructure

### Bottleneck Decision Framework

| CPU      | Memory   | Latency            | Throughput    | Diagnosis              | Action                    |
| -------- | -------- | ------------------ | ------------- | ---------------------- | ------------------------- |
| Low      | Low      | Low                | Increasing    | Healthy                | Nothing needed            |
| **High** | Low      | **Stable**         | Increasing    | **CPU bound**          | **SCALE** (add instances) |
| Low      | **High** | **Rising**         | **Falling**   | **Memory pressure**    | **OPTIMIZE** (code)       |
| **High** | **High** | **Rising**         | **Plateaued** | **Hit limit**          | **SCALE**                 |
| Mid      | Mid      | **Rising sharply** | **Flat**      | **Queuing bottleneck** | **SCALE**                 |

### Evidence Collection

**SCALE Hypothesis** (need more compute):

- ✅ CPU utilization approaches 100%
- ✅ Response time remains reasonable
- ✅ Doubling instances doubles throughput
- ✅ Memory usage stays under 70%

**OPTIMIZE Hypothesis** (code inefficiency):

- ✅ Low CPU but high latency
- ✅ Memory usage grows unexpectedly
- ✅ Goroutines or connections leak
- ✅ Thread contention causes context switching

### Expected Behavior with Current Architecture

**Prediction**: With 100-product limit and `sync.Map`, bottleneck is **CPU** at ~50-100 concurrent users.

**Rationale**:

- 100 product iterations per request = fixed CPU work
- `sync.Map` has minimal locking overhead
- No algorithmic improvements possible
- Must scale horizontally (add instances)

**Expected Results**:

- ✅ Phase 1-3: CPU <50%, response times <10ms
- ✅ Phase 4: CPU 70-80%, latency starts rising
- ✅ Phase 5: CPU at 100%, latency >100ms, errors appear
- ✅ With 2 instances: Phase 4 moves to 100 users

## Files Overview

```
hw6/
├── src/
│   ├── main.go              # Product search service (184 lines)
│   ├── go.mod               # Go module definition
│   └── Dockerfile           # Multi-stage build
├── docker-compose.yml       # Local dev environment
├── locustfile.py            # Load testing scenarios
├── terraform/
│   ├── main.tf              # ECS, ALB, networking, autoscaling
│   ├── variables.tf         # Configurable parameters
│   ├── outputs.tf           # Output values
│   └── provider.tf          # Terraform provider config
├── run_load_tests.sh        # Progressive load testing
├── test_local.sh            # Basic local performance test
├── analyze_bottleneck.sh    # Analysis framework
└── README.md                # This file
```

## Code Overview

### Key Components

**Product Generation** (main.go:80-100):

- Creates 100,000 products at startup
- Consistent naming: "Product {Brand} {ID}"
- Rotates through 8 brands and 8 categories

**Search Function** (main.go:135-165):

- Accepts query parameter `q`
- Iterates through products (max 100)
- Searches name and category case-insensitively
- Returns up to 20 results with total count

**Thread Safety**:

```go
// Using sync.Map for lock-free reads
store.products.Range(func(key, value interface{}) bool {
    // ... process products
    return true
})
```

## Real-World Application

This exercise demonstrates real problems:

1. **Fixed Computation Work**: Not all performance issues can be solved by optimization
   - AI model inference (fixed latency)
   - Video frame processing (N GB/sec limit)
   - Complex calculations (math bound)

2. **Recognizing When to Scale**:
   - Monitor: CPU, memory, latency, throughput
   - Before hitting the wall, predict and scale
   - Auto-scaling policies prevent manual intervention

3. **Cost-Effective Decisions**:
   - Optimize code when ROI is high
   - Scale infrastructure when bottleneck is resource bound
   - Combination of both strategies

## Success Criteria

✅ Service deploys successfully to ECS Fargate  
✅ Generates 100,000 products at startup  
✅ Handles multiple concurrent requests with sync.Map  
✅ Load tests identify clear breaking point  
✅ Metrics show CPU as bottleneck (not memory or locks)  
✅ Scaling to 2 instances improves throughput  
✅ Evidence collected demonstrates infrastructure vs code tradeoff

## Troubleshooting

### Local service won't start

```bash
# Check Docker
docker ps -a
docker logs [container_name]

# Check port conflicts
lsof -i :8080
```

### Locust not working

```bash
# Install/upgrade
pip install --upgrade locust

# Test connectivity
curl http://localhost:8080/health
```

### Terraform issues

```bash
# Validate configuration
terraform validate

# Check AWS credentials
aws sts get-caller-identity
```

## References

- [Sync.Map Documentation](https://pkg.go.dev/sync)
- [Locust Testing Framework](https://locust.io/)
- [AWS ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/)
- [CloudWatch Metrics](https://docs.aws.amazon.com/AmazonCloudWatch/latest/)
