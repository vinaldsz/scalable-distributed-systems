# Distributed KV Store: Leader-Follower + Leaderless

## Architecture Decisions

### Version Numbers (Logical Clocks)
Each KV entry is a `(value, version)` pair. Version is a monotonically incrementing integer assigned by the write coordinator (leader in LF, receiving node in leaderless). Versions start at 0 (absent). Replication messages carry the pre-assigned version so followers store it directly without re-incrementing.

### Delay Model (creates testable inconsistency windows)
| Event | Sleep |
|---|---|
| Leader after sending to each follower (sequential) | 200ms |
| Follower before responding to replicate | 100ms |
| Follower before responding to internal read | 50ms |

W=5 write latency = 4 × (200ms + 100ms) ≈ **1.2 seconds** — intentionally slow.

### Replication Strategies
| Strategy | Write | Read | Behavior |
|---|---|---|---|
| W=5, R=1 | Sequential to all 4 followers, 200ms sleep after each | From leader only | Strongly consistent writes |
| W=1, R=5 | Write to self only, async fan-out to followers | All 5 nodes, return highest version | Eventually consistent writes |
| W=3, R=3 | Sequential to 2 followers + self, async rest | 3 nodes, return highest version | Quorum |
| Leaderless W=5, R=1 | Concurrent fan-out to all peers, wait for all | Return own value | Write coordinator per request |

### Critical Implementation Notes
1. **Flask must run with `--threaded`** — leader makes outbound HTTP calls during request handling; single-threaded Flask deadlocks.
2. **Sequential fan-out for LF leader** — assignment says "sleep 200ms after each message" meaning sequential, NOT concurrent.
3. **Concurrent fan-out for leaderless** — use `ThreadPoolExecutor`, total wait ~100ms not 400ms.
4. **Version increment is atomic with write** — acquire lock, increment, store, release — then replicate with pre-assigned version.
5. **W=1 must still replicate async** — fire-and-forget `threading.Thread` before returning 201, or R=5 reads will always be stale.

---

## File Structure

```
leader-follower-replication/
├── PLAN.md                        <- this file
├── README.md
├── docker-compose.yml             <- 20 services: 3×LF clusters + 1 leaderless
│
├── node/                          <- single Flask app, configured by env vars
│   ├── Dockerfile
│   ├── requirements.txt
│   └── src/
│       ├── app.py                 <- Flask routes
│       ├── store.py               <- In-memory KV store with version tracking
│       ├── replication.py         <- Fan-out logic (LF sequential, leaderless concurrent)
│       └── config.py              <- Reads NODE_ROLE, PEER_URLS, W, R from env
│
├── tests/
│   ├── helpers.py                 <- wait_for_nodes, set_key, get_key, local_read
│   ├── test_leader_follower.py    <- Unit tests for all 3 LF strategies
│   └── test_leaderless.py         <- Unit tests for leaderless + inconsistency window
│
├── load_tester/
│   ├── config.py                  <- Key space size, ratios, cluster URLs
│   ├── locustfile.py              <- Locust tasks with stale-read detection + key clustering
│   └── run_load_tests.sh          <- Runs all 16 scenarios (4 configs × 4 ratios)
│
└── analysis/
    ├── generate_graphs.py         <- Reads CSVs → 5 graph types
    ├── results/                   <- CSV output from Locust (gitignored)
    └── graphs/                    <- PNG outputs
```

---

## API Contract

All responses are JSON.

```
POST /set                 Body: {"key": "k1", "value": "abc"}
                          201: {"key": "k1", "value": "abc", "version": 3}
                          400: {"error": "key cannot be empty"}

GET  /get/<key>           200: {"key": "k1", "value": "abc", "version": 3}
                          404: {"error": "key not found"}

GET  /local_read/<key>    200: {"key": "k1", "value": "abc", "version": 3, "node_id": 2}
                          404: {"error": "key not found"}

POST /internal/replicate  Body: {"key": "k1", "value": "abc", "version": 3}
                          [sleeps 100ms before writing]
                          200: {"ack": true, "node_id": 2, "version": 3}

GET  /internal/read/<key> [sleeps 50ms before responding]
                          200: {"key": "k1", "value": "abc", "version": 3, "node_id": 2}
                          404: {"error": "key not found"}

GET  /health              200: {"status": "ok", "node_id": 2, "role": "leader"}
```

---

## Environment Variables

| Variable | Example | Description |
|---|---|---|
| `NODE_ROLE` | `leader` / `follower` / `leaderless` | Role of this node |
| `NODE_ID` | `0` | Integer 0-4 |
| `PEER_URLS` | `http://lf1-n1:5000,http://lf1-n2:5000,...` | Comma-separated peer URLs (excludes self) |
| `WRITE_QUORUM` | `5` | W value |
| `READ_QUORUM` | `1` | R value |
| `LEADER_URL` | `http://lf1-leader:5000` | (Followers only) leader URL for redirects |

---

## Docker Port Mapping

| Cluster | Service | Host Port |
|---|---|---|
| LF W=5,R=1 | leader | 8010 |
| LF W=5,R=1 | node1-4 | 8011-8014 |
| LF W=1,R=5 | leader | 8020 |
| LF W=1,R=5 | node1-4 | 8021-8024 |
| LF W=3,R=3 | leader | 8030 |
| LF W=3,R=3 | node1-4 | 8031-8034 |
| Leaderless | node0-4 | 8040-8044 |

---

## Implementation Order

1. `node/src/store.py` — KV store with versioning (no deps)
2. `node/src/config.py` — env var parsing (no deps)
3. `node/src/replication.py` — fan-out logic
4. `node/src/app.py` — Flask routes (depends on 1-3)
5. `node/Dockerfile` + `requirements.txt`
6. `docker-compose.yml` — spin up all 20 nodes, verify `/health`
7. `tests/helpers.py` — shared test utilities
8. `tests/test_leader_follower.py`
9. `tests/test_leaderless.py`
10. `load_tester/config.py` + `load_tester/locustfile.py`
11. `load_tester/run_load_tests.sh`
12. `analysis/generate_graphs.py`
13. `README.md`

---

## Unit Test Cases

### Leader-Follower Tests

| Test | What it proves |
|---|---|
| `test_lf_w5r1_leader_read_consistent` | set → get from leader → same value |
| `test_lf_w5r1_follower_read_after_ack` | set (W=5) → local_read from follower → consistent (leader waited for all) |
| `test_lf_w1r5_sneaky_inconsistency` | set (W=1) → immediately local_read from all followers → at least 1 stale |
| `test_lf_w1r5_read_all_returns_latest` | set (W=1) → /get (R=5 quorum read) → returns latest version |
| `test_lf_w3r3_quorum_consistent` | set (W=3) → /get (R=3) → returns latest; non-quorum follower may be stale |
| `test_version_monotonic` | write same key 3 times → versions are 1, 2, 3 |

### Leaderless Tests

| Test | What it proves |
|---|---|
| `test_ll_write_ack_means_all_nodes_updated` | set to node0 → after 201 → local_read all nodes → consistent |
| `test_ll_sneaky_inconsistency_during_write` | start set in background thread → immediately local_read others → stale |
| `test_ll_coordinator_read_immediately_consistent` | set to node0 → read from node0 → consistent |

---

## Load Testing

### Key Clustering Strategy
Each Locust user maintains a `recent_keys` deque (max 50) from a global key space of `k0`-`k99`. Writes append to the deque; reads sample from the last 20 entries. This ensures reads and writes to the same key are clustered in time.

### Stale Read Detection
Client-side dict `{key: last_written_version}` updated on every write. On each read, if `response.version < client_versions[key]` → stale read. Log to `stale_events.csv` with `{key, write_ts, read_ts, config, ratio}`.

### Scenarios (16 total)
4 configs × 4 ratios (1/99, 10/90, 50/50, 90/10)
Each run: 60 seconds, 20 users, spawn rate 5

---

## Graphs to Produce

1. **Latency CDF** — Reads vs Writes per config (4 subplots, shows long tail)
2. **P50/P95/P99 Bar Chart** — Config comparison for reads and writes
3. **Stale Read Rate** — % stale reads by config × R/W ratio
4. **Time Interval Histogram** — Write-to-read interval for same key per config
5. **Throughput vs R/W Ratio** — One line per config

---

## Tricky Bits for the Report

- **W=5 write latency is ~1.2s** because it is sequential: 4 followers × (200ms leader sleep + 100ms follower sleep). This is a design choice to make the window observable.
- **W=1 R=5 reads are slower than W=5 R=1 reads** because R=5 fans out to all nodes (with 50ms each) and picks the max version.
- **Leaderless inconsistency window is ~100ms** (concurrent fan-out, each peer sleeps 100ms). The sneaky test must fire within this window.
- **Quorum (W=3, R=3) guarantees overlap** — any 3-node write set and any 3-node read set must share at least 1 node (since 3+3 > 5). That node will have the latest version.
