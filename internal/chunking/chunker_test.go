package chunking

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultChunkOptions(t *testing.T) {
	opts := DefaultChunkOptions()

	if opts.Language != "text" {
		t.Errorf("expected language 'text', got %s", opts.Language)
	}
	if opts.MaxSize != 1500 {
		t.Errorf("expected MaxSize 1500, got %d", opts.MaxSize)
	}
	if opts.Overlap != 100 {
		t.Errorf("expected Overlap 100, got %d", opts.Overlap)
	}
	if !opts.Semantic {
		t.Error("expected Semantic to be true")
	}
}

func TestLineChunker_Chunk_Empty(t *testing.T) {
	chunker := NewLineChunker(100, 10)
	chunks, err := chunker.Chunk(context.Background(), "", ChunkOptions{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestLineChunker_Chunk_SingleChunk(t *testing.T) {
	chunker := NewLineChunker(1000, 10)
	content := "line 1\nline 2\nline 3"

	chunks, err := chunker.Chunk(context.Background(), content, ChunkOptions{MaxSize: 1000})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0].StartLine != 1 {
		t.Errorf("expected StartLine 1, got %d", chunks[0].StartLine)
	}
}

func TestLineChunker_Chunk_MultipleChunks(t *testing.T) {
	chunker := NewLineChunker(30, 5)
	content := "line 1 with some text\nline 2 with more text\nline 3 here\nline 4 is last"

	chunks, err := chunker.Chunk(context.Background(), content, ChunkOptions{MaxSize: 30, Overlap: 5})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks, got %d", len(chunks))
	}

	// Verify chunks have proper line numbers
	for i, chunk := range chunks {
		if chunk.StartLine < 1 {
			t.Errorf("chunk %d: invalid StartLine %d", i, chunk.StartLine)
		}
		if chunk.EndLine < chunk.StartLine {
			t.Errorf("chunk %d: EndLine %d < StartLine %d", i, chunk.EndLine, chunk.StartLine)
		}
	}
}

func TestLineChunker_ChunkFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	chunker := NewLineChunker(1000, 10)
	chunks, err := chunker.ChunkFile(context.Background(), tmpFile)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Should detect Go language
	if chunks[0].Type != "go" {
		t.Errorf("expected type 'go', got %s", chunks[0].Type)
	}
}

func TestLineChunker_ChunkFile_NotFound(t *testing.T) {
	chunker := NewLineChunker(1000, 10)
	_, err := chunker.ChunkFile(context.Background(), "/nonexistent/file.go")

	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLineChunker_SupportedLanguages(t *testing.T) {
	chunker := NewLineChunker(1000, 10)
	languages := chunker.SupportedLanguages()

	if len(languages) == 0 {
		t.Error("expected at least one supported language")
	}

	// Check for common languages
	hasGo := false
	for _, lang := range languages {
		if lang == "go" {
			hasGo = true
			break
		}
	}
	if !hasGo {
		t.Error("expected 'go' to be in supported languages")
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".go", "go"},
		{".py", "python"},
		{".js", "javascript"},
		{".ts", "typescript"},
		{".tsx", "typescript"},
		{".rs", "rust"},
		{".java", "java"},
		{".c", "c"},
		{".h", "c"},
		{".cpp", "cpp"},
		{".hpp", "cpp"},
		{".rb", "ruby"},
		{".php", "php"},
		{".swift", "swift"},
		{".kt", "kotlin"},
		{".cs", "csharp"},
		{".md", "markdown"},
		{".json", "json"},
		{".yaml", "yaml"},
		{".yml", "yaml"},
		{".toml", "toml"},
		{".sql", "sql"},
		{".sh", "shell"},
		{".unknown", "text"},
		{"", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := detectLanguage(tt.ext)
			if result != tt.expected {
				t.Errorf("detectLanguage(%s) = %s, want %s", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestFindOverlapStart(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		overlap  int
		expected string
	}{
		{
			name:     "short content",
			content:  "hi",
			overlap:  10,
			expected: "hi",
		},
		{
			name:     "with newline",
			content:  "line 1\nline 2",
			overlap:  8,
			expected: "line 2",
		},
		{
			name:     "no newline in overlap",
			content:  "abcdefghij",
			overlap:  5,
			expected: "fghij",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findOverlapStart(tt.content, tt.overlap)
			if result != tt.expected {
				t.Errorf("findOverlapStart() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Code Chunker Tests

func TestCodeChunker_ChunkGo(t *testing.T) {
	chunker := NewCodeChunker(2000, 100)
	content := `package main

import "fmt"

func hello() {
	fmt.Println("Hello")
}

func world() {
	fmt.Println("World")
}
`
	opts := ChunkOptions{
		Language: "go",
		MaxSize:  2000,
		Semantic: true,
	}

	chunks, err := chunker.Chunk(context.Background(), content, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have chunks for the functions
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Check function names are extracted
	foundHello := false
	foundWorld := false
	for _, chunk := range chunks {
		if chunk.Name == "hello" {
			foundHello = true
		}
		if chunk.Name == "world" {
			foundWorld = true
		}
	}

	if !foundHello || !foundWorld {
		t.Error("expected to find 'hello' and 'world' functions")
	}
}

func TestCodeChunker_ChunkPython(t *testing.T) {
	chunker := NewCodeChunker(2000, 100)
	content := `import os

def greet(name):
    print(f"Hello, {name}")

class Person:
    def __init__(self, name):
        self.name = name
`
	opts := ChunkOptions{
		Language: "python",
		MaxSize:  2000,
		Semantic: true,
	}

	chunks, err := chunker.Chunk(context.Background(), content, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestCodeChunker_NonSemantic(t *testing.T) {
	chunker := NewCodeChunker(100, 10)
	content := "line 1\nline 2\nline 3"

	opts := ChunkOptions{
		Language: "go",
		MaxSize:  100,
		Semantic: false, // Disable semantic chunking
	}

	chunks, err := chunker.Chunk(context.Background(), content, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should fall back to line-based chunking
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestCodeChunker_ChunkFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
	println("test")
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	chunker := NewCodeChunker(2000, 100)
	chunks, err := chunker.ChunkFile(context.Background(), tmpFile)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestCodeChunker_SupportedLanguages(t *testing.T) {
	chunker := NewCodeChunker(1000, 10)
	languages := chunker.SupportedLanguages()

	expected := []string{"go", "python", "javascript", "typescript"}
	if len(languages) != len(expected) {
		t.Errorf("expected %d languages, got %d", len(expected), len(languages))
	}
}

// Benchmarks

func BenchmarkLineChunker_SmallFile(b *testing.B) {
	chunker := NewLineChunker(1000, 100)
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	opts := ChunkOptions{MaxSize: 1000}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Chunk(ctx, content, opts)
	}
}

func BenchmarkLineChunker_LargeFile(b *testing.B) {
	chunker := NewLineChunker(1000, 100)

	// Generate a large content
	var content string
	for i := 0; i < 1000; i++ {
		content += "This is line " + string(rune(i)) + " of the content\n"
	}
	opts := ChunkOptions{MaxSize: 1000}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Chunk(ctx, content, opts)
	}
}

func BenchmarkCodeChunker_Go(b *testing.B) {
	chunker := NewCodeChunker(2000, 100)
	content := `package main

import "fmt"

func hello() {
	fmt.Println("Hello")
}

func world() {
	fmt.Println("World")
}

func main() {
	hello()
	world()
}
`
	opts := ChunkOptions{
		Language: "go",
		MaxSize:  2000,
		Semantic: true,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Chunk(ctx, content, opts)
	}
}
