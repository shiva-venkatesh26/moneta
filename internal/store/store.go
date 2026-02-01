// Package store defines the vector storage interface
package store

import (
	"context"

	"github.com/shivavenkatesh/moneta/pkg/types"
)

// Store handles persistence of memories and vector search
type Store interface {
	// Add creates a new memory
	Add(ctx context.Context, memory *types.Memory) error

	// Get retrieves a memory by ID
	Get(ctx context.Context, id string) (*types.Memory, error)

	// Update modifies an existing memory
	Update(ctx context.Context, memory *types.Memory) error

	// Delete removes a memory by ID
	Delete(ctx context.Context, id string) error

	// AddBatch adds multiple memories efficiently
	AddBatch(ctx context.Context, memories []*types.Memory) error

	// DeleteByProject removes all memories for a project
	DeleteByProject(ctx context.Context, project string) error

	// Search finds similar memories using vector search
	Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]types.SearchResult, error)

	// List returns memories with filtering and pagination
	List(ctx context.Context, opts ListOptions) ([]*types.Memory, error)

	// Count returns the number of memories, optionally filtered by project
	Count(ctx context.Context, project string) (int, error)

	// Stats returns storage statistics
	Stats(ctx context.Context) (*types.StatsResponse, error)

	// Close releases resources
	Close() error

	// Compact optimizes storage (VACUUM)
	Compact(ctx context.Context) error
}

// SearchOptions configures vector search
type SearchOptions struct {
	Project   string
	Types     []types.MemoryType
	Limit     int
	Threshold float32  // Minimum similarity score (0-1)
	FilePaths []string // Filter by file paths (prefix match)
}

// ListOptions configures listing queries
type ListOptions struct {
	Project    string
	Type       types.MemoryType
	Limit      int
	Offset     int
	OrderBy    string // "created_at", "updated_at"
	Descending bool
}
