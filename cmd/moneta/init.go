package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shivavenkatesh/moneta/internal/chunking"
	"github.com/shivavenkatesh/moneta/internal/embeddings"
	"github.com/shivavenkatesh/moneta/internal/memory"
	"github.com/shivavenkatesh/moneta/internal/store/sqlite"
)

// initService creates and initializes the memory service
func initService() (memory.Service, error) {
	// Determine data directory
	dir := dataDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(home, ".moneta")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize store
	dbPath := filepath.Join(dir, "moneta.db")
	store, err := sqlite.New(sqlite.Config{
		Path:       dbPath,
		Dimensions: 768, // nomic-embed-text dimensions
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	// Initialize embedder
	embedder := embeddings.NewOllamaClient(embeddings.OllamaConfig{
		Dimensions: 768,
		CacheSize:  1000,
	})

	// Initialize chunker
	chunker := chunking.NewCodeChunker(1500, 100)

	// Create service
	cfg := memory.Config{
		DataDir:                dir,
		EmbedBatchSize:         50,
		IndexIgnore:            []string{".git", "node_modules", "vendor", "__pycache__", ".venv", "dist", "build"},
		DefaultSearchLimit:     10,
		DefaultSearchThreshold: 0.5,
	}

	svc := memory.NewService(store, embedder, chunker, cfg)

	if verbose {
		fmt.Printf("Data directory: %s\n", dir)
		fmt.Printf("Database: %s\n", dbPath)
	}

	return svc, nil
}
