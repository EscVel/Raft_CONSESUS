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
	Type        string  `json:"type"`
	Color       string  `json:"color"`
	WeightGrams float64 `json:"weight_grams"`
}

type PrintJob struct {
	ID          string  `json:"id"`
	FilePath    string  `json:"file_path"`
	GramsNeeded float64 `json:"grams_needed"`
	PrinterID   string  `json:"printer_id"`
	FilamentID  string  `json:"filament_id"`
	Status      string  `json:"status"`
}

type UpdateJobStatusData struct {
	JobID     string `json:"job_id"`
	NewStatus string `json:"new_status"`
}

type command struct {
	Op   string          `json:"op,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// --- FSM Struct and Methods ---

type fsmData struct {
	Printers  map[string]Printer
	Filaments map[string]Filament
	PrintJobs map[string]PrintJob
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
			PrintJobs: make(map[string]PrintJob),
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

	case "add_print_job":
		var job PrintJob
		if err := json.Unmarshal(cmd.Data, &job); err != nil {
			return fmt.Errorf("failed to unmarshal print job data: %w", err)
		}

		if _, ok := f.data.Printers[job.PrinterID]; !ok {
			return fmt.Errorf("printer with ID %s not found", job.PrinterID)
		}
		filament, ok := f.data.Filaments[job.FilamentID]
		if !ok {
			return fmt.Errorf("filament with ID %s not found", job.FilamentID)
		}

		var reservedWeight float64
		for _, existingJob := range f.data.PrintJobs {
			if existingJob.FilamentID == job.FilamentID && (existingJob.Status == "Queued" || existingJob.Status == "Running") {
				reservedWeight += existingJob.GramsNeeded
			}
		}
		availableWeight := filament.WeightGrams - reservedWeight
		if availableWeight < job.GramsNeeded {
			return fmt.Errorf("insufficient filament: required %.2fg, available %.2fg (%.2fg total - %.2fg reserved)",
				job.GramsNeeded, availableWeight, filament.WeightGrams, reservedWeight)
		}

		job.Status = "Queued"
		f.data.PrintJobs[job.ID] = job
		return nil

	case "update_job_status":
		var updateData UpdateJobStatusData
		if err := json.Unmarshal(cmd.Data, &updateData); err != nil {
			return fmt.Errorf("failed to unmarshal update job data: %w", err)
		}

		job, ok := f.data.PrintJobs[updateData.JobID]
		if !ok {
			return fmt.Errorf("print job with ID %s not found", updateData.JobID)
		}

		currentStatus := job.Status
		newStatus := updateData.NewStatus
		isValidTransition := false

		switch newStatus {
		case "Running":
			if currentStatus == "Queued" {
				isValidTransition = true
			}
		case "Done":
			if currentStatus == "Running" {
				isValidTransition = true
				filament := f.data.Filaments[job.FilamentID]
				filament.WeightGrams -= job.GramsNeeded
				f.data.Filaments[job.FilamentID] = filament
			}
		case "Canceled":
			if currentStatus == "Queued" || currentStatus == "Running" {
				isValidTransition = true
			}
		}

		if !isValidTransition {
			return fmt.Errorf("invalid status transition from '%s' to '%s'", currentStatus, newStatus)
		}

		job.Status = newStatus
		f.data.PrintJobs[job.ID] = job
		return nil

	default:
		return fmt.Errorf("unrecognized command op: %s", cmd.Op)
	}
}

// --- Snapshot and Restore Methods ---
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	clone := fsmData{
		Printers:  make(map[string]Printer),
		Filaments: make(map[string]Filament),
		PrintJobs: make(map[string]PrintJob),
	}
	for k, v := range f.data.Printers {
		clone.Printers[k] = v
	}
	for k, v := range f.data.Filaments {
		clone.Filaments[k] = v
	}
	for k, v := range f.data.PrintJobs {
		clone.PrintJobs[k] = v
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
