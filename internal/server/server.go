// Package server provides the HTTP API server for Moneta
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shivavenkatesh/moneta/internal/memory"
	"github.com/shivavenkatesh/moneta/internal/store"
	"github.com/shivavenkatesh/moneta/pkg/types"
)

// Server is the HTTP API server
type Server struct {
	svc    memory.Service
	config Config
	server *http.Server
}

// Config configures the server
type Config struct {
	Host string
	Port int
}

// New creates a new server
func New(svc memory.Service, cfg Config) *Server {
	return &Server{
		svc:    svc,
		config: cfg,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/memory", s.handleMemory)
	mux.HandleFunc("/memory/", s.handleMemoryByID)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/index", s.handleIndex)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/projects", s.handleProjects)

	// CORS middleware for Claude Code integration
	handler := corsMiddleware(mux)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// corsMiddleware adds CORS headers for Claude Code integration
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleMemory handles POST /memory (add memory)
func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.AddMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	memory, err := s.svc.Add(r.Context(), req)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, memory, http.StatusCreated)
}

// handleMemoryByID handles GET/DELETE /memory/:id
func (s *Server) handleMemoryByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/memory/")
	if id == "" {
		writeError(w, "Memory ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		memory, err := s.svc.Get(r.Context(), id)
		if err != nil {
			writeError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, memory, http.StatusOK)

	case http.MethodDelete:
		if err := s.svc.Delete(r.Context(), id); err != nil {
			writeError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]bool{"deleted": true}, http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSearch handles POST /search
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.svc.Search(r.Context(), req)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, resp, http.StatusOK)
}

// handleIndex handles POST /index
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	count, err := s.svc.Index(r.Context(), req)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]int{"indexed": count}, http.StatusOK)
}

// handleStats handles GET /stats
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := s.svc.Stats(r.Context())
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, stats, http.StatusOK)
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "version": "0.1.0"}, http.StatusOK)
}

// handleProjects handles GET /projects (list projects)
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get list of projects from stats
	// This is a simplified implementation - could be optimized
	memories, err := s.svc.List(r.Context(), store.ListOptions{
		Limit: 1000,
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract unique projects
	projectSet := make(map[string]bool)
	for _, m := range memories {
		projectSet[m.Project] = true
	}

	projects := make([]string, 0, len(projectSet))
	for p := range projectSet {
		projects = append(projects, p)
	}

	writeJSON(w, map[string][]string{"projects": projects}, http.StatusOK)
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
