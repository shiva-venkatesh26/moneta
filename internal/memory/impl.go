// Package memory provides the core memory service implementation
package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shivavenkatesh/moneta/internal/chunking"
	"github.com/shivavenkatesh/moneta/internal/embeddings"
	"github.com/shivavenkatesh/moneta/internal/store"
	"github.com/shivavenkatesh/moneta/pkg/types"

	"github.com/google/uuid"
)

// serviceImpl implements the Service interface
type serviceImpl struct {
	store    store.Store
	embedder embeddings.Embedder
	chunker  chunking.Chunker
	config   Config
}

// NewService creates a new memory service
func NewService(st store.Store, emb embeddings.Embedder, ch chunking.Chunker, cfg Config) Service {
	if cfg.DefaultSearchLimit <= 0 {
		cfg.DefaultSearchLimit = 10
	}
	if cfg.DefaultSearchThreshold <= 0 {
		cfg.DefaultSearchThreshold = 0.5
	}
	if cfg.DefaultProject == "" {
		cfg.DefaultProject = "default"
	}
	if cfg.EmbedBatchSize <= 0 {
		cfg.EmbedBatchSize = 50
	}

	return &serviceImpl{
		store:    st,
		embedder: emb,
		chunker:  ch,
		config:   cfg,
	}
}

// Add creates a new memory with automatic embedding generation
func (s *serviceImpl) Add(ctx context.Context, req types.AddMemoryRequest) (*types.Memory, error) {
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	project := req.Project
	if project == "" {
		project = s.config.DefaultProject
	}

	memType := req.Type
	if memType == "" {
		memType = types.TypeContext
	}

	// Generate embedding
	embedding, err := s.embedder.Embed(ctx, req.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	memory := &types.Memory{
		ID:        uuid.New().String(),
		Content:   req.Content,
		Project:   project,
		Type:      memType,
		FilePath:  req.FilePath,
		Language:  req.Language,
		Metadata:  req.Metadata,
		Embedding: embedding,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.Add(ctx, memory); err != nil {
		return nil, fmt.Errorf("failed to store memory: %w", err)
	}

	return memory, nil
}

// Search finds relevant memories using semantic search
func (s *serviceImpl) Search(ctx context.Context, req types.SearchRequest) (*types.SearchResponse, error) {
	start := time.Now()

	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Generate query embedding
	queryEmbedding, err := s.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = s.config.DefaultSearchLimit
	}

	threshold := req.Threshold
	if threshold <= 0 {
		threshold = s.config.DefaultSearchThreshold
	}

	opts := store.SearchOptions{
		Project:   req.Project,
		Limit:     limit,
		Threshold: threshold,
	}

	if req.Type != "" {
		opts.Types = []types.MemoryType{req.Type}
	}

	results, err := s.store.Search(ctx, queryEmbedding, opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return &types.SearchResponse{
		Results: results,
		Total:   len(results),
		Timing:  time.Since(start).Milliseconds(),
	}, nil
}

// Index processes a file or directory and stores as memories
func (s *serviceImpl) Index(ctx context.Context, req types.IndexRequest) (int, error) {
	if req.Path == "" {
		return 0, fmt.Errorf("path is required")
	}

	project := req.Project
	if project == "" {
		project = s.config.DefaultProject
	}

	// Expand ~ to home directory
	path := req.Path
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to access path: %w", err)
	}

	var count int
	if info.IsDir() {
		count, err = s.indexDirectory(ctx, path, project)
	} else {
		count, err = s.indexFile(ctx, path, project)
	}

	return count, err
}

// indexDirectory recursively indexes all files in a directory
func (s *serviceImpl) indexDirectory(ctx context.Context, dir, project string) (int, error) {
	var count int

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip ignored patterns
		for _, pattern := range s.config.IndexIgnore {
			if matched, _ := filepath.Match(pattern, info.Name()); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		// Only index known file types
		ext := strings.ToLower(filepath.Ext(path))
		if !isIndexableFile(ext) {
			return nil
		}

		n, err := s.indexFile(ctx, path, project)
		if err != nil {
			// Log error but continue indexing other files
			fmt.Fprintf(os.Stderr, "Warning: failed to index %s: %v\n", path, err)
			return nil
		}
		count += n

		return nil
	})

	return count, err
}

// indexFile indexes a single file
func (s *serviceImpl) indexFile(ctx context.Context, path, project string) (int, error) {
	chunks, err := s.chunker.ChunkFile(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("failed to chunk file: %w", err)
	}

	if len(chunks) == 0 {
		return 0, nil
	}

	// Generate embeddings in batches
	memories := make([]*types.Memory, 0, len(chunks))

	for i := 0; i < len(chunks); i += s.config.EmbedBatchSize {
		end := i + s.config.EmbedBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		texts := make([]string, len(batch))
		for j, chunk := range batch {
			texts[j] = chunk.Content
		}

		embeddings, err := s.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			return len(memories), fmt.Errorf("failed to generate embeddings: %w", err)
		}

		for j, chunk := range batch {
			memory := &types.Memory{
				ID:       uuid.New().String(),
				Content:  chunk.Content,
				Project:  project,
				Type:     types.TypeContext,
				FilePath: path,
				Language: chunk.Type,
				Metadata: map[string]string{
					"start_line": fmt.Sprintf("%d", chunk.StartLine),
					"end_line":   fmt.Sprintf("%d", chunk.EndLine),
					"chunk_name": chunk.Name,
				},
				Embedding: embeddings[j],
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			memories = append(memories, memory)
		}
	}

	// Batch add to store
	if err := s.store.AddBatch(ctx, memories); err != nil {
		return 0, fmt.Errorf("failed to store memories: %w", err)
	}

	return len(memories), nil
}

// Get retrieves a single memory by ID
func (s *serviceImpl) Get(ctx context.Context, id string) (*types.Memory, error) {
	return s.store.Get(ctx, id)
}

// Delete removes a memory by ID
func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// DeleteByProject removes all memories for a project
func (s *serviceImpl) DeleteByProject(ctx context.Context, project string) error {
	return s.store.DeleteByProject(ctx, project)
}

// List returns memories with filtering
func (s *serviceImpl) List(ctx context.Context, opts store.ListOptions) ([]*types.Memory, error) {
	return s.store.List(ctx, opts)
}

// Stats returns system statistics
func (s *serviceImpl) Stats(ctx context.Context) (*types.StatsResponse, error) {
	stats, err := s.store.Stats(ctx)
	if err != nil {
		return nil, err
	}
	stats.EmbeddingModel = s.embedder.Model()
	return stats, nil
}

// Close releases resources
func (s *serviceImpl) Close() error {
	if err := s.embedder.Close(); err != nil {
		return err
	}
	return s.store.Close()
}

// isIndexableFile returns true if the file extension is indexable
func isIndexableFile(ext string) bool {
	indexable := map[string]bool{
		".go":    true,
		".py":    true,
		".js":    true,
		".ts":    true,
		".tsx":   true,
		".jsx":   true,
		".rs":    true,
		".java":  true,
		".c":     true,
		".cpp":   true,
		".h":     true,
		".hpp":   true,
		".rb":    true,
		".php":   true,
		".swift": true,
		".kt":    true,
		".cs":    true,
		".md":    true,
		".txt":   true,
		".yaml":  true,
		".yml":   true,
		".toml":  true,
		".json":  true,
		".sql":   true,
		".sh":    true,
	}
	return indexable[ext]
}
