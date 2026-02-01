// Package cache provides caching utilities for Moneta
package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
)

// LRU implements a thread-safe LRU cache with generics
type LRU[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	items    map[K]*list.Element
	order    *list.List

	// Stats
	hits   atomic.Int64
	misses atomic.Int64
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// NewLRU creates a new LRU cache with the specified capacity
func NewLRU[K comparable, V any](capacity int) *LRU[K, V] {
	return &LRU[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves a value from the cache, returning (value, true) if found
func (c *LRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		c.misses.Add(1)
		var zero V
		return zero, false
	}

	c.hits.Add(1)
	c.order.MoveToFront(elem)
	return elem.Value.(*entry[K, V]).value, true
}

// Put adds or updates a value in the cache
func (c *LRU[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*entry[K, V]).value = value
		return
	}

	// Evict oldest if at capacity
	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			delete(c.items, oldest.Value.(*entry[K, V]).key)
			c.order.Remove(oldest)
		}
	}

	// Add new entry
	elem := c.order.PushFront(&entry[K, V]{key: key, value: value})
	c.items[key] = elem
}

// Delete removes a key from the cache
func (c *LRU[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		delete(c.items, key)
		c.order.Remove(elem)
	}
}

// Len returns the current number of items in the cache
func (c *LRU[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache
func (c *LRU[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*list.Element)
	c.order = list.New()
}

// Stats returns cache hit/miss statistics
func (c *LRU[K, V]) Stats() (hits, misses int64) {
	return c.hits.Load(), c.misses.Load()
}

// HitRate returns the cache hit rate as a percentage
func (c *LRU[K, V]) HitRate() float64 {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// EmbeddingCache is a specialized cache for text embeddings
type EmbeddingCache struct {
	cache *LRU[string, []float32]
}

// NewEmbeddingCache creates a cache for embeddings with content hashing
func NewEmbeddingCache(capacity int) *EmbeddingCache {
	return &EmbeddingCache{
		cache: NewLRU[string, []float32](capacity),
	}
}

// Get retrieves an embedding by content hash
func (c *EmbeddingCache) Get(content string) ([]float32, bool) {
	key := hashContent(content)
	return c.cache.Get(key)
}

// Put stores an embedding by content hash
func (c *EmbeddingCache) Put(content string, embedding []float32) {
	key := hashContent(content)
	// Store a copy to prevent external modification
	embCopy := make([]float32, len(embedding))
	copy(embCopy, embedding)
	c.cache.Put(key, embCopy)
}

// Stats returns cache statistics
func (c *EmbeddingCache) Stats() (hits, misses int64, hitRate float64) {
	hits, misses = c.cache.Stats()
	hitRate = c.cache.HitRate()
	return
}

// hashContent creates a hash of the content for cache keys
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:16]) // Use first 16 bytes (128 bits)
}
