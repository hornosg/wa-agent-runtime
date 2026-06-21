// agent-runtime (SVC-02) — worker que consume `inbound_message` de River, clasifica
// la intención (L1), responde por rama y emite la respuesta. Hexagonal (P-04),
// canonical logs (P-20), Go (ADR-0001). Contrato y puertos: ADR-0003. LLM: ADR-0002.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hornosg/wa-agent-runtime/src/agent"
	"github.com/hornosg/wa-agent-runtime/src/config"
	"github.com/hornosg/wa-agent-runtime/src/contract"
	"github.com/hornosg/wa-agent-runtime/src/knowledge"
	"github.com/hornosg/wa-agent-runtime/src/logging"
	"github.com/hornosg/wa-agent-runtime/src/scheduling"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

const serviceName = "agent-runtime"

func main() {
	cfg := config.Load()
	log := logging.New(serviceName)
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		log.Error("startup.db_pool_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	defer pool.Close()

	driver := riverpgxv5.New(pool)

	// Migraciones de River (idempotente; por si el runtime arranca primero).
	if migrator, err := rivermigrate.New(driver, nil); err == nil {
		if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
			log.Error("startup.river_migrate_failed", map[string]any{"error": err.Error()})
			os.Exit(1)
		}
	}

	// Selección del clasificador (LLM): anthropic (real) o stub (sin key).
	var classifier agent.IntentClassifier
	if cfg.LLMDriver == "stub" || cfg.AnthropicKey == "" {
		classifier = agent.NewStubClassifier()
		log.Warn("startup.llm_driver", map[string]any{"driver": "stub", "reason": "LLM_DRIVER=stub o ANTHROPIC_API_KEY vacío"})
	} else {
		classifier = agent.NewAnthropicClassifier(cfg.AnthropicKey)
		log.Info("startup.llm_driver", map[string]any{"driver": "anthropic", "model": "claude-haiku-4-5"})
	}

	// RAG (E05): embeddings (Voyage/stub) → retriever pgvector → answerer (Sonnet/stub).
	var emb knowledge.Embeddings
	if cfg.EmbeddingsDriver == "stub" || cfg.VoyageKey == "" {
		emb = knowledge.NewStubEmbeddings()
		log.Warn("startup.embeddings_driver", map[string]any{"driver": "stub", "reason": "EMBEDDINGS_DRIVER=stub o VOYAGE_API_KEY vacío"})
	} else {
		emb = knowledge.NewVoyageEmbeddings(cfg.VoyageKey)
		log.Info("startup.embeddings_driver", map[string]any{"driver": "voyage", "model": "voyage-3.5"})
	}
	retriever := knowledge.NewPgKnowledge(pool, emb, cfg.KnowledgeMaxDist)

	var answerer agent.FAQAnswerer
	if cfg.LLMDriver == "stub" || cfg.AnthropicKey == "" {
		answerer = knowledge.NewStubAnswerer()
	} else {
		answerer = knowledge.NewAnthropicAnswerer(cfg.AnthropicKey)
	}

	// Cliente insert-only para encolar outbound (rompe el ciclo worker→rt→outbound).
	// Registra el kind "outbound_message" (no-op) sólo para validar Insert; sin Queues no trabaja.
	insertWorkers := river.NewWorkers()
	river.AddWorker(insertWorkers, agent.NewNoopOutboundWorker())
	insertClient, err := river.NewClient(driver, &river.Config{Workers: insertWorkers})
	if err != nil {
		log.Error("startup.river_insert_client_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}

	// Outbound: encola en la cola "outbound" (gateway → Kapso) o loguea (dev).
	var outbound agent.Outbound
	if cfg.OutboundDriver == "log" {
		outbound = agent.NewLogOutbound(log)
		log.Warn("startup.outbound_driver", map[string]any{"driver": "log"})
	} else {
		outbound = agent.NewRiverOutbound(insertClient)
	}

	rt := agent.New(
		agent.NewPgTenants(pool, log),
		classifier,
		agent.NewGuadaReplier(retriever, answerer, scheduling.NewPgScheduler(pool)),
		outbound,
		log,
	)

	// Cliente consumidor de la cola "inbound".
	workers := river.NewWorkers()
	river.AddWorker(workers, agent.NewInboundWorker(rt))
	client, err := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{contract.QueueInbound: {MaxWorkers: cfg.MaxWorkers}},
		Workers: workers,
	})
	if err != nil {
		log.Error("startup.river_client_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}

	if err := client.Start(ctx); err != nil {
		log.Error("startup.river_start_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	log.Info("startup.consuming", map[string]any{"queue": contract.QueueInbound, "max_workers": cfg.MaxWorkers})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutdown.start", nil)
	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_ = client.Stop(shutdownCtx)
	log.Info("shutdown.done", nil)
}
