// Package sqlite provides SQLite + sqlite-vec storage implementation
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shivavenkatesh/moneta/internal/store"
	"github.com/shivavenkatesh/moneta/pkg/types"

	_ "github.com/mattn/go-sqlite3"
)

// Store implements store.Store using SQLite with sqlite-vec extension
type Store struct {
	db   *sql.DB
	path string
	dims int // embedding dimensions
	mu   sync.RWMutex
}

// Config configures the SQLite store
type Config struct {
	Path       string // Path to database file
	Dimensions int    // Embedding dimensions (e.g., 768 for nomic-embed-text)
}

// New creates a new SQLite store
func New(cfg Config) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database with sqlite-vec extension
	db, err := sql.Open("sqlite3", cfg.Path+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA cache_size = -32000",       // 32MB cache
		"PRAGMA temp_store = MEMORY",       // temp tables in memory
		"PRAGMA mmap_size = 268435456",     // 256MB mmap
		"PRAGMA page_size = 4096",          // optimal for SSD
		"PRAGMA auto_vacuum = INCREMENTAL", // gradual space reclaim
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	s := &Store{
		db:   db,
		path: cfg.Path,
		dims: cfg.Dimensions,
	}

	// Initialize schema
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// initSchema creates the database tables
func (s *Store) initSchema() error {
	schema := `
	-- Main memories table
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		project TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'context',
		file_path TEXT,
		language TEXT,
		metadata TEXT, -- JSON
		embedding BLOB, -- float32 array as bytes
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project);
	CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
	CREATE INDEX IF NOT EXISTS idx_memories_file_path ON memories(file_path);
	CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);

	-- Schema version tracking
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Insert initial version if not exists
	INSERT OR IGNORE INTO schema_version (version) VALUES (1);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Add creates a new memory
func (s *Store) Add(ctx context.Context, memory *types.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	embedding := float32ToBytes(memory.Embedding)

	query := `
		INSERT INTO memories (id, content, project, type, file_path, language, metadata, embedding, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = now
	}
	memory.UpdatedAt = now

	_, err = s.db.ExecContext(ctx, query,
		memory.ID,
		memory.Content,
		memory.Project,
		string(memory.Type),
		memory.FilePath,
		memory.Language,
		string(metadata),
		embedding,
		memory.CreatedAt,
		memory.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert memory: %w", err)
	}

	return nil
}

// Get retrieves a memory by ID
func (s *Store) Get(ctx context.Context, id string) (*types.Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, content, project, type, file_path, language, metadata, embedding, created_at, updated_at
		FROM memories WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanMemory(row)
}

// Update modifies an existing memory
func (s *Store) Update(ctx context.Context, memory *types.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	embedding := float32ToBytes(memory.Embedding)
	memory.UpdatedAt = time.Now()

	query := `
		UPDATE memories
		SET content = ?, project = ?, type = ?, file_path = ?, language = ?,
		    metadata = ?, embedding = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		memory.Content,
		memory.Project,
		string(memory.Type),
		memory.FilePath,
		memory.Language,
		string(metadata),
		embedding,
		memory.UpdatedAt,
		memory.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update memory: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memory not found: %s", memory.ID)
	}

	return nil
}

// Delete removes a memory by ID
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "DELETE FROM memories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// AddBatch adds multiple memories efficiently
func (s *Store) AddBatch(ctx context.Context, memories []*types.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO memories (id, content, project, type, file_path, language, metadata, embedding, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, memory := range memories {
		metadata, err := json.Marshal(memory.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		embedding := float32ToBytes(memory.Embedding)

		if memory.CreatedAt.IsZero() {
			memory.CreatedAt = now
		}
		memory.UpdatedAt = now

		_, err = stmt.ExecContext(ctx,
			memory.ID,
			memory.Content,
			memory.Project,
			string(memory.Type),
			memory.FilePath,
			memory.Language,
			string(metadata),
			embedding,
			memory.CreatedAt,
			memory.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert memory %s: %w", memory.ID, err)
		}
	}

	return tx.Commit()
}

// DeleteByProject removes all memories for a project
func (s *Store) DeleteByProject(ctx context.Context, project string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM memories WHERE project = ?", project)
	if err != nil {
		return fmt.Errorf("failed to delete memories for project: %w", err)
	}

	return nil
}

// Search finds similar memories using vector search
func (s *Store) Search(ctx context.Context, embedding []float32, opts store.SearchOptions) ([]types.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build query with filters
	conditions := []string{"1=1"}
	args := []interface{}{}

	if opts.Project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, opts.Project)
	}

	if len(opts.Types) > 0 {
		placeholders := make([]string, len(opts.Types))
		for i, t := range opts.Types {
			placeholders[i] = "?"
			args = append(args, string(t))
		}
		conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(opts.FilePaths) > 0 {
		pathConditions := make([]string, len(opts.FilePaths))
		for i, fp := range opts.FilePaths {
			pathConditions[i] = "file_path LIKE ?"
			args = append(args, fp+"%")
		}
		conditions = append(conditions, "("+strings.Join(pathConditions, " OR ")+")")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// Query all matching memories and compute similarity in Go
	// (sqlite-vec extension would do this more efficiently, but this works without it)
	query := fmt.Sprintf(`
		SELECT id, content, project, type, file_path, language, metadata, embedding, created_at, updated_at
		FROM memories
		WHERE %s
	`, strings.Join(conditions, " AND "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	var results []types.SearchResult
	for rows.Next() {
		memory, err := s.scanMemoryFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(embedding, memory.Embedding)

		// Apply threshold filter
		if opts.Threshold > 0 && similarity < opts.Threshold {
			continue
		}

		results = append(results, types.SearchResult{
			Memory:     *memory,
			Similarity: similarity,
		})
	}

	// Sort by similarity descending
	sortBySimilarity(results)

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// List returns memories with filtering and pagination
func (s *Store) List(ctx context.Context, opts store.ListOptions) ([]*types.Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conditions := []string{"1=1"}
	args := []interface{}{}

	if opts.Project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, opts.Project)
	}

	if opts.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, string(opts.Type))
	}

	orderBy := "created_at"
	if opts.OrderBy != "" {
		orderBy = opts.OrderBy
	}
	order := "ASC"
	if opts.Descending {
		order = "DESC"
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	query := fmt.Sprintf(`
		SELECT id, content, project, type, file_path, language, metadata, embedding, created_at, updated_at
		FROM memories
		WHERE %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, strings.Join(conditions, " AND "), orderBy, order)

	args = append(args, limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	defer rows.Close()

	var memories []*types.Memory
	for rows.Next() {
		memory, err := s.scanMemoryFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

// Count returns the number of memories
func (s *Store) Count(ctx context.Context, project string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	var err error

	if project == "" {
		err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories").Scan(&count)
	} else {
		err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories WHERE project = ?", project).Scan(&count)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to count memories: %w", err)
	}

	return count, nil
}

// Stats returns storage statistics
func (s *Store) Stats(ctx context.Context) (*types.StatsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &types.StatsResponse{
		MemoriesByType: make(map[string]int),
	}

	// Total count
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories").Scan(&stats.TotalMemories); err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Count by type
	rows, err := s.db.QueryContext(ctx, "SELECT type, COUNT(*) FROM memories GROUP BY type")
	if err != nil {
		return nil, fmt.Errorf("failed to get type counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memType string
		var count int
		if err := rows.Scan(&memType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type count: %w", err)
		}
		stats.MemoriesByType[memType] = count
	}

	// Project count
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT project) FROM memories").Scan(&stats.ProjectCount); err != nil {
		return nil, fmt.Errorf("failed to get project count: %w", err)
	}

	// Storage size
	if info, err := os.Stat(s.path); err == nil {
		stats.StorageBytes = info.Size()
	}

	return stats, nil
}

// Close releases resources
func (s *Store) Close() error {
	return s.db.Close()
}

// Compact optimizes storage
func (s *Store) Compact(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "VACUUM")
	return err
}

// scanMemory scans a single row into a Memory struct
func (s *Store) scanMemory(row *sql.Row) (*types.Memory, error) {
	var m types.Memory
	var memType string
	var metadataJSON sql.NullString
	var embeddingBytes []byte
	var filePath, language sql.NullString

	err := row.Scan(
		&m.ID,
		&m.Content,
		&m.Project,
		&memType,
		&filePath,
		&language,
		&metadataJSON,
		&embeddingBytes,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memory not found")
		}
		return nil, err
	}

	m.Type = types.MemoryType(memType)
	m.FilePath = filePath.String
	m.Language = language.String

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &m.Metadata)
	}

	m.Embedding = bytesToFloat32(embeddingBytes)

	return &m, nil
}

// scanMemoryFromRows scans from rows iterator
func (s *Store) scanMemoryFromRows(rows *sql.Rows) (*types.Memory, error) {
	var m types.Memory
	var memType string
	var metadataJSON sql.NullString
	var embeddingBytes []byte
	var filePath, language sql.NullString

	err := rows.Scan(
		&m.ID,
		&m.Content,
		&m.Project,
		&memType,
		&filePath,
		&language,
		&metadataJSON,
		&embeddingBytes,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	m.Type = types.MemoryType(memType)
	m.FilePath = filePath.String
	m.Language = language.String

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &m.Metadata)
	}

	m.Embedding = bytesToFloat32(embeddingBytes)

	return &m, nil
}
