// Package types defines the core data structures for Moneta
package types

import "time"

// Memory represents a single memory entry
type Memory struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Project   string            `json:"project"`
	Type      MemoryType        `json:"type"`
	FilePath  string            `json:"file_path,omitempty"`
	Language  string            `json:"language,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Embedding []float32         `json:"-"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// MemoryType categorizes memories for better organization
type MemoryType string

const (
	TypeArchitecture MemoryType = "architecture" // High-level design decisions
	TypePattern      MemoryType = "pattern"      // Code patterns and conventions
	TypeDecision     MemoryType = "decision"     // Why something was done
	TypeGotcha       MemoryType = "gotcha"       // Bugs, edge cases, warnings
	TypeContext      MemoryType = "context"      // General context about code
	TypePreference   MemoryType = "preference"   // User coding preferences
)

// Chunk represents a piece of code or text that was chunked
type Chunk struct {
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Type      string `json:"type"` // function, class, import, etc.
	Name      string `json:"name"` // function/class name if applicable
}

// SearchResult represents a memory match with similarity score
type SearchResult struct {
	Memory     Memory  `json:"memory"`
	Similarity float32 `json:"similarity"`
}

// AddMemoryRequest is the request payload for adding a memory
type AddMemoryRequest struct {
	Content  string            `json:"content"`
	Project  string            `json:"project"`
	Type     MemoryType        `json:"type,omitempty"`
	FilePath string            `json:"file_path,omitempty"`
	Language string            `json:"language,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SearchRequest is the request payload for searching memories
type SearchRequest struct {
	Query     string     `json:"query"`
	Project   string     `json:"project,omitempty"`
	Type      MemoryType `json:"type,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Threshold float32    `json:"threshold,omitempty"`
}

// SearchResponse is the response payload for search
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Timing  int64          `json:"timing_ms"`
}

// IndexRequest is the request payload for indexing a file or directory
type IndexRequest struct {
	Path     string `json:"path"`
	Project  string `json:"project"`
	Language string `json:"language,omitempty"` // Auto-detect if empty
}

// StatsResponse contains statistics about the memory store
type StatsResponse struct {
	TotalMemories  int            `json:"total_memories"`
	MemoriesByType map[string]int `json:"memories_by_type"`
	ProjectCount   int            `json:"project_count"`
	EmbeddingModel string         `json:"embedding_model"`
	StorageBytes   int64          `json:"storage_bytes"`
}
