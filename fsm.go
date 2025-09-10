// fsm.go

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

// command represents a command to be applied to the FSM.
// We will serialize this to JSON and send it over the Raft log.
type command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// fsm is the Finite State Machine for our key-value store.
type fsm struct {
	mu   sync.Mutex
	data map[string]string
}

// newFSM creates a new, empty FSM.
func newFSM() *fsm {
	return &fsm{
		data: make(map[string]string),
	}
}

// Apply applies a Raft log entry to the key-value store.
// This is the core of the FSM, where state changes happen.
func (f *fsm) Apply(log *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var cmd command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch cmd.Op {
	case "set":
		f.data[cmd.Key] = cmd.Value
		return nil
	case "delete":
		delete(f.data, cmd.Key)
		return nil
	default:
		return fmt.Errorf("unrecognized command op: %s", cmd.Op)
	}
}

// Snapshot returns a snapshot of the current state.
// Raft uses this to compact its log.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Clone the data map
	clone := make(map[string]string)
	for k, v := range f.data {
		clone[k] = v
	}

	return &fsmSnapshot{data: clone}, nil
}

// Restore restores the FSM to a previous state from a snapshot.
func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var data map[string]string
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = data
	return nil
}

// fsmSnapshot is used to represent a snapshot of the FSM's state.
type fsmSnapshot struct {
	data map[string]string
}

// Persist writes the FSM state to a sink (a file).
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode the data as JSON and write it to the sink.
		if err := json.NewEncoder(sink).Encode(s.data); err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

// Release is a no-op.
func (s *fsmSnapshot) Release() {}
