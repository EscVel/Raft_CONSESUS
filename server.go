package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/hashicorp/raft" // Make sure raft is imported if not already
)

// Server is the HTTP server for the Raft node.
type Server struct {
	store    *Store
	httpAddr string
}

// NewServer creates a new server instance.
func NewServer(addr string, store *Store) *Server {
	return &Server{
		store:    store,
		httpAddr: addr,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/join", s.handleJoin)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/printers", s.handlePrinters)

	log.Printf("HTTP server listening on %s\n", s.httpAddr)
	return http.ListenAndServe(s.httpAddr, mux)
}

// handlePrinters routes requests for the /printers endpoint based on the HTTP method.
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

// handleGetPrinters handles GET requests to list all printers.
func (s *Server) handleGetPrinters(w http.ResponseWriter, r *http.Request) {
	printers := s.store.GetPrinters()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(printers); err != nil {
		http.Error(w, "Failed to encode printers", http.StatusInternalServerError)
	}
}

// handleAddPrinter handles POST requests to add a new printer.
// If the node is not the leader, it redirects the request.
func (s *Server) handleAddPrinter(w http.ResponseWriter, r *http.Request) {
	if s.store.raft.State() != raft.Leader {
		// Get the leader's Raft address.
		leaderRaftAddr := string(s.store.raft.Leader())
		if leaderRaftAddr == "" {
			http.Error(w, "No leader found", http.StatusServiceUnavailable)
			return
		}

		// Derive the HTTP address from the Raft address (Raft Port + 1000).
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
		redirectURL := fmt.Sprintf("http://%s:%d%s", host, httpPort, r.URL.Path)

		log.Printf("I am not the leader. Redirecting request to leader at %s", redirectURL)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// If this node is the leader, process the request.
	var p Printer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Failed to decode printer from request", http.StatusBadRequest)
		return
	}
	cmdData, err := json.Marshal(p)
	if err != nil {
		http.Error(w, "Failed to marshal printer data", http.StatusInternalServerError)
		return
	}
	cmd := command{
		Op:   "add_printer",
		Data: cmdData,
	}
	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		http.Error(w, "Failed to marshal command", http.StatusInternalServerError)
		return
	}
	if err := s.store.Apply(cmdBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleJoin is the HTTP handler for joining a node to the cluster.
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

// handleStatus is the HTTP handler for checking the node's Raft status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := map[string]string{
		"status": s.store.raft.State().String(),
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode status: %s", err)
		http.Error(w, "Failed to encode status", http.StatusInternalServerError)
	}
}
