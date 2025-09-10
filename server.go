package main

import (
	"encoding/json"
	"log"
	"net/http"
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
	// We will add more handlers here later for /set, /get, etc.

	log.Printf("HTTP server listening on %s\n", s.httpAddr)
	return http.ListenAndServe(s.httpAddr, mux)
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
