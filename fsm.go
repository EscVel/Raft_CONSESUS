package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

// --- Struct Definitions ---

type Printer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Filament struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"` // e.g., PLA, PETG, ABS, TPU
	Color       string  `json:"color"`
	WeightGrams float64 `json:"weight_grams"` // Remaining weight
}

type command struct {
	Op   string          `json:"op,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// --- FSM Struct and Methods ---

// fsmData holds all the data for our state machine.
// We use this helper struct to make snapshotting easier.
type fsmData struct {
	Printers  map[string]Printer
	Filaments map[string]Filament
}

type fsm struct {
	mu   sync.Mutex
	data fsmData
}

func newFSM() *fsm {
	return &fsm{
		data: fsmData{
			Printers:  make(map[string]Printer),
			Filaments: make(map[string]Filament),
		},
	}
}

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
		f.data.Printers[p.ID] = p
		return nil

	case "add_filament":
		var filament Filament
		if err := json.Unmarshal(cmd.Data, &filament); err != nil {
			return fmt.Errorf("failed to unmarshal filament data: %w", err)
		}
		f.data.Filaments[filament.ID] = filament
		return nil

	default:
		return fmt.Errorf("unrecognized command op: %s", cmd.Op)
	}
}

// --- Snapshot and Restore Methods (Updated for fsmData) ---

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// It's important to clone the data struct for safety.
	clone := fsmData{
		Printers:  make(map[string]Printer),
		Filaments: make(map[string]Filament),
	}
	for k, v := range f.data.Printers {
		clone.Printers[k] = v
	}
	for k, v := range f.data.Filaments {
		clone.Filaments[k] = v
	}

	return &fsmSnapshot{data: clone}, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var data fsmData
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = data
	return nil
}

type fsmSnapshot struct {
	data fsmData
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
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

func (s *fsmSnapshot) Release() {}
