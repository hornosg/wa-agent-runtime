// ingest — carga FAQs de un tenant a la KnowledgeBase (knowledge.faq_chunk).
// Uso: ingest -tenant demo -file faqs.txt   (una FAQ por línea)
//
//	EMBEDDINGS_DRIVER=voyage VOYAGE_API_KEY=... para embeddings reales; stub por defecto si no hay key.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hornosg/wa-agent-runtime/src/config"
	"github.com/hornosg/wa-agent-runtime/src/knowledge"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	tenant := flag.String("tenant", "", "tenant slug")
	file := flag.String("file", "", "archivo de FAQs (una por línea)")
	flag.Parse()
	if *tenant == "" || *file == "" {
		fmt.Fprintln(os.Stderr, "uso: ingest -tenant <slug> -file <faqs.txt>")
		os.Exit(2)
	}

	contents, err := readLines(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error leyendo archivo:", err)
		os.Exit(1)
	}

	cfg := config.Load()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error pool:", err)
		os.Exit(1)
	}
	defer pool.Close()

	var emb knowledge.Embeddings
	if cfg.EmbeddingsDriver == "stub" || cfg.VoyageKey == "" {
		emb = knowledge.NewStubEmbeddings()
		fmt.Fprintln(os.Stderr, "[ingest] embeddings: stub")
	} else {
		emb = knowledge.NewVoyageEmbeddings(cfg.VoyageKey)
		fmt.Fprintln(os.Stderr, "[ingest] embeddings: voyage-3.5")
	}

	store := knowledge.NewPgKnowledge(pool, emb, cfg.KnowledgeMaxDist)
	n, err := store.Upsert(ctx, *tenant, contents)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error upsert:", err)
		os.Exit(1)
	}
	fmt.Printf("ingestados %d chunks para tenant %q\n", n, *tenant)
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			out = append(out, line)
		}
	}
	return out, sc.Err()
}
