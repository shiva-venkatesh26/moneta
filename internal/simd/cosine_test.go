package simd

import (
	"math"
	"testing"
)

const epsilon = 1e-5

func almostEqual(a, b, eps float32) bool {
	return float32(math.Abs(float64(a-b))) < eps
}

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	a := []float32{1, 2, 3, 4, 5, 6, 7, 8}
	b := []float32{1, 2, 3, 4, 5, 6, 7, 8}

	sim := CosineSimilarity(a, b)
	if !almostEqual(sim, 1.0, epsilon) {
		t.Errorf("expected 1.0 for identical vectors, got %f", sim)
	}
}

func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	a := []float32{1, 2, 3, 4}
	b := []float32{-1, -2, -3, -4}

	sim := CosineSimilarity(a, b)
	if !almostEqual(sim, -1.0, epsilon) {
		t.Errorf("expected -1.0 for opposite vectors, got %f", sim)
	}
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}

	sim := CosineSimilarity(a, b)
	if !almostEqual(sim, 0.0, epsilon) {
		t.Errorf("expected 0.0 for orthogonal vectors, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1, 2}

	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 for different length vectors, got %f", sim)
	}
}

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	a := []float32{}
	b := []float32{}

	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 for empty vectors, got %f", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}

	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("expected 0 when one vector is zero, got %f", sim)
	}
}

func TestCosineSimilarity_NonAligned(t *testing.T) {
	// Test with non-8-aligned length to test remainder handling
	a := []float32{1, 2, 3, 4, 5}
	b := []float32{1, 2, 3, 4, 5}

	sim := CosineSimilarity(a, b)
	if !almostEqual(sim, 1.0, epsilon) {
		t.Errorf("expected 1.0, got %f", sim)
	}
}

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{
			name:     "simple",
			a:        []float32{1, 2, 3},
			b:        []float32{4, 5, 6},
			expected: 32, // 1*4 + 2*5 + 3*6 = 4 + 10 + 18 = 32
		},
		{
			name:     "zeros",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 2, 3},
			expected: 0,
		},
		{
			name:     "eight elements",
			a:        []float32{1, 1, 1, 1, 1, 1, 1, 1},
			b:        []float32{2, 2, 2, 2, 2, 2, 2, 2},
			expected: 16,
		},
		{
			name:     "different lengths",
			a:        []float32{1, 2},
			b:        []float32{1},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DotProduct(tt.a, tt.b)
			if !almostEqual(result, tt.expected, epsilon) {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestL2Norm(t *testing.T) {
	tests := []struct {
		name     string
		v        []float32
		expected float32
	}{
		{
			name:     "unit vector",
			v:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "3-4-5 triangle",
			v:        []float32{3, 4},
			expected: 5.0,
		},
		{
			name:     "zero vector",
			v:        []float32{0, 0, 0},
			expected: 0.0,
		},
		{
			name:     "all ones (8 elements)",
			v:        []float32{1, 1, 1, 1, 1, 1, 1, 1},
			expected: float32(math.Sqrt(8)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := L2Norm(tt.v)
			if !almostEqual(result, tt.expected, 0.01) {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	v := []float32{3, 4, 0}
	Normalize(v)

	expected := []float32{0.6, 0.8, 0}
	for i := range v {
		if !almostEqual(v[i], expected[i], epsilon) {
			t.Errorf("at index %d: expected %f, got %f", i, expected[i], v[i])
		}
	}

	// After normalization, L2 norm should be 1
	norm := L2Norm(v)
	if !almostEqual(norm, 1.0, epsilon) {
		t.Errorf("normalized vector should have norm 1.0, got %f", norm)
	}
}

func TestNormalize_ZeroVector(t *testing.T) {
	v := []float32{0, 0, 0}
	Normalize(v)

	// Should remain zeros (no division by zero)
	for i, val := range v {
		if val != 0 {
			t.Errorf("at index %d: expected 0, got %f", i, val)
		}
	}
}

func TestBatchCosineSimilarity(t *testing.T) {
	query := []float32{1, 0, 0, 0, 0, 0, 0, 0}
	targets := [][]float32{
		{1, 0, 0, 0, 0, 0, 0, 0}, // identical
		{0, 1, 0, 0, 0, 0, 0, 0}, // orthogonal
		{-1, 0, 0, 0, 0, 0, 0, 0}, // opposite
	}
	similarities := make([]float32, len(targets))

	BatchCosineSimilarity(query, targets, similarities)

	expectedSims := []float32{1.0, 0.0, -1.0}
	for i, expected := range expectedSims {
		if !almostEqual(similarities[i], expected, epsilon) {
			t.Errorf("at index %d: expected %f, got %f", i, expected, similarities[i])
		}
	}
}

func TestBatchCosineSimilarity_ZeroQuery(t *testing.T) {
	query := []float32{0, 0, 0, 0}
	targets := [][]float32{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}
	similarities := make([]float32, len(targets))

	BatchCosineSimilarity(query, targets, similarities)

	for i, sim := range similarities {
		if sim != 0 {
			t.Errorf("at index %d: expected 0 for zero query, got %f", i, sim)
		}
	}
}

func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{
			name:     "same point",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "3-4-5 triangle",
			a:        []float32{0, 0},
			b:        []float32{3, 4},
			expected: 5.0,
		},
		{
			name:     "unit distance",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EuclideanDistance(tt.a, tt.b)
			if !almostEqual(result, tt.expected, 0.01) {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestVectorPool(t *testing.T) {
	dims := 768
	pool := NewVectorPool(dims)

	// Get a vector
	v := pool.Get()
	if len(v) != dims {
		t.Errorf("expected vector length %d, got %d", dims, len(v))
	}

	// Modify it
	v[0] = 1.0
	v[100] = 2.0

	// Return it
	pool.Put(v)

	// Get another - should be zeroed
	v2 := pool.Get()
	if v2[0] != 0 || v2[100] != 0 {
		t.Error("pooled vector should be zeroed")
	}
}

func TestVectorPool_WrongSize(t *testing.T) {
	pool := NewVectorPool(10)

	// Try to put a wrong-sized vector
	wrongSize := make([]float32, 20)
	pool.Put(wrongSize) // Should be silently ignored

	// Get should still work
	v := pool.Get()
	if len(v) != 10 {
		t.Errorf("expected length 10, got %d", len(v))
	}
}

// Benchmarks

func BenchmarkCosineSimilarity_Small(b *testing.B) {
	a := make([]float32, 64)
	vec := make([]float32, 64)
	for i := range a {
		a[i] = float32(i) * 0.1
		vec[i] = float32(i) * 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, vec)
	}
}

func BenchmarkCosineSimilarity_Medium(b *testing.B) {
	a := make([]float32, 384)
	vec := make([]float32, 384)
	for i := range a {
		a[i] = float32(i) * 0.1
		vec[i] = float32(i) * 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, vec)
	}
}

func BenchmarkCosineSimilarity_Large(b *testing.B) {
	a := make([]float32, 768)
	vec := make([]float32, 768)
	for i := range a {
		a[i] = float32(i) * 0.1
		vec[i] = float32(i) * 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, vec)
	}
}

func BenchmarkBatchCosineSimilarity_100(b *testing.B) {
	dims := 768
	query := make([]float32, dims)
	targets := make([][]float32, 100)
	for i := range targets {
		targets[i] = make([]float32, dims)
		for j := range targets[i] {
			targets[i][j] = float32(j) * 0.1
		}
	}
	similarities := make([]float32, len(targets))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchCosineSimilarity(query, targets, similarities)
	}
}

func BenchmarkBatchCosineSimilarity_1000(b *testing.B) {
	dims := 768
	query := make([]float32, dims)
	targets := make([][]float32, 1000)
	for i := range targets {
		targets[i] = make([]float32, dims)
		for j := range targets[i] {
			targets[i][j] = float32(j) * 0.1
		}
	}
	similarities := make([]float32, len(targets))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchCosineSimilarity(query, targets, similarities)
	}
}

func BenchmarkL2Norm(b *testing.B) {
	v := make([]float32, 768)
	for i := range v {
		v[i] = float32(i) * 0.1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		L2Norm(v)
	}
}

func BenchmarkNormalize(b *testing.B) {
	v := make([]float32, 768)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for j := range v {
			v[j] = float32(j) * 0.1
		}
		b.StartTimer()
		Normalize(v)
	}
}

func BenchmarkVectorPool(b *testing.B) {
	pool := NewVectorPool(768)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := pool.Get()
		pool.Put(v)
	}
}
