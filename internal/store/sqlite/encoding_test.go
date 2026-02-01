package sqlite

import (
	"math"
	"testing"

	"github.com/shivavenkatesh/moneta/pkg/types"
)

func TestFloat32ToBytes(t *testing.T) {
	input := []float32{1.0, 2.0, 3.0, 4.0}
	bytes := float32ToBytes(input)

	if len(bytes) != len(input)*4 {
		t.Errorf("expected %d bytes, got %d", len(input)*4, len(bytes))
	}

	// Convert back
	result := bytesToFloat32(bytes)
	for i, v := range input {
		if result[i] != v {
			t.Errorf("at index %d: expected %f, got %f", i, v, result[i])
		}
	}
}

func TestFloat32ToBytes_Empty(t *testing.T) {
	bytes := float32ToBytes(nil)
	if bytes != nil {
		t.Error("expected nil for empty input")
	}

	bytes = float32ToBytes([]float32{})
	if bytes != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestBytesToFloat32_Empty(t *testing.T) {
	result := bytesToFloat32(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}

	result = bytesToFloat32([]byte{})
	if result != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestBytesToFloat32_InvalidLength(t *testing.T) {
	// Not divisible by 4
	result := bytesToFloat32([]byte{1, 2, 3})
	if result != nil {
		t.Error("expected nil for invalid byte length")
	}
}

func TestFloat32ToBytesAlloc(t *testing.T) {
	input := []float32{1.0, 2.0, 3.0, 4.0}
	bytes := float32ToBytesAlloc(input)

	if len(bytes) != len(input)*4 {
		t.Errorf("expected %d bytes, got %d", len(input)*4, len(bytes))
	}

	// Modify original - shouldn't affect bytes (it's a copy)
	original := input[0]
	input[0] = 999.0

	// Convert back and verify original value
	result := bytesToFloat32Alloc(bytes)
	if result[0] != original {
		t.Error("bytesToFloat32Alloc should be independent of original")
	}
}

func TestBytesToFloat32Alloc(t *testing.T) {
	input := []float32{1.5, 2.5, 3.5}
	bytes := float32ToBytesAlloc(input)
	result := bytesToFloat32Alloc(bytes)

	for i, v := range input {
		if result[i] != v {
			t.Errorf("at index %d: expected %f, got %f", i, v, result[i])
		}
	}
}

func TestBytesToFloat32Alloc_Invalid(t *testing.T) {
	// Nil
	result := bytesToFloat32Alloc(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}

	// Invalid length
	result = bytesToFloat32Alloc([]byte{1, 2, 3})
	if result != nil {
		t.Error("expected nil for invalid length")
	}
}

func TestSortBySimilarity(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.5},
		{Memory: types.Memory{ID: "2"}, Similarity: 0.9},
		{Memory: types.Memory{ID: "3"}, Similarity: 0.3},
		{Memory: types.Memory{ID: "4"}, Similarity: 0.7},
	}

	sortBySimilarity(results)

	// Should be sorted descending
	expected := []float32{0.9, 0.7, 0.5, 0.3}
	for i, exp := range expected {
		if results[i].Similarity != exp {
			t.Errorf("at index %d: expected %f, got %f", i, exp, results[i].Similarity)
		}
	}
}

func TestSortBySimilarity_Empty(t *testing.T) {
	var results []types.SearchResult
	sortBySimilarity(results) // Should not panic
}

func TestSortBySimilarity_Single(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.5},
	}
	sortBySimilarity(results)
	if results[0].Similarity != 0.5 {
		t.Error("single element should remain unchanged")
	}
}

func TestInsertionSortResults(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.2},
		{Memory: types.Memory{ID: "2"}, Similarity: 0.8},
		{Memory: types.Memory{ID: "3"}, Similarity: 0.5},
	}

	insertionSortResults(results)

	if results[0].Similarity != 0.8 || results[1].Similarity != 0.5 || results[2].Similarity != 0.2 {
		t.Error("insertion sort did not sort correctly")
	}
}

func TestTopKResults(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.2},
		{Memory: types.Memory{ID: "2"}, Similarity: 0.8},
		{Memory: types.Memory{ID: "3"}, Similarity: 0.5},
		{Memory: types.Memory{ID: "4"}, Similarity: 0.9},
		{Memory: types.Memory{ID: "5"}, Similarity: 0.3},
	}

	top := topKResults(results, 3)

	if len(top) != 3 {
		t.Errorf("expected 3 results, got %d", len(top))
	}

	// Should have top 3 similarities
	if top[0].Similarity != 0.9 {
		t.Errorf("expected top similarity 0.9, got %f", top[0].Similarity)
	}
}

func TestTopKResults_KGreaterThanN(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.5},
		{Memory: types.Memory{ID: "2"}, Similarity: 0.3},
	}

	top := topKResults(results, 10)

	if len(top) != 2 {
		t.Errorf("expected 2 results, got %d", len(top))
	}
}

func TestSelectTopK(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.2},
		{Memory: types.Memory{ID: "2"}, Similarity: 0.8},
		{Memory: types.Memory{ID: "3"}, Similarity: 0.5},
		{Memory: types.Memory{ID: "4"}, Similarity: 0.9},
	}

	top := selectTopK(results, 2)

	if len(top) != 2 {
		t.Errorf("expected 2 results, got %d", len(top))
	}
	if top[0].Similarity != 0.9 || top[1].Similarity != 0.8 {
		t.Error("selectTopK did not return correct top 2")
	}
}

func TestHeapTopK(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.2},
		{Memory: types.Memory{ID: "2"}, Similarity: 0.8},
		{Memory: types.Memory{ID: "3"}, Similarity: 0.5},
		{Memory: types.Memory{ID: "4"}, Similarity: 0.9},
		{Memory: types.Memory{ID: "5"}, Similarity: 0.3},
		{Memory: types.Memory{ID: "6"}, Similarity: 0.7},
		{Memory: types.Memory{ID: "7"}, Similarity: 0.6},
	}

	top := heapTopK(results, 3)

	if len(top) != 3 {
		t.Errorf("expected 3 results, got %d", len(top))
	}

	// Verify we got the top 3
	expectedSims := []float32{0.9, 0.8, 0.7}
	for i, exp := range expectedSims {
		if top[i].Similarity != exp {
			t.Errorf("at index %d: expected %f, got %f", i, exp, top[i].Similarity)
		}
	}
}

func TestHeapTopK_Zero(t *testing.T) {
	results := []types.SearchResult{
		{Memory: types.Memory{ID: "1"}, Similarity: 0.5},
	}

	top := heapTopK(results, 0)
	if top != nil {
		t.Error("expected nil for k=0")
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	a := []float32{1, 2, 3, 4, 5, 6, 7, 8}
	b := []float32{1, 2, 3, 4, 5, 6, 7, 8}

	sim := cosineSimilarity(a, b)
	if math.Abs(float64(sim-1.0)) > 0.01 {
		t.Errorf("expected ~1.0 for identical vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1, 2, 3, 4}
	b := []float32{-1, -2, -3, -4}

	sim := cosineSimilarity(a, b)
	if math.Abs(float64(sim+1.0)) > 0.01 {
		t.Errorf("expected ~-1.0 for opposite vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}

	sim := cosineSimilarity(a, b)
	if math.Abs(float64(sim)) > 0.01 {
		t.Errorf("expected ~0 for orthogonal vectors, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1, 2}

	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 for different length vectors, got %f", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}

	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 when one vector is zero, got %f", sim)
	}
}

func TestSqrt32(t *testing.T) {
	tests := []struct {
		input    float32
		expected float32
	}{
		{4.0, 2.0},
		{9.0, 3.0},
		{16.0, 4.0},
		{25.0, 5.0},
		{100.0, 10.0},
	}

	for _, tt := range tests {
		result := sqrt32(tt.input)
		if math.Abs(float64(result-tt.expected)) > 0.01 {
			t.Errorf("sqrt32(%f) = %f, want %f", tt.input, result, tt.expected)
		}
	}
}

func TestSqrt32_Zero(t *testing.T) {
	if sqrt32(0) != 0 {
		t.Error("sqrt32(0) should be 0")
	}
}

func TestSqrt32_Negative(t *testing.T) {
	if sqrt32(-1) != 0 {
		t.Error("sqrt32(-1) should be 0")
	}
}

// Benchmarks

func BenchmarkFloat32ToBytes(b *testing.B) {
	input := make([]float32, 768)
	for i := range input {
		input[i] = float32(i) * 0.1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		float32ToBytes(input)
	}
}

func BenchmarkFloat32ToBytesAlloc(b *testing.B) {
	input := make([]float32, 768)
	for i := range input {
		input[i] = float32(i) * 0.1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		float32ToBytesAlloc(input)
	}
}

func BenchmarkBytesToFloat32(b *testing.B) {
	input := make([]float32, 768)
	for i := range input {
		input[i] = float32(i) * 0.1
	}
	bytes := float32ToBytesAlloc(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesToFloat32(bytes)
	}
}

func BenchmarkSortBySimilarity_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		results := make([]types.SearchResult, 10)
		for j := range results {
			results[j].Similarity = float32(j) * 0.1
		}
		sortBySimilarity(results)
	}
}

func BenchmarkSortBySimilarity_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		results := make([]types.SearchResult, 1000)
		for j := range results {
			results[j].Similarity = float32(j) * 0.001
		}
		sortBySimilarity(results)
	}
}

func BenchmarkTopKResults(b *testing.B) {
	results := make([]types.SearchResult, 1000)
	for j := range results {
		results[j].Similarity = float32(j) * 0.001
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultsCopy := make([]types.SearchResult, len(results))
		copy(resultsCopy, results)
		topKResults(resultsCopy, 10)
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	a := make([]float32, 768)
	vec := make([]float32, 768)
	for i := range a {
		a[i] = float32(i) * 0.1
		vec[i] = float32(i) * 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cosineSimilarity(a, vec)
	}
}
