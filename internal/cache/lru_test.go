package cache

import (
	"testing"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRU[string, int](3)

	// Test Put and Get
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Errorf("expected 1, got %v", v)
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Errorf("expected 2, got %v", v)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Errorf("expected 3, got %v", v)
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRU[string, int](2)

	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3) // should evict "a"

	if _, ok := cache.Get("a"); ok {
		t.Error("expected 'a' to be evicted")
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Errorf("expected 2, got %v", v)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Errorf("expected 3, got %v", v)
	}
}

func TestLRUCache_AccessOrder(t *testing.T) {
	cache := NewLRU[string, int](2)

	cache.Put("a", 1)
	cache.Put("b", 2)

	// Access "a" to make it recently used
	cache.Get("a")

	// Add "c", should evict "b" not "a"
	cache.Put("c", 3)

	if _, ok := cache.Get("b"); ok {
		t.Error("expected 'b' to be evicted")
	}
	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Errorf("expected 1, got %v", v)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Errorf("expected 3, got %v", v)
	}
}

func TestLRUCache_Update(t *testing.T) {
	cache := NewLRU[string, int](2)

	cache.Put("a", 1)
	cache.Put("a", 10) // update

	if v, ok := cache.Get("a"); !ok || v != 10 {
		t.Errorf("expected 10, got %v", v)
	}

	if cache.Len() != 1 {
		t.Errorf("expected len 1, got %d", cache.Len())
	}
}

func TestLRUCache_Len(t *testing.T) {
	cache := NewLRU[string, int](5)

	if cache.Len() != 0 {
		t.Errorf("expected 0, got %d", cache.Len())
	}

	cache.Put("a", 1)
	cache.Put("b", 2)

	if cache.Len() != 2 {
		t.Errorf("expected 2, got %d", cache.Len())
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRU[string, int](5)

	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("expected 0 after clear, got %d", cache.Len())
	}

	if _, ok := cache.Get("a"); ok {
		t.Error("expected 'a' to be cleared")
	}
}

func TestEmbeddingCache_BasicOperations(t *testing.T) {
	cache := NewEmbeddingCache(100)

	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	cache.Put("hello world", embedding)

	if got, ok := cache.Get("hello world"); !ok {
		t.Error("expected to find embedding")
	} else {
		if len(got) != len(embedding) {
			t.Errorf("expected len %d, got %d", len(embedding), len(got))
		}
		for i := range embedding {
			if got[i] != embedding[i] {
				t.Errorf("expected %f at index %d, got %f", embedding[i], i, got[i])
			}
		}
	}
}

func TestEmbeddingCache_Stats(t *testing.T) {
	cache := NewEmbeddingCache(100)

	embedding := []float32{0.1, 0.2, 0.3}
	cache.Put("test", embedding)

	// First access - hit
	cache.Get("test")
	// Second access - hit
	cache.Get("test")
	// Miss
	cache.Get("nonexistent")

	hits, misses, hitRate := cache.Stats()
	if hits != 2 {
		t.Errorf("expected 2 hits, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("expected 1 miss, got %d", misses)
	}
	expectedRate := 2.0 / 3.0 * 100
	if hitRate < expectedRate-1 || hitRate > expectedRate+1 {
		t.Errorf("expected hit rate ~%.1f%%, got %.1f%%", expectedRate, hitRate)
	}
}

func BenchmarkLRUCache_Put(b *testing.B) {
	cache := NewLRU[int, int](1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(i%1000, i)
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	cache := NewLRU[int, int](1000)
	for i := 0; i < 1000; i++ {
		cache.Put(i, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(i % 1000)
	}
}

func BenchmarkEmbeddingCache_Put(b *testing.B) {
	cache := NewEmbeddingCache(1000)
	embedding := make([]float32, 768)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put("test", embedding)
	}
}
