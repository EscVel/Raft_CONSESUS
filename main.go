package main

import (
	"flag"
	"log"
)

func main() {
	nodeID := flag.String("id", "", "Node ID")
	raftAddr := flag.String("raft-addr", "127.0.0.1:7001", "Raft communication address")
	httpAddr := flag.String("http-addr", "127.0.0.1:8001", "HTTP API address")
	dataDir := flag.String("data-dir", "data/", "Directory to store Raft data")
	bootstrap := flag.Bool("bootstrap", false, "Bootstrap the cluster")

	flag.Parse()

	if *nodeID == "" {
		log.Fatalf("Error: -id flag is required")
	}

	store := NewStore(*nodeID, *raftAddr, *dataDir)

	if err := store.Open(*bootstrap); err != nil {
		log.Fatalf("failed to open store: %s", err)
	}

	server := NewServer(*httpAddr, store)
	if err := server.Start(); err != nil {
		log.Fatalf("failed to start server: %s", err)
	}
}
