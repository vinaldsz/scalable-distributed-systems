package main

import (
	"fmt"
	"leader-follower-replication/internal/config"
	"leader-follower-replication/internal/handler"
	"leader-follower-replication/internal/store"
	"log"
	"net/http"
	"os"
)

func main() {
	cfg := config.Load()
	s := store.New()
	h := handler.New(s, cfg)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Printf("node %d starting as %q on :%s (W=%d R=%d peers=%v)",
		cfg.NodeID, cfg.Role, port, cfg.WriteQuorum, cfg.ReadQuorum, cfg.PeerURLs)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), mux); err != nil {
		log.Fatal(err)
	}
}
