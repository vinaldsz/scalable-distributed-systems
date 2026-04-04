package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// Cluster URLs — tests run against a live docker-compose cluster on localhost.
var (
	// LF W=5 R=1
	lf1Leader  = "http://localhost:8010"
	lf1Nodes   = []string{"http://localhost:8011", "http://localhost:8012", "http://localhost:8013", "http://localhost:8014"}
	lf1All     = append([]string{lf1Leader}, lf1Nodes...)

	// LF W=1 R=5
	lf2Leader = "http://localhost:8020"
	lf2Nodes  = []string{"http://localhost:8021", "http://localhost:8022", "http://localhost:8023", "http://localhost:8024"}
	lf2All    = append([]string{lf2Leader}, lf2Nodes...)

	// LF W=3 R=3
	lf3Leader = "http://localhost:8030"
	lf3Nodes  = []string{"http://localhost:8031", "http://localhost:8032", "http://localhost:8033", "http://localhost:8034"}

	// Leaderless
	llNodes = []string{
		"http://localhost:8040", "http://localhost:8041", "http://localhost:8042",
		"http://localhost:8043", "http://localhost:8044",
	}
)

type setResp struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
}

type getResp struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
	NodeID  int    `json:"node_id"`
	Error   string `json:"error"`
}

func setKey(t *testing.T, baseURL, key, value string) (int, setResp) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"key": key, "value": value})
	resp, err := http.Post(baseURL+"/set", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("setKey %s/%s: %v", baseURL, key, err)
	}
	defer resp.Body.Close()
	var r setResp
	json.NewDecoder(resp.Body).Decode(&r)
	return resp.StatusCode, r
}

func getKey(t *testing.T, baseURL, key string) (int, getResp) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("%s/get/%s", baseURL, key))
	if err != nil {
		t.Fatalf("getKey %s/%s: %v", baseURL, key, err)
	}
	defer resp.Body.Close()
	var r getResp
	json.NewDecoder(resp.Body).Decode(&r)
	return resp.StatusCode, r
}

// localRead returns (statusCode, body, error). Does NOT call t.Fatalf so
// callers can decide how to handle connection errors (e.g. sneaky tests).
func localRead(t *testing.T, baseURL, key string) (int, getResp, error) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("%s/local_read/%s", baseURL, key))
	if err != nil {
		return 0, getResp{}, err
	}
	defer resp.Body.Close()
	var r getResp
	json.NewDecoder(resp.Body).Decode(&r)
	return resp.StatusCode, r, nil
}

// waitForNodes polls /health on each URL with a 30s per-node deadline.
func waitForNodes(t *testing.T, urls []string) {
	t.Helper()
	for _, u := range urls {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			resp, err := http.Get(u + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// uniqueKey generates a test-unique key to avoid cross-test contamination.
func uniqueKey(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
