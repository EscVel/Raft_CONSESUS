package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	boltdb "github.com/hashicorp/raft-boltdb"
)

// Store is a wrapper around the Raft node.
type Store struct {
	NodeID   string
	RaftAddr string
	DataDir  string

	raft *raft.Raft // The consensus mechanism
	fsm  *fsm       // The state machine
}

// NewStore creates a new Store.
func NewStore(nodeID, raftAddr, dataDir string) *Store {
	return &Store{
		NodeID:   nodeID,
		RaftAddr: raftAddr,
		DataDir:  dataDir,
		fsm:      newFSM(),
	}
}

// Open initializes the store, including the Raft node.
func (s *Store) Open(bootstrap bool) error {
	nodeDataDir := filepath.Join(s.DataDir, s.NodeID)
	if err := os.MkdirAll(nodeDataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(s.NodeID)

	addr, err := net.ResolveTCPAddr("tcp", s.RaftAddr)
	if err != nil {
		return err
	}
	transport, err := raft.NewTCPTransport(s.RaftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	logStore, err := boltdb.NewBoltStore(filepath.Join(nodeDataDir, "raft-log.db"))
	if err != nil {
		return err
	}
	stableStore, err := boltdb.NewBoltStore(filepath.Join(nodeDataDir, "raft-stable.db"))
	if err != nil {
		return err
	}

	snapshots, err := raft.NewFileSnapshotStore(nodeDataDir, 2, os.Stderr)
	if err != nil {
		return err
	}

	r, err := raft.NewRaft(config, s.fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		return err
	}
	s.raft = r

	if bootstrap {
		log.Println("Bootstrapping the cluster...")
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		future := s.raft.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			return fmt.Errorf("failed to bootstrap cluster: %w", err)
		}
	}
	return nil
}

// Join adds a new node to the cluster.
func (s *Store) Join(nodeID, addr string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader, cannot join")
	}
	log.Printf("Received join request for node %s at %s", nodeID, addr)
	future := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := future.Error(); err != nil {
		log.Printf("Failed to add voter: %s", err)
		return err
	}
	log.Printf("Node %s at %s joined successfully", nodeID, addr)
	return nil
}
