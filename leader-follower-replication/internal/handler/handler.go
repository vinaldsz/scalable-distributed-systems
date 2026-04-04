package handler

import (
	"encoding/json"
	"leader-follower-replication/internal/config"
	"leader-follower-replication/internal/replication"
	"leader-follower-replication/internal/store"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	store *store.KVStore
	cfg   config.Config
}

func New(s *store.KVStore, cfg config.Config) *Handler {
	return &Handler{store: s, cfg: cfg}
}

// RegisterRoutes wires all endpoints onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/set", h.handleSet)
	mux.HandleFunc("/get/", h.handleGet)
	mux.HandleFunc("/local_read/", h.handleLocalRead)
	mux.HandleFunc("/internal/replicate", h.handleInternalReplicate)
	mux.HandleFunc("/internal/read/", h.handleInternalRead)
	mux.HandleFunc("/health", h.handleHealth)
}

// ---------- response helpers ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func keyFromPath(r *http.Request, prefix string) string {
	return strings.TrimPrefix(r.URL.Path, prefix)
}

// ---------- client-facing endpoints ----------

// POST /set  body: {"key":"k","value":"v"}
func (h *Handler) handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key cannot be empty"})
		return
	}

	switch h.cfg.Role {
	case "leader":
		h.leaderSet(w, req.Key, req.Value)
	case "leaderless":
		h.leaderlessSet(w, req.Key, req.Value)
	default:
		// Followers should not receive client writes; return 400
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "writes must go to the leader"})
	}
}

func (h *Handler) leaderSet(w http.ResponseWriter, key, value string) {
	W := h.cfg.WriteQuorum
	peers := h.cfg.PeerURLs // peers does NOT include self

	// Assign version and write locally first
	version := h.store.Set(key, value, 0)

	syncCount := W - 1 // -1 because leader itself counts as 1
	if syncCount < 0 {
		syncCount = 0
	}
	if syncCount > len(peers) {
		syncCount = len(peers)
	}

	// Synchronous replication to syncCount followers (sequential + 200ms sleep)
	replication.LeaderReplicateSequential(peers, syncCount, key, value, version)

	// Async replication to remaining followers
	if syncCount < len(peers) {
		replication.LeaderReplicateAsync(peers, syncCount, key, value, version)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"key": key, "value": value, "version": version,
	})
}

func (h *Handler) leaderlessSet(w http.ResponseWriter, key, value string) {
	// This node is the write coordinator
	version := h.store.Set(key, value, 0)

	// Fan out concurrently to all peers, wait for all (W=N)
	replication.LeaderlessCoordinate(h.cfg.PeerURLs, key, value, version)

	writeJSON(w, http.StatusCreated, map[string]any{
		"key": key, "value": value, "version": version,
	})
}

// GET /get/{key}
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := keyFromPath(r, "/get/")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key cannot be empty"})
		return
	}

	switch h.cfg.Role {
	case "leader":
		h.leaderGet(w, key)
	case "leaderless":
		// R=1: return own value
		h.localGet(w, key)
	default:
		// Follower: return own value (caller decided to read from this node)
		h.localGet(w, key)
	}
}

func (h *Handler) leaderGet(w http.ResponseWriter, key string) {
	R := h.cfg.ReadQuorum
	peers := h.cfg.PeerURLs

	if R <= 1 {
		// R=1: serve from own store
		h.localGet(w, key)
		return
	}

	// R>1: quorum read — include self + (R-1) peers
	selfEntry, selfFound := h.store.Get(key)

	peerCount := R - 1
	if peerCount > len(peers) {
		peerCount = len(peers)
	}

	bestValue, bestVersion, _ := replication.QuorumRead(peers, peerCount, key)

	if selfFound && selfEntry.Version > bestVersion {
		bestValue = selfEntry.Value
		bestVersion = selfEntry.Version
	}

	if !selfFound && bestVersion == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key": key, "value": bestValue, "version": bestVersion,
	})
}

func (h *Handler) localGet(w http.ResponseWriter, key string) {
	entry, ok := h.store.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key": key, "value": entry.Value, "version": entry.Version,
	})
}

// GET /local_read/{key} — test-only, always returns own store value
func (h *Handler) handleLocalRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := keyFromPath(r, "/local_read/")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key cannot be empty"})
		return
	}
	entry, ok := h.store.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found", "node_id": intStr(h.cfg.NodeID)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key": key, "value": entry.Value, "version": entry.Version, "node_id": h.cfg.NodeID,
	})
}

// ---------- internal (node-to-node) endpoints ----------

// POST /internal/replicate  body: {"key","value","version"}
// Follower sleeps 100ms before writing to simulate storage latency.
func (h *Handler) handleInternalReplicate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var msg replication.ReplicateMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// Simulate storage write latency
	time.Sleep(100 * time.Millisecond)
	h.store.Set(msg.Key, msg.Value, msg.Version)
	writeJSON(w, http.StatusOK, map[string]any{
		"ack": true, "node_id": h.cfg.NodeID, "version": msg.Version,
	})
}

// GET /internal/read/{key}
// Follower sleeps 50ms before responding to simulate read latency.
func (h *Handler) handleInternalRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := keyFromPath(r, "/internal/read/")
	// Simulate read latency
	time.Sleep(50 * time.Millisecond)
	entry, ok := h.store.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key": key, "value": entry.Value, "version": entry.Version, "node_id": h.cfg.NodeID,
	})
}

// GET /health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok", "node_id": h.cfg.NodeID, "role": h.cfg.Role,
	})
}

func intStr(n int) string {
	return strconv.Itoa(n)
}
