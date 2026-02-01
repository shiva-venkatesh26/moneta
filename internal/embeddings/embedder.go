// Package embeddings provides vector embedding generation
package embeddings

import "context"

// Embedder generates vector embeddings from text
type Embedder interface {
	// Embed generates an embedding for a single text
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts efficiently
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the embedding vector dimensions
	Dimensions() int

	// Model returns the model identifier
	Model() string

	// Close releases any resources
	Close() error
}
