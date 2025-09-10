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

type Store struct {
	NodeID   string
	RaftAddr string
	DataDir  string

	raft *raft.Raft
	fsm  *fsm
}

func NewStore(nodeID, raftAddr, dataDir string) *Store {
	return &Store{
		NodeID:   nodeID,
		RaftAddr: raftAddr,
		DataDir:  dataDir,
		fsm:      newFSM(),
	}
}

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

func (s *Store) Apply(cmdBytes []byte) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader, cannot apply command")
	}
	future := s.raft.Apply(cmdBytes, 500*time.Millisecond)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply command: %w", err)
	}
	return nil
}

func (s *Store) GetPrinters() []Printer {
	s.fsm.mu.Lock()
	defer s.fsm.mu.Unlock()
	printers := make([]Printer, 0, len(s.fsm.data.Printers))
	for _, p := range s.fsm.data.Printers {
		printers = append(printers, p)
	}
	return printers
}

func (s *Store) GetFilaments() []Filament {
	s.fsm.mu.Lock()
	defer s.fsm.mu.Unlock()
	filaments := make([]Filament, 0, len(s.fsm.data.Filaments))
	for _, f := range s.fsm.data.Filaments {
		filaments = append(filaments, f)
	}
	return filaments
}
