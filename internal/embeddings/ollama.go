// Package embeddings provides embedding generation via Ollama
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/shivavenkatesh/moneta/internal/cache"
)

// OllamaClient handles communication with Ollama for embeddings
type OllamaClient struct {
	baseURL    string
	model      string
	dims       int
	httpClient *http.Client
	cache      *cache.EmbeddingCache

	// Stats
	requests atomic.Int64
	latency  atomic.Int64 // cumulative latency in microseconds
}

// ollamaRequest is the request payload for Ollama embed API
type ollamaRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// ollamaResponse is the response from Ollama embed API
type ollamaResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// OllamaConfig configures the Ollama client
type OllamaConfig struct {
	BaseURL    string
	Model      string
	Dimensions int
	CacheSize  int
	Timeout    time.Duration
}

// DefaultOllamaConfig returns sensible defaults
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		BaseURL:    getEnvOrDefault("OLLAMA_HOST", "http://localhost:11434"),
		Model:      getEnvOrDefault("EMBEDDING_MODEL", "nomic-embed-text"),
		Dimensions: 768, // nomic-embed-text dimensions
		CacheSize:  1000,
		Timeout:    30 * time.Second,
	}
}

// NewOllamaClient creates a new Ollama embeddings client
func NewOllamaClient(cfg OllamaConfig) *OllamaClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultOllamaConfig().BaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultOllamaConfig().Model
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 1000
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &OllamaClient{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		dims:    cfg.Dimensions,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		cache: cache.NewEmbeddingCache(cfg.CacheSize),
	}
}

// Embed generates an embedding for the given text
func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// Check cache first
	if embedding, ok := c.cache.Get(text); ok {
		return embedding, nil
	}

	start := time.Now()

	reqBody := ollamaRequest{
		Model: c.model,
		Input: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	// Use streaming parser for better performance
	embedding, err := c.parseEmbeddingStream(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Update stats
	c.requests.Add(1)
	c.latency.Add(time.Since(start).Microseconds())

	// Cache the result
	c.cache.Put(text, embedding)

	return embedding, nil
}

// parseEmbeddingStream extracts embeddings without full JSON parse
// This is faster than json.Unmarshal for large embedding arrays
func (c *OllamaClient) parseEmbeddingStream(r io.Reader) ([]float32, error) {
	dec := json.NewDecoder(r)

	// Find the embeddings array
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if key, ok := t.(string); ok && key == "embeddings" {
			// Read opening bracket of outer array
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			// Read opening bracket of inner array
			if _, err := dec.Token(); err != nil {
				return nil, err
			}

			// Pre-allocate with expected dimensions
			embedding := make([]float32, 0, c.dims)

			// Read floats until closing bracket
			for dec.More() {
				var f float64
				if err := dec.Decode(&f); err != nil {
					return nil, err
				}
				embedding = append(embedding, float32(f))
			}
			return embedding, nil
		}
	}

	return nil, fmt.Errorf("no embeddings found in response")
}

// EmbedBatch generates embeddings for multiple texts
// Uses concurrent requests for better throughput
func (c *OllamaClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	// For now, process sequentially (Ollama doesn't batch well)
	// TODO: Add concurrent processing with semaphore
	for i, text := range texts {
		emb, err := c.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// Dimensions returns the embedding vector dimensions
func (c *OllamaClient) Dimensions() int {
	return c.dims
}

// Model returns the current embedding model name
func (c *OllamaClient) Model() string {
	return c.model
}

// Close releases resources
func (c *OllamaClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// Ping checks if Ollama is reachable and the model is available
func (c *OllamaClient) Ping(ctx context.Context) error {
	// Try to generate a tiny embedding to verify everything works
	_, err := c.Embed(ctx, "test")
	if err != nil {
		return fmt.Errorf("Ollama health check failed: %w", err)
	}
	return nil
}

// Stats returns client statistics
func (c *OllamaClient) Stats() (requests int64, avgLatencyMs float64, cacheHitRate float64) {
	requests = c.requests.Load()
	if requests > 0 {
		avgLatencyMs = float64(c.latency.Load()) / float64(requests) / 1000
	}
	_, _, cacheHitRate = c.cache.Stats()
	return
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
