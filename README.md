# Scalable Distributed Systems

A collection of hands-on assignments covering distributed systems patterns, concurrency, cloud infrastructure, and load testing.

## Assignments

| Assignment | Focus                     | Key Concepts                                                                                         |
| ---------- | ------------------------- | ---------------------------------------------------------------------------------------------------- |
| **HW1a**   | Go Fundamentals           | Basic Go programming, module setup                                                                   |
| **HW1b**   | Performance Analysis      | Benchmarking, profiling, optimization                                                                |
| **HW2**    | Cloud Infrastructure      | Terraform, AWS (EC2, VPC, security groups), infrastructure-as-code                                   |
| **HW3**    | Concurrency Patterns      | Mutexes (RWMutex), sync.Map, atomic operations, context switching, thread safety                     |
| **HW4**    | MapReduce Framework       | Distributed computing, containerization (Docker), data processing pipeline                           |
| **HW5**    | Load Testing & API Design | Product API (Go), Locust load testing, HttpUser vs FastHttpUser comparison, lock contention analysis |

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

## Key Learnings

- **Concurrency**: RWMutex enables concurrent reads but exclusive writes create bottlenecks
- **Load Testing**: HttpUser vs FastHttpUser shows negligible difference when server-side locks are the constraint
- **Infrastructure**: Terraform enables reproducible cloud deployments
- **Distributed Computing**: MapReduce patterns scale data processing across multiple nodes

Each assignment builds on distributed systems principles: scalability, concurrency, fault tolerance, and performance optimization.
