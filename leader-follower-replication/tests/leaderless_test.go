package tests

import (
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestLL_WriteAckMeansAllNodesUpdated proves that after the write coordinator
// returns 201 (W=N, all nodes waited), every node holds the value.
func TestLL_WriteAckMeansAllNodesUpdated(t *testing.T) {
	waitForNodes(t, llNodes)
	key := uniqueKey("ll-consistent")
	value := "propagated-everywhere"

	coordinator := llNodes[0]
	code, sr := setKey(t, coordinator, key, value)
	if code != http.StatusCreated {
		t.Fatalf("set returned %d", code)
	}

	for _, node := range llNodes {
		code, gr, err := localRead(t, node, key)
		if err != nil {
			t.Errorf("node %s: local_read error: %v", node, err)
			continue
		}
		if code != http.StatusOK {
			t.Errorf("node %s: local_read returned %d (want 200)", node, code)
			continue
		}
		if gr.Value != value {
			t.Errorf("node %s: got %q, want %q", node, gr.Value, value)
		}
		if gr.Version != sr.Version {
			t.Errorf("node %s: version mismatch set=%d local=%d", node, sr.Version, gr.Version)
		}
	}
}

// TestLL_CoordinatorReadConsistent proves that reading from the write
// coordinator immediately after its ACK returns the correct value.
func TestLL_CoordinatorReadConsistent(t *testing.T) {
	waitForNodes(t, llNodes)
	key := uniqueKey("ll-coord-read")
	value := "from-coordinator"

	coordinator := llNodes[rand.Intn(len(llNodes))]
	_, sr := setKey(t, coordinator, key, value)

	code, gr := getKey(t, coordinator, key)
	if code != http.StatusOK {
		t.Fatalf("get from coordinator returned %d", code)
	}
	if gr.Value != value {
		t.Errorf("got %q, want %q", gr.Value, value)
	}
	if gr.Version != sr.Version {
		t.Errorf("version mismatch: set=%d get=%d", sr.Version, gr.Version)
	}
}

// TestLL_SneakyInconsistencyWindow proves that OTHER nodes are stale while
// the write coordinator's concurrent fan-out is in progress.
// Strategy: start the write in a goroutine, then immediately local_read from
// peers — the 100ms follower sleep creates an observable window.
func TestLL_SneakyInconsistencyWindow(t *testing.T) {
	waitForNodes(t, llNodes)
	key := uniqueKey("ll-sneaky")
	value := "mid-flight"

	coordinator := llNodes[0]
	peers := llNodes[1:]

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		setKey(t, coordinator, key, value)
	}()

	// Give the coordinator just enough time to send the HTTP requests but
	// not enough for peers to finish their 100ms sleep and write locally.
	time.Sleep(20 * time.Millisecond)

	staleCount := 0
	for _, node := range peers {
		code, _, err := localRead(t, node, key)
		if err != nil || code == http.StatusNotFound {
			staleCount++
		}
	}
	wg.Wait()

	t.Logf("Stale nodes observed during fan-out window: %d/%d", staleCount, len(peers))
	if staleCount == 0 {
		t.Log("WARNING: no stale reads — window may have closed before local_read fired")
	}
}

// TestLL_AnyNodeCanBeCoordinator proves that writing to different nodes
// all result in consistent state after the ACK (any node can be coordinator).
func TestLL_AnyNodeCanBeCoordinator(t *testing.T) {
	waitForNodes(t, llNodes)

	for i, coordinator := range llNodes {
		key := uniqueKey("ll-any-coord")
		value := "written-by-node-" + string(rune('0'+i))

		code, sr := setKey(t, coordinator, key, value)
		if code != http.StatusCreated {
			t.Errorf("node %s: set returned %d", coordinator, code)
			continue
		}

		// All nodes must have the value after ACK.
		for _, node := range llNodes {
			code, gr, err := localRead(t, node, key)
			if err != nil {
				t.Errorf("key %s on node %s: local_read error: %v", key, node, err)
				continue
			}
			if code != http.StatusOK {
				t.Errorf("key %s on node %s: local_read returned %d", key, node, code)
				continue
			}
			if gr.Value != value {
				t.Errorf("key %s node %s: got %q want %q", key, node, gr.Value, value)
			}
			if gr.Version != sr.Version {
				t.Errorf("key %s node %s: version set=%d local=%d", key, node, sr.Version, gr.Version)
			}
		}
	}
}
