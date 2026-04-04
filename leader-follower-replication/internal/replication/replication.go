package replication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ReplicateMsg is sent from coordinator to peer nodes.
type ReplicateMsg struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// ReadResult holds a peer's response to an internal read.
type ReadResult struct {
	Value   string `json:"value"`
	Version int    `json:"version"`
	NodeID  int    `json:"node_id"`
	Found   bool
}

var client = &http.Client{Timeout: 10 * time.Second}

// LeaderReplicateSequential sends replicate messages to peers one-by-one,
// sleeping 200ms after each send (as required by the assignment).
// It sends to exactly `count` peers and returns the number of acks received.
func LeaderReplicateSequential(peers []string, count int, key, value string, version int) int {
	acks := 0
	msg := ReplicateMsg{Key: key, Value: value, Version: version}
	body, _ := json.Marshal(msg)

	for i := 0; i < count && i < len(peers); i++ {
		url := peers[i] + "/internal/replicate"
		resp, err := client.Post(url, "application/json", bytes.NewReader(body))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			acks++
		}
		// 200ms sleep after each send — creates the inconsistency window
		time.Sleep(200 * time.Millisecond)
	}
	return acks
}

// LeaderReplicateAsync fans out to remaining peers in a background goroutine
// (used for W<N: replicate to quorum sync, rest async).
func LeaderReplicateAsync(peers []string, skip int, key, value string, version int) {
	go func() {
		msg := ReplicateMsg{Key: key, Value: value, Version: version}
		body, _ := json.Marshal(msg)
		for i := skip; i < len(peers); i++ {
			url := peers[i] + "/internal/replicate"
			resp, err := client.Post(url, "application/json", bytes.NewReader(body))
			if err == nil {
				resp.Body.Close()
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
}

// LeaderlessCoordinate fans out to all peers concurrently and waits for all.
// Returns number of successful acks.
func LeaderlessCoordinate(peers []string, key, value string, version int) int {
	msg := ReplicateMsg{Key: key, Value: value, Version: version}
	body, _ := json.Marshal(msg)

	var mu sync.Mutex
	acks := 0
	var wg sync.WaitGroup

	for _, peer := range peers {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			resp, err := client.Post(url+"/internal/replicate", "application/json", bytes.NewReader(body))
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				mu.Lock()
				acks++
				mu.Unlock()
			}
		}(peer)
	}
	wg.Wait()
	return acks
}

// QuorumRead fetches from `count` peers via /internal/read and returns the
// entry with the highest version (plus acks count).
func QuorumRead(peers []string, count int, key string) (string, int, int) {
	type result struct {
		value   string
		version int
		ok      bool
	}

	results := make(chan result, count)
	for i := 0; i < count && i < len(peers); i++ {
		go func(url string) {
			r, err := client.Get(fmt.Sprintf("%s/internal/read/%s", url, key))
			if err != nil {
				results <- result{}
				return
			}
			defer r.Body.Close()
			if r.StatusCode == http.StatusNotFound {
				results <- result{}
				return
			}
			var rr ReadResult
			if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
				results <- result{}
				return
			}
			results <- result{value: rr.Value, version: rr.Version, ok: true}
		}(peers[i])
	}

	best := result{}
	acks := 0
	for i := 0; i < count && i < len(peers); i++ {
		r := <-results
		if r.ok {
			acks++
			if r.version > best.version {
				best = r
			}
		}
	}
	return best.value, best.version, acks
}
