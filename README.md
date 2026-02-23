# Scalable Distributed Systems

A collection of hands-on assignments covering distributed systems patterns, concurrency, cloud infrastructure, and load testing.

## Assignments

| Assignment   | Focus                     | Key Concepts                                                                                         |
| ------------ | ------------------------- | ---------------------------------------------------------------------------------------------------- |
| **Topic 1**  | Go Fundamentals           | Basic Go programming, module setup                                                                   |
| **Topic 1b** | Performance Analysis      | Benchmarking, profiling, optimization                                                                |
| **Topic 2**  | Cloud Infrastructure      | Terraform, AWS (EC2, VPC, security groups), infrastructure-as-code                                   |
| **Topic 3**  | Concurrency Patterns      | Mutexes (RWMutex), sync.Map, atomic operations, context switching, thread safety                     |
| **Topic 4**  | MapReduce Framework       | Distributed computing, containerization (Docker), data processing pipeline                           |
| **Topic 5**  | Load Testing & API Design | Product API (Go), Locust load testing, HttpUser vs FastHttpUser comparison, lock contention analysis |
| **Topic 6**  | Horizontal Scaling        | ECS Fargate, ALB, auto-scaling policies, CloudWatch metrics, load testing at scale                   |

## Quick Start

**HW5 - Product API with Load Testing** (Latest)

```bash
cd hw5

# Start API and load tester
docker-compose up

# Run automated stress tests
chmod +x automated_tests.sh
./automated_tests.sh

# View results
cat STRESS_TEST_RESULTS.md
```

**HW6 - Horizontal Scaling with ALB & Auto-Scaling**

- Deploys the product-search service to ECS Fargate behind an ALB with auto-scaling.
- Uses CloudWatch metrics to observe CPU and scaling behavior under load.
- Guides: [hw6/DEPLOYMENT_AND_LOAD_TESTING.md](hw6/DEPLOYMENT_AND_LOAD_TESTING.md), [hw6/images/PART3_DEPLOYMENT_GUIDE.md](hw6/images/PART3_DEPLOYMENT_GUIDE.md)

## Key Learnings

- **Concurrency**: RWMutex enables concurrent reads but exclusive writes create bottlenecks
- **Load Testing**: HttpUser vs FastHttpUser shows negligible difference when server-side locks are the constraint
- **Infrastructure**: Terraform enables reproducible cloud deployments
- **Distributed Computing**: MapReduce patterns scale data processing across multiple nodes
- **Auto-Scaling**: ALB + ECS scale-out handles burst load and improves resilience

Each assignment builds on distributed systems principles: scalability, concurrency, fault tolerance, and performance optimization.
