// Package embeddings provides ONNX-based local embedding generation
// This is a leaner alternative to Ollama for embeddings-only use cases
package embeddings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/shivavenkatesh/moneta/internal/cache"
)

// ONNXClient provides embeddings using ONNX Runtime
// This is significantly leaner than Ollama for embeddings-only use cases
//
// Supported models:
// - all-MiniLM-L6-v2 (23MB, 384 dims) - fast, good quality
// - nomic-embed-text-v1 (274MB, 768 dims) - higher quality
// - bge-small-en-v1.5 (33MB, 384 dims) - good balance
type ONNXClient struct {
	modelPath string
	dims      int
	cache     *cache.EmbeddingCache
	mu        sync.Mutex

	// ONNX runtime session (lazily initialized)
	// session *ort.Session
	initialized bool
}

// ONNXConfig configures the ONNX embedder
type ONNXConfig struct {
	ModelPath  string // Path to .onnx model file
	Dimensions int    // Embedding dimensions
	CacheSize  int    // LRU cache size
}

// DefaultONNXConfig returns config for all-MiniLM-L6-v2
func DefaultONNXConfig() ONNXConfig {
	home, _ := os.UserHomeDir()
	return ONNXConfig{
		ModelPath:  filepath.Join(home, ".moneta", "models", "all-MiniLM-L6-v2.onnx"),
		Dimensions: 384,
		CacheSize:  1000,
	}
}

// NewONNXClient creates a new ONNX-based embedder
// Note: This requires the ONNX runtime library to be available
func NewONNXClient(cfg ONNXConfig) (*ONNXClient, error) {
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 1000
	}

	return &ONNXClient{
		modelPath: cfg.ModelPath,
		dims:      cfg.Dimensions,
		cache:     cache.NewEmbeddingCache(cfg.CacheSize),
	}, nil
}

// Embed generates an embedding for the given text
func (c *ONNXClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// Check cache first
	if embedding, ok := c.cache.Get(text); ok {
		return embedding, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Lazy initialization
	if !c.initialized {
		if err := c.initialize(); err != nil {
			return nil, err
		}
	}

	// TODO: Implement actual ONNX inference
	// For now, return an error indicating ONNX support is not yet implemented
	// The actual implementation would:
	// 1. Tokenize the input text
	// 2. Run inference through the ONNX model
	// 3. Return the embedding vector

	return nil, fmt.Errorf("ONNX support not yet implemented - use Ollama for now")
}

// EmbedBatch generates embeddings for multiple texts
func (c *ONNXClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := c.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

// Dimensions returns the embedding dimensions
func (c *ONNXClient) Dimensions() int {
	return c.dims
}

// Model returns the model name
func (c *ONNXClient) Model() string {
	return filepath.Base(c.modelPath)
}

// Close releases resources
func (c *ONNXClient) Close() error {
	// Close ONNX session if initialized
	return nil
}

// initialize loads the ONNX model
func (c *ONNXClient) initialize() error {
	// Check if model file exists
	if _, err := os.Stat(c.modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s\n\nTo use ONNX embeddings, download a model:\n"+
			"  mkdir -p ~/.moneta/models\n"+
			"  curl -L -o ~/.moneta/models/all-MiniLM-L6-v2.onnx \\\n"+
			"    https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx",
			c.modelPath)
	}

	// TODO: Initialize ONNX runtime session
	// This would use github.com/yalue/onnxruntime_go or similar

	c.initialized = true
	return nil
}

// DownloadModel downloads a pre-trained embedding model
func DownloadModel(modelName, destPath string) error {
	// Model URLs for common embedding models
	models := map[string]string{
		"all-MiniLM-L6-v2":  "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx",
		"bge-small-en-v1.5": "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/onnx/model.onnx",
	}

	url, ok := models[modelName]
	if !ok {
		return fmt.Errorf("unknown model: %s (available: all-MiniLM-L6-v2, bge-small-en-v1.5)", modelName)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	fmt.Printf("Downloading %s to %s...\n", modelName, destPath)
	fmt.Printf("URL: %s\n", url)

	// TODO: Implement actual download with progress bar
	// For now, just print instructions
	fmt.Printf("\nRun manually:\n  curl -L -o %s %s\n", destPath, url)

	return nil
}
