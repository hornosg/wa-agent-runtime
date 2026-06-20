package knowledge

import (
	"context"
	"strconv"
	"strings"

	"github.com/hornosg/wa-agent-runtime/src/agent"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgKnowledge — store pgvector. Implementa agent.FAQRetriever y expone Upsert
// para la ingestión. Distancia coseno (operador <=>), umbral configurable (P-05).
type PgKnowledge struct {
	pool        *pgxpool.Pool
	emb         Embeddings
	maxDistance float64
	topK        int
}

func NewPgKnowledge(pool *pgxpool.Pool, emb Embeddings, maxDistance float64) *PgKnowledge {
	return &PgKnowledge{pool: pool, emb: emb, maxDistance: maxDistance, topK: 4}
}

// Retrieve recupera los chunks más cercanos del tenant que estén bajo el umbral.
func (k *PgKnowledge) Retrieve(ctx context.Context, tenantSlug, query string) ([]agent.FAQChunk, error) {
	vecs, err := k.emb.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	qv := vectorLiteral(vecs[0])

	rows, err := k.pool.Query(ctx, `
		SELECT content, embedding <=> $2::vector AS distance
		FROM knowledge.faq_chunk
		WHERE tenant_slug = $1
		ORDER BY embedding <=> $2::vector
		LIMIT $3`, tenantSlug, qv, k.topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []agent.FAQChunk
	for rows.Next() {
		var c agent.FAQChunk
		if err := rows.Scan(&c.Content, &c.Distance); err != nil {
			return nil, err
		}
		if c.Distance <= k.maxDistance { // umbral: sin esto → handoff (P-05)
			chunks = append(chunks, c)
		}
	}
	return chunks, rows.Err()
}

// Upsert inserta chunks de FAQ del tenant (ingestión, E05).
func (k *PgKnowledge) Upsert(ctx context.Context, tenantSlug string, contents []string) (int, error) {
	vecs, err := k.emb.Embed(ctx, contents)
	if err != nil {
		return 0, err
	}
	n := 0
	for i, content := range contents {
		if _, err := k.pool.Exec(ctx,
			`INSERT INTO knowledge.faq_chunk (tenant_slug, content, embedding) VALUES ($1, $2, $3::vector)`,
			tenantSlug, content, vectorLiteral(vecs[i]),
		); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// vectorLiteral formatea un []float32 como literal pgvector "[v1,v2,...]".
func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
