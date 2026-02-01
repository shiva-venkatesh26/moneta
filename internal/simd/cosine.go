// Package simd provides SIMD-accelerated vector operations
package simd

import (
	"sync"
	"unsafe"
)

// VectorPool pools float32 slices to avoid allocations in hot paths
type VectorPool struct {
	pool sync.Pool
	dims int
}

// NewVectorPool creates a pool for vectors of the specified dimensions
func NewVectorPool(dims int) *VectorPool {
	return &VectorPool{
		dims: dims,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]float32, dims)
			},
		},
	}
}

// Get retrieves a vector from the pool
func (p *VectorPool) Get() []float32 {
	return p.pool.Get().([]float32)
}

// Put returns a vector to the pool
func (p *VectorPool) Put(v []float32) {
	if len(v) != p.dims {
		return // Don't pool wrong-sized vectors
	}
	// Clear before returning to pool
	for i := range v {
		v[i] = 0
	}
	p.pool.Put(v)
}

// CosineSimilarity computes cosine similarity between two vectors.
// Returns a value between -1 and 1, where 1 means identical direction.
// Uses loop unrolling for compiler auto-vectorization (AVX2/NEON).
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float32
	n := len(a)

	// Process 8 elements at a time for SIMD optimization
	// The compiler will auto-vectorize this pattern
	limit := n - (n % 8)

	for i := 0; i < limit; i += 8 {
		// Dot product accumulation
		dotProduct += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]

		// Norm A accumulation
		normA += a[i]*a[i] + a[i+1]*a[i+1] + a[i+2]*a[i+2] + a[i+3]*a[i+3] +
			a[i+4]*a[i+4] + a[i+5]*a[i+5] + a[i+6]*a[i+6] + a[i+7]*a[i+7]

		// Norm B accumulation
		normB += b[i]*b[i] + b[i+1]*b[i+1] + b[i+2]*b[i+2] + b[i+3]*b[i+3] +
			b[i+4]*b[i+4] + b[i+5]*b[i+5] + b[i+6]*b[i+6] + b[i+7]*b[i+7]
	}

	// Handle remaining elements
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

// DotProduct computes the dot product of two vectors
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var sum float32
	n := len(a)
	limit := n - (n % 8)

	for i := 0; i < limit; i += 8 {
		sum += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]
	}

	for i := limit; i < n; i++ {
		sum += a[i] * b[i]
	}

	return sum
}

// L2Norm computes the L2 (Euclidean) norm of a vector
func L2Norm(v []float32) float32 {
	var sum float32
	n := len(v)
	limit := n - (n % 8)

	for i := 0; i < limit; i += 8 {
		sum += v[i]*v[i] + v[i+1]*v[i+1] + v[i+2]*v[i+2] + v[i+3]*v[i+3] +
			v[i+4]*v[i+4] + v[i+5]*v[i+5] + v[i+6]*v[i+6] + v[i+7]*v[i+7]
	}

	for i := limit; i < n; i++ {
		sum += v[i] * v[i]
	}

	return sqrt32(sum)
}

// Normalize normalizes a vector in-place to unit length
func Normalize(v []float32) {
	norm := L2Norm(v)
	if norm == 0 {
		return
	}
	invNorm := 1.0 / norm
	for i := range v {
		v[i] *= invNorm
	}
}

// BatchCosineSimilarity computes similarities between one query and many targets
// Results are written to the similarities slice (must be pre-allocated)
func BatchCosineSimilarity(query []float32, targets [][]float32, similarities []float32) {
	// Pre-compute query norm once
	var queryNorm float32
	for _, v := range query {
		queryNorm += v * v
	}
	queryNorm = sqrt32(queryNorm)

	if queryNorm == 0 {
		for i := range similarities {
			similarities[i] = 0
		}
		return
	}

	invQueryNorm := 1.0 / queryNorm

	for i, target := range targets {
		var dot, targetNorm float32

		n := len(target)
		limit := n - (n % 8)

		for j := 0; j < limit; j += 8 {
			dot += query[j]*target[j] + query[j+1]*target[j+1] +
				query[j+2]*target[j+2] + query[j+3]*target[j+3] +
				query[j+4]*target[j+4] + query[j+5]*target[j+5] +
				query[j+6]*target[j+6] + query[j+7]*target[j+7]

			targetNorm += target[j]*target[j] + target[j+1]*target[j+1] +
				target[j+2]*target[j+2] + target[j+3]*target[j+3] +
				target[j+4]*target[j+4] + target[j+5]*target[j+5] +
				target[j+6]*target[j+6] + target[j+7]*target[j+7]
		}

		for j := limit; j < n; j++ {
			dot += query[j] * target[j]
			targetNorm += target[j] * target[j]
		}

		if targetNorm == 0 {
			similarities[i] = 0
		} else {
			similarities[i] = dot * invQueryNorm / sqrt32(targetNorm)
		}
	}
}

// sqrt32 is a fast float32 square root using the Quake III inverse sqrt trick
// with additional Newton-Raphson iterations for precision
func sqrt32(x float32) float32 {
	if x <= 0 {
		return 0
	}

	// Fast inverse square root (Quake III algorithm)
	i := *(*uint32)(unsafe.Pointer(&x))
	i = 0x5f3759df - (i >> 1) // Magic number for initial guess
	y := *(*float32)(unsafe.Pointer(&i))

	// Two Newton-Raphson iterations for precision
	y = y * (1.5 - (0.5 * x * y * y))
	y = y * (1.5 - (0.5 * x * y * y))

	// sqrt(x) = x * (1/sqrt(x))
	return x * y
}

// EuclideanDistance computes the Euclidean distance between two vectors
func EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var sum float32
	n := len(a)
	limit := n - (n % 8)

	for i := 0; i < limit; i += 8 {
		d0, d1 := a[i]-b[i], a[i+1]-b[i+1]
		d2, d3 := a[i+2]-b[i+2], a[i+3]-b[i+3]
		d4, d5 := a[i+4]-b[i+4], a[i+5]-b[i+5]
		d6, d7 := a[i+6]-b[i+6], a[i+7]-b[i+7]

		sum += d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5 + d6*d6 + d7*d7
	}

	for i := limit; i < n; i++ {
		d := a[i] - b[i]
		sum += d * d
	}

	return sqrt32(sum)
}
