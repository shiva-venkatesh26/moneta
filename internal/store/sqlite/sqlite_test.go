package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shivavenkatesh/moneta/internal/store"
	"github.com/shivavenkatesh/moneta/pkg/types"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := New(Config{
		Path:       dbPath,
		Dimensions: 768,
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestStore_AddAndGet(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()
	memory := &types.Memory{
		ID:        "test-1",
		Content:   "Test memory content",
		Project:   "test-project",
		Type:      types.TypeContext,
		FilePath:  "/path/to/file.go",
		Language:  "go",
		Metadata:  map[string]string{"key": "value"},
		Embedding: generateTestEmbedding(768),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add
	if err := s.Add(ctx, memory); err != nil {
		t.Fatalf("failed to add memory: %v", err)
	}

	// Get
	got, err := s.Get(ctx, "test-1")
	if err != nil {
		t.Fatalf("failed to get memory: %v", err)
	}

	if got.ID != memory.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, memory.ID)
	}
	if got.Content != memory.Content {
		t.Errorf("Content mismatch: got %s, want %s", got.Content, memory.Content)
	}
	if got.Project != memory.Project {
		t.Errorf("Project mismatch: got %s, want %s", got.Project, memory.Project)
	}
	if got.Type != memory.Type {
		t.Errorf("Type mismatch: got %s, want %s", got.Type, memory.Type)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()
	_, err := s.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent memory")
	}
}

func TestStore_Update(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()
	memory := &types.Memory{
		ID:        "test-1",
		Content:   "Original content",
		Project:   "test-project",
		Type:      types.TypeContext,
		Embedding: generateTestEmbedding(768),
	}

	if err := s.Add(ctx, memory); err != nil {
		t.Fatalf("failed to add memory: %v", err)
	}

	// Update
	memory.Content = "Updated content"
	if err := s.Update(ctx, memory); err != nil {
		t.Fatalf("failed to update memory: %v", err)
	}

	// Verify
	got, err := s.Get(ctx, "test-1")
	if err != nil {
		t.Fatalf("failed to get memory: %v", err)
	}
	if got.Content != "Updated content" {
		t.Errorf("Content not updated: got %s", got.Content)
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()
	memory := &types.Memory{
		ID:      "nonexistent",
		Content: "Test",
		Project: "test",
	}

	err := s.Update(ctx, memory)
	if err == nil {
		t.Error("expected error for updating nonexistent memory")
	}
}

func TestStore_Delete(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()
	memory := &types.Memory{
		ID:        "test-1",
		Content:   "Test content",
		Project:   "test-project",
		Type:      types.TypeContext,
		Embedding: generateTestEmbedding(768),
	}

	if err := s.Add(ctx, memory); err != nil {
		t.Fatalf("failed to add memory: %v", err)
	}

	// Delete
	if err := s.Delete(ctx, "test-1"); err != nil {
		t.Fatalf("failed to delete memory: %v", err)
	}

	// Verify deleted
	_, err := s.Get(ctx, "test-1")
	if err == nil {
		t.Error("memory should be deleted")
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()
	err := s.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for deleting nonexistent memory")
	}
}

func TestStore_AddBatch(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()
	memories := make([]*types.Memory, 10)
	for i := 0; i < 10; i++ {
		memories[i] = &types.Memory{
			ID:        "batch-" + string(rune('0'+i)),
			Content:   "Batch content",
			Project:   "test-project",
			Type:      types.TypeContext,
			Embedding: generateTestEmbedding(768),
		}
	}

	if err := s.AddBatch(ctx, memories); err != nil {
		t.Fatalf("failed to add batch: %v", err)
	}

	// Verify all added
	count, err := s.Count(ctx, "test-project")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 10 {
		t.Errorf("expected 10 memories, got %d", count)
	}
}

func TestStore_DeleteByProject(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()

	// Add memories to two projects
	for i := 0; i < 5; i++ {
		s.Add(ctx, &types.Memory{
			ID:        "proj1-" + string(rune('0'+i)),
			Content:   "Content",
			Project:   "project1",
			Type:      types.TypeContext,
			Embedding: generateTestEmbedding(768),
		})
		s.Add(ctx, &types.Memory{
			ID:        "proj2-" + string(rune('0'+i)),
			Content:   "Content",
			Project:   "project2",
			Type:      types.TypeContext,
			Embedding: generateTestEmbedding(768),
		})
	}

	// Delete project1
	if err := s.DeleteByProject(ctx, "project1"); err != nil {
		t.Fatalf("failed to delete by project: %v", err)
	}

	// Verify project1 deleted
	count1, _ := s.Count(ctx, "project1")
	if count1 != 0 {
		t.Errorf("project1 should be empty, got %d", count1)
	}

	// Verify project2 untouched
	count2, _ := s.Count(ctx, "project2")
	if count2 != 5 {
		t.Errorf("project2 should have 5 memories, got %d", count2)
	}
}

func TestStore_Search(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()

	// Add some memories with different embeddings
	for i := 0; i < 5; i++ {
		embedding := generateTestEmbedding(768)
		embedding[0] = float32(i) // Make them slightly different
		s.Add(ctx, &types.Memory{
			ID:        "search-" + string(rune('0'+i)),
			Content:   "Search content",
			Project:   "test-project",
			Type:      types.TypeContext,
			Embedding: embedding,
		})
	}

	// Search
	queryEmbedding := generateTestEmbedding(768)
	queryEmbedding[0] = 2.5 // Most similar to search-2 or search-3

	results, err := s.Search(ctx, queryEmbedding, store.SearchOptions{
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one result")
	}

	// Results should be sorted by similarity
	for i := 1; i < len(results); i++ {
		if results[i].Similarity > results[i-1].Similarity {
			t.Error("results not sorted by similarity descending")
		}
	}
}

func TestStore_Search_WithFilters(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()

	// Add memories with different types and projects
	s.Add(ctx, &types.Memory{
		ID:        "mem-1",
		Content:   "Architecture decision",
		Project:   "project1",
		Type:      types.TypeArchitecture,
		Embedding: generateTestEmbedding(768),
	})
	s.Add(ctx, &types.Memory{
		ID:        "mem-2",
		Content:   "Code pattern",
		Project:   "project1",
		Type:      types.TypePattern,
		Embedding: generateTestEmbedding(768),
	})
	s.Add(ctx, &types.Memory{
		ID:        "mem-3",
		Content:   "Different project",
		Project:   "project2",
		Type:      types.TypeContext,
		Embedding: generateTestEmbedding(768),
	})

	// Search with project filter
	results, err := s.Search(ctx, generateTestEmbedding(768), store.SearchOptions{
		Project: "project1",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	for _, r := range results {
		if r.Memory.Project != "project1" {
			t.Errorf("got memory from wrong project: %s", r.Memory.Project)
		}
	}

	// Search with type filter
	results, err = s.Search(ctx, generateTestEmbedding(768), store.SearchOptions{
		Types: []types.MemoryType{types.TypeArchitecture},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	for _, r := range results {
		if r.Memory.Type != types.TypeArchitecture {
			t.Errorf("got wrong type: %s", r.Memory.Type)
		}
	}
}

func TestStore_List(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()

	// Add memories
	for i := 0; i < 10; i++ {
		s.Add(ctx, &types.Memory{
			ID:        "list-" + string(rune('0'+i)),
			Content:   "List content",
			Project:   "test-project",
			Type:      types.TypeContext,
			Embedding: generateTestEmbedding(768),
		})
	}

	// List with limit
	results, err := s.List(ctx, store.ListOptions{
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// List with offset
	results, err = s.List(ctx, store.ListOptions{
		Limit:  5,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestStore_Count(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()

	// Add memories to different projects
	for i := 0; i < 3; i++ {
		s.Add(ctx, &types.Memory{
			ID:        "proj1-" + string(rune('0'+i)),
			Content:   "Content",
			Project:   "project1",
			Type:      types.TypeContext,
			Embedding: generateTestEmbedding(768),
		})
	}
	for i := 0; i < 2; i++ {
		s.Add(ctx, &types.Memory{
			ID:        "proj2-" + string(rune('0'+i)),
			Content:   "Content",
			Project:   "project2",
			Type:      types.TypeContext,
			Embedding: generateTestEmbedding(768),
		})
	}

	// Count all
	total, err := s.Count(ctx, "")
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected 5 total, got %d", total)
	}

	// Count project1
	count1, err := s.Count(ctx, "project1")
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count1 != 3 {
		t.Errorf("expected 3 for project1, got %d", count1)
	}
}

func TestStore_Stats(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()

	// Add memories
	s.Add(ctx, &types.Memory{
		ID:        "stats-1",
		Content:   "Content",
		Project:   "project1",
		Type:      types.TypeArchitecture,
		Embedding: generateTestEmbedding(768),
	})
	s.Add(ctx, &types.Memory{
		ID:        "stats-2",
		Content:   "Content",
		Project:   "project2",
		Type:      types.TypePattern,
		Embedding: generateTestEmbedding(768),
	})

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	if stats.TotalMemories != 2 {
		t.Errorf("expected 2 total memories, got %d", stats.TotalMemories)
	}
	if stats.ProjectCount != 2 {
		t.Errorf("expected 2 projects, got %d", stats.ProjectCount)
	}
	if len(stats.MemoriesByType) == 0 {
		t.Error("expected memories by type to be populated")
	}
}

func TestStore_Compact(t *testing.T) {
	s := createTestStore(t)
	defer s.Close()

	ctx := context.Background()

	// Add and delete some memories
	for i := 0; i < 10; i++ {
		s.Add(ctx, &types.Memory{
			ID:        "compact-" + string(rune('0'+i)),
			Content:   "Content",
			Project:   "test",
			Type:      types.TypeContext,
			Embedding: generateTestEmbedding(768),
		})
	}
	for i := 0; i < 5; i++ {
		s.Delete(ctx, "compact-"+string(rune('0'+i)))
	}

	// Compact
	if err := s.Compact(ctx); err != nil {
		t.Fatalf("compact failed: %v", err)
	}

	// Verify remaining memories still accessible
	count, err := s.Count(ctx, "")
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 memories after compact, got %d", count)
	}
}

// Helper functions

func createTestStore(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := New(Config{
		Path:       dbPath,
		Dimensions: 768,
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s
}

func generateTestEmbedding(dims int) []float32 {
	embedding := make([]float32, dims)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}
	return embedding
}

// Benchmarks

func BenchmarkStore_Add(b *testing.B) {
	tmpDir := b.TempDir()
	s, _ := New(Config{
		Path:       filepath.Join(tmpDir, "bench.db"),
		Dimensions: 768,
	})
	defer s.Close()

	ctx := context.Background()
	embedding := generateTestEmbedding(768)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(ctx, &types.Memory{
			ID:        "bench-" + string(rune(i)),
			Content:   "Benchmark content",
			Project:   "bench",
			Type:      types.TypeContext,
			Embedding: embedding,
		})
	}
}

func BenchmarkStore_Search(b *testing.B) {
	tmpDir := b.TempDir()
	s, _ := New(Config{
		Path:       filepath.Join(tmpDir, "bench.db"),
		Dimensions: 768,
	})
	defer s.Close()

	ctx := context.Background()
	embedding := generateTestEmbedding(768)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		s.Add(ctx, &types.Memory{
			ID:        "search-" + string(rune(i)),
			Content:   "Search content",
			Project:   "bench",
			Type:      types.TypeContext,
			Embedding: embedding,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Search(ctx, embedding, store.SearchOptions{
			Limit: 10,
		})
	}
}
