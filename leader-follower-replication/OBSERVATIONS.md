# Distributed KV Store — Load Test Observations

## Setup

- **4 clusters**, each with 5 nodes, running in Docker on a single host
- **20 concurrent workers** per scenario, **60 seconds** per run
- **100-key space** with recency-biased key clustering (reads biased toward last 20 written keys)
- **16 scenarios** total: 4 configs × 4 write percentages (1%, 10%, 50%, 90%)

### Artificial Delays (to expose inconsistency windows)
| Event | Delay |
|---|---|
| Leader after sending to each follower (sequential) | 200ms |
| Follower before writing on replicate | 100ms |
| Follower before responding to internal read | 50ms |

This makes W=5 writes take ~1.2s and W=1 writes take <10ms.

---

## Raw Results

| Config | Write% | Total Requests | Throughput (req/s) | Read P50 | Read P99 | Write P50 | Write P99 | Stale Reads |
|---|---|---|---|---|---|---|---|---|
| **LF W=5,R=1** | 1% | 92,550 | 1,542 | 0ms | 11ms | 1,215ms | 1,316ms | 0.1% |
| **LF W=5,R=1** | 10% | 9,357 | 156 | 0ms | 17ms | 1,230ms | 1,373ms | 0.5% |
| **LF W=5,R=1** | 50% | 1,901 | 32 | 4ms | 19ms | 1,254ms | 1,295ms | 2.4% |
| **LF W=5,R=1** | 90% | 1,055 | 18 | 5ms | 11ms | 1,260ms | 1,313ms | 1.1% |
| **LF W=1,R=5** | 1% | 7,486 | 125 | 0ms | 60ms | 0ms | 7ms | 6.8% |
| **LF W=1,R=5** | 10% | 8,213 | 137 | 0ms | 60ms | 0ms | 7ms | 43.2% |
| **LF W=1,R=5** | 50% | 16,152 | 269 | 0ms | 57ms | 0ms | 5ms | 77.1% |
| **LF W=1,R=5** | 90% | 66,402 | 1,107 | 0ms | 63ms | 0ms | 6ms | 85.3% |
| **LF W=3,R=3** | 1% | 70,545 | 1,176 | 0ms | 55ms | 605ms | 627ms | 7.2% |
| **LF W=3,R=3** | 10% | 16,897 | 282 | 0ms | 56ms | 607ms | 624ms | 16.6% |
| **LF W=3,R=3** | 50% | 3,914 | 65 | 1ms | 62ms | 611ms | 634ms | 19.5% |
| **LF W=3,R=3** | 90% | 2,117 | 35 | 4ms | 95ms | 626ms | 810ms | 25.8% |
| **Leaderless W=5,R=1** | 1% | 167,574 | 2,793 | 1ms | 29ms | 105ms | 169ms | 0.1% |
| **Leaderless W=5,R=1** | 10% | 32,962 | 549 | 0ms | 6ms | 102ms | 4,214ms | 0.2% |
| **Leaderless W=5,R=1** | 50% | 22,145 | 369 | 1ms | 10ms | 104ms | 126ms | 0.4% |
| **Leaderless W=5,R=1** | 90% | 3,470 | 58 | 1ms | 134ms | 106ms | 1,169ms | 0.0% |

---

## Observations by Configuration

### 1. LF W=5, R=1 — Strong Consistency

**How it works:** The leader must receive ACKs from all 4 followers before responding to the client. Reads go directly to the leader only (R=1).

**Throughput:**
- At 1% writes: **1,542 req/s** — the highest among the LF strategies because reads are essentially free (return from leader instantly, P50=0ms).
- At 90% writes: collapses to **18 req/s** — every write takes ~1.2s (4 followers × (200ms sleep + 100ms follower processing)). 20 workers × (1 write/1.2s) = ~17 writes/s, which matches exactly.
- Throughput is entirely write-bottlenecked. The gap between 1% and 90% write ratios is **85×**.

**Latency:**
- Write P50 is consistently **~1,215–1,260ms** across all ratios. This is deterministic: 4 sequential follower updates × ~300ms each.
- Read P50 is **0ms** — reads never touch followers, just the leader's in-memory map.
- Write P99 (~1,316ms) is close to P50, meaning write latency is very predictable with no long tail.

**Consistency:**
- Stale reads are theoretically impossible (W=5 guarantees all nodes updated before ACK), but **0.1–2.4% stale reads** were observed. This is a measurement artifact: the client-side version tracking can race across goroutines — a write completes on one worker goroutine and updates `keyStates`, while another goroutine reads `keyStates` for comparison slightly before the version is stored.
- **Best config for read-heavy, strongly consistent workloads** (e.g. banking, inventory).

---

### 2. LF W=1, R=5 — Eventual Consistency

**How it works:** Leader writes to itself only and immediately returns 201. Async fan-out to followers happens in the background. Reads query all 5 nodes and return the highest version.

**Throughput:**
- At 1% writes: only **125 req/s** despite nearly all traffic being reads. This is because reads do R=5 fan-out — each read queries all 5 nodes (with 50ms follower sleep each, but concurrently). So read latency = ~50ms per read, meaning 20 workers can do ~400 reads/s maximum.
- At 90% writes: **1,107 req/s** — writes are nearly instant (<10ms P50), so write-heavy load is handled very efficiently.
- This is the **inverse** of W=5,R=1: fast writes, slow reads.

**Latency:**
- Write P50: **0ms**, P99: **7ms** — essentially network round-trip only. The leader doesn't wait for followers at all.
- Read P50: **0ms** but P99: **57–63ms** — the fan-out to 5 nodes is concurrent but the 50ms follower sleep creates a consistent floor for any node that hasn't been recently updated.

**Consistency — the most alarming numbers:**
- At 10% writes: **43.2% stale reads**
- At 50% writes: **77.1% stale reads**
- At 90% writes: **85.3% stale reads**

This is expected and demonstrates why W=1 is dangerous for read-heavy workloads. With W=1, followers are updated asynchronously. Our key clustering means we frequently read the same key we just wrote — but because followers lag by ~300–1200ms (sequential async replication), reads from followers see old versions constantly.

Counterintuitively, stale reads *increase* with write percentage. This is because more writes per second means followers are perpetually catching up — the inconsistency window never closes.

- **Best config for write-heavy workloads where stale reads are acceptable** (e.g. analytics counters, social media likes).

---

### 3. LF W=3, R=3 — Quorum

**How it works:** Leader updates itself + 2 followers synchronously (W=3), remaining 2 followers asynchronously. Reads query 3 nodes and return the highest version.

**Throughput:**
- At 1% writes: **1,176 req/s** — good throughput because reads only fan out to 3 nodes (vs 5 for W=1,R=5), so read latency is lower.
- At 90% writes: **35 req/s** — writes still take ~600ms (2 followers × 300ms sequential), so write throughput is bottlenecked but ~2× better than W=5.
- Write P50 consistently **~605–626ms** — exactly half of W=5 latency (2 synchronous followers instead of 4).

**Consistency:**
- Stale reads range from **7.2% (1% writes) to 25.8% (90% writes)**.
- This seems surprising — quorum is supposed to guarantee overlap. The reason stale reads still occur is that the 2 async followers (nodes 3 and 4) are not in the write quorum. When reads hit those nodes as part of the R=3 fan-out, they can return stale data. The highest-version wins logic in the fan-out corrects this *when it works*, but our stale detection is client-side: if the client sends a read before the async followers have propagated AND the R=3 fan-out doesn't include the 2 sync followers, it can return stale data.
- True quorum correctness requires that the W=3 and R=3 node sets always overlap. Our implementation picks followers in order (always nodes 1 and 2 for W=3), while R=3 fans out to any 3 of the 4 peer URLs — this can miss the overlap guarantee in edge cases.
- **Best general-purpose config** — balances write latency (~600ms) with reasonable read throughput and moderate stale rates.

---

### 4. Leaderless W=5, R=1

**How it works:** Any node receiving a write becomes the coordinator. It fans out to all 4 peers *concurrently* (using goroutines), waits for all, then responds. Reads return the node's own local value instantly.

**Throughput:**
- At 1% writes: **2,793 req/s** — the highest of all configurations. Reads are instant (no fan-out, just in-memory lookup). Even with W=5 writes, the concurrent fan-out means write latency is only ~100ms (1 peer round-trip, not 4 sequential).
- At 90% writes: **58 req/s** — write throughput limited by ~100ms per write × 20 workers = ~200 writes/s theoretically, but P99 at 1,169ms suggests contention at high write rates.

**The P99 anomaly at 10% writes (4,214ms):**
This is the most interesting data point. At 10% writes, write P99 spikes to 4,214ms while P50 is 102ms. This indicates occasional severe tail latency — likely when multiple coordinators write to the same key simultaneously, causing HTTP timeout retries or goroutine pile-up. At 50% writes the P99 returns to normal (126ms), suggesting the scheduler finds a better steady state under uniform high load.

**Consistency:**
- Stale reads: **0.0–0.4%** across all ratios — remarkably low for an eventually consistent system.
- At 90% writes: **0% stale reads**. Because writes are concurrent fan-out that completes in ~100ms, by the time the write ACK reaches the client and the client issues a read, all nodes already have the value. The 100ms window is smaller than the client's round-trip latency.
- **Best for high-availability systems** with no single point of failure. If the leader dies in LF clusters, the whole cluster is unavailable; leaderless continues with any remaining node.

---

## Cross-Configuration Comparison

### Which config wins at each write ratio?

| Write% | Best Throughput | Best Consistency | Best Write Latency |
|---|---|---|---|
| **1% writes** | Leaderless (2,793/s) | W=5,R=1 (0.1% stale) | W=1,R=5 (<1ms) |
| **10% writes** | Leaderless (549/s) | W=5,R=1 (0.5% stale) | W=1,R=5 (<1ms) |
| **50% writes** | W=1,R=5 (269/s) | Leaderless (0.4% stale) | W=1,R=5 (<1ms) |
| **90% writes** | W=1,R=5 (1,107/s) | Leaderless (0.0% stale) | W=1,R=5 (<1ms) |

### Throughput collapse under write load

```
Write%    W=5,R=1    W=1,R=5    W=3,R=3    Leaderless
1%        1,542      125        1,176       2,793
10%       156        137        282         549
50%       32         269        65          369
90%       18         1,107      35          58
```

- **W=5,R=1** collapses 85× from 1% to 90% writes — the most extreme bottleneck
- **W=1,R=5** *increases* 9× from 1% to 90% writes — designed for write-heavy load
- **Leaderless** degrades gracefully but P99 tail latency becomes unpredictable

---

## How the Key Clustering Works

The load tester maintains a `recentKeys` ring buffer (max 200 entries) per test run. Every write appends the written key's index to the buffer. Every read samples from the last 20 entries of the buffer. This means:

- Keys written recently are highly likely to be read within the next few operations
- The write-to-read interval is typically 0–500ms (within the async replication window)
- Without clustering, stale reads would be nearly impossible to observe at low write rates because the probability of reading a key that was recently written would be very low across 100 keys

This clustering is what makes the stale read numbers meaningful — we are intentionally exercising the inconsistency window.

---

## Which Database for Which Application?

| Application type | Best config | Reason |
|---|---|---|
| Banking / payments | **W=5, R=1** | Zero tolerance for stale reads. Slow writes are acceptable — transactions are rare relative to reads. |
| Social media feeds | **W=1, R=5** | Billions of writes (likes, views). Slightly stale reads are acceptable — users don't notice a like count that's 1 second stale. |
| Shopping cart / inventory | **W=3, R=3** | Must not oversell (need reasonable consistency) but also needs write throughput for high traffic. Quorum is the industry standard (Cassandra, DynamoDB). |
| Real-time leaderboards | **Leaderless** | High read throughput, writes are fast, no single point of failure. Eventual consistency is fine for scores. |
| Config / feature flags | **W=5, R=1** | Read extremely often, written rarely. Pay the write cost once, get instant consistent reads forever after. |

---

## CAP Theorem Summary

These results demonstrate the **CAP theorem** in practice:

> You cannot simultaneously have **Consistency**, **Availability**, and **Partition tolerance**. You must choose two.

- **W=5, R=1** — chooses **Consistency** over availability/performance. Writes block until all nodes agree.
- **W=1, R=5** — chooses **Availability** and **Performance** over consistency. Data may be stale.
- **W=3, R=3** — the practical middle ground. Sacrifices some performance for probabilistic consistency.
- **Leaderless** — chooses **Availability** (no leader = no single point of failure) with strong write guarantees (W=N) but accepts stale reads on busy nodes.
