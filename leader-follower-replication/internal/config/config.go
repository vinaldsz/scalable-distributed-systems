package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	NodeID       int
	Role         string   // "leader" | "follower" | "leaderless"
	PeerURLs     []string // URLs of all peer nodes (excludes self)
	WriteQuorum  int      // W
	ReadQuorum   int      // R
	LeaderURL    string   // followers only
}

func Load() Config {
	nodeID, _ := strconv.Atoi(getenv("NODE_ID", "0"))
	writeQuorum, _ := strconv.Atoi(getenv("WRITE_QUORUM", "5"))
	readQuorum, _ := strconv.Atoi(getenv("READ_QUORUM", "1"))

	var peers []string
	if raw := os.Getenv("PEER_URLS"); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				peers = append(peers, p)
			}
		}
	}

	return Config{
		NodeID:      nodeID,
		Role:        getenv("NODE_ROLE", "leader"),
		PeerURLs:    peers,
		WriteQuorum: writeQuorum,
		ReadQuorum:  readQuorum,
		LeaderURL:   os.Getenv("LEADER_URL"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
