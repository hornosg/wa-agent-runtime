package agent

import "context"

// FAQChunk — fragmento recuperado de la KnowledgeBase del tenant (G-07).
type FAQChunk struct {
	Content  string
	Distance float64 // distancia coseno (menor = más relevante)
}

// FAQRetriever — puerto: recupera chunks relevantes sobre umbral. Vacío = sin evidencia.
type FAQRetriever interface {
	Retrieve(ctx context.Context, tenantSlug, query string) ([]FAQChunk, error)
}

// FAQAnswerer — puerto: compone la respuesta GROUNDED solo en los chunks (P-05).
type FAQAnswerer interface {
	Answer(ctx context.Context, query string, chunks []FAQChunk) (Reply, error)
}
