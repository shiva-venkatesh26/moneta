// Package chunking provides text and code chunking functionality
package chunking

import (
	"context"

	"github.com/shivavenkatesh/moneta/pkg/types"
)

// Chunker splits content into semantic chunks
type Chunker interface {
	// Chunk splits content into pieces based on options
	Chunk(ctx context.Context, content string, opts ChunkOptions) ([]types.Chunk, error)

	// ChunkFile reads and chunks a file, detecting language automatically
	ChunkFile(ctx context.Context, path string) ([]types.Chunk, error)

	// SupportedLanguages returns list of supported programming languages
	SupportedLanguages() []string
}

// ChunkOptions configures chunking behavior
type ChunkOptions struct {
	Language string // Programming language or "text" for plain text
	MaxSize  int    // Maximum chunk size in characters
	Overlap  int    // Overlap between chunks in characters
	Semantic bool   // Use semantic boundaries (functions, classes)
}

// DefaultChunkOptions returns sensible defaults
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		Language: "text",
		MaxSize:  1500,
		Overlap:  100,
		Semantic: true,
	}
}
