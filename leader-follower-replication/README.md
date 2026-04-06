# Distributed Key-Value Store — Leader-Follower & Leaderless

An in-memory distributed Key-Value store implementing the CAP theorem tradeoffs
through three Leader-Follower replication strategies and a Leaderless design.
Built in Go, deployed via Docker Compose and AWS EC2/ECR using Terraform.

---

## How to Run

### Local (Docker Compose)
```bash
docker compose up --build -d     # start all 20 containers
go test ./tests/ -v -timeout 120s  # run unit tests
./load_tester/run_load_tests.sh  # run all 16 load test scenarios
python3 analysis/generate_graphs.py  # generate graphs
```

### AWS
```bash
cd terraform && terraform init && terraform apply
cd .. && ./scripts/deploy.sh          # build + push to ECR
# SSH in and start containers (see deploy steps in OBSERVATIONS.md)
HOST=<ec2-ip> ./load_tester/run_load_tests.sh
```

---

## Architecture Overview

### One Binary, Four Roles

There is a single Go binary (`cmd/node`) that runs on every node. Its behavior
is entirely controlled by environment variables:

```
NODE_ROLE     = leader | follower | leaderless
WRITE_QUORUM  = W  (1, 3, or 5)
READ_QUORUM   = R  (1, 3, or 5)
PEER_URLS     = comma-separated HTTP URLs of all other nodes
NODE_ID       = 0-4
```

This means Docker Compose can spin up 20 identical containers from the same
image and configure them differently. The Dockerfile builds the binary once;
the role is injected at runtime.

### The Four Clusters

| Cluster | Ports | W | R | Strategy |
|---|---|---|---|---|
| LF W=5,R=1 | 8010–8014 | 5 | 1 | Strong consistency |
| LF W=1,R=5 | 8020–8024 | 1 | 5 | Eventual consistency |
| LF W=3,R=3 | 8030–8034 | 3 | 3 | Quorum |
| Leaderless | 8040–8044 | 5 | 1 | No leader, concurrent fan-out |

---

## Code Walkthrough

### `internal/store/store.go` — The KV Store

The store is a plain Go `map[string]Entry` protected by a `sync.RWMutex`.
Each entry holds a `(value, version)` pair where version is a monotonically
incrementing integer.

```go
type Entry struct {
    Value   string
    Version int
}
```

The `Set` method has two modes:
- **version == 0** (original write): auto-increments the current version
- **version > 0** (replication message): stores the exact version passed in

This is critical. When the leader replicates to a follower, it sends the
pre-assigned version number. The follower stores it directly without
re-incrementing. This ensures all nodes end up with the same version number
for the same write.

**Tricky bit:** The version assignment and store write happen inside the same
mutex lock. This prevents two concurrent writes from assigning the same version
number.

---

### `internal/config/config.go` — Configuration

Reads all environment variables at startup. Nothing happens at request time —
the config is read once and passed into the handler. This keeps request
handling fast and the configuration explicit.

---

### `internal/replication/replication.go` — The Fan-Out Logic

This is where the replication strategies differ.

#### Leader-Follower: Sequential Fan-Out (`LeaderReplicateSequential`)

```
for each peer (up to syncCount):
    POST /internal/replicate
    sleep 200ms   ← this is the intentional delay
```

The 200ms sleep after each send is **sequential** — the assignment says
"sleep 200ms after each message", not "sleep 200ms total". This makes W=5
writes take ~1.2 seconds (4 followers × 300ms each). This is intentional to
create an observable inconsistency window.

After sync replication to `W-1` peers, the rest are replicated asynchronously
via `LeaderReplicateAsync`, which fires a goroutine that does the same loop in
the background without blocking the client response.

#### Leaderless: Concurrent Fan-Out (`LeaderlessCoordinate`)

```
for each peer: go func() { POST /internal/replicate }()
wait for all goroutines
```

All peers are notified simultaneously. Total wait ≈ 100ms (one round-trip),
not 400ms sequential. This is why Leaderless W=5 writes are ~10× faster than
LF W=5 writes despite updating the same number of nodes.

#### Quorum Reads (`QuorumRead`)

For R>1 reads, the leader fans out to `R-1` peers via `/internal/read` using
goroutines (parallel), collects results, and returns the entry with the highest
version. This guarantees that even if one node has stale data, the most
recently written value is returned — as long as the write quorum and read
quorum sets overlap (which W+R > N guarantees when W=3, R=3, N=5).

---

### `internal/handler/handler.go` — HTTP Routes

Every node exposes the same six routes:

```
POST /set                  ← client writes
GET  /get/<key>            ← client reads
GET  /local_read/<key>     ← test-only: always returns own store value
POST /internal/replicate   ← node-to-node: receives a replication message
GET  /internal/read/<key>  ← node-to-node: used for quorum reads
GET  /health               ← Docker/load balancer health check
```

#### What happens when a write arrives at `/set`

**Leader node:**
1. Validate the key is not empty
2. Write to own store (assigns version via `store.Set(key, value, 0)`)
3. Calculate `syncCount = W - 1` (leader counts as 1)
4. Call `LeaderReplicateSequential(peers, syncCount, ...)` — blocks until done
5. Fire `LeaderReplicateAsync` for remaining peers (goroutine, non-blocking)
6. Return `201 Created` with `{key, value, version}`

**Leaderless node:**
1. Validate key
2. Write to own store
3. Call `LeaderlessCoordinate(peers, ...)` — blocks until all peers ACK
4. Return `201 Created`

**Follower node:**
- Returns `400 Bad Request` — clients must send writes to the leader directly

#### What happens when a read arrives at `/get/<key>`

**Leader with R=1:** Return from own store. No network calls. O(1).

**Leader with R>1 (quorum read):**
1. Read own store
2. Fan out to `R-1` peers via `/internal/read` concurrently
3. Collect responses (all goroutines run in parallel)
4. Return the entry with the highest version number

**Leaderless node (R=1):** Return own store value. No coordination.
This is where the inconsistency window lives — if a write coordinator hasn't
finished its fan-out yet, this node returns stale data.

**Follower node:** Return own store value. The client chose to read from this
follower directly. Stale reads are possible.

#### The internal endpoints

`POST /internal/replicate` — called by leader or leaderless coordinator:
- Sleeps **100ms** before writing (simulates disk/storage latency)
- Writes to store using the version from the message body (no re-increment)
- Returns `{ack: true, node_id, version}`

`GET /internal/read/<key>` — called by leader for quorum reads:
- Sleeps **50ms** before responding (simulates read latency)
- Returns own store value

These sleeps are what make the inconsistency window observable in tests.

---

### The Delays and Why They Matter

| Location | Sleep | Why |
|---|---|---|
| Leader after each follower send | 200ms | Simulates network + processing delay between message sends |
| Follower on `/internal/replicate` | 100ms | Simulates slow storage write |
| Follower on `/internal/read` | 50ms | Simulates slow storage read |

Without these delays, replication completes in <1ms and inconsistency windows
are nearly impossible to observe in tests. The delays are artificial but
representative of what happens in real geographically-distributed systems.

**Resulting windows:**
- W=5: ~1.2s write latency, followers 1–4 updated sequentially
- Leaderless W=5: ~100ms write latency, all followers updated in parallel
- The "sneaky" unit tests fire `local_read` during these windows

---

### `tests/` — Unit and Integration Tests

Tests run against a live Docker Compose cluster. They are **integration tests**,
not mocks — they send real HTTP requests to real containers.

Key tests:

**`TestLF_W5R1_FollowerReadConsistent`** — Proves W=5 guarantee works.
After the leader ACKs a write, every follower must have the value. This is
only possible because W=5 waited for all follower ACKs before returning.

**`TestLF_W1R5_SneakyInconsistency`** — The "gotcha" test.
Fires a W=1 write (returns instantly), then immediately calls `local_read`
on all followers. Followers should still be stale because async replication
hasn't completed. If all are up to date, the test logs a warning (replication
was unexpectedly fast) but does not fail.

**`TestLL_SneakyInconsistencyWindow`** — Leaderless inconsistency.
Starts a write in a goroutine, sleeps 20ms, then local_reads from peers.
The 100ms follower sleep means peers should still be stale after 20ms.

**`local_read` vs `get`** — This distinction is important.
`/get` goes through quorum logic (may fan out). `/local_read` always returns
the raw in-memory value of that specific node, bypassing quorum. It exists
only for testing — it would never be exposed in a real system.

**`waitForNodes`** — Polls `/health` on each node with a 30s per-node timeout.
Each node gets its own independent deadline, so a slow-starting container
doesn't starve other nodes of wait time. This was a bug in the original
implementation (shared deadline) that caused flaky tests.

---

### `cmd/loadtester/main.go` — Load Tester

The load tester is a Go program (not Locust) to avoid Python dependencies.

**Key clustering:** Each worker maintains a `recentKeys` ring buffer of the last
200 written key indices. Read operations sample from the last 20 entries. This
means reads are biased toward recently-written keys, creating "local in time"
clustering. Without this, stale reads at low write rates would be extremely
rare across a 100-key space.

**Stale detection:** The tester maintains a per-key `{version}` map
(`keyStates` array). After every write, it stores the returned version. On
every read, it compares the read version against the stored version. If
`read_version < stored_version`, it records a stale read.

**Output:** Each scenario writes a CSV with columns:
`op, latency_ms, stale, write_to_read_ms`

`write_to_read_ms` is the time elapsed since the last write to that key.
This is used to plot the distribution of how quickly reads follow writes.

---

### Error Handling Philosophy

- **Replication errors are silently dropped.** If a follower is unreachable,
  the leader continues. This is intentional: in a real distributed system,
  nodes fail, and the cluster must make progress. The ack count determines
  whether quorum was reached.
- **Client errors return JSON**, never plain text, so callers can always
  `json.Unmarshal` the response body.
- **No retries.** If a replication message fails, it is not retried. The
  follower will receive the next write's replication and catch up.
- **Followers don't redirect writes.** Clients must know the leader's address.
  In a real system you'd redirect, but for this assignment the load tester
  is configured with the correct endpoints.

---

## AI Disclosure

This project was developed with AI assistance (Claude). The following components
were AI-generated and have been reviewed and understood:

- **`internal/store/store.go`** — Standard Go map with RWMutex pattern.
  The version-assignment logic (auto-increment vs explicit version) was
  designed collaboratively.
- **`internal/replication/replication.go`** — The sequential vs concurrent
  fan-out distinction is the core architectural decision. Sequential loop with
  `time.Sleep` for LF; `sync.WaitGroup` + goroutines for leaderless.
- **`internal/handler/handler.go`** — Route dispatch logic. The `syncCount = W-1`
  calculation (leader counts as 1 toward quorum) is a subtle but important detail.
- **`cmd/loadtester/main.go`** — The key clustering via `recentKeys` ring buffer
  and client-side stale detection via `keyStates` array.
- **`terraform/`** — Standard EC2 + ECR pattern matching prior assignments.
- **`tests/`** — Integration test structure. The probabilistic "sneaky" tests
  (log rather than fail) are a deliberate design choice to handle timing variance.

All code has been read, understood, and is explainable line by line.

---

## File Structure

```
leader-follower-replication/
├── cmd/
│   ├── node/main.go          ← server entry point
│   └── loadtester/main.go    ← load test client
├── internal/
│   ├── store/store.go        ← thread-safe KV store with versioning
│   ├── config/config.go      ← env var parsing
│   ├── replication/replication.go  ← fan-out strategies
│   └── handler/handler.go    ← HTTP routes and request logic
├── tests/
│   ├── helpers_test.go       ← shared test utilities
│   ├── leader_follower_test.go
│   └── leaderless_test.go
├── node/Dockerfile           ← multi-stage Go build
├── docker-compose.yml        ← 20 containers, 4 clusters
├── terraform/                ← EC2 + ECR on AWS
├── load_tester/
│   └── run_load_tests.sh     ← runs all 16 scenarios
├── analysis/
│   ├── generate_graphs.py    ← produces 5 graph types from CSVs
│   ├── results/              ← CSV output from load tests
│   └── graphs/               ← PNG graph outputs
├── OBSERVATIONS.md           ← full results analysis
└── README.md                 ← this file
```
