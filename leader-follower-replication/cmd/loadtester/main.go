// Load tester for the distributed KV store.
//
// Usage:
//
//	go run ./cmd/loadtester \
//	  -config lf1 \
//	  -write-pct 10 \
//	  -duration 60s \
//	  -concurrency 20 \
//	  -host localhost \
//	  -out analysis/results/lf1_w10.csv
//
// Configs: lf1 (W=5,R=1)  lf2 (W=1,R=5)  lf3 (W=3,R=3)  ll (leaderless)
// Set -host to your EC2 public IP to run against AWS.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// ─── cluster definitions ─────────────────────────────────────────────────────

type cluster struct {
	writeURL   string   // where to send writes (leader, or any node for leaderless)
	readURLs   []string // where reads can go
	leaderless bool
}

func buildClusters(host string) map[string]cluster {
	u := func(port int) string { return fmt.Sprintf("http://%s:%d", host, port) }
	return map[string]cluster{
		"lf1": {
			writeURL: u(8010),
			readURLs: []string{u(8010), u(8011), u(8012), u(8013), u(8014)},
		},
		"lf2": {
			writeURL: u(8020),
			readURLs: []string{u(8020), u(8021), u(8022), u(8023), u(8024)},
		},
		"lf3": {
			writeURL: u(8030),
			readURLs: []string{u(8030), u(8031), u(8032), u(8033), u(8034)},
		},
		"ll": {
			leaderless: true,
			readURLs:   []string{u(8040), u(8041), u(8042), u(8043), u(8044)},
		},
	}
}

// ─── per-key client state ─────────────────────────────────────────────────────

const keySpaceSize = 100

type keyState struct {
	mu          sync.Mutex
	version     int
	lastWriteTS time.Time
}

var (
	keyStates  [keySpaceSize]keyState
	recentKeys []int // ring buffer of recently-written key indices
	recentMu   sync.Mutex
)

func pushRecent(idx int) {
	recentMu.Lock()
	recentKeys = append(recentKeys, idx)
	if len(recentKeys) > 200 {
		recentKeys = recentKeys[len(recentKeys)-200:]
	}
	recentMu.Unlock()
}

// pickKey returns a key index biased toward recently-written keys for reads.
func pickKey(isRead bool) int {
	recentMu.Lock()
	defer recentMu.Unlock()
	if isRead && len(recentKeys) > 0 {
		pool := recentKeys
		if len(pool) > 20 {
			pool = pool[len(pool)-20:]
		}
		return pool[rand.Intn(len(pool))]
	}
	return rand.Intn(keySpaceSize)
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 10 * time.Second}

type setResp struct {
	Version int    `json:"version"`
	Error   string `json:"error"`
}
type getResp struct {
	Value   string `json:"value"`
	Version int    `json:"version"`
	Error   string `json:"error"`
}

func doSet(url, key, value string) (int, time.Duration, error) {
	body, _ := json.Marshal(map[string]string{"key": key, "value": value})
	start := time.Now()
	resp, err := httpClient.Post(url+"/set", "application/json", bytes.NewReader(body))
	lat := time.Since(start)
	if err != nil {
		return 0, lat, err
	}
	defer resp.Body.Close()
	var r setResp
	json.NewDecoder(resp.Body).Decode(&r)
	if resp.StatusCode != http.StatusCreated {
		return 0, lat, fmt.Errorf("set status %d", resp.StatusCode)
	}
	return r.Version, lat, nil
}

func doGet(url, key string) (string, int, time.Duration, error) {
	start := time.Now()
	resp, err := httpClient.Get(fmt.Sprintf("%s/get/%s", url, key))
	lat := time.Since(start)
	if err != nil {
		return "", 0, lat, err
	}
	defer resp.Body.Close()
	var r getResp
	json.NewDecoder(resp.Body).Decode(&r)
	if resp.StatusCode == http.StatusNotFound {
		return "", 0, lat, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, lat, fmt.Errorf("get status %d", resp.StatusCode)
	}
	return r.Value, r.Version, lat, nil
}

// ─── result record ─────────────────────────────────────────────────────────────

type record struct {
	op          string  // "read" | "write"
	latencyMs   float64
	stale       bool
	writeToRead float64 // ms since last write to this key; -1 if unknown
}

// ─── worker ───────────────────────────────────────────────────────────────────

func worker(cfg cluster, writePct int, duration time.Duration, results chan<- record, wg *sync.WaitGroup) {
	defer wg.Done()
	deadline := time.Now().Add(duration)

	for time.Now().Before(deadline) {
		isWrite := rand.Intn(100) < writePct

		if isWrite {
			idx := pickKey(false)
			key := fmt.Sprintf("k%d", idx)
			value := fmt.Sprintf("v%d-%d", idx, time.Now().UnixNano())

			writeURL := cfg.writeURL
			if cfg.leaderless {
				writeURL = cfg.readURLs[rand.Intn(len(cfg.readURLs))]
			}

			version, lat, err := doSet(writeURL, key, value)
			if err != nil {
				continue
			}

			ks := &keyStates[idx]
			ks.mu.Lock()
			ks.version = version
			ks.lastWriteTS = time.Now()
			ks.mu.Unlock()

			pushRecent(idx)
			results <- record{op: "write", latencyMs: float64(lat.Milliseconds())}

		} else {
			idx := pickKey(true)
			key := fmt.Sprintf("k%d", idx)
			readURL := cfg.readURLs[rand.Intn(len(cfg.readURLs))]

			_, version, lat, err := doGet(readURL, key)
			if err != nil {
				continue
			}

			ks := &keyStates[idx]
			ks.mu.Lock()
			lastVer := ks.version
			lastWriteTS := ks.lastWriteTS
			ks.mu.Unlock()

			stale := lastVer > 0 && version < lastVer
			writeToRead := float64(-1)
			if !lastWriteTS.IsZero() {
				writeToRead = float64(time.Since(lastWriteTS).Milliseconds())
			}
			results <- record{op: "read", latencyMs: float64(lat.Milliseconds()), stale: stale, writeToRead: writeToRead}
		}
	}
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	configName := flag.String("config", "lf1", "Cluster config: lf1|lf2|lf3|ll")
	writePct := flag.Int("write-pct", 10, "Percentage of requests that are writes (0-100)")
	duration := flag.Duration("duration", 60*time.Second, "Test duration")
	concurrency := flag.Int("concurrency", 20, "Number of concurrent workers")
	outFile := flag.String("out", "analysis/results/results.csv", "Output CSV file")
	host := flag.String("host", "localhost", "Host where the cluster is running (EC2 IP or localhost)")
	flag.Parse()

	clusters := buildClusters(*host)
	cfg, ok := clusters[*configName]
	if !ok {
		log.Fatalf("unknown config %q — use lf1, lf2, lf3, or ll", *configName)
	}

	// Wait for cluster health.
	checkURL := cfg.writeURL
	if cfg.leaderless {
		checkURL = cfg.readURLs[0]
	}
	log.Printf("Waiting for cluster %s at %s...", *configName, checkURL)
	for {
		resp, err := httpClient.Get(checkURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Printf("Cluster ready. Running %s for %v, %d workers, write%%=%d%%",
		*configName, *duration, *concurrency, *writePct)

	results := make(chan record, 10000)
	var wg sync.WaitGroup
	var totalReqs int64

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			worker(cfg, *writePct, *duration, results, &wg)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var records []record
	for r := range results {
		records = append(records, r)
		atomic.AddInt64(&totalReqs, 1)
	}

	// Write CSV.
	if err := os.MkdirAll("analysis/results", 0755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"op", "latency_ms", "stale", "write_to_read_ms"})
	staleCount := 0
	for _, r := range records {
		stale := "0"
		if r.stale {
			stale = "1"
			staleCount++
		}
		w.Write([]string{
			r.op,
			strconv.FormatFloat(r.latencyMs, 'f', 2, 64),
			stale,
			strconv.FormatFloat(r.writeToRead, 'f', 2, 64),
		})
	}
	w.Flush()

	log.Printf("Done. Total: %d requests, %d stale reads. Output: %s",
		len(records), staleCount, *outFile)
}
