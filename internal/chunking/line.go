// Package chunking provides text and code chunking implementations
package chunking

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivavenkatesh/moneta/pkg/types"
)

// LineChunker implements line-based chunking with overlap
type LineChunker struct {
	maxSize int
	overlap int
}

// NewLineChunker creates a new line-based chunker
func NewLineChunker(maxSize, overlap int) *LineChunker {
	if maxSize <= 0 {
		maxSize = 1500
	}
	if overlap < 0 {
		overlap = 100
	}
	return &LineChunker{
		maxSize: maxSize,
		overlap: overlap,
	}
}

// Chunk splits content into chunks based on lines
func (c *LineChunker) Chunk(ctx context.Context, content string, opts ChunkOptions) ([]types.Chunk, error) {
	if content == "" {
		return nil, nil
	}

	maxSize := opts.MaxSize
	if maxSize <= 0 {
		maxSize = c.maxSize
	}

	overlap := opts.Overlap
	if overlap < 0 {
		overlap = c.overlap
	}

	lines := strings.Split(content, "\n")
	var chunks []types.Chunk

	var currentChunk strings.Builder
	startLine := 1
	currentLine := 1

	for _, line := range lines {
		// Check if adding this line would exceed max size
		if currentChunk.Len()+len(line)+1 > maxSize && currentChunk.Len() > 0 {
			// Save current chunk
			chunks = append(chunks, types.Chunk{
				Content:   strings.TrimSpace(currentChunk.String()),
				StartLine: startLine,
				EndLine:   currentLine - 1,
				Type:      "text",
			})

			// Start new chunk with overlap
			currentChunk.Reset()
			overlapStart := findOverlapStart(chunks[len(chunks)-1].Content, overlap)
			if overlapStart != "" {
				currentChunk.WriteString(overlapStart)
				currentChunk.WriteString("\n")
			}
			startLine = currentLine
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
		currentLine++
	}

	// Add final chunk if not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, types.Chunk{
			Content:   strings.TrimSpace(currentChunk.String()),
			StartLine: startLine,
			EndLine:   currentLine - 1,
			Type:      "text",
		})
	}

	return chunks, nil
}

// ChunkFile reads and chunks a file
func (c *LineChunker) ChunkFile(ctx context.Context, path string) ([]types.Chunk, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	language := detectLanguage(ext)

	opts := ChunkOptions{
		Language: language,
		MaxSize:  c.maxSize,
		Overlap:  c.overlap,
	}

	chunks, err := c.Chunk(ctx, string(content), opts)
	if err != nil {
		return nil, err
	}

	// Set language on all chunks
	for i := range chunks {
		chunks[i].Type = language
	}

	return chunks, nil
}

// SupportedLanguages returns list of supported programming languages
func (c *LineChunker) SupportedLanguages() []string {
	return []string{"text", "go", "python", "javascript", "typescript", "rust", "java", "c", "cpp"}
}

// findOverlapStart returns the last N characters of content for overlap
func findOverlapStart(content string, overlap int) string {
	if len(content) <= overlap {
		return content
	}

	// Find a good break point (end of line) within overlap range
	lastPart := content[len(content)-overlap:]
	if idx := strings.Index(lastPart, "\n"); idx != -1 {
		return lastPart[idx+1:]
	}
	return lastPart
}

// detectLanguage maps file extensions to language names
func detectLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".cs":
		return "csharp"
	case ".md", ".markdown":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "shell"
	default:
		return "text"
	}
}

// CodeChunker implements code-aware chunking that respects function boundaries
type CodeChunker struct {
	lineChunker *LineChunker
}

// NewCodeChunker creates a code-aware chunker
func NewCodeChunker(maxSize, overlap int) *CodeChunker {
	return &CodeChunker{
		lineChunker: NewLineChunker(maxSize, overlap),
	}
}

// Chunk splits code content respecting semantic boundaries
func (c *CodeChunker) Chunk(ctx context.Context, content string, opts ChunkOptions) ([]types.Chunk, error) {
	if !opts.Semantic {
		return c.lineChunker.Chunk(ctx, content, opts)
	}

	// For semantic chunking, detect function/class boundaries
	// This is a simplified version - tree-sitter would be more accurate
	switch opts.Language {
	case "go":
		return c.chunkGo(ctx, content, opts)
	case "python":
		return c.chunkPython(ctx, content, opts)
	case "javascript", "typescript":
		return c.chunkJS(ctx, content, opts)
	default:
		return c.lineChunker.Chunk(ctx, content, opts)
	}
}

// chunkGo chunks Go code by function boundaries
func (c *CodeChunker) chunkGo(ctx context.Context, content string, opts ChunkOptions) ([]types.Chunk, error) {
	var chunks []types.Chunk
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentChunk strings.Builder
	var currentName string
	startLine := 1
	lineNum := 0
	braceDepth := 0
	inFunc := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Detect function start
		if strings.HasPrefix(strings.TrimSpace(line), "func ") {
			// Save previous chunk if exists
			if currentChunk.Len() > 0 {
				chunks = append(chunks, types.Chunk{
					Content:   strings.TrimSpace(currentChunk.String()),
					StartLine: startLine,
					EndLine:   lineNum - 1,
					Type:      "function",
					Name:      currentName,
				})
				currentChunk.Reset()
			}
			startLine = lineNum
			inFunc = true

			// Extract function name
			parts := strings.Fields(strings.TrimSpace(line))
			if len(parts) >= 2 {
				name := parts[1]
				if idx := strings.Index(name, "("); idx != -1 {
					name = name[:idx]
				}
				currentName = name
			}
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")

		// Track brace depth
		braceDepth += strings.Count(line, "{") - strings.Count(line, "}")

		// End of function
		if inFunc && braceDepth == 0 && strings.Contains(line, "}") {
			chunks = append(chunks, types.Chunk{
				Content:   strings.TrimSpace(currentChunk.String()),
				StartLine: startLine,
				EndLine:   lineNum,
				Type:      "function",
				Name:      currentName,
			})
			currentChunk.Reset()
			currentName = ""
			startLine = lineNum + 1
			inFunc = false
		}

		// Check max size
		if currentChunk.Len() > opts.MaxSize && !inFunc {
			chunks = append(chunks, types.Chunk{
				Content:   strings.TrimSpace(currentChunk.String()),
				StartLine: startLine,
				EndLine:   lineNum,
				Type:      "text",
			})
			currentChunk.Reset()
			startLine = lineNum + 1
		}
	}

	// Add remaining content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, types.Chunk{
			Content:   strings.TrimSpace(currentChunk.String()),
			StartLine: startLine,
			EndLine:   lineNum,
			Type:      "text",
			Name:      currentName,
		})
	}

	return chunks, nil
}

// chunkPython chunks Python code by function/class boundaries
func (c *CodeChunker) chunkPython(ctx context.Context, content string, opts ChunkOptions) ([]types.Chunk, error) {
	var chunks []types.Chunk
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentChunk strings.Builder
	var currentName string
	var currentType string
	startLine := 1
	lineNum := 0
	baseIndent := -1

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Calculate indent level
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Detect function/class start at base level
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") {
			// Save previous chunk if at same or lower indent
			if currentChunk.Len() > 0 && (baseIndent == -1 || indent <= baseIndent) {
				chunks = append(chunks, types.Chunk{
					Content:   strings.TrimSpace(currentChunk.String()),
					StartLine: startLine,
					EndLine:   lineNum - 1,
					Type:      currentType,
					Name:      currentName,
				})
				currentChunk.Reset()
				startLine = lineNum
			}

			baseIndent = indent
			if strings.HasPrefix(trimmed, "def ") {
				currentType = "function"
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 {
					name := parts[1]
					if idx := strings.Index(name, "("); idx != -1 {
						name = name[:idx]
					}
					currentName = name
				}
			} else {
				currentType = "class"
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 {
					name := parts[1]
					if idx := strings.Index(name, "("); idx != -1 {
						name = name[:idx]
					}
					if idx := strings.Index(name, ":"); idx != -1 {
						name = name[:idx]
					}
					currentName = name
				}
			}
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")

		// Check max size
		if currentChunk.Len() > opts.MaxSize {
			chunks = append(chunks, types.Chunk{
				Content:   strings.TrimSpace(currentChunk.String()),
				StartLine: startLine,
				EndLine:   lineNum,
				Type:      currentType,
				Name:      currentName,
			})
			currentChunk.Reset()
			startLine = lineNum + 1
			currentName = ""
			currentType = "text"
			baseIndent = -1
		}
	}

	// Add remaining content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, types.Chunk{
			Content:   strings.TrimSpace(currentChunk.String()),
			StartLine: startLine,
			EndLine:   lineNum,
			Type:      currentType,
			Name:      currentName,
		})
	}

	return chunks, nil
}

// chunkJS chunks JavaScript/TypeScript code
func (c *CodeChunker) chunkJS(ctx context.Context, content string, opts ChunkOptions) ([]types.Chunk, error) {
	// For now, use the Go chunker logic as JS is similar with braces
	// TODO: Handle arrow functions, class methods, etc.
	return c.chunkGo(ctx, content, opts)
}

// ChunkFile reads and chunks a file with code awareness
func (c *CodeChunker) ChunkFile(ctx context.Context, path string) ([]types.Chunk, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	language := detectLanguage(ext)

	opts := ChunkOptions{
		Language: language,
		MaxSize:  c.lineChunker.maxSize,
		Overlap:  c.lineChunker.overlap,
		Semantic: true,
	}

	return c.Chunk(ctx, string(content), opts)
}

// SupportedLanguages returns list of supported programming languages
func (c *CodeChunker) SupportedLanguages() []string {
	return []string{"go", "python", "javascript", "typescript"}
}
