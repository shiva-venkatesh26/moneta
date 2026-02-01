// Package sqlite provides encoding utilities for SQLite storage
package sqlite

import (
	"sort"
	"unsafe"

	"github.com/shivavenkatesh/moneta/pkg/types"
)

// float32ToBytes converts a float32 slice to bytes using zero-copy
// WARNING: The returned slice shares memory with the input - do not modify input after calling
// For safe usage across goroutines, use float32ToBytesAlloc instead
func float32ToBytes(f []float32) []byte {
	if len(f) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&f[0])), len(f)*4)
}

// bytesToFloat32 converts bytes to float32 slice using zero-copy
// WARNING: The returned slice shares memory with the input - do not modify input after calling
// For safe usage across goroutines, use bytesToFloat32Alloc instead
func bytesToFloat32(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}
	if len(b)%4 != 0 {
		return nil // Invalid byte length for float32
	}
	return unsafe.Slice((*float32)(unsafe.Pointer(&b[0])), len(b)/4)
}

// float32ToBytesAlloc creates a safe copy suitable for concurrent use
func float32ToBytesAlloc(f []float32) []byte {
	if len(f) == 0 {
		return nil
	}
	b := make([]byte, len(f)*4)
	for i, v := range f {
		bits := *(*uint32)(unsafe.Pointer(&v))
		b[i*4] = byte(bits)
		b[i*4+1] = byte(bits >> 8)
		b[i*4+2] = byte(bits >> 16)
		b[i*4+3] = byte(bits >> 24)
	}
	return b
}

// bytesToFloat32Alloc creates a safe copy suitable for concurrent use
func bytesToFloat32Alloc(b []byte) []float32 {
	if len(b) == 0 || len(b)%4 != 0 {
		return nil
	}
	f := make([]float32, len(b)/4)
	for i := range f {
		bits := uint32(b[i*4]) |
			uint32(b[i*4+1])<<8 |
			uint32(b[i*4+2])<<16 |
			uint32(b[i*4+3])<<24
		f[i] = *(*float32)(unsafe.Pointer(&bits))
	}
	return f
}

// sortBySimilarity sorts search results by similarity in descending order
// Uses optimized sorting for typical result sizes
func sortBySimilarity(results []types.SearchResult) {
	n := len(results)
	if n <= 1 {
		return
	}

	// For small slices, insertion sort is faster due to lower overhead
	if n <= 16 {
		insertionSortResults(results)
		return
	}

	// For larger slices, use Go's optimized sort (pdqsort)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
}

// insertionSortResults performs insertion sort for small slices
// Faster than quicksort for n <= 16 due to lower overhead
func insertionSortResults(results []types.SearchResult) {
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1

		// Move elements that are smaller than key to one position ahead
		for j >= 0 && results[j].Similarity < key.Similarity {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}

// topKResults extracts top K results efficiently using partial sort
// This is faster than full sort when K << N
func topKResults(results []types.SearchResult, k int) []types.SearchResult {
	n := len(results)
	if k >= n {
		sortBySimilarity(results)
		return results
	}

	// Use selection algorithm for top-k
	// For very small k, simple selection is faster
	if k <= 5 {
		return selectTopK(results, k)
	}

	// For moderate k, partial heap sort
	return heapTopK(results, k)
}

// selectTopK uses simple selection for very small k
func selectTopK(results []types.SearchResult, k int) []types.SearchResult {
	top := make([]types.SearchResult, 0, k)

	for i := 0; i < k; i++ {
		maxIdx := i
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[maxIdx].Similarity {
				maxIdx = j
			}
		}
		results[i], results[maxIdx] = results[maxIdx], results[i]
		top = append(top, results[i])
	}

	return top
}

// heapTopK uses a min-heap to efficiently find top-k elements
func heapTopK(results []types.SearchResult, k int) []types.SearchResult {
	if k <= 0 {
		return nil
	}

	// Build min-heap of size k
	heap := make([]types.SearchResult, 0, k)

	for _, r := range results {
		if len(heap) < k {
			heap = append(heap, r)
			heapifyUp(heap, len(heap)-1)
		} else if r.Similarity > heap[0].Similarity {
			heap[0] = r
			heapifyDown(heap, 0)
		}
	}

	// Sort the heap for ordered results
	sortBySimilarity(heap)
	return heap
}

func heapifyUp(heap []types.SearchResult, i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if heap[parent].Similarity <= heap[i].Similarity {
			break
		}
		heap[parent], heap[i] = heap[i], heap[parent]
		i = parent
	}
}

func heapifyDown(heap []types.SearchResult, i int) {
	n := len(heap)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && heap[left].Similarity < heap[smallest].Similarity {
			smallest = left
		}
		if right < n && heap[right].Similarity < heap[smallest].Similarity {
			smallest = right
		}

		if smallest == i {
			break
		}

		heap[i], heap[smallest] = heap[smallest], heap[i]
		i = smallest
	}
}

// cosineSimilarity computes cosine similarity between two vectors
// This is a fallback for when the simd package is not used
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float32
	n := len(a)

	// Process 8 elements at a time for better performance
	limit := n - (n % 8)

	for i := 0; i < limit; i += 8 {
		dotProduct += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]

		normA += a[i]*a[i] + a[i+1]*a[i+1] + a[i+2]*a[i+2] + a[i+3]*a[i+3] +
			a[i+4]*a[i+4] + a[i+5]*a[i+5] + a[i+6]*a[i+6] + a[i+7]*a[i+7]

		normB += b[i]*b[i] + b[i+1]*b[i+1] + b[i+2]*b[i+2] + b[i+3]*b[i+3] +
			b[i+4]*b[i+4] + b[i+5]*b[i+5] + b[i+6]*b[i+6] + b[i+7]*b[i+7]
	}

	for i := limit; i < n; i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 is a fast float32 square root
func sqrt32(x float32) float32 {
	if x <= 0 {
		return 0
	}
	i := *(*uint32)(unsafe.Pointer(&x))
	i = 0x5f3759df - (i >> 1)
	y := *(*float32)(unsafe.Pointer(&i))
	y = y * (1.5 - (0.5 * x * y * y))
	y = y * (1.5 - (0.5 * x * y * y))
	return x * y
}
