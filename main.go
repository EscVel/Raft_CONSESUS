// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	boltdb "github.com/hashicorp/raft-boltdb"
)

func main() {
	// 1. Define command-line flags
	nodeID := flag.String("id", "", "Node ID")
	raftAddr := flag.String("raft-addr", "127.0.0.1:7001", "Raft communication address")
	httpAddr := flag.String("http-addr", "127.0.0.1:8001", "HTTP API address")
	dataDir := flag.String("data-dir", "data/", "Directory to store Raft data")
	bootstrap := flag.Bool("bootstrap", false, "Bootstrap the cluster")

	// 2. Parse the flags
	flag.Parse()

	// 3. Basic validation
	if *nodeID == "" {
		log.Fatalf("Error: -id flag is required")
	}

	// Create the data directory path for this node
	nodeDataDir := filepath.Join(*dataDir, *nodeID)
	if err := os.MkdirAll(nodeDataDir, 0700); err != nil {
		log.Fatalf("failed to create data directory: %s", err)
	}

	// 4. Set up Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(*nodeID)

	// 5. Set up the transport layer (networking)
	addr, err := net.ResolveTCPAddr("tcp", *raftAddr)
	if err != nil {
		log.Fatalf("failed to resolve tcp addr: %s", err)
	}
	transport, err := raft.NewTCPTransport(*raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatalf("failed to create tcp transport: %s", err)
	}

	// 6. Set up the log and stable stores (storage)
	logStore, err := boltdb.NewBoltStore(filepath.Join(nodeDataDir, "raft-log.db"))
	if err != nil {
		log.Fatalf("failed to create bolt store for logs: %s", err)
	}
	stableStore, err := boltdb.NewBoltStore(filepath.Join(nodeDataDir, "raft-stable.db"))
	if err != nil {
		log.Fatalf("failed to create bolt store for stable: %s", err)
	}

	// 7. Set up snapshots
	snapshots, err := raft.NewFileSnapshotStore(nodeDataDir, 2, os.Stderr)
	if err != nil {
		log.Fatalf("failed to create snapshot store: %s", err)
	}

	// 8. Create the FSM
	fsm := newFSM()

	// 9. Instantiate the Raft system
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		log.Fatalf("failed to create raft instance: %s", err)
	}

	// 10. Bootstrap the cluster if necessary
	if *bootstrap {
		log.Println("Bootstrapping the cluster...")
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		future := r.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			log.Fatalf("failed to bootstrap cluster: %s", err)
		}
	}

	log.Printf("Node %s started at Raft address %s\n", *nodeID, *raftAddr)

	// 11. Start a simple HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Raft node %s", *nodeID)
	})

	log.Printf("HTTP server listening on %s\n", *httpAddr)
	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatalf("failed to start HTTP server: %s", err)
	}
}
