// Package memory provides the core memory service
package memory

import (
	"context"

	"github.com/shivavenkatesh/moneta/internal/store"
	"github.com/shivavenkatesh/moneta/pkg/types"
)

// Service orchestrates memory operations
type Service interface {
	// Add creates a new memory with automatic embedding generation
	Add(ctx context.Context, req types.AddMemoryRequest) (*types.Memory, error)

	// Search finds relevant memories using semantic search
	Search(ctx context.Context, req types.SearchRequest) (*types.SearchResponse, error)

	// Index processes a file or directory and stores as memories
	Index(ctx context.Context, req types.IndexRequest) (int, error)

	// Get retrieves a single memory by ID
	Get(ctx context.Context, id string) (*types.Memory, error)

	// Delete removes a memory by ID
	Delete(ctx context.Context, id string) error

	// DeleteByProject removes all memories for a project
	DeleteByProject(ctx context.Context, project string) error

	// List returns memories with filtering
	List(ctx context.Context, opts store.ListOptions) ([]*types.Memory, error)

	// Stats returns system statistics
	Stats(ctx context.Context) (*types.StatsResponse, error)

	// Close releases resources
	Close() error
}

// Config configures the memory service
type Config struct {
	DataDir        string   // Directory for data storage
	EmbedBatchSize int      // Batch size for embedding generation
	IndexIgnore    []string // Glob patterns to ignore during indexing
	DefaultProject string   // Default project name if not specified

	// Search defaults
	DefaultSearchLimit     int
	DefaultSearchThreshold float32
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		DataDir:                "~/.moneta",
		EmbedBatchSize:         50,
		IndexIgnore:            []string{".git", "node_modules", "vendor", "__pycache__", ".venv"},
		DefaultProject:         "default",
		DefaultSearchLimit:     10,
		DefaultSearchThreshold: 0.5,
	}
}
