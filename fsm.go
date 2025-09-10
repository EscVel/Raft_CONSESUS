package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

// Printer represents a 3D printer in the system, as per the project spec.
type Printer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// command represents a command that will be sent through the Raft log.
// Using json.RawMessage lets us handle different data types for different operations.
type command struct {
	Op   string          `json:"op,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// fsm is the Finite State Machine. It now stores a map of Printers.
type fsm struct {
	mu       sync.Mutex
	printers map[string]Printer // Keyed by Printer ID
}

// newFSM creates a new, empty FSM.
func newFSM() *fsm {
	return &fsm{
		printers: make(map[string]Printer),
	}
}

// Apply is where all state changes happen. It's called after a command
// has been committed to the Raft log.
func (f *fsm) Apply(log *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var cmd command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch cmd.Op {
	case "add_printer":
		var p Printer
		if err := json.Unmarshal(cmd.Data, &p); err != nil {
			return fmt.Errorf("failed to unmarshal printer data: %w", err)
		}
		f.printers[p.ID] = p
		return nil
	default:
		return fmt.Errorf("unrecognized command op: %s", cmd.Op)
	}
}

// Snapshot is called by Raft to create a snapshot of the current state.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	clone := make(map[string]Printer)
	for k, v := range f.printers {
		clone[k] = v
	}

	return &fsmSnapshot{printers: clone}, nil
}

// Restore is called by Raft to restore the FSM from a snapshot.
func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var data map[string]Printer
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.printers = data
	return nil
}

// fsmSnapshot is the struct that Raft uses to persist a snapshot.
type fsmSnapshot struct {
	printers map[string]Printer
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		if err := json.NewEncoder(sink).Encode(s.printers); err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		sink.Cancel()
	}
	return err
}

func (s *fsmSnapshot) Release() {}
