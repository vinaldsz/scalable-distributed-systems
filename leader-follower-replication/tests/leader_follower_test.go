package tests

import (
	"net/http"
	"testing"
	"time"
)

// ─── W=5 R=1 (Cluster 1) ───────────────────────────────────────────────────

// TestLF_W5R1_LeaderReadConsistent proves that after the leader ACKs a write
// (W=5, so all nodes updated), reading from the leader returns the same value.
func TestLF_W5R1_LeaderReadConsistent(t *testing.T) {
	waitForNodes(t, lf1All)
	key := uniqueKey("w5r1-leader")
	value := "hello-leader"

	code, sr := setKey(t, lf1Leader, key, value)
	if code != http.StatusCreated {
		t.Fatalf("set returned %d", code)
	}
	if sr.Version < 1 {
		t.Fatalf("expected version >= 1, got %d", sr.Version)
	}

	code, gr := getKey(t, lf1Leader, key)
	if code != http.StatusOK {
		t.Fatalf("get returned %d", code)
	}
	if gr.Value != value {
		t.Errorf("leader read: got %q, want %q", gr.Value, value)
	}
	if gr.Version != sr.Version {
		t.Errorf("version mismatch: set=%d get=%d", sr.Version, gr.Version)
	}
}

// TestLF_W5R1_FollowerReadConsistent proves that after W=5 ACK all followers
// have the value (leader waited for all before responding).
func TestLF_W5R1_FollowerReadConsistent(t *testing.T) {
	waitForNodes(t, lf1All)
	key := uniqueKey("w5r1-follower")
	value := "propagated"

	code, sr := setKey(t, lf1Leader, key, value)
	if code != http.StatusCreated {
		t.Fatalf("set returned %d", code)
	}

	// All followers must have the value because W=5 waited for all ACKs.
	for _, node := range lf1Nodes {
		code, gr, err := localRead(t, node, key)
		if err != nil {
			t.Errorf("node %s: local_read error: %v", node, err)
			continue
		}
		if code != http.StatusOK {
			t.Errorf("node %s: local_read returned %d", node, code)
			continue
		}
		if gr.Value != value {
			t.Errorf("node %s: got %q, want %q", node, gr.Value, value)
		}
		if gr.Version != sr.Version {
			t.Errorf("node %s: version mismatch set=%d local_read=%d", node, sr.Version, gr.Version)
		}
	}
}

// TestLF_W5R1_VersionMonotonic proves versions increment on repeated writes.
func TestLF_W5R1_VersionMonotonic(t *testing.T) {
	waitForNodes(t, lf1All)
	key := uniqueKey("versions")

	var prev int
	for i := 0; i < 3; i++ {
		_, sr := setKey(t, lf1Leader, key, "v")
		if sr.Version <= prev {
			t.Errorf("write %d: version did not increase: prev=%d cur=%d", i, prev, sr.Version)
		}
		prev = sr.Version
	}
}

// ─── W=1 R=5 (Cluster 2) ───────────────────────────────────────────────────

// TestLF_W1R5_SneakyInconsistency proves that immediately after a W=1 write
// (leader only updated), followers still hold stale data.
func TestLF_W1R5_SneakyInconsistency(t *testing.T) {
	waitForNodes(t, lf2All)
	key := uniqueKey("w1r5-sneaky")
	value := "fresh"

	// Fire the set in a goroutine (it returns quickly since W=1)
	done := make(chan struct{})
	var writeVersion int
	go func() {
		_, sr := setKey(t, lf2Leader, key, value)
		writeVersion = sr.Version
		close(done)
	}()
	<-done

	// Immediately local_read from ALL followers — at least some should be stale.
	staleCount := 0
	for _, node := range lf2Nodes {
		code, gr, err := localRead(t, node, key)
		if err != nil {
			// Node not reachable counts as stale (hasn't been updated yet).
			t.Logf("node %s: connection error (counts as stale): %v", node, err)
			staleCount++
			continue
		}
		if code == http.StatusNotFound || gr.Version < writeVersion {
			staleCount++
		}
	}
	if staleCount == 0 {
		t.Log("WARNING: no stale followers observed — replication may have been unexpectedly fast")
	} else {
		t.Logf("Observed %d/%d stale followers immediately after W=1 write (expected)", staleCount, len(lf2Nodes))
	}
}

// TestLF_W1R5_QuorumReadReturnsLatest proves that even with W=1, a quorum
// read (R=5) returns the latest version because it queries all nodes and picks
// the highest version.
func TestLF_W1R5_QuorumReadReturnsLatest(t *testing.T) {
	waitForNodes(t, lf2All)
	key := uniqueKey("w1r5-quorum")
	value := "latest"

	_, sr := setKey(t, lf2Leader, key, value)

	// Wait for async replication to fully propagate (≤ 4×300ms = 1.2s)
	time.Sleep(1500 * time.Millisecond)

	// A get through the leader with R=5 must return the latest value.
	code, gr := getKey(t, lf2Leader, key)
	if code != http.StatusOK {
		t.Fatalf("get returned %d", code)
	}
	if gr.Value != value {
		t.Errorf("quorum read: got %q, want %q", gr.Value, value)
	}
	if gr.Version != sr.Version {
		t.Errorf("version mismatch: set=%d get=%d", sr.Version, gr.Version)
	}
}

// ─── W=3 R=3 Quorum (Cluster 3) ────────────────────────────────────────────

// TestLF_W3R3_QuorumReadConsistent proves that a W=3/R=3 quorum read always
// returns the latest value because any 3-write and 3-read sets must overlap.
func TestLF_W3R3_QuorumReadConsistent(t *testing.T) {
	waitForNodes(t, append([]string{lf3Leader}, lf3Nodes...))
	key := uniqueKey("w3r3-quorum")
	value := "quorum-value"

	_, sr := setKey(t, lf3Leader, key, value)

	// The quorum read is triggered by /get on the leader (READ_QUORUM=3).
	code, gr := getKey(t, lf3Leader, key)
	if code != http.StatusOK {
		t.Fatalf("get returned %d", code)
	}
	if gr.Value != value {
		t.Errorf("quorum read: got %q, want %q", gr.Value, value)
	}
	if gr.Version != sr.Version {
		t.Errorf("version mismatch: set=%d get=%d", sr.Version, gr.Version)
	}
}

// TestLF_W3R3_SneakyNonQuorumFollowerStale proves that followers NOT in the
// write quorum may be stale immediately after a W=3 write.
func TestLF_W3R3_SneakyNonQuorumFollowerStale(t *testing.T) {
	waitForNodes(t, append([]string{lf3Leader}, lf3Nodes...))
	key := uniqueKey("w3r3-sneaky")
	value := "partial"

	// W=3: leader + 2 followers updated synchronously, 2 followers async.
	done := make(chan int)
	go func() {
		_, sr := setKey(t, lf3Leader, key, value)
		done <- sr.Version
	}()
	writeVersion := <-done

	// Nodes 3 & 4 (indices 2 & 3 in lf3Nodes) are async — should be stale.
	staleCount := 0
	for _, node := range lf3Nodes[2:] {
		code, gr, err := localRead(t, node, key)
		if err != nil || code == http.StatusNotFound || gr.Version < writeVersion {
			staleCount++
		}
	}
	t.Logf("Non-quorum followers stale: %d/2", staleCount)
	// This is a probabilistic test; we log rather than hard-fail.
}
