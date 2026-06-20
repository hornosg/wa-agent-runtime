// Package knowledge — RAG (E05): embeddings, store pgvector y answerer.
package knowledge

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
)

const Dim = 1024 // voyage-3.5 (D-02)

// Embeddings — puerto: texto → vector. Adaptadores: Voyage (real) / Stub (dev/CI).
type Embeddings interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// StubEmbeddings — embedding léxico determinístico (bag-of-tokens hasheado, L2-normalizado).
// No es semántico; sirve para probar el pipeline sin API key. El real es Voyage.
type StubEmbeddings struct{}

func NewStubEmbeddings() *StubEmbeddings { return &StubEmbeddings{} }

func (StubEmbeddings) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = lexicalVec(t)
	}
	return out, nil
}

func lexicalVec(text string) []float32 {
	v := make([]float32, Dim)
	for _, tok := range strings.Fields(strings.ToLower(text)) {
		tok = strings.Trim(tok, ".,;:¿?¡!()\"'")
		if tok == "" {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(tok))
		v[h.Sum32()%Dim] += 1
	}
	// L2-normalizar para que la distancia coseno sea estable.
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm > 0 {
		n := float32(math.Sqrt(norm))
		for j := range v {
			v[j] /= n
		}
	}
	return v
}
