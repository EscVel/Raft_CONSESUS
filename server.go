package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/raft"
)

type Server struct {
	store    *Store
	httpAddr string
}

func NewServer(addr string, store *Store) *Server {
	return &Server{
		store:    store,
		httpAddr: addr,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/join", s.handleJoin)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/printers", s.handlePrinters)
	mux.HandleFunc("/filaments", s.handleFilaments)
	mux.HandleFunc("/print_jobs", s.handlePrintJobs)
	mux.HandleFunc("/print_jobs/", s.handleUpdateJobStatus)

	log.Printf("HTTP server listening on %s\n", s.httpAddr)
	return http.ListenAndServe(s.httpAddr, mux)
}

// --- Helper for redirection ---
func (s *Server) redirectToLeader(w http.ResponseWriter, r *http.Request) {
	leaderRaftAddr := string(s.store.raft.Leader())
	if leaderRaftAddr == "" {
		http.Error(w, "No leader found", http.StatusServiceUnavailable)
		return
	}
	host, raftPortStr, err := net.SplitHostPort(leaderRaftAddr)
	if err != nil {
		http.Error(w, "Failed to parse leader address", http.StatusInternalServerError)
		return
	}
	raftPort, err := strconv.Atoi(raftPortStr)
	if err != nil {
		http.Error(w, "Failed to parse leader port", http.StatusInternalServerError)
		return
	}
	httpPort := raftPort + 1000
	redirectURL := fmt.Sprintf("http://%s:%d%s?%s", host, httpPort, r.URL.Path, r.URL.RawQuery)
	log.Printf("I am not the leader. Redirecting request to leader at %s", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// --- Handlers ---

func (s *Server) handlePrinters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetPrinters(w, r)
	case http.MethodPost:
		s.handleAddPrinter(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetPrinters(w http.ResponseWriter, r *http.Request) {
	printers := s.store.GetPrinters()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(printers); err != nil {
		http.Error(w, "Failed to encode printers", http.StatusInternalServerError)
	}
}

func (s *Server) handleAddPrinter(w http.ResponseWriter, r *http.Request) {
	if s.store.raft.State() != raft.Leader {
		s.redirectToLeader(w, r)
		return
	}
	var p Printer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Failed to decode printer from request", http.StatusBadRequest)
		return
	}
	cmdData, _ := json.Marshal(p)
	cmd := command{Op: "add_printer", Data: cmdData}
	cmdBytes, _ := json.Marshal(cmd)
	resp, err := s.store.Apply(cmdBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fsmErr, ok := resp.(error); ok {
		http.Error(w, fsmErr.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleFilaments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetFilaments(w, r)
	case http.MethodPost:
		s.handleAddFilament(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetFilaments(w http.ResponseWriter, r *http.Request) {
	filaments := s.store.GetFilaments()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filaments); err != nil {
		http.Error(w, "Failed to encode filaments", http.StatusInternalServerError)
	}
}

func (s *Server) handleAddFilament(w http.ResponseWriter, r *http.Request) {
	if s.store.raft.State() != raft.Leader {
		s.redirectToLeader(w, r)
		return
	}
	var f Filament
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, "Failed to decode filament from request", http.StatusBadRequest)
		return
	}
	cmdData, _ := json.Marshal(f)
	cmd := command{Op: "add_filament", Data: cmdData}
	cmdBytes, _ := json.Marshal(cmd)
	resp, err := s.store.Apply(cmdBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fsmErr, ok := resp.(error); ok {
		http.Error(w, fsmErr.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePrintJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetPrintJobs(w, r)
	case http.MethodPost:
		s.handleAddPrintJob(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetPrintJobs(w http.ResponseWriter, r *http.Request) {
	jobs := s.store.GetPrintJobs()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		http.Error(w, "Failed to encode jobs", http.StatusInternalServerError)
	}
}

func (s *Server) handleAddPrintJob(w http.ResponseWriter, r *http.Request) {
	if s.store.raft.State() != raft.Leader {
		s.redirectToLeader(w, r)
		return
	}
	var job PrintJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Failed to decode job from request", http.StatusBadRequest)
		return
	}
	cmdData, _ := json.Marshal(job)
	cmd := command{Op: "add_print_job", Data: cmdData}
	cmdBytes, _ := json.Marshal(cmd)
	resp, err := s.store.Apply(cmdBytes)
	if err != nil {
		http.Error(w, "Failed to apply command to raft", http.StatusInternalServerError)
		return
	}
	if fsmErr, ok := resp.(error); ok {
		http.Error(w, fsmErr.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.store.raft.State() != raft.Leader {
		s.redirectToLeader(w, r)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 || parts[3] != "status" {
		http.Error(w, "Invalid URL path, expected /print_jobs/{id}/status", http.StatusBadRequest)
		return
	}
	jobID := parts[2]
	newStatus := r.URL.Query().Get("status")
	if newStatus == "" {
		http.Error(w, "Missing 'status' query parameter", http.StatusBadRequest)
		return
	}
	updateData := UpdateJobStatusData{JobID: jobID, NewStatus: newStatus}
	cmdData, _ := json.Marshal(updateData)
	cmd := command{Op: "update_job_status", Data: cmdData}
	cmdBytes, _ := json.Marshal(cmd)
	resp, err := s.store.Apply(cmdBytes)
	if err != nil {
		http.Error(w, "Failed to apply command to raft", http.StatusInternalServerError)
		return
	}
	if fsmErr, ok := resp.(error); ok {
		http.Error(w, fsmErr.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID   string `json:"id"`
		Addr string `json:"addr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode join request: %s", err)
		http.Error(w, "Failed to decode request", http.StatusBadRequest)
		return
	}
	if err := s.store.Join(req.ID, req.Addr); err != nil {
		log.Printf("Failed to join node: %s", err)
		http.Error(w, "Failed to join node", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := s.store.raft.Stats()
	status := map[string]string{
		"state":          stats["state"],
		"leader":         stats["leader_addr"],
		"commit_index":   stats["commit_index"],
		"last_applied":   stats["last_applied"],
		"last_log_index": stats["last_log_index"],
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode status: %s", err)
		http.Error(w, "Failed to encode status", http.StatusInternalServerError)
	}
}
